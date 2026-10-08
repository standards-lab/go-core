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

	"github.com/standards-lab/go-core/config"
	"github.com/standards-lab/go-core/graph"
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

// recorder collects events, in the order they happen, across goroutines.
type recorder struct {
	mu     sync.Mutex
	events []string
}

func (r *recorder) record(event string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
}

func (r *recorder) list() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.events)
}

// sameSet reports whether got holds exactly want, in any order.
func sameSet(got []string, want ...string) bool {
	return len(got) == len(want) &&
		slices.Equal(slices.Sorted(slices.Values(got)), slices.Sorted(slices.Values(want)))
}

// phased reports whether got is exactly phases, in order, each phase's
// events in any order among themselves: a layer starts and stops
// concurrently, and the layers one after another.
func phased(got []string, phases ...[]string) bool {
	for _, phase := range phases {
		if len(got) < len(phase) || !sameSet(got[:len(phase)], phase...) {
			return false
		}
		got = got[len(phase):]
	}
	return len(got) == 0
}

// fake is a Subsystem that records "start name" and "stop name" on r, when
// r is set, then runs its optional start and stop behaviour; nil behaviour
// succeeds.
type fake struct {
	name        string
	r           *recorder
	start, stop func(context.Context) error
}

func (f *fake) Start(ctx context.Context) error {
	if f.r != nil {
		f.r.record("start " + f.name)
	}
	if f.start != nil {
		return f.start(ctx)
	}
	return nil
}

func (f *fake) Shutdown(ctx context.Context) error {
	if f.r != nil {
		f.r.record("stop " + f.name)
	}
	if f.stop != nil {
		return f.stop(ctx)
	}
	return nil
}

// starter is a start-only participant: a Starter and not a Stopper.
type starter func(context.Context) error

func (s starter) Start(ctx context.Context) error { return s(ctx) }

// stopper is a stop-only participant: a Stopper and not a Starter.
type stopper func(context.Context) error

func (s stopper) Shutdown(ctx context.Context) error { return s(ctx) }

// node defines f on g under f's name, using deps, so f sits one layer
// above the highest of them.
func node(g *graph.Graph, f *fake, deps ...graph.Ref) *graph.Node[*fake] {
	return value(g, f.name, f, deps...)
}

// value defines v on g under name, using deps, so v sits one layer above
// the highest of them.
func value[T any](g *graph.Graph, name string, v T, deps ...graph.Ref) *graph.Node[T] {
	return g.Define(name, func(s *graph.Scope) (T, error) {
		for _, d := range deps {
			use(s, d)
		}
		return v, nil
	})
}

// use uses d through s, for each node type these tests define: a Use, unlike
// an After, brings d into the System.
func use(s *graph.Scope, d graph.Ref) {
	switch n := d.(type) {
	case *graph.Node[*fake]:
		s.Use(n)
	case *graph.Node[starter]:
		s.Use(n)
	case *graph.Node[stopper]:
		s.Use(n)
	default:
		panic(fmt.Sprintf("use: unexpected node type %T", d))
	}
}

// coordinator builds g from roots and returns a Coordinator for the System
// that shuts down within timeout, failing the test on a build error.
func coordinator(
	t *testing.T,
	timeout time.Duration,
	g *graph.Graph,
	roots ...graph.Ref,
) *lifecycle.Coordinator {
	t.Helper()
	sys, err := g.Build(roots...)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return lifecycle.New(sys, lifecycle.Config{ShutdownTimeout: config.Duration(timeout)})
}

// run starts Run on its own goroutine and returns the channel its result
// lands on.
func run(ctx context.Context, lc *lifecycle.Coordinator) <-chan error {
	done := make(chan error, 1)
	go func() { done <- lc.Run(ctx) }()
	return done
}

// runThrough runs lc until it becomes ready, then stops it: an OnReady hook
// cancels Run's context, so Run shuts down immediately after startup.
func runThrough(lc *lifecycle.Coordinator) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lc.OnReady(cancel)
	return lc.Run(ctx)
}

// cancelled returns a context that is already cancelled.
func cancelled() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

// noop is an Exec function that succeeds.
func noop(context.Context) error { return nil }

