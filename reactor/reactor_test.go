package reactor_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/standards-lab/go-core/reactor"
)

// failsafe bounds every wait for something that should happen, so a broken
// reactor fails the test instead of hanging it.
const failsafe = 2 * time.Second

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

func eventually(t *testing.T, cond func() bool, what string) {
	t.Helper()
	deadline := time.Now().Add(failsafe)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(time.Millisecond)
	}
}

func shutdown(t *testing.T, r interface{ Shutdown(context.Context) error }, d time.Duration) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	return r.Shutdown(ctx)
}

func TestReactor_ShutdownDrainsInFlight(t *testing.T) {
	started := make(chan struct{}, 1)
	var calls atomic.Int32
	var ended, finished atomic.Bool
	r := reactor.New(reactor.Every(5*time.Millisecond), func(ctx context.Context, _ time.Time) error {
		calls.Add(1)
		select {
		case started <- struct{}{}:
		default:
		}
		time.Sleep(50 * time.Millisecond)
		ended.Store(ctx.Err() != nil)
		finished.Store(true)
		return nil
	})

	runCtx, cancelRun := context.WithCancel(context.Background())
	if err := r.Start(runCtx); err != nil {
		t.Fatal(err)
	}
	recvOrFail(t, started, "the first tick")
	cancelRun() // the coordinator cancels the run context before draining

	if err := shutdown(t, r, failsafe); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	if !finished.Load() {
		t.Error("Shutdown returned before the in-flight handler finished")
	}
	if ended.Load() {
		t.Error("the handler's context ended before the drain finished")
	}
	n := calls.Load()
	time.Sleep(30 * time.Millisecond)
	if calls.Load() != n {
		t.Error("a tick was handled after Shutdown returned")
	}
}

