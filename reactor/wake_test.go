package reactor_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/standards-lab/go-core/reactor"
)

// counter is a Func that counts its runs and reports each on runs.
type counter struct {
	calls atomic.Int32
	runs  chan struct{}
}

func newCounter() *counter { return &counter{runs: make(chan struct{}, 64)} }

func (c *counter) fn(context.Context, time.Time) error {
	c.calls.Add(1)
	c.runs <- struct{}{}
	return nil
}

// settle waits long enough for a delivery that should not happen to show.
func settle() { time.Sleep(50 * time.Millisecond) }

// Wake without a nudge is an interval: it delivers on every tick.
func TestWake_Interval(t *testing.T) {
	c := newCounter()
	r := reactor.New(reactor.Wake(5*time.Millisecond), c.fn)
	if err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	eventually(t, func() bool { return c.calls.Load() >= 3 }, "three ticks")
	if err := shutdown(t, r, failsafe); err != nil {
		t.Fatal(err)
	}
}

// Nudges before the next delivery coalesce into one prompt run, long
// before the hour-long interval; a nudge after it is a run of its own.
// The nudges here come before Start, so the first run also shows that a
// nudge before the source starts is delivered when it does.
func TestWake_NudgesCoalesce(t *testing.T) {
	c := newCounter()
	w := reactor.Wake(time.Hour)
	for range 5 {
		w.Nudge()
	}
	r := reactor.New(w, c.fn)
	if err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	recvOrFail(t, c.runs, "the nudged run")
	settle()
	if n := c.calls.Load(); n != 1 {
		t.Fatalf("five nudges before the run ran %d times, want 1", n)
	}

	w.Nudge()
	recvOrFail(t, c.runs, "the second nudged run")
	settle()
	if n := c.calls.Load(); n != 2 {
		t.Errorf("a nudge after the run brought the total to %d, want 2", n)
	}
	if err := shutdown(t, r, failsafe); err != nil {
		t.Fatal(err)
	}
}

// A nudge while the handler runs, however many, produces exactly one run
// after it: work that arrived during the run is not lost, and is not run
// twice.
func TestWake_NudgeDuringRun(t *testing.T) {
	var calls atomic.Int32
	running := make(chan struct{}, 1)
	release := make(chan struct{})
	w := reactor.Wake(time.Hour)
	r := reactor.New(w, func(context.Context, time.Time) error {
		if calls.Add(1) == 1 {
			running <- struct{}{}
			<-release
		}
		return nil
	})
	if err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	w.Nudge()
	recvOrFail(t, running, "the first run")
	for range 3 {
		w.Nudge()
	}
	close(release)
	eventually(t, func() bool { return calls.Load() == 2 }, "the run after the nudges")
	settle()
	if n := calls.Load(); n != 2 {
		t.Errorf("nudges during a run brought the total to %d, want 2", n)
	}
	if err := shutdown(t, r, failsafe); err != nil {
		t.Fatal(err)
	}
}

// A nudge after Shutdown runs nothing and does not block.
func TestWake_NoRunAfterShutdown(t *testing.T) {
	c := newCounter()
	w := reactor.Wake(time.Hour)
	r := reactor.New(w, c.fn)
	if err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	eventually(t, r.Ready, "readiness")
	if err := shutdown(t, r, failsafe); err != nil {
		t.Fatal(err)
	}
	w.Nudge()
	w.Nudge()
	settle()
	if n := c.calls.Load(); n != 0 {
		t.Errorf("nudges after Shutdown ran %d times, want 0", n)
	}
	if w.Ready() || r.Ready() {
		t.Error("ready after Shutdown")
	}
}

// A handler's error ends Wake as it ends Every, reported on Err.
func TestWake_HandlerErrorOnErr(t *testing.T) {
	boom := errors.New("boom")
	w := reactor.Wake(time.Hour)
	r := reactor.New(w, func(context.Context, time.Time) error { return boom })
	if err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	w.Nudge()
	if err := recvOrFail(t, r.Err(), "the failure"); !errors.Is(err, boom) {
		t.Fatalf("Err yielded %v", err)
	}
	if err := shutdown(t, r, failsafe); err != nil {
		t.Errorf("Shutdown after a reported failure = %v, want nil", err)
	}
}

func TestWake_RejectsNonPositive(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("Wake(0) did not panic")
		}
	}()
	reactor.Wake(0)
}