// mustPanic fails the test unless fn panics with exactly want.
func mustPanic(t *testing.T, want string, fn func()) {
	t.Helper()
	defer func() {
		t.Helper()
		if r := recover(); r != want {
			t.Errorf("panic = %v, want %q", r, want)
		}
	}()
	fn()
}

func TestNew_PanicsOnWiringMistakes(t *testing.T) {
	sys, err := graph.New().Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	mustPanic(t, "lifecycle: New: nil System", func() {
		lifecycle.New(nil, lifecycle.Config{ShutdownTimeout: config.Duration(time.Second)})
	})
	for _, timeout := range []time.Duration{0, -time.Second} {
		t.Run(timeout.String(), func(t *testing.T) {
			want := fmt.Sprintf(
				"lifecycle: New: shutdown timeout %v is not positive; finalize the Config",
				config.Duration(timeout),
			)
			mustPanic(t, want, func() {
				lifecycle.New(sys, lifecycle.Config{ShutdownTimeout: config.Duration(timeout)})
			})
		})
	}
}

func TestRun_LayerStartsConcurrently(t *testing.T) {
	const n = 5
	arrived := make(chan struct{}, n)
	release := make(chan struct{})
	var count atomic.Int64

	g := graph.New()
	var roots []graph.Ref
	for i := range n {
		roots = append(roots, value(g, fmt.Sprintf("svc-%d", i), starter(func(context.Context) error {
			count.Add(1)
			arrived <- struct{}{}
			<-release
			return nil
		})))
	}
	lc := coordinator(t, failsafe, g, roots...)

	ctx, cancel := context.WithCancel(context.Background())
	done := run(ctx, lc)

	// Every participant must reach its arrival send before any is released,
	// which only holds if the layer starts them simultaneously.
	for range n {
		recvOrFail(t, arrived, "layer member arrival")
	}
	close(release)
	cancel()

	if err := recvOrFail(t, done, "Run to return"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := count.Load(); got != n {
		t.Fatalf("started %d participants, want %d", got, n)
	}
}

func TestRun_ReadyLifecycle(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	g := graph.New()
	svc := value(g, "svc", starter(func(context.Context) error {
		close(started)
		<-release
		return nil
	}))
	lc := coordinator(t, failsafe, g, svc)

	// OnReady observes readiness from inside the hook: the flip must precede
	// the ready hooks.
	readyInHook := make(chan bool, 1)
	lc.OnReady(func() { readyInHook <- lc.Ready() })

	if lc.Ready() {
		t.Fatal("Ready() is true before Run")
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := run(ctx, lc)

	recvOrFail(t, started, "the participant's Start")
	if lc.Ready() {
		t.Fatal("Ready() is true while a Start is still running")
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
	sentinel := errors.New("db connect failed")
	var failedStopped, drained atomic.Bool
	g := graph.New()
	db := node(g, &fake{
		name:  "db",
		start: func(context.Context) error { return sentinel },
		stop:  func(context.Context) error { failedStopped.Store(true); return nil },
	})
	flusher := value(g, "flusher", stopper(func(context.Context) error {
		drained.Store(true)
		return nil
	}))
	lc := coordinator(t, failsafe, g, db, flusher)

	var wasReady atomic.Bool
	lc.OnReady(func() { wasReady.Store(true) })

	err := lc.Run(context.Background())
	if err == nil {
		t.Fatal("Run returned nil for a failing Start")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("error = %v, want errors.Is(err, sentinel)", err)
	}
	if !strings.Contains(err.Error(), "startup:") || !strings.Contains(err.Error(), "db:") {
		t.Errorf("error = %v, want the startup wrap naming the failed participant", err)
	}
	if wasReady.Load() {
		t.Error("OnReady hooks ran despite a startup failure")
	}
	if lc.Ready() {
		t.Error("Ready() is true after a startup failure")
	}
	if !drained.Load() {
		t.Error("the failing layer's Stopper was not shut down")
	}
	if !failedStopped.Load() {
		t.Error("the participant whose Start failed was not shut down")
	}
}

func TestRun_JoinsAllStartupFailures(t *testing.T) {
	first := errors.New("first subsystem failed")
	second := errors.New("second subsystem failed")
	g := graph.New()
	a := value(g, "a", starter(func(context.Context) error { return first }))
	b := value(g, "b", starter(func(context.Context) error { return second }))

	err := coordinator(t, failsafe, g, a, b).Run(context.Background())
	if !errors.Is(err, first) || !errors.Is(err, second) {
		t.Fatalf("error = %v, want both startup failures joined", err)
	}
}

func TestRun_CancelReturnsNilAndShutsDown(t *testing.T) {
	// Start captures the run context; Shutdown observes both contexts at
	// shutdown. The chain is race-free: the Start goroutine completes before
	// the shutdown goroutine launches.
	var runCtx context.Context
	type observation struct{ runErr, shutdownErr error }
	obs := make(chan observation, 1)
	g := graph.New()
	svc := node(g, &fake{
		name: "svc",
		start: func(ctx context.Context) error {
			runCtx = ctx
			return nil
		},
		stop: func(ctx context.Context) error {
			obs <- observation{runErr: runCtx.Err(), shutdownErr: ctx.Err()}
			return nil
		},
	})
	lc := coordinator(t, failsafe, g, svc)

	ready := make(chan struct{})
	lc.OnReady(func() { close(ready) })

	ctx, cancel := context.WithCancel(context.Background())
	done := run(ctx, lc)

	recvOrFail(t, ready, "coordinator to become ready")
	cancel()

	if err := recvOrFail(t, done, "Run to return"); err != nil {
		t.Fatalf("Run after a clean cancel: %v", err)
	}

	o := recvOrFail(t, obs, "Shutdown invocation")
	if o.runErr == nil {
		t.Error("run context was not cancelled when Shutdown ran")
	}
	if o.shutdownErr != nil {
		t.Errorf("shutdown context was already cancelled when Shutdown ran: %v", o.shutdownErr)
	}
}

func TestRun_MonitorFailureShutsDown(t *testing.T) {
	var drained atomic.Bool
	g := graph.New()
	flusher := value(g, "flusher", stopper(func(context.Context) error {
		drained.Store(true)
		return nil
	}))
	lc := coordinator(t, failsafe, g, flusher)

	errs := make(chan error, 1)
	lc.Monitor(errs)

	ready := make(chan struct{})
	lc.OnReady(func() { close(ready) })

	done := run(t.Context(), lc)

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
		t.Error("Stopper did not run after a monitor failure")
	}
	if lc.Ready() {
		t.Error("Ready() is true after a monitor failure")
	}
}

func TestRun_MonitorIgnoresNilAndClose(t *testing.T) {
	lc := coordinator(t, failsafe, graph.New())

	errs := make(chan error)
	lc.Monitor(errs)

	ready := make(chan struct{})
	lc.OnReady(func() { close(ready) })

	ctx, cancel := context.WithCancel(context.Background())
	done := run(ctx, lc)

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

// A context ended before Run is the clean stop before any layer begins:
// nothing starts, so nothing shuts down.
func TestRun_PreCancelledContextStartsNothing(t *testing.T) {
	var r recorder
	g := graph.New()
	a := node(g, &fake{name: "a", r: &r})
	lc := coordinator(t, failsafe, g, a)

	var wasReady atomic.Bool
	lc.OnReady(func() { wasReady.Store(true) })

	if err := lc.Run(cancelled()); err != nil {
		t.Fatalf("Run with a pre-cancelled context: %v", err)
	}
	if wasReady.Load() {
		t.Error("OnReady hooks ran under a pre-cancelled context")
	}
	if got := r.list(); len(got) != 0 {
		t.Errorf("events = %q, want nothing started or stopped", got)
	}
}

func TestRun_ShutdownTimeout(t *testing.T) {
	// Shutdown outlives the timeout; releasing it only at cleanup keeps the
	// layer-done path closed, so shutdown must end via the deadline.
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	g := graph.New()
	svc := value(g, "svc", stopper(func(context.Context) error {
		<-release
		return nil
	}))
	lc := coordinator(t, 20*time.Millisecond, g, svc)

	err := runThrough(lc)
	if err == nil {
		t.Fatal("Run returned nil for a Shutdown that outlived the timeout")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error = %v, want errors.Is(err, context.DeadlineExceeded)", err)
	}
	if !strings.Contains(err.Error(), "drain timeout after") {
		t.Errorf("error = %v, want the drain timeout description", err)
	}
	if lc.Ready() {
		t.Error("Ready() is true while a Shutdown straggles past the timeout")
	}
}

func TestRun_JoinsShutdownErrorsWithNames(t *testing.T) {
	poolErr := errors.New("pool close failed")
	busErr := errors.New("bus close failed")
	g := graph.New()
	pool := value(g, "pool", stopper(func(context.Context) error { return poolErr }))
	bus := value(g, "bus", stopper(func(context.Context) error { return busErr }), pool)

	err := runThrough(coordinator(t, failsafe, g, bus))
	if !errors.Is(err, poolErr) || !errors.Is(err, busErr) {
		t.Fatalf("error = %v, want both shutdown failures joined", err)
	}
	if !strings.Contains(err.Error(), "shutdown:") {
		t.Errorf("error = %v, want the shutdown phase wrap", err)
	}
	if !strings.Contains(err.Error(), "pool:") || !strings.Contains(err.Error(), "bus:") {
		t.Errorf("error = %v, want each failure labeled with its participant's name", err)
	}
}

func TestRegistration_PanicsAfterRun(t *testing.T) {
	lc := coordinator(t, failsafe, graph.New())
	if err := lc.Run(cancelled()); err != nil {
		t.Fatalf("Run with no participants: %v", err)
	}

	for _, tc := range []struct {
		name string
		call func()
	}{
		{"OnReady", func() { lc.OnReady(func() {}) }},
		{"Monitor", func() { lc.Monitor(make(chan error)) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mustPanic(t, "lifecycle: "+tc.name+" after Run", tc.call)
		})
	}
}

func TestCoordinator_RunsOnce(t *testing.T) {
	execOnce := func(c *lifecycle.Coordinator) { _ = c.Exec(context.Background(), noop) }
	runOnce := func(c *lifecycle.Coordinator) { _ = c.Run(cancelled()) }
	for _, tc := range []struct {
		name          string
		first, second func(*lifecycle.Coordinator)
		op            string
	}{
		{"RunTwice", runOnce, runOnce, "Run"},
		{"ExecTwice", execOnce, execOnce, "Exec"},
		{"RunAfterExec", execOnce, runOnce, "Run"},
		{"ExecAfterRun", runOnce, execOnce, "Exec"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lc := coordinator(t, failsafe, graph.New())
			tc.first(lc)
			want := "lifecycle: " + tc.op + " on a Coordinator that already ran"
			mustPanic(t, want, func() { tc.second(lc) })
		})
	}
}

func TestRun_LayersStartLowestFirst(t *testing.T) {
	var r recorder
	g := graph.New()
	pool := node(g, &fake{name: "pool", r: &r})
	consumer := node(g, &fake{name: "consumer", r: &r}, pool)
	server := node(g, &fake{name: "server", r: &r}, consumer)

	if err := runThrough(coordinator(t, failsafe, g, server)); err != nil {
		t.Fatalf("Run: %v", err)
	}

	want := []string{
		"start pool", "start consumer", "start server",
		"stop server", "stop consumer", "stop pool",
	}
	if got := r.list(); !slices.Equal(got, want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
}

func TestRun_LayerBarrierBlocksTheNextLayer(t *testing.T) {
	blocked := make(chan struct{})
	release := make(chan struct{})
	var second atomic.Bool
	g := graph.New()
	first := value(g, "first", starter(func(context.Context) error {
		close(blocked)
		<-release
		return nil
	}))
	top := value(g, "second", starter(func(context.Context) error {
		second.Store(true)
		return nil
	}), first)
	lc := coordinator(t, failsafe, g, top)

	ready := make(chan struct{})
	lc.OnReady(func() { close(ready) })

	ctx, cancel := context.WithCancel(context.Background())
	done := run(ctx, lc)

	recvOrFail(t, blocked, "layer 0 to start")
	time.Sleep(20 * time.Millisecond)
	if second.Load() {
		t.Fatal("layer 1 started while layer 0 was still starting")
	}

	close(release)
	recvOrFail(t, ready, "coordinator to become ready")
	cancel()
	if err := recvOrFail(t, done, "Run to return"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !second.Load() {
		t.Fatal("layer 1 never started")
	}
}

func TestRun_ShutdownReversesLayers(t *testing.T) {
	var r recorder
	g := graph.New()
	pool := node(g, &fake{name: "pool", r: &r})
	// A stop-only participant shuts down with its layer.
	flusher := value(g, "flusher", stopper(func(context.Context) error {
		r.record("stop flusher")
		return nil
	}), pool)
	server := node(g, &fake{name: "server", r: &r}, flusher)

	if err := runThrough(coordinator(t, failsafe, g, server)); err != nil {
		t.Fatalf("Run: %v", err)
	}

	want := []string{"start pool", "start server", "stop server", "stop flusher", "stop pool"}
	if got := r.list(); !slices.Equal(got, want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
}

func TestRun_StartupFailureSkipsHigherLayersAndShutsDownBegunLayers(t *testing.T) {
	var r recorder
	sentinel := errors.New("bus connect failed")
	g := graph.New()
	pool := node(g, &fake{name: "pool", r: &r})
	bus := node(g, &fake{name: "bus", r: &r, start: func(context.Context) error {
		return sentinel
	}}, pool)
	cache := node(g, &fake{name: "cache", r: &r}, pool)
	server := node(g, &fake{name: "server", r: &r}, bus, cache)

	err := coordinator(t, failsafe, g, server).Run(context.Background())
	if !errors.Is(err, sentinel) {
		t.Fatalf("error = %v, want errors.Is(err, sentinel)", err)
	}
	if !strings.Contains(err.Error(), "startup:") || !strings.Contains(err.Error(), "bus:") {
		t.Errorf("error = %v, want the startup wrap naming the failed participant", err)
	}

	// The failing layer finishes its concurrent starts, and the whole layer
	// shuts down, bus whose Start failed included; server never starts.
	got := r.list()
	if !phased(got,
		[]string{"start pool"},
		[]string{"start bus", "start cache"},
		[]string{"stop bus", "stop cache"},
		[]string{"stop pool"},
	) {
		t.Fatalf("events = %v, want pool, {bus cache} started, then {bus cache} and pool stopped", got)
	}
}

// A startup failure cancels the run context before shutdown, so work a
// started participant left running on it stops rather than outliving the
// shutdown.
func TestRun_StartupFailureCancelsRunContextBeforeShutdown(t *testing.T) {
	var runCtx context.Context
	runErr := make(chan error, 1)
	g := graph.New()
	pool := node(g, &fake{
		name: "pool",
		start: func(ctx context.Context) error {
			runCtx = ctx
			return nil
		},
		stop: func(context.Context) error {
			runErr <- runCtx.Err()
			return nil
		},
	})
	bus := value(g, "bus", starter(func(context.Context) error {
		return errors.New("bus connect failed")
	}), pool)

	if err := coordinator(t, failsafe, g, bus).Run(context.Background()); err == nil {
		t.Fatal("Run returned nil for a failing layer")
	}
	if err := recvOrFail(t, runErr, "pool shutdown"); err == nil {
		t.Error("run context was live while the startup failure shut down")
	}
}

// The first failure in a layer cancels its siblings, and the cancellation
// they return is the failure's consequence, not a failure of its own.
func TestRun_StartupFailureCancelsSiblings(t *testing.T) {
	sentinel := errors.New("bus connect failed")
	g := graph.New()
	bus := value(g, "bus", starter(func(context.Context) error { return sentinel }))
	cache := value(g, "cache", starter(func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}))
	lc := coordinator(t, failsafe, g, bus, cache)

	err := recvOrFail(t, run(context.Background(), lc), "Run to return")
	if !errors.Is(err, sentinel) {
		t.Fatalf("error = %v, want errors.Is(err, sentinel)", err)
	}
	if errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want the sibling's consequent cancellation dropped", err)
	}
}

// A cancellation a participant returns from a context of its own is its
// failure, not a consequence: with the run context live and nothing on
// record, it is kept.
func TestRun_StartupKeepsAnUnrelatedCancellation(t *testing.T) {
	g := graph.New()
	dial := value(g, "dial", starter(func(context.Context) error {
		own, cancel := context.WithCancel(context.Background())
		cancel()
		return fmt.Errorf("dial: %w", own.Err())
	}))

	err := coordinator(t, failsafe, g, dial).Run(context.Background())
	if !errors.Is(err, context.Canceled) || !strings.Contains(err.Error(), "startup:") {
		t.Fatalf("error = %v, want the startup failure wrapping context.Canceled", err)
	}
}

// A signal during startup is the signal-driven exit: the Starts that honor
// the run context return its cancellation, the begun layers shut down, and
// Run returns nil.
func TestRun_SignalDuringStartupShutsDownCleanly(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var r recorder
	g := graph.New()
	pool := node(g, &fake{name: "pool", r: &r})
	var layer1 []graph.Ref
	for _, name := range []string{"bus", "cache"} {
		layer1 = append(layer1, node(g, &fake{name: name, r: &r, start: func(ctx context.Context) error {
			cancel() // the signal arrives mid-startup
			<-ctx.Done()
			return fmt.Errorf("connect: %w", ctx.Err())
		}}, pool))
	}
	server := node(g, &fake{name: "server", r: &r}, layer1...)
	lc := coordinator(t, failsafe, g, server)

	readied := false
	lc.OnReady(func() { readied = true })

	if err := lc.Run(ctx); err != nil {
		t.Fatalf("Run after a signal during startup = %v, want nil", err)
	}
	if readied {
		t.Error("OnReady hooks ran for a startup the signal cut short")
	}
	// bus and cache returned the cancellation and still shut down; server
	// never starts.
	got := r.list()
	if !phased(got,
		[]string{"start pool"},
		[]string{"start bus", "start cache"},
		[]string{"stop bus", "stop cache"},
		[]string{"stop pool"},
	) {
		t.Fatalf("events = %v, want the begun layers shut down and server never started", got)
	}
}

// A signal during startup does not mask a real failure: a Start whose error
// is not a cancellation still fails the run.
func TestRun_SignalDuringStartupKeepsARealFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sentinel := errors.New("flush failed")

	g := graph.New()
	bus := value(g, "bus", starter(func(ctx context.Context) error {
		cancel()
		<-ctx.Done()
		return ctx.Err()
	}))
	cache := value(g, "cache", starter(func(ctx context.Context) error {
		<-ctx.Done()
		return sentinel
	}))

	err := coordinator(t, failsafe, g, bus, cache).Run(ctx)
	if !errors.Is(err, sentinel) || !strings.Contains(err.Error(), "startup:") {
		t.Fatalf("error = %v, want the startup failure wrapping the sentinel", err)
	}
}

func TestExec_StartsLayersThenRunsThenShutsDownInReverse(t *testing.T) {
	var r recorder
	g := graph.New()
	// a's Start yields before it returns, so a missing barrier lets layer
	// 1 start before it finishes.
	a := node(g, &fake{name: "a", r: &r, start: func(context.Context) error {
		time.Sleep(20 * time.Millisecond)
		r.record("started a")
		return nil
	}})
	b := node(g, &fake{name: "b", r: &r}, a)
	c := node(g, &fake{name: "c", r: &r}, a)
	d := node(g, &fake{name: "d", r: &r}, b, c)

	err := coordinator(t, failsafe, g, d).Exec(context.Background(), func(context.Context) error {
		r.record("fn")
		return nil
	})
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}

	if got := r.list(); !phased(got,
		[]string{"start a"}, []string{"started a"},
		[]string{"start b", "start c"},
		[]string{"start d"}, []string{"fn"}, []string{"stop d"},
		[]string{"stop b", "stop c"},
		[]string{"stop a"},
	) {
		t.Errorf("events = %q, want a, {b c}, d, fn, then d, {b c}, a", got)
	}
}

// rendezvous returns two functions, each of which marks its side begun and
// waits for the other's, failing with an error when the other never begins:
// the pair completes only when both run at once.
func rendezvous() (a, b func(context.Context) error) {
	aBegun, bBegun := make(chan struct{}), make(chan struct{})
	meet := func(mine chan struct{}, theirs <-chan struct{}) func(context.Context) error {
		return func(context.Context) error {
			close(mine)
			select {
			case <-theirs:
				return nil
			case <-time.After(failsafe):
				return errors.New("sibling never began")
			}
		}
	}
	return meet(aBegun, bBegun), meet(bBegun, aBegun)
}

func TestExec_LayerShutsDownConcurrently(t *testing.T) {
	a, b := rendezvous()
	g := graph.New()
	na := value(g, "a", stopper(a))
	nb := value(g, "b", stopper(b))
	if err := coordinator(t, 2*failsafe, g, na, nb).Exec(context.Background(), noop); err != nil {
		t.Fatalf("Exec: %v", err)
	}
}

func TestExec_ContextEndedBeforeStartupReturnsTheCancellation(t *testing.T) {
	var r recorder
	g := graph.New()
	a := node(g, &fake{name: "a", r: &r})
	ran := false
	err := coordinator(t, failsafe, g, a).Exec(cancelled(), func(context.Context) error {
		ran = true
		return nil
	})
	if !errors.Is(err, context.Canceled) || !strings.Contains(err.Error(), "startup:") {
		t.Fatalf("Exec = %v, want the startup wrap of context.Canceled", err)
	}
	if ran {
		t.Error("Exec ran fn after its context ended")
	}
	if got := r.list(); len(got) != 0 {
		t.Errorf("events = %q, want nothing started", got)
	}
}

// A value takes part in each phase it implements, and a value that
// implements neither takes no part.
func TestExec_ParticipationFollowsTheValue(t *testing.T) {
	var r recorder
	g := graph.New()
	roots := []graph.Ref{
		node(g, &fake{name: "subsystem", r: &r}),
		value(g, "start-only", starter(func(context.Context) error {
			r.record("start start-only")
			return nil
		})),
		value(g, "stop-only", stopper(func(context.Context) error {
			r.record("stop stop-only")
			return nil
		})),
		value(g, "inert", 2),
	}
	if err := coordinator(t, failsafe, g, roots...).Exec(context.Background(), noop); err != nil {
		t.Fatalf("Exec: %v", err)
	}

	got := r.list()
	starts := []string{"start subsystem", "start start-only"}
	stops := []string{"stop subsystem", "stop stop-only"}
	if !phased(got, starts, stops) {
		t.Errorf("events = %q, want starts %q then stops %q", got, starts, stops)
	}
}

func TestExec_StartFailureSkipsFn(t *testing.T) {
	var r recorder
	errBoom := errors.New("boom")
	waiting := make(chan struct{})
	g := graph.New()
	base := node(g, &fake{name: "base", r: &r})
	database := node(g, &fake{name: "database", r: &r, start: func(context.Context) error {
		select {
		case <-waiting:
			return errBoom
		case <-time.After(failsafe):
			return errors.New("sibling never began")
		}
	}}, base)
	slow := node(g, &fake{name: "slow", r: &r, start: func(ctx context.Context) error {
		close(waiting)
		select {
		case <-ctx.Done():
			r.record("slow cancelled")
			return ctx.Err()
		case <-time.After(failsafe):
			return errors.New("never cancelled")
		}
	}}, base)
	top := node(g, &fake{name: "top", r: &r}, database, slow)

	ran := false
	err := coordinator(t, failsafe, g, top).Exec(context.Background(), func(context.Context) error {
		ran = true
		return nil
	})

	if want := "startup: database: boom"; err == nil || err.Error() != want {
		t.Fatalf("Exec = %v, want the single error %q", err, want)
	}
	if !errors.Is(err, errBoom) {
		t.Errorf("Exec = %v, want errors.Is(err, errBoom)", err)
	}
	if ran {
		t.Error("Exec ran fn after a start failure")
	}

	if got := r.list(); !phased(got,
		[]string{"start base"},
		[]string{"start database", "start slow"},
		[]string{"slow cancelled"},
		[]string{"stop database", "stop slow"},
		[]string{"stop base"},
	) {
		t.Errorf("events = %q, want base, {database slow}, slow cancelled, "+
			"then {database slow} and base shut down, top never started", got)
	}
}

func TestExec_JoinsShutdownErrors(t *testing.T) {
	errA, errB := errors.New("a stuck"), errors.New("b stuck")
	newGraph := func(r *recorder) (*graph.Graph, graph.Ref) {
		g := graph.New()
		a := node(g, &fake{name: "a", r: r, stop: func(context.Context) error { return errA }})
		b := node(g, &fake{name: "b", r: r, stop: func(context.Context) error { return errB }}, a)
		return g, b
	}

	t.Run("FailsACleanRun", func(t *testing.T) {
		var r recorder
		g, root := newGraph(&r)
		err := coordinator(t, failsafe, g, root).Exec(context.Background(), noop)
		if want := "shutdown: b: b stuck\na: a stuck"; err == nil || err.Error() != want {
			t.Fatalf("Exec = %v, want %q", err, want)
		}
		if !errors.Is(err, errA) || !errors.Is(err, errB) {
			t.Errorf("Exec = %v, want errA and errB", err)
		}
	})

	t.Run("JoinsFnsError", func(t *testing.T) {
		var r recorder
		g, root := newGraph(&r)
		errFn := errors.New("fn failed")
		err := coordinator(t, failsafe, g, root).Exec(context.Background(), func(context.Context) error {
			return errFn
		})
		if !errors.Is(err, errFn) || !errors.Is(err, errA) || !errors.Is(err, errB) {
			t.Fatalf("Exec = %v, want errFn, errA and errB", err)
		}
		want := []string{"start a", "start b", "stop b", "stop a"}
		if got := r.list(); !slices.Equal(got, want) {
			t.Errorf("events = %q, want %q", got, want)
		}
	})
}

func TestExec_ShutdownContextIsDetachedAndBounded(t *testing.T) {
	const timeout = time.Minute
	var (
		r        recorder
		stopErr  error
		deadline time.Time
		bounded  bool
	)
	g := graph.New()
	a := node(g, &fake{name: "a", r: &r, stop: func(ctx context.Context) error {
		stopErr = ctx.Err()
		deadline, bounded = ctx.Deadline()
		return nil
	}})
	ctx, cancel := context.WithCancel(context.Background())
	begun := time.Now()

	err := coordinator(t, timeout, g, a).Exec(ctx, func(ctx context.Context) error {
		cancel()
		return ctx.Err()
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Exec = %v, want fn's cancellation", err)
	}
	if !slices.Contains(r.list(), "stop a") {
		t.Fatal("a was not shut down after its context was cancelled mid-fn")
	}
	if stopErr != nil {
		t.Errorf("shutdown context err = %v, want live", stopErr)
	}
	if !bounded || deadline.Before(begun.Add(timeout)) || deadline.After(time.Now().Add(timeout)) {
		t.Errorf("shutdown deadline = %v (set %v), want ShutdownTimeout from shutdown", deadline, bounded)
	}
}

func TestExec_ShutdownOverrunStillAttemptsLowerLayers(t *testing.T) {
	var r recorder
	release := make(chan struct{})
	defer close(release)
	baseStopped := make(chan struct{})
	errTop := errors.New("top failed")
	g := graph.New()
	base := node(g, &fake{name: "base", r: &r, stop: func(context.Context) error {
		close(baseStopped)
		return nil
	}})
	hang := node(g, &fake{name: "hang", r: &r, stop: func(context.Context) error {
		<-release
		return errors.New("late")
	}}, base)
	top := node(g, &fake{name: "top", r: &r, stop: func(context.Context) error { return errTop }}, hang)

	err := coordinator(t, 20*time.Millisecond, g, top).Exec(context.Background(), noop)
	// Past the deadline shutdown no longer waits on a layer, so base's
	// shutdown may begin after Exec returns; it must begin.
	recvOrFail(t, baseStopped, "base's shutdown")

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Exec = %v, want a deadline error", err)
	}
	if !strings.Contains(err.Error(), "drain timeout after") {
		t.Errorf("Exec = %v, want the drain timeout description", err)
	}
	if !errors.Is(err, errTop) {
		t.Errorf("Exec = %v, want errTop from the layer before the overrun", err)
	}
	if strings.Contains(err.Error(), "late") {
		t.Errorf("Exec = %v, kept the straggler's late error", err)
	}
	if n := strings.Count(err.Error(), context.DeadlineExceeded.Error()); n != 1 {
		t.Errorf("Exec = %v, want exactly one deadline error, got %d", err, n)
	}
	want := []string{"start base", "start hang", "start top", "stop top", "stop hang", "stop base"}
	if got := r.list(); !slices.Equal(got, want) {
		t.Errorf("events = %q, want %q", got, want)
	}
}