func TestReactor_RunContextDoesNotStop(t *testing.T) {
	var calls atomic.Int32
	r := reactor.New(reactor.Every(2*time.Millisecond), func(context.Context, time.Time) error {
		calls.Add(1)
		return nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	if err := r.Start(ctx); err != nil {
		t.Fatal(err)
	}
	cancel()
	eventually(t, func() bool { return calls.Load() >= 3 }, "ticks after the run context ended")
	if err := shutdown(t, r, failsafe); err != nil {
		t.Fatal(err)
	}
}

func TestReactor_ShutdownDeadlineCancelsHandler(t *testing.T) {
	started := make(chan struct{}, 1)
	cancelled := make(chan error, 1)
	r := reactor.New(reactor.Every(time.Millisecond), func(ctx context.Context, _ time.Time) error {
		started <- struct{}{}
		<-ctx.Done()
		cancelled <- ctx.Err()
		return nil
	})
	if err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	recvOrFail(t, started, "the handler")
	err := shutdown(t, r, 20*time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Shutdown = %v, want DeadlineExceeded", err)
	}
	if got := recvOrFail(t, cancelled, "the handler's cancellation"); !errors.Is(got, context.Canceled) {
		t.Errorf("handler context error = %v", got)
	}
}

func TestReactor_GraceCancelsThenWaits(t *testing.T) {
	started := make(chan struct{}, 1)
	unwound := make(chan struct{})
	r := reactor.New(reactor.Every(time.Millisecond), func(ctx context.Context, _ time.Time) error {
		started <- struct{}{}
		<-ctx.Done()
		time.Sleep(20 * time.Millisecond) // the handler takes time to unwind
		close(unwound)
		return ctx.Err()
	}, reactor.Grace(20*time.Millisecond))
	if err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	recvOrFail(t, started, "the handler")
	err := shutdown(t, r, failsafe)
	select {
	case <-unwound:
	default:
		t.Fatal("Shutdown returned before the handler unwound")
	}
	if err == nil || !strings.Contains(err.Error(), "handlers cancelled after grace 20ms") {
		t.Fatalf("Shutdown = %v, want the grace report", err)
	}
	if errors.Is(err, context.Canceled) {
		t.Errorf("Shutdown = %v, reports the handler's own cancellation", err)
	}
}

func TestReactor_GraceStillBoundedByContext(t *testing.T) {
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	defer close(release)
	r := reactor.New(reactor.Every(time.Millisecond), func(context.Context, time.Time) error {
		started <- struct{}{}
		<-release // ignores cancellation
		return nil
	}, reactor.Grace(10*time.Millisecond))
	if err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	recvOrFail(t, started, "the handler")
	if err := shutdown(t, r, 50*time.Millisecond); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Shutdown = %v, want DeadlineExceeded", err)
	}
}

func TestReactor_GraceUnusedIsClean(t *testing.T) {
	r := reactor.New(reactor.Every(time.Millisecond), func(context.Context, time.Time) error {
		return nil
	}, reactor.Grace(time.Second))
	if err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	eventually(t, r.Ready, "readiness")
	if err := shutdown(t, r, failsafe); err != nil {
		t.Fatalf("Shutdown = %v", err)
	}
}

// GraceWithin grants half the drain timeout it is given: a handler that
// ignores the drain is cancelled once that half passes.
func TestReactor_GraceWithinIsHalfTheDrain(t *testing.T) {
	started := make(chan struct{}, 1)
	r := reactor.New(reactor.Every(time.Millisecond), func(ctx context.Context, _ time.Time) error {
		started <- struct{}{}
		<-ctx.Done()
		return ctx.Err()
	}, reactor.GraceWithin(40*time.Millisecond))
	if err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	recvOrFail(t, started, "the handler")
	begun := time.Now()
	err := shutdown(t, r, failsafe)
	if err == nil || !strings.Contains(err.Error(), "handlers cancelled after grace 20ms") {
		t.Fatalf("Shutdown = %v, want the report of a 20ms grace", err)
	}
	if took := time.Since(begun); took < 20*time.Millisecond {
		t.Errorf("handlers cancelled after %v, before the 20ms grace", took)
	}
}

// A handler's error ends an interval source, and so the reactor, which
// reports it on Err, closes Err, and reads not ready; the failure is not
// reported a second time by Shutdown.
func TestReactor_HandlerErrorOnErr(t *testing.T) {
	boom := errors.New("boom")
	r := reactor.New(reactor.Every(time.Millisecond), func(context.Context, time.Time) error {
		return boom
	})
	if err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := recvOrFail(t, r.Err(), "the failure"); !errors.Is(err, boom) {
		t.Fatalf("Err yielded %v", err)
	}
	if _, open := <-r.Err(); open {
		t.Error("Err did not close after the source ended")
	}
	if r.Ready() {
		t.Error("a failed reactor reports ready")
	}
	if err := shutdown(t, r, failsafe); err != nil {
		t.Errorf("Shutdown after a reported failure = %v, want nil", err)
	}
}

func TestReactor_UnreadErrReachesShutdown(t *testing.T) {
	boom := errors.New("boom")
	src := failing{err: boom, fail: make(chan struct{})}
	r := reactor.New[int](src, func(context.Context, int) error { return nil })
	if err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	// Nothing reads Err, as when the coordinator stopped monitoring at the
	// signal just before the source failed. The reactor turns not ready
	// as it puts the failure on Err, so Shutdown begins only once the
	// failure has landed there.
	eventually(t, r.Ready, "readiness")
	close(src.fail)
	eventually(t, func() bool { return !r.Ready() }, "the failure to land on Err")
	if err := shutdown(t, r, failsafe); !errors.Is(err, boom) {
		t.Fatalf("Shutdown = %v, want the unread failure", err)
	}
}

func TestReactor_ErrorWhileDrainingGoesToShutdown(t *testing.T) {
	boom := errors.New("boom")
	src := &stub{err: boom}
	r := reactor.New[int](src, func(context.Context, int) error { return nil })
	if err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	eventually(t, r.Ready, "readiness")
	if err := shutdown(t, r, failsafe); !errors.Is(err, boom) {
		t.Fatalf("Shutdown = %v, want boom", err)
	}
	if _, open := <-r.Err(); open {
		t.Error("Err yielded an error sent to Shutdown")
	}
}

func TestReactor_Ready(t *testing.T) {
	r := reactor.New(reactor.Every(time.Hour), func(context.Context, time.Time) error { return nil })
	if r.Ready() {
		t.Error("ready before Start")
	}
	if err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	eventually(t, r.Ready, "readiness")
	if err := shutdown(t, r, failsafe); err != nil {
		t.Fatal(err)
	}
	if r.Ready() {
		t.Error("ready after Shutdown")
	}
}

// Readiness turns false when Shutdown begins, not when it returns: a handler
// held in flight keeps the drain open while Ready is read.
func TestReactor_NotReadyWhileDraining(t *testing.T) {
	inFlight := make(chan struct{})
	drainSeen := make(chan struct{}, 1)
	release := make(chan struct{})
	var first sync.Once
	r := reactor.New(reactor.Every(time.Millisecond), func(ctx context.Context, _ time.Time) error {
		first.Do(func() { close(inFlight) })
		<-reactor.Draining(ctx)
		select {
		case drainSeen <- struct{}{}:
		default:
		}
		<-release
		return nil
	})
	if err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	recvOrFail(t, inFlight, "the handler to start")
	eventually(t, r.Ready, "readiness")

	done := make(chan error, 1)
	go func() { done <- shutdown(t, r, failsafe) }()
	recvOrFail(t, drainSeen, "the drain to begin")
	if r.Ready() {
		t.Error("ready while Shutdown drains a handler in flight")
	}
	close(release)
	if err := recvOrFail(t, done, "Shutdown to return"); err != nil {
		t.Fatalf("Shutdown = %v, want a clean drain", err)
	}
}

func TestReactor_StartTwice(t *testing.T) {
	r := reactor.New(reactor.Every(time.Hour), func(context.Context, time.Time) error { return nil })
	if err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := r.Start(context.Background()); err == nil {
		t.Error("second Start succeeded")
	}
	if err := shutdown(t, r, failsafe); err != nil {
		t.Fatal(err)
	}
}

func TestReactor_ShutdownBeforeStart(t *testing.T) {
	r := reactor.New(reactor.Every(time.Hour), func(context.Context, time.Time) error { return nil })
	if err := shutdown(t, r, failsafe); err != nil {
		t.Errorf("Shutdown before Start = %v", err)
	}
	if err := r.Start(context.Background()); err == nil {
		t.Error("Start after Shutdown succeeded")
	}
	if _, open := <-r.Err(); open {
		t.Error("Err is open on a retired reactor")
	}
}

// A second Shutdown of a reactor retired before Start returns nil: it has
// no source to stop.
func TestReactor_ShutdownTwiceBeforeStart(t *testing.T) {
	r := reactor.New(reactor.Every(time.Hour), func(context.Context, time.Time) error { return nil })
	for i := range 2 {
		if err := shutdown(t, r, failsafe); err != nil {
			t.Errorf("Shutdown %d before Start = %v", i+1, err)
		}
	}
}

// A second Shutdown of a drained reactor returns at once, with the first's
// outcome.
func TestReactor_ShutdownTwiceAfterStart(t *testing.T) {
	r := reactor.New(reactor.Every(time.Hour), func(context.Context, time.Time) error { return nil })
	if err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	for i := range 2 {
		if err := shutdown(t, r, failsafe); err != nil {
			t.Errorf("Shutdown %d = %v", i+1, err)
		}
	}
}

// The drain signal closes when Shutdown begins, while the handler in
// flight still runs on a live context, so a handler that works in steps
// stops between them and the drain ends clean, without a grace.
func TestReactor_DrainingClosesWhenShutdownBegins(t *testing.T) {
	inFlight := make(chan struct{})
	var first sync.Once
	type seen struct {
		drained bool
		ctxErr  error
		steps   int
	}
	got := make(chan seen, 1)
	r := reactor.New(reactor.Every(time.Millisecond), func(ctx context.Context, _ time.Time) error {
		first.Do(func() { close(inFlight) })
		steps := 0
		for {
			steps++
			select {
			case <-reactor.Draining(ctx):
				got <- seen{true, ctx.Err(), steps}
				return nil
			case <-time.After(time.Millisecond):
			}
		}
	})
	if err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	recvOrFail(t, inFlight, "the handler to start")
	if err := shutdown(t, r, failsafe); err != nil {
		t.Fatalf("Shutdown = %v, want a clean drain", err)
	}
	res := recvOrFail(t, got, "the handler to see the drain")
	if !res.drained || res.ctxErr != nil || res.steps == 0 {
		t.Errorf("handler saw %+v; want the drain signal on a live context", res)
	}
}

// A context no reactor gave has no drain signal: Draining is nil, which
// never closes.
func TestDraining_OutsideAReactor(t *testing.T) {
	if ch := reactor.Draining(context.Background()); ch != nil {
		t.Errorf("Draining(Background) = %v, want nil", ch)
	}
}

// ctxKey is a context key the tests put values under.
type ctxKey string

// A handler's context carries the start context's values, though not its
// cancellation.
func TestReactor_HandlerKeepsStartValues(t *testing.T) {
	got := make(chan any, 1)
	r := reactor.New(reactor.Every(time.Millisecond), func(ctx context.Context, _ time.Time) error {
		select {
		case got <- ctx.Value(ctxKey("start")):
		default:
		}
		return nil
	})
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), ctxKey("start"), "run"))
	if err := r.Start(ctx); err != nil {
		t.Fatal(err)
	}
	cancel()
	if v := recvOrFail(t, got, "a tick"); v != "run" {
		t.Errorf("handler saw start value %v, want run", v)
	}
	if err := shutdown(t, r, failsafe); err != nil {
		t.Fatal(err)
	}
}

