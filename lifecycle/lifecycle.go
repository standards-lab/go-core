package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/standards-lab/go-core/graph"
)

// Coordinator runs one built [graph.System]: it starts the System's layers,
// runs a function or serves, and shuts the layers down in reverse. Declare
// hooks and monitors, then call Exec or Run once.
type Coordinator struct {
	sys     *graph.System
	timeout time.Duration
	checks  []Check
	// monitored are the System's [Monitored] values, in layer then
	// definition order; their channels are read once startup completes.
	monitored []Monitored

	mu       sync.Mutex
	state    state
	onReady  []func()
	monitors []<-chan error
}

// New returns a Coordinator for sys, shutting down within
// cfg.ShutdownTimeout, binds every [Readiness] in sys to it, and watches
// every [Monitored] value in sys as [Coordinator.Monitor] would. It panics
// on a nil sys, a ShutdownTimeout that is not positive, as an unfinalized
// Config's is, or a Readiness already bound to another Coordinator: each is
// a wiring mistake.
func New(sys *graph.System, cfg Config) *Coordinator {
	if sys == nil {
		panic("lifecycle: New: nil System")
	}
	if cfg.ShutdownTimeout <= 0 {
		panic(fmt.Sprintf(
			"lifecycle: New: shutdown timeout %v is not positive; finalize the Config",
			cfg.ShutdownTimeout,
		))
	}
	c := &Coordinator{sys: sys, timeout: cfg.ShutdownTimeout.Duration()}
	var bind []graph.Dependency
	for _, layer := range sys.Layers() {
		for _, d := range layer {
			switch v := d.Value.(type) {
			case *Readiness:
				bind = append(bind, d)
			case ReadinessChecker:
				c.checks = append(c.checks, Check{Name: d.Name, Checker: v})
			}
			if v, ok := d.Value.(Monitored); ok {
				c.monitored = append(c.monitored, v)
			}
		}
	}
	// Bind once the arguments are checked, so a New that panics on them
	// binds nothing.
	for _, d := range bind {
		d.Value.(*Readiness).bind(d.Name, c)
	}
	return c
}

// OnReady registers a hook [Coordinator.Exec] or [Coordinator.Run] invokes
// synchronously, in registration order, once startup has completed and the
// Coordinator is ready: before Exec's function runs, and before Run serves.
// Registration after Exec or Run panics.
func (c *Coordinator) OnReady(fn func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state != stateWaiting {
		panic("lifecycle: OnReady after Run")
	}
	c.onReady = append(c.onReady, fn)
}

// Monitor registers a channel [Coordinator.Exec] and [Coordinator.Run]
// watch once startup has completed, alongside the System's [Monitored]
// values: the first non-nil error received ends the run and joins the
// return wrapped "run:". A nil error is ignored, and a closed channel
// retires quietly — the expected end of a source that stopped cleanly.
// Registration after Exec or Run panics.
func (c *Coordinator) Monitor(errs <-chan error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state != stateWaiting {
		panic("lifecycle: Monitor after Run")
	}
	c.monitors = append(c.monitors, errs)
}

// Ready reports whether startup has completed and shutdown has not begun.
func (c *Coordinator) Ready() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state == stateRunning
}

// Checks returns a [Check] for each value in the System that implements
// [ReadinessChecker], named by its node's name, in layer order and, within a
// layer, definition order. A [Readiness] is not among them: it reports the
// Coordinator, not a dependency. Each call returns a fresh copy, nil when
// there are none.
func (c *Coordinator) Checks() []Check {
	return slices.Clone(c.checks)
}

// Exec starts the System, runs fn under the run context, and shuts the
// System down. Exec keeps Run's runtime contract: the Coordinator is ready
// and its OnReady hooks have run when fn starts, and it is not ready once
// shutdown begins, and a monitored failure ends fn's context. It returns
// fn's error joined with the monitored failure, wrapped "run:", and the
// shutdown's. When
// startup fails, or ctx ends before startup completes, fn does not run and
// Exec returns the startup error, or ctx's, wrapped "startup:" and joined
// with the shutdown's. A second call to Exec or [Coordinator.Run] panics.
func (c *Coordinator) Exec(ctx context.Context, fn func(context.Context) error) error {
	return c.execute(ctx, "Exec", false, func(runCtx context.Context, _ context.CancelCauseFunc, monitored func() error) error {
		err := fn(runCtx)
		if m := monitored(); m != nil {
			return errors.Join(err, fmt.Errorf("run: %w", m))
		}
		return err
	})
}

