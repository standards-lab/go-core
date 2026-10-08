package lifecycle

import (
	"context"
	"errors"
	"fmt"
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

	mu       sync.Mutex
	state    state
	onReady  []func()
	monitors []<-chan error
}

// New returns a Coordinator for sys, shutting down within
// cfg.ShutdownTimeout. It panics on a nil sys or a ShutdownTimeout that is
// not positive, as an unfinalized Config's is: both are wiring mistakes.
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
	return &Coordinator{sys: sys, timeout: cfg.ShutdownTimeout.Duration()}
}

// OnReady registers a hook [Coordinator.Run] invokes synchronously, in
// registration order, once startup has completed: every layer started.
// Registration after Exec or Run panics.
func (c *Coordinator) OnReady(fn func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state != stateWaiting {
		panic("lifecycle: OnReady after Run")
	}
	c.onReady = append(c.onReady, fn)
}

// Monitor registers a channel [Coordinator.Run] watches while running: the
// first non-nil error received ends the run and joins Run's return. A nil
// error is ignored, and a closed channel retires quietly — the expected end of
// a source that stopped cleanly. Registration after Exec or Run panics.
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

// Exec starts the System, runs fn under the run context, and shuts the
// System down. It returns fn's error joined with the shutdown's. When
// startup fails, or ctx ends before startup completes, fn does not run and
// Exec returns the startup error, or ctx's, wrapped "startup:" and joined
// with the shutdown's. A second call to Exec or [Coordinator.Run] panics.
func (c *Coordinator) Exec(ctx context.Context, fn func(context.Context) error) error {
	return c.execute(ctx, "Exec", false, func(runCtx context.Context, _ context.CancelCauseFunc) error {
		return fn(runCtx)
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
	return c.execute(ctx, "Run", true, func(runCtx context.Context, fail context.CancelCauseFunc) error {
		for _, fn := range c.onReady {
			fn()
		}
		c.watch(runCtx, fail)
		<-runCtx.Done()
		if cause := context.Cause(runCtx); !errors.Is(cause, context.Canceled) {
			return fmt.Errorf("run: %w", cause)
		}
		return nil
	})
}

// execute is the runtime path Exec and Run share: it claims the
// Coordinator for op, derives the run context from ctx, starts the layers
// under it, runs body once every layer has started, and shuts down on every
// path, cancelling the run context first. body receives the run context and
// the function that cancels it with a cause. A startup that ctx's end cut
// short returns only the shutdown's error when cutShortIsClean; otherwise
// it is a startup failure like any other.
func (c *Coordinator) execute(
	ctx context.Context,
	op string,
	cutShortIsClean bool,
	body func(context.Context, context.CancelCauseFunc) error,
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
	err := body(runCtx, fail)
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

// watch watches every monitored channel until ctx ends, failing the run
// with the first non-nil error any of them yields.
func (c *Coordinator) watch(
	ctx context.Context,
	fail context.CancelCauseFunc,
) {
	for _, ch := range c.monitors {
		go func() {
			for {
				select {
				case err, ok := <-ch:
					if !ok {
						return
					}
					if err != nil {
						fail(err)
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}
}