// A handler's context keeps the deadline and values the source set for the
// occurrence, and ends at that deadline rather than at the source's stop.
func TestReactor_HandlerKeepsSourceDeadline(t *testing.T) {
	src := &deadlineSource{d: 50 * time.Millisecond, stopped: make(chan struct{})}
	type result struct {
		hasDeadline bool
		value       any
		err         error
	}
	got := make(chan result, 1)
	r := reactor.New[int](src, func(ctx context.Context, _ int) error {
		<-src.stopped
		_, ok := ctx.Deadline()
		v := ctx.Value(ctxKey("occurrence"))
		<-ctx.Done()
		got <- result{ok, v, ctx.Err()}
		return nil
	})
	if err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	res := recvOrFail(t, got, "the handler's context to end")
	if !res.hasDeadline {
		t.Error("the handler lost the source's deadline")
	}
	if res.value != "one" {
		t.Errorf("handler saw occurrence value %v, want one", res.value)
	}
	if !errors.Is(res.err, context.DeadlineExceeded) {
		t.Errorf("handler context ended with %v, want the deadline, not the source's stop", res.err)
	}
	if err := shutdown(t, r, failsafe); err != nil {
		t.Errorf("Shutdown: %v", err)
	}
}

// stub is a source that is ready until its context ends, then returns err.
type stub struct {
	err   error
	ready atomic.Bool
}

func (s *stub) Receive(ctx context.Context, _ reactor.Func[int]) error {
	s.ready.Store(true)
	<-ctx.Done()
	s.ready.Store(false)
	return s.err
}

func (s *stub) Ready() bool { return s.ready.Load() }

// failing is a source that is ready from the start and fails once fail
// closes.
type failing struct {
	err  error
	fail chan struct{}
}

func (f failing) Receive(context.Context, reactor.Func[int]) error {
	<-f.fail
	return f.err
}

func (failing) Ready() bool { return true }

// deadlineSource delivers one occurrence under a per-occurrence deadline
// and value, then stops its own context while the handler still runs.
type deadlineSource struct {
	d       time.Duration
	stopped chan struct{}
}

func (s *deadlineSource) Receive(ctx context.Context, fn reactor.Func[int]) error {
	occ, cancel := context.WithTimeout(context.WithValue(ctx, ctxKey("occurrence"), "one"), s.d)
	defer cancel()
	stop, stopNow := context.WithCancel(occ)
	done := make(chan error, 1)
	go func() { done <- fn(stop, 1) }()
	stopNow()
	close(s.stopped)
	<-done
	<-ctx.Done()
	return nil
}

func (s *deadlineSource) Ready() bool { return true }
