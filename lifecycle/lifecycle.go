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

// Coordinator hosts a process's lifecycle: declare services, hooks, and
// monitors, then call Run once. The zero value is ready to use.
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
	return &Coordinator{}
}

// Add declares svc; Run starts and drains it with its stage. Add panics on
// an empty or duplicate Name, a negative Stage, a Service declaring nothing,
// or a call after Run.
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
	if c.stages == nil {
		c.stages = make(map[int][]*service)
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
// registration order, once startup has completed: every startup hook and
// every service stage succeeded. Registration after Run panics.
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

// Ready reports whether startup has completed and draining has not begun.
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

// Run drives startup, readiness, the run, and a drain bounded by
// drainTimeout, blocking until done. It returns nil for a cancellation of
// ctx with a clean drain, a startup that cancellation cut short included,
// else the joined startup, run, and shutdown errors. A second call, or a
// drainTimeout that is not positive, panics.
func (c *Coordinator) Run(
	ctx context.Context,
	drainTimeout time.Duration,
) error {
	if drainTimeout <= 0 {
		panic(fmt.Sprintf("lifecycle: Run: drain timeout %v is not positive", drainTimeout))
	}
	c.mu.Lock()
	if c.state != stateWaiting {
		c.mu.Unlock()
		panic("lifecycle: Run called twice")
	}
	c.state = stateStarting
	c.mu.Unlock()

	runCtx, fail := context.WithCancelCause(ctx)
	defer fail(nil)

	failed := c.startup(runCtx, fail)
	// A signal before or during startup ends the run like one after it: a
	// startup cut short only by that cancellation drains as a clean exit.
	if ctx.Err() != nil && failed.only(context.Canceled) {
		return c.drain(drainTimeout)
	}
	if err := failed.join(); err != nil {
		return errors.Join(fmt.Errorf("startup: %w", err), c.drain(drainTimeout))
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

	err := c.shutdown(timeout)
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

	var failed failures
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
					failed.record(fmt.Errorf(
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
		await(launch(drainCtx, shutdowns, failed.record))
	}
	await(launch(drainCtx, c.onShutdown, failed.record))

	if err := failed.join(); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}

// startup runs the hooks, then the stages, and returns what failed. The
// first failure cancels ctx, so its siblings in the phase stop early, and
// the cancellations they return in consequence are not recorded.
func (c *Coordinator) startup(ctx context.Context, fail context.CancelCauseFunc) *failures {
	failed := &failures{}
	record := func(err error) {
		if failed.recordUnlessConsequent(ctx, err) {
			fail(err)
		}
	}

	phases := [][]func(context.Context) error{c.onStartup}
	for _, stage := range slices.Sorted(maps.Keys(c.stages)) {
		var starts []func(context.Context) error
		for _, svc := range c.stages[stage] {
			starts = append(starts, svc.start)
		}
		phases = append(phases, starts)
	}
	for _, phase := range phases {
		<-launch(ctx, phase, record)
		if failed.any() {
			break
		}
	}
	return failed
}

// failures collects the errors of a phase's concurrent participants.
type failures struct {
	mu   sync.Mutex
	errs []error
}

func (f *failures) record(err error) {
	if err == nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.errs = append(f.errs, err)
}

// recordUnlessConsequent records err and reports true, unless err is nil
// or a cancellation consequent on a failure already recorded: it wraps
// context.Canceled, ctx is cancelled, and a failure is on record. The check
// and the record hold one lock, so two failures cannot both see none.
func (f *failures) recordUnlessConsequent(ctx context.Context, err error) bool {
	if err == nil {
		return false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if errors.Is(err, context.Canceled) && ctx.Err() != nil && len(f.errs) > 0 {
		return false
	}
	f.errs = append(f.errs, err)
	return true
}

// any reports whether a failure is on record.
func (f *failures) any() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.errs) > 0
}

// only reports whether every recorded failure wraps target; with none on
// record it is true.
func (f *failures) only(target error) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, err := range f.errs {
		if !errors.Is(err, target) {
			return false
		}
	}
	return true
}

func (f *failures) join() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return errors.Join(f.errs...)
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
