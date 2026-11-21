package reactor

import (
	"context"
	"sync/atomic"
	"time"
)

// Every returns a source that delivers the time on every tick of d. The
// handler runs synchronously, and a tick that comes due while it runs is
// dropped, so the handler never receives a stale time. A handler error ends
// the source, and so the reactor, which reports it on Err; a handler that
// tolerates a failure returns nil. Every panics if d is not positive, as
// time.NewTicker does.
func Every(d time.Duration) Source[time.Time] {
	if d <= 0 {
		panic("reactor: Every needs a positive interval")
	}
	return &every{d: d}
}

type every struct {
	d     time.Duration
	ready atomic.Bool
}

func (s *every) Receive(ctx context.Context, fn Func[time.Time]) error {
	return receiveTicks(ctx, s.d, nil, &s.ready, fn)
}

func receiveTicks(ctx context.Context, d time.Duration, nudge <-chan struct{}, ready *atomic.Bool, fn Func[time.Time]) error {
	t := time.NewTicker(d)
	defer t.Stop()
	ready.Store(true)
	defer ready.Store(false)
	for {
		var now time.Time
		select {
		case <-ctx.Done():
			return nil
		case now = <-t.C:
		case <-nudge:
			now = time.Now()
		}
		if ctx.Err() != nil {
			return nil
		}
		if err := fn(ctx, now); err != nil {
			return err
		}
		select {
		case <-t.C:
		default:
		}
	}
}

func (s *every) Ready() bool { return s.ready.Load() }
