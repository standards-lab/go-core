package lifecycle_test

import (
	"context"
	"errors"
	"strings"
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
