package lifecycle_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/standards-lab/go-core/lifecycle"
)

var _ lifecycle.ReadinessChecker = (*lifecycle.Coordinator)(nil)

// failsafe bounds every wait for an event that should occur, so a broken
// coordinator fails the test instead of hanging it.
const failsafe = 2 * time.Second

// recvOrFail receives from ch, or fails the test if nothing arrives within the
// failsafe window.
func recvOrFail[T any](t *testing.T, ch <-chan T, what string) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(failsafe):
		t.Fatalf("timed out waiting for %s", what)
		var zero T
		return zero
	}
}

// run starts Run on its own goroutine and returns the channel its result
// lands on.
func run(ctx context.Context, lc *lifecycle.Coordinator, drainTimeout time.Duration) <-chan error {
	done := make(chan error, 1)
	go func() { done <- lc.Run(ctx, drainTimeout) }()
	return done
}

// cancelled returns a context that is already cancelled, so Run drains
// immediately after startup without blocking.
func cancelled() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestRun_StartupHooksRunConcurrently(t *testing.T) {
	lc := lifecycle.New()

	const n = 5
	arrived := make(chan struct{}, n)
	release := make(chan struct{})
	var count atomic.Int64

	for range n {
		lc.OnStartup(func(context.Context) error {
			count.Add(1)
			arrived <- struct{}{}
			<-release
			return nil
		})
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := run(ctx, lc, failsafe)

	// Every hook must reach its arrival send before any is released, which only
	// holds if they run simultaneously rather than one after another.
	for range n {
		recvOrFail(t, arrived, "startup hook arrival")
	}
	close(release)
	cancel()

	if err := recvOrFail(t, done, "Run to return"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := count.Load(); got != n {
		t.Fatalf("ran %d startup hooks, want %d", got, n)
	}
}

func TestRun_ReadyLifecycle(t *testing.T) {
	lc := lifecycle.New()

	started := make(chan struct{})
	release := make(chan struct{})
	lc.OnStartup(func(context.Context) error {
		close(started)
		<-release
		return nil
	})

	// OnReady observes readiness from inside the hook: the flip must precede
	// the ready hooks.
	readyInHook := make(chan bool, 1)
	lc.OnReady(func() { readyInHook <- lc.Ready() })

	if lc.Ready() {
		t.Fatal("Ready() is true before Run")
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := run(ctx, lc, failsafe)

	recvOrFail(t, started, "startup hook to start")
	if lc.Ready() {
		t.Fatal("Ready() is true while a startup hook is still running")
	}

	close(release)
	if !recvOrFail(t, readyInHook, "OnReady hook") {
		t.Fatal("Ready() is false inside an OnReady hook")
	}
	if !lc.Ready() {
		t.Fatal("Ready() is false while running")
	}

	cancel()
	if err := recvOrFail(t, done, "Run to return"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if lc.Ready() {
		t.Fatal("Ready() is true after Run returned")
	}
}

func TestRun_StartupFailureNeverReady(t *testing.T) {
	lc := lifecycle.New()

	sentinel := errors.New("db connect failed")
	lc.OnStartup(func(context.Context) error { return sentinel })

	var wasReady atomic.Bool
	lc.OnReady(func() { wasReady.Store(true) })

	var drained atomic.Bool
	lc.OnShutdown(func(context.Context) error {
		drained.Store(true)
		return nil
	})

	err := lc.Run(context.Background(), failsafe)
	if err == nil {
		t.Fatal("Run returned nil for a failing startup hook")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("error = %v, want errors.Is(err, sentinel)", err)
	}
	if !strings.Contains(err.Error(), "startup:") {
		t.Errorf("error = %v, want the startup phase wrap", err)
	}
	if wasReady.Load() {
		t.Error("OnReady hooks ran despite a startup failure")
	}
	if lc.Ready() {
		t.Error("Ready() is true after a startup failure")
	}
	if !drained.Load() {
		t.Error("shutdown hooks did not drain the partial start")
	}
}

func TestRun_JoinsAllStartupFailures(t *testing.T) {
	lc := lifecycle.New()

	first := errors.New("first subsystem failed")
	second := errors.New("second subsystem failed")
	lc.OnStartup(func(context.Context) error { return first })
	lc.OnStartup(func(context.Context) error { return second })

	err := lc.Run(context.Background(), failsafe)
	if !errors.Is(err, first) || !errors.Is(err, second) {
		t.Fatalf("error = %v, want both startup failures joined", err)
	}
}

func TestRun_CancelReturnsNilAndDrains(t *testing.T) {
	lc := lifecycle.New()

	// The startup hook captures the run context; the shutdown hook observes
	// both contexts at drain time. The chain is race-free: the startup
	// goroutine completes before Run launches the shutdown goroutine.
	var runCtx context.Context
	lc.OnStartup(func(ctx context.Context) error {
		runCtx = ctx
		return nil
	})

	type observation struct{ runErr, drainErr error }
	obs := make(chan observation, 1)
	lc.OnShutdown(func(drainCtx context.Context) error {
		obs <- observation{runErr: runCtx.Err(), drainErr: drainCtx.Err()}
		return nil
	})

	ready := make(chan struct{})
	lc.OnReady(func() { close(ready) })

	ctx, cancel := context.WithCancel(context.Background())
	done := run(ctx, lc, failsafe)

	recvOrFail(t, ready, "coordinator to become ready")
	cancel()

	if err := recvOrFail(t, done, "Run to return"); err != nil {
		t.Fatalf("Run after a clean cancel: %v", err)
	}

	o := recvOrFail(t, obs, "shutdown hook invocation")
	if o.runErr == nil {
		t.Error("run context was not cancelled when the shutdown hook ran")
	}
	if o.drainErr != nil {
		t.Errorf("drain context was already cancelled when the hook ran: %v", o.drainErr)
	}
}

func TestRun_MonitorFailureDrains(t *testing.T) {
	lc := lifecycle.New()

	errs := make(chan error, 1)
	lc.Monitor(errs)

	var drained atomic.Bool
	lc.OnShutdown(func(context.Context) error {
		drained.Store(true)
		return nil
	})

	ready := make(chan struct{})
	lc.OnReady(func() { close(ready) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := run(ctx, lc, failsafe)

	recvOrFail(t, ready, "coordinator to become ready")

	sentinel := errors.New("serve exploded")
	errs <- sentinel

	err := recvOrFail(t, done, "Run to return")
	if !errors.Is(err, sentinel) {
		t.Fatalf("error = %v, want errors.Is(err, sentinel)", err)
	}
	if !strings.Contains(err.Error(), "run:") {
		t.Errorf("error = %v, want the run phase wrap", err)
	}
	if !drained.Load() {
		t.Error("shutdown hooks did not run after a monitor failure")
	}
	if lc.Ready() {
		t.Error("Ready() is true after a monitor failure")
	}
}

func TestRun_MonitorIgnoresNilAndClose(t *testing.T) {
	lc := lifecycle.New()

	errs := make(chan error)
	lc.Monitor(errs)

	ready := make(chan struct{})
	lc.OnReady(func() { close(ready) })

	ctx, cancel := context.WithCancel(context.Background())
	done := run(ctx, lc, failsafe)

	recvOrFail(t, ready, "coordinator to become ready")

	// A nil error and a closing channel are both the quiet end of a monitored
	// source, not failures.
	errs <- nil
	close(errs)

	time.Sleep(20 * time.Millisecond)
	if !lc.Ready() {
		t.Fatal("coordinator left running after a nil monitor error or close")
	}

	cancel()
	if err := recvOrFail(t, done, "Run to return"); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestRun_PreCancelledContextDrainsWithoutReady(t *testing.T) {
	lc := lifecycle.New()

	var wasReady atomic.Bool
	lc.OnReady(func() { wasReady.Store(true) })

	var drained atomic.Bool
	lc.OnShutdown(func(context.Context) error {
		drained.Store(true)
		return nil
	})

	if err := lc.Run(cancelled(), failsafe); err != nil {
		t.Fatalf("Run with a pre-cancelled context: %v", err)
	}
	if wasReady.Load() {
		t.Error("OnReady hooks ran under a pre-cancelled context")
	}
	if !drained.Load() {
		t.Error("shutdown hooks did not run under a pre-cancelled context")
	}
}

func TestRun_DrainTimeout(t *testing.T) {
	lc := lifecycle.New()

	// The hook outlives the timeout; releasing it only at cleanup keeps the
	// hooks-done path closed, so the drain must end via the deadline.
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	lc.OnShutdown(func(context.Context) error {
		<-release
		return nil
	})

	err := lc.Run(cancelled(), 20*time.Millisecond)
	if err == nil {
		t.Fatal("Run returned nil for a shutdown hook that outlived the timeout")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error = %v, want errors.Is(err, context.DeadlineExceeded)", err)
	}
	if !strings.Contains(err.Error(), "drain timeout after") {
		t.Errorf("error = %v, want the drain timeout description", err)
	}
	if lc.Ready() {
		t.Error("Ready() is true while a shutdown hook straggles past the timeout")
	}
}

func TestRun_JoinsShutdownHookErrors(t *testing.T) {
	lc := lifecycle.New()

	sentinel := errors.New("close failed")
	lc.OnShutdown(func(context.Context) error { return sentinel })

	err := lc.Run(cancelled(), failsafe)
	if !errors.Is(err, sentinel) {
		t.Fatalf("error = %v, want errors.Is(err, sentinel)", err)
	}
	if !strings.Contains(err.Error(), "shutdown:") {
		t.Errorf("error = %v, want the shutdown phase wrap", err)
	}
}

func TestRegistration_PanicsAfterRun(t *testing.T) {
	lc := lifecycle.New()
	if err := lc.Run(cancelled(), failsafe); err != nil {
		t.Fatalf("Run with no hooks: %v", err)
	}

	for _, tc := range []struct {
		name string
		call func()
	}{
		{"OnStartup", func() { lc.OnStartup(func(context.Context) error { return nil }) }},
		{"OnShutdown", func() { lc.OnShutdown(func(context.Context) error { return nil }) }},
		{"OnReady", func() { lc.OnReady(func() {}) }},
		{"Monitor", func() { lc.Monitor(make(chan error)) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if r == nil {
					t.Fatalf("%s after Run did not panic", tc.name)
				}
				if want := "lifecycle: " + tc.name + " after Run"; r != want {
					t.Fatalf("panic = %v, want %q", r, want)
				}
			}()
			tc.call()
		})
	}
}

func TestRun_CalledTwicePanics(t *testing.T) {
	lc := lifecycle.New()
	if err := lc.Run(cancelled(), failsafe); err != nil {
		t.Fatalf("first Run: %v", err)
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("a second Run did not panic")
		}
		if want := "lifecycle: Run called twice"; r != want {
			t.Fatalf("panic = %v, want %q", r, want)
		}
	}()
	_ = lc.Run(cancelled(), failsafe)
}

// recorder collects ordered step labels from hooks and services across
// goroutines.
type recorder struct {
	mu    sync.Mutex
	steps []string
}

func (r *recorder) add(step string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.steps = append(r.steps, step)
}

func (r *recorder) list() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.steps)
}

// staticChecker reports a fixed readiness.
type staticChecker bool

func (s staticChecker) Ready() bool { return bool(s) }

func TestRun_StagesStartInOrderRootLast(t *testing.T) {
	lc := lifecycle.New()
	rec := &recorder{}

	add := func(name string, stage int) {
		lc.Add(lifecycle.Service{
			Name:  name,
			Stage: stage,
			Start: func(context.Context) error {
				rec.add(name)
				return nil
			},
		})
	}
	// Added out of start order: sorting the stages, not Add order, decides.
	add("server", lifecycle.StageRoot)
	add("consumer", 1)
	add("pool", 0)

	ready := make(chan struct{})
	lc.OnReady(func() { close(ready) })

	ctx, cancel := context.WithCancel(context.Background())
	done := run(ctx, lc, failsafe)

	recvOrFail(t, ready, "coordinator to become ready")
	cancel()
	if err := recvOrFail(t, done, "Run to return"); err != nil {
		t.Fatalf("Run: %v", err)
	}

	want := []string{"pool", "consumer", "server"}
	if got := rec.list(); !slices.Equal(got, want) {
		t.Fatalf("start order = %v, want %v", got, want)
	}
}

func TestRun_ServicesWithinAStageStartConcurrently(t *testing.T) {
	lc := lifecycle.New()

	const n = 3
	arrived := make(chan struct{}, n)
	release := make(chan struct{})
	for i := range n {
		lc.Add(lifecycle.Service{
			Name:  fmt.Sprintf("svc-%d", i),
			Stage: 0,
			Start: func(context.Context) error {
				arrived <- struct{}{}
				<-release
				return nil
			},
		})
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := run(ctx, lc, failsafe)

	// Every member must reach its arrival send before any is released, which
	// only holds if the stage starts them simultaneously.
	for range n {
		recvOrFail(t, arrived, "stage member arrival")
	}
	close(release)
	cancel()

	if err := recvOrFail(t, done, "Run to return"); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestRun_StageBarrierBlocksTheNextStage(t *testing.T) {
	lc := lifecycle.New()

	blocked := make(chan struct{})
	release := make(chan struct{})
	lc.Add(lifecycle.Service{
		Name:  "first",
		Stage: 0,
		Start: func(context.Context) error {
			close(blocked)
			<-release
			return nil
		},
	})

	var second atomic.Bool
	lc.Add(lifecycle.Service{
		Name:  "second",
		Stage: 1,
		Start: func(context.Context) error {
			second.Store(true)
			return nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := run(ctx, lc, failsafe)

	recvOrFail(t, blocked, "stage 0 to start")
	time.Sleep(20 * time.Millisecond)
	if second.Load() {
		t.Fatal("stage 1 started while stage 0 was still starting")
	}

	close(release)
	cancel()
	if err := recvOrFail(t, done, "Run to return"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !second.Load() {
		t.Fatal("stage 1 never started")
	}
}

func TestRun_DrainReversesStagesRootFirst(t *testing.T) {
	lc := lifecycle.New()
	rec := &recorder{}

	lc.Add(lifecycle.Service{
		Name:     "pool",
		Stage:    0,
		Start:    func(context.Context) error { return nil },
		Shutdown: func(context.Context) error { rec.add("pool"); return nil },
	})
	// A Start-less service still counts as started once its stage runs, so
	// its Shutdown participates in the drain.
	lc.Add(lifecycle.Service{
		Name:     "flusher",
		Stage:    1,
		Shutdown: func(context.Context) error { rec.add("flusher"); return nil },
	})
	lc.Add(lifecycle.Service{
		Name:     "server",
		Stage:    lifecycle.StageRoot,
		Start:    func(context.Context) error { return nil },
		Shutdown: func(context.Context) error { rec.add("server"); return nil },
	})

	if err := lc.Run(cancelled(), failsafe); err != nil {
		t.Fatalf("Run: %v", err)
	}

	want := []string{"server", "flusher", "pool"}
	if got := rec.list(); !slices.Equal(got, want) {
		t.Fatalf("drain order = %v, want %v", got, want)
	}
}

func TestRun_JoinsServiceShutdownErrorsWithNames(t *testing.T) {
	lc := lifecycle.New()

	poolErr := errors.New("pool close failed")
	busErr := errors.New("bus close failed")
	lc.Add(lifecycle.Service{
		Name:     "pool",
		Stage:    0,
		Shutdown: func(context.Context) error { return poolErr },
	})
	lc.Add(lifecycle.Service{
		Name:     "bus",
		Stage:    1,
		Shutdown: func(context.Context) error { return busErr },
	})

	err := lc.Run(cancelled(), failsafe)
	if !errors.Is(err, poolErr) || !errors.Is(err, busErr) {
		t.Fatalf("error = %v, want both shutdown failures joined", err)
	}
	if !strings.Contains(err.Error(), "shutdown:") {
		t.Errorf("error = %v, want the shutdown phase wrap", err)
	}
	if !strings.Contains(err.Error(), "pool:") || !strings.Contains(err.Error(), "bus:") {
		t.Errorf("error = %v, want each failure labeled with its service name", err)
	}
}

func TestRun_StageFailureSkipsLaterStagesAndUnwindsStartedOnly(t *testing.T) {
	lc := lifecycle.New()
	rec := &recorder{}
	sentinel := errors.New("bus connect failed")

	stopRecording := func(name string) func(context.Context) error {
		return func(context.Context) error {
			rec.add(name)
			return nil
		}
	}

	lc.Add(lifecycle.Service{
		Name:     "pool",
		Stage:    0,
		Start:    func(context.Context) error { return nil },
		Shutdown: stopRecording("pool"),
	})
	lc.Add(lifecycle.Service{
		Name:     "bus",
		Stage:    1,
		Start:    func(context.Context) error { return sentinel },
		Shutdown: stopRecording("bus"),
	})
	lc.Add(lifecycle.Service{
		Name:     "cache",
		Stage:    1,
		Start:    func(context.Context) error { return nil },
		Shutdown: stopRecording("cache"),
	})

	var serverStarted atomic.Bool
	lc.Add(lifecycle.Service{
		Name:  "server",
		Stage: lifecycle.StageRoot,
		Start: func(context.Context) error {
			serverStarted.Store(true)
			return nil
		},
		Shutdown: stopRecording("server"),
	})

	err := lc.Run(context.Background(), failsafe)
	if !errors.Is(err, sentinel) {
		t.Fatalf("error = %v, want errors.Is(err, sentinel)", err)
	}
	if !strings.Contains(err.Error(), "startup:") || !strings.Contains(err.Error(), "bus:") {
		t.Errorf("error = %v, want the startup wrap naming the failed service", err)
	}
	if serverStarted.Load() {
		t.Error("a later stage started after an earlier stage failed")
	}

	// The failing stage finishes its concurrent starts, so cache is started
	// and unwinds; bus failed and does not; server never ran.
	want := []string{"cache", "pool"}
	if got := rec.list(); !slices.Equal(got, want) {
		t.Fatalf("unwind = %v, want %v (started services only, in reverse)", got, want)
	}
}

func TestRun_HooksBracketTheServiceStages(t *testing.T) {
	lc := lifecycle.New()
	rec := &recorder{}

	lc.OnStartup(func(context.Context) error {
		rec.add("hook-up")
		return nil
	})
	lc.OnShutdown(func(context.Context) error {
		rec.add("hook-down")
		return nil
	})
	lc.Add(lifecycle.Service{
		Name:     "svc",
		Stage:    0,
		Start:    func(context.Context) error { rec.add("svc-up"); return nil },
		Shutdown: func(context.Context) error { rec.add("svc-down"); return nil },
	})

	if err := lc.Run(cancelled(), failsafe); err != nil {
		t.Fatalf("Run: %v", err)
	}

	want := []string{"hook-up", "svc-up", "svc-down", "hook-down"}
	if got := rec.list(); !slices.Equal(got, want) {
		t.Fatalf("sequence = %v, want %v", got, want)
	}
}

func TestChecks_CollectsInStartOrderSkippingNil(t *testing.T) {
	lc := lifecycle.New()

	lc.Add(lifecycle.Service{
		Name:  "server",
		Stage: lifecycle.StageRoot,
		Check: staticChecker(true),
	})
	lc.Add(lifecycle.Service{
		Name:  "pool",
		Stage: 0,
		Check: staticChecker(true),
	})
	lc.Add(lifecycle.Service{
		Name:     "flusher",
		Stage:    0,
		Shutdown: func(context.Context) error { return nil },
	})
	lc.Add(lifecycle.Service{
		Name:  "bus",
		Stage: 1,
		Check: staticChecker(false),
	})

	checks := lc.Checks()
	var names []string
	for _, check := range checks {
		names = append(names, check.Name)
	}
	want := []string{"pool", "bus", "server"}
	if !slices.Equal(names, want) {
		t.Fatalf("checks = %v, want %v", names, want)
	}
	if !checks[0].Checker.Ready() || checks[1].Checker.Ready() {
		t.Error("checkers were not carried through")
	}
}

func TestAdd_Validation(t *testing.T) {
	valid := lifecycle.Service{
		Name:  "svc",
		Start: func(context.Context) error { return nil },
	}

	for _, tc := range []struct {
		name string
		want string
		add  func(lc *lifecycle.Coordinator)
	}{
		{
			name: "EmptyName",
			want: "lifecycle: Add: empty service name",
			add: func(lc *lifecycle.Coordinator) {
				svc := valid
				svc.Name = ""
				lc.Add(svc)
			},
		},
		{
			name: "NegativeStage",
			want: `lifecycle: Add: service "svc": negative stage -1`,
			add: func(lc *lifecycle.Coordinator) {
				svc := valid
				svc.Stage = -1
				lc.Add(svc)
			},
		},
		{
			name: "DeclaresNothing",
			want: `lifecycle: Add: service "svc" declares nothing`,
			add: func(lc *lifecycle.Coordinator) {
				lc.Add(lifecycle.Service{Name: "svc"})
			},
		},
		{
			name: "DuplicateName",
			want: `lifecycle: Add: duplicate service "svc"`,
			add: func(lc *lifecycle.Coordinator) {
				lc.Add(valid)
				lc.Add(valid)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lc := lifecycle.New()
			defer func() {
				r := recover()
				if r == nil {
					t.Fatal("Add did not panic")
				}
				if r != tc.want {
					t.Fatalf("panic = %v, want %q", r, tc.want)
				}
			}()
			tc.add(lc)
		})
	}

	t.Run("AfterRun", func(t *testing.T) {
		lc := lifecycle.New()
		if err := lc.Run(cancelled(), failsafe); err != nil {
			t.Fatalf("Run: %v", err)
		}
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("Add after Run did not panic")
			}
			if want := "lifecycle: Add after Run"; r != want {
				t.Fatalf("panic = %v, want %q", r, want)
			}
		}()
		lc.Add(valid)
	})
}
