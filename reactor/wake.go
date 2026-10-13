package reactor

import (
	"context"
	"sync/atomic"
	"time"
)

// Wake returns a source that delivers the time on every tick of d, as
// [Every] does, and also on a [Waker.Nudge]: the interval is the backstop
// and the nudge the prompt, for work whose producer can say that some is
// waiting. Nudges coalesce: any number before the next delivery produce
// one, and a nudge while the handler runs produces exactly one delivery
// after it. A tick that comes due while the handler runs is dropped, as
// Every drops it. A handler error ends the source, as it ends Every. Wake
// panics if d is not positive.
func Wake(d time.Duration) *Waker {
	if d <= 0 {
		panic("reactor: Wake needs a positive interval")
	}
	return &Waker{d: d, nudge: make(chan struct{}, 1)}
}

// Waker is the [Wake] source: a [Source] of times with a Nudge its
// producers call.
type Waker struct {
	d     time.Duration
	nudge chan struct{} // one pending nudge; a second coalesces into it
	ready atomic.Bool
}

// Nudge asks for a delivery as soon as the handler is free. It returns at
// once and never blocks: a nudge already pending absorbs it. A nudge
// before the source starts is delivered when it does; one after the
// source stops delivers nothing.
func (s *Waker) Nudge() {
	select {
	case s.nudge <- struct{}{}:
	default:
	}
}

// Receive delivers the time on each tick and each pending nudge until ctx
// ends.
func (s *Waker) Receive(ctx context.Context, fn Func[time.Time]) error {
	return receiveTicks(ctx, s.d, s.nudge, &s.ready, fn)
}

// Ready reports whether the source is receiving.
func (s *Waker) Ready() bool { return s.ready.Load() }
