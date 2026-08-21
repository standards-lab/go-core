package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"sync"
	"time"
)

// StageRoot marks the root stage: the request edge of the process, where an
// HTTP server belongs. Root services start only after every numbered stage
// is up, and drain first. The value is reserved; numbered stages count
// upward from 0.
const StageRoot = math.MaxInt

// Coordinator hosts a process's lifecycle. Services, hooks, and monitors are
// declared while it waits; one blocking [Coordinator.Run] then owns the whole
// sequence — the bracketing startup hooks, the service stages, readiness,
// monitored running, and the timed drain.
type Coordinator struct {
	mu         sync.Mutex
	state      state
	stages     map[int][]*service
	onStartup  []func(context.Context) error
	onShutdown []func(context.Context) error
	onReady    []func()
	monitors   []<-chan error
}

// New returns a Coordinator awaiting registrations and [Coordinator.Run].
func New() *Coordinator {
	return &Coordinator{
		state:  stateWaiting,
		stages: make(map[int][]*service),
	}
}

// Add declares svc on the coordinator. [Coordinator.Run] starts the services
// stage by stage — numbered stages ascending, [StageRoot] last — with every
// service in a stage started concurrently and a failure ending startup once
// its stage completes. The drain runs the stages in reverse: the root stage
// first, then the numbered stages descending, joining every error. Services
// order against each other through stages; hooks carry no ordering and
// bracket the stages instead. Add panics on an empty or duplicate Name, a
// negative Stage, a Service declaring none of Start, Shutdown, or Check, and
// registration after Run — each a wiring mistake surfaced at cold start.
func (c *Coordinator) Add(svc Service) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state != stateWaiting {
		panic("lifecycle: Add after Run")
	}
	if svc.Name == "" {
		panic("lifecycle: Add: empty service name")
	}
	if svc.Stage < 0 {
		panic(fmt.Sprintf(
			"lifecycle: Add: service %q: negative stage %d", svc.Name, svc.Stage,
		))
	}
	if svc.Start == nil && svc.Shutdown == nil && svc.Check == nil {
		panic(fmt.Sprintf("lifecycle: Add: service %q declares nothing", svc.Name))
	}
	for _, members := range c.stages {
		for _, existing := range members {
			if existing.Name == svc.Name {
				panic(fmt.Sprintf("lifecycle: Add: duplicate service %q", svc.Name))
			}
		}
	}
	c.stages[svc.Stage] = append(c.stages[svc.Stage], &service{Service: svc})
}

// OnStartup registers a hook [Coordinator.Run] launches concurrently with
// every other startup hook, before the first service stage, passing the run
// context. A non-nil return fails startup. Registration after Run panics.
func (c *Coordinator) OnStartup(fn func(context.Context) error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state != stateWaiting {
		panic("lifecycle: OnStartup after Run")
	}
	c.onStartup = append(c.onStartup, fn)
}

// OnShutdown registers a hook the drain runs concurrently with every other
// shutdown hook, after the last service stage, passing a fresh drain context
// bounded by Run's timeout. Errors join [Coordinator.Run]'s return.
// Registration after Run panics.
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

// Checks returns the added services' named readiness checks, in start order,
// skipping services without one. The coordinator itself is not included; an
// aggregate that wants process readiness prepends the coordinator under a
// name of its own choosing.
func (c *Coordinator) Checks() []Check {
	c.mu.Lock()
	defer c.mu.Unlock()
	var checks []Check
	for _, stage := range slices.Sorted(maps.Keys(c.stages)) {
		for _, svc := range c.stages[stage] {
			if svc.Check != nil {
				checks = append(checks, Check{Name: svc.Name, Checker: svc.Check})
			}
		}
	}
	return checks
}

// Run drives the lifecycle and blocks until it ends. It launches the startup
// hooks concurrently and, once every hook has succeeded, starts the services
// stage by stage; a failure drains what did start and returns the
// joined errors, with readiness never flipped. Otherwise it marks the
// coordinator ready, invokes the OnReady hooks, and blocks until ctx is
// cancelled or a monitored channel yields an error. It then drains — the
// root stage first, the numbered stages descending, and finally every
// shutdown hook, all against a fresh context bounded by drainTimeout — and
// returns nil for a cancellation with a clean drain, or the joined startup,
// run, and shutdown errors otherwise. Work still running when the drain
// times out continues on the expired context, its late errors dropped. Run
// is legal exactly once; a second call panics.
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
	if len(c.onShutdown) > 0 || len(c.stages) > 0 {
		err = c.shutdown(timeout)
	}
	c.setState(stateStopped)
	return err
}

// launch is the one concurrency pattern every phase shares: it runs fns
// concurrently, records what failed, and returns a channel that closes when
// the phase completes. Startup waits by receiving; the drain selects the
// channel against its deadline.
func launch(
	ctx context.Context,
	fns []func(context.Context) error,
	record func(error),
) <-chan struct{} {
	done := make(chan struct{})
	var wg sync.WaitGroup
	for _, fn := range fns {
		wg.Go(func() { record(fn(ctx)) })
	}
	go func() {
		wg.Wait()
		close(done)
	}()
	return done
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

	timedOut := false
	await := func(done <-chan struct{}) {
		select {
		case <-done:
		case <-drainCtx.Done():
			select {
			case <-done:
			default:
				if !timedOut {
					timedOut = true
					record(fmt.Errorf(
						"drain timeout after %v: %w", timeout, drainCtx.Err(),
					))
				}
			}
		}
	}

	for _, stage := range slices.Backward(slices.Sorted(maps.Keys(c.stages))) {
		var shutdowns []func(context.Context) error
		for _, svc := range c.stages[stage] {
			shutdowns = append(shutdowns, svc.shutdown)
		}
		await(launch(drainCtx, shutdowns, record))
	}
	await(launch(drainCtx, c.onShutdown, record))

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
	record := func(err error) {
		if err != nil {
			mu.Lock()
			failed = append(failed, err)
			mu.Unlock()
		}
	}

	<-launch(ctx, c.onStartup, record)
	if len(failed) > 0 {
		return fmt.Errorf("startup: %w", errors.Join(failed...))
	}

	for _, stage := range slices.Sorted(maps.Keys(c.stages)) {
		var starts []func(context.Context) error
		for _, svc := range c.stages[stage] {
			starts = append(starts, svc.start)
		}
		<-launch(ctx, starts, record)
		if len(failed) > 0 {
			return fmt.Errorf("startup: %w", errors.Join(failed...))
		}
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
