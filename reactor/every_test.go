package reactor_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/standards-lab/go-core/reactor"
)

func TestEvery_DropsStaleTicks(t *testing.T) {
	const interval = 20 * time.Millisecond
	lags := make(chan time.Duration, 3)
	r := reactor.New(reactor.Every(interval), func(_ context.Context, at time.Time) error {
		select {
		case lags <- time.Since(at):
		default:
		}
		time.Sleep(3 * interval) // outlast several ticks
		return nil
	})
	if err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if lag := recvOrFail(t, lags, "a tick"); lag > interval {
			t.Errorf("tick delivered %v late; a stale tick was kept", lag)
		}
	}
	if err := shutdown(t, r, failsafe); err != nil {
		t.Fatal(err)
	}
}

func TestEvery_RejectsNonPositive(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("Every(0) did not panic")
		}
	}()
	reactor.Every(0)
}

// A handler's error ends Every, and so the reactor, which reports it on Err.
func TestEvery_HandlerErrorEndsIt(t *testing.T) {
	boom := errors.New("boom")
	var calls int
	r := reactor.New(reactor.Every(time.Millisecond), func(context.Context, time.Time) error {
		calls++
		return boom
	})
	if err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := recvOrFail(t, r.Err(), "the failure"); !errors.Is(err, boom) {
		t.Fatalf("Err yielded %v", err)
	}
	if _, open := <-r.Err(); open {
		t.Error("Err did not close after Every ended")
	}
	if calls != 1 {
		t.Errorf("the handler ran %d times, want 1: the error ends Every", calls)
	}
	if err := shutdown(t, r, failsafe); err != nil {
		t.Errorf("Shutdown after a reported failure = %v, want nil", err)
	}
}
