package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// ReadinessChecker reports whether a subsystem is ready to serve.
// [Coordinator] satisfies it, as does any subsystem a readiness probe
// aggregates.
type ReadinessChecker interface {
	Ready() bool
}

type state string

const (
	stateWaiting  state = "WAITING"
	stateStarting state = "STARTING"
	stateRunning  state = "RUNNING"
	stateDraining state = "DRAINING"
	stateStopped  state = "STOPPED"
)

// Coordinator hosts a process's lifecycle. Hooks and monitors are declared
// while it waits; one blocking [Coordinator.Run] then owns the whole sequence —
// parallel startup, readiness, monitored running, and the timed drain.
type Coordinator struct {
	mu         sync.Mutex
	state      state
	onStartup  []func(context.Context) error
	onShutdown []func(context.Context) error
	onReady    []func()
	monitors   []<-chan error
}

// New returns a Coordinator awaiting registrations and [Coordinator.Run].
func New() *Coordinator {
	return &Coordinator{
		state: stateWaiting,
	}
}

// OnStartup registers a hook [Coordinator.Run] launches concurrently with
// every other startup hook, passing the run context. A non-nil return fails
// startup. Registration after Run panics.
func (c *Coordinator) OnStartup(fn func(context.Context) error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state != stateWaiting {
		panic("lifecycle: OnStartup after Run")
	}
	c.onStartup = append(c.onStartup, fn)
}

// OnShutdown registers a hook the drain runs concurrently with every other
// shutdown hook, passing a fresh drain context bounded by Run's timeout.
// Errors join [Coordinator.Run]'s return. Registration after Run panics.
func (c *Coordinator) OnShutdown(fn func(context.Context) error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state != stateWaiting {
		panic("lifecycle: OnShutdown after Run")
	}
	c.onShutdown = append(c.onShutdown, fn)
}

// OnReady registers a hook [Coordinator.Run] invokes synchronously, in
// registration order, once every startup hook has succeeded. Registration
// after Run panics.
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
// a source that stopped cleanly. Registration after Run panics.
func (c *Coordinator) Monitor(errs <-chan error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state != stateWaiting {
		panic("lifecycle: Monitor after Run")
	}
	c.monitors = append(c.monitors, errs)
}

// Ready reports whether the coordinator is running: true once every startup
// hook has succeeded, and false again the moment draining begins.
func (c *Coordinator) Ready() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state == stateRunning
}

// Run drives the lifecycle and blocks until it ends. It launches the startup
// hooks and waits; a failure drains what did start and returns the joined
// errors, with readiness never flipped. Otherwise it marks the coordinator
// ready, invokes the OnReady hooks, and blocks until ctx is cancelled or a
// monitored channel yields an error. It then drains — every shutdown hook
// against a fresh context bounded by drainTimeout — and returns nil for a
// cancellation with a clean drain, or the joined startup, run, and shutdown
// errors otherwise. Hooks still running when the drain times out continue on
// the expired context, their late errors dropped. Run is legal exactly once; a
// second call panics.
func (c *Coordinator) Run(
	ctx context.Context,
	drainTimeout time.Duration,
) error {
	c.mu.Lock()
	if c.state != stateWaiting {
		c.mu.Unlock()
		panic("lifecycle: Run called twice")
	}
	c.state = stateStarting
	c.mu.Unlock()

	runCtx, fail := context.WithCancelCause(ctx)
	defer fail(nil)

	if err := c.startup(runCtx); err != nil {
		return errors.Join(err, c.drain(drainTimeout))
	}

	if ctx.Err() != nil {
		return c.drain(drainTimeout)
	}

	c.setState(stateRunning)
	for _, fn := range c.onReady {
		fn()
	}

	c.watch(runCtx, fail)
	<-runCtx.Done()

	var runErr error
	if cause := context.Cause(runCtx); !errors.Is(cause, context.Canceled) {
		runErr = fmt.Errorf("run: %w", cause)
	}

	return errors.Join(runErr, c.drain(drainTimeout))
}

func (c *Coordinator) drain(timeout time.Duration) error {
	c.setState(stateDraining)

	var err error
	if len(c.onShutdown) > 0 {
		err = c.shutdown(timeout)
	}
	c.setState(stateStopped)
	return err
}

func (c *Coordinator) setState(s state) {
	c.mu.Lock()
	c.state = s
	c.mu.Unlock()
}

func (c *Coordinator) shutdown(timeout time.Duration) error {
	drainCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var mu sync.Mutex
	var failed []error
	record := func(err error) {
		if err != nil {
			mu.Lock()
			failed = append(failed, err)
			mu.Unlock()
		}
	}

	var wg sync.WaitGroup
	for _, fn := range c.onShutdown {
		wg.Go(func() { record(fn(drainCtx)) })
	}

	drained := make(chan struct{})
	go func() {
		wg.Wait()
		close(drained)
	}()

	select {
	case <-drained:
	case <-drainCtx.Done():
		select {
		case <-drained:
		default:
			record(fmt.Errorf(
				"drain timeout after %v: %w", timeout, drainCtx.Err(),
			))
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if len(failed) > 0 {
		return fmt.Errorf("shutdown: %w", errors.Join(failed...))
	}
	return nil
}

func (c *Coordinator) startup(ctx context.Context) error {
	var mu sync.Mutex
	var failed []error

	var wg sync.WaitGroup
	for _, fn := range c.onStartup {
		wg.Go(func() {
			if err := fn(ctx); err != nil {
				mu.Lock()
				failed = append(failed, err)
				mu.Unlock()
			}
		})
	}
	wg.Wait()

	if len(failed) > 0 {
		return fmt.Errorf("startup: %w", errors.Join(failed...))
	}
	return nil
}

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