// Run starts the System, serves until ctx ends or a monitored channel
// fails, and shuts the System down. The end of ctx is the clean stop, and
// so is a ctx that ends before or during startup: whatever began to start is
// shut down, and Run returns only the shutdown's error, nil when it is
// clean. A startup failure returns the startup error wrapped "startup:", and
// a monitored failure returns it wrapped "run:", each joined with the
// shutdown's. A second call to Run or [Coordinator.Exec] panics.
func (c *Coordinator) Run(ctx context.Context) error {
	return c.execute(ctx, "Run", true, func(runCtx context.Context, _ context.CancelCauseFunc, _ func() error) error {
		<-runCtx.Done()
		if cause := context.Cause(runCtx); !errors.Is(cause, context.Canceled) {
			return fmt.Errorf("run: %w", cause)
		}
		return nil
	})
}

// execute is the runtime path Exec and Run share: it claims the
// Coordinator for op, derives the run context from ctx, starts the layers
// under it, and once every layer has started, flips ready, runs the
// OnReady hooks in registration order, starts watching the monitored
// channels, and runs body. It shuts down on every path, cancelling the run
// context first. body receives the run context, the function that cancels
// it with a cause, and a function reporting the first monitored failure,
// nil when none has occurred. A startup that ctx's end cut
// short returns only the shutdown's error when cutShortIsClean; otherwise
// it is a startup failure like any other.
func (c *Coordinator) execute(
	ctx context.Context,
	op string,
	cutShortIsClean bool,
	body func(context.Context, context.CancelCauseFunc, func() error) error,
) error {
	c.claim(op)

	runCtx, fail := context.WithCancelCause(ctx)
	defer fail(nil)

	var e engine
	failed := c.startup(runCtx, fail, &e)
	if failed.any() {
		// A context end before or during startup ends the run like one
		// after it: a startup cut short only by that end is not a failure.
		if cutShortIsClean && ctx.Err() != nil && failed.only(ctx.Err()) {
			return c.shutdown(fail, &e)
		}
		return errors.Join(fmt.Errorf("startup: %w", failed.join()), c.shutdown(fail, &e))
	}

	c.setState(stateRunning)
	for _, fn := range c.onReady {
		fn()
	}
	monitored := c.watch(runCtx, fail)
	err := body(runCtx, fail, monitored)
	return errors.Join(err, c.shutdown(fail, &e))
}

// claim moves the Coordinator out of waiting for op, panicking when it
// already ran.
func (c *Coordinator) claim(op string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state != stateWaiting {
		panic("lifecycle: " + op + " on a Coordinator that already ran")
	}
	c.state = stateStarting
}

// startup starts the System's layers, lowest first, each as one engine
// phase, and returns what failed. The first failure cancels ctx, so the rest
// of its layer can stop early, and no higher layer starts. A layer starts
// only while ctx is live: once ctx has ended, startup records its error and
// starts nothing further.
func (c *Coordinator) startup(
	ctx context.Context,
	fail context.CancelCauseFunc,
	e *engine,
) *failures {
	failed := &failures{}
	for _, layer := range c.sys.Layers() {
		if err := ctx.Err(); err != nil {
			failed.record(err)
			break
		}
		steps := participants(layer)
		if len(steps) == 0 {
			continue
		}
		e.start(ctx, fail, steps, failed)
		if failed.any() {
			break
		}
	}
	return failed
}

// shutdown cancels the run context, so work started under it stops, then
// unwinds every phase startup pushed within the configured timeout, and
// returns the errors joined under "shutdown: ", or nil.
func (c *Coordinator) shutdown(fail context.CancelCauseFunc, e *engine) error {
	c.setState(stateDraining)
	fail(nil)
	errs := e.unwind(c.timeout)
	c.setState(stateStopped)
	if len(errs) > 0 {
		return fmt.Errorf("shutdown: %w", errors.Join(errs...))
	}
	return nil
}

func (c *Coordinator) setState(s state) {
	c.mu.Lock()
	c.state = s
	c.mu.Unlock()
}

// watch watches every channel registered with [Coordinator.Monitor] and
// every [Monitored] value's Err channel, read now that startup has
// completed, until ctx ends. The first non-nil error any of them yields
// fails the run. A nil channel never yields, so it is not watched. watch
// returns a function reporting the first monitored failure, nil until one
// occurs.
func (c *Coordinator) watch(
	ctx context.Context,
	fail context.CancelCauseFunc,
) func() error {
	channels := slices.Clone(c.monitors)
	for _, m := range c.monitored {
		channels = append(channels, m.Err())
	}

	var (
		mu    sync.Mutex
		first error
	)
	for _, ch := range channels {
		if ch == nil {
			continue
		}
		go func() {
			for {
				select {
				case err, ok := <-ch:
					if !ok {
						return
					}
					if err != nil {
						// Recording and failing under one lock keeps the
						// failure reported the run's cause.
						mu.Lock()
						if first == nil {
							first = err
							fail(err)
						}
						mu.Unlock()
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	return func() error {
		mu.Lock()
		defer mu.Unlock()
		return first
	}
}
