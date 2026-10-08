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

// step is one participant in a phase: a name that labels its errors, and an
// optional start and stop. A nil start counts as started; a nil stop is
// skipped.
type step struct {
	name        string
	start, stop func(context.Context) error
}

// participants returns the steps for one layer's dependencies, in layer
// order. A value takes part in each phase it implements: a [Starter] in
// startup, a [Stopper] in shutdown, and a value that is neither is skipped.
func participants(layer []graph.Dependency) []step {
	var steps []step
	for _, d := range layer {
		s := step{name: d.Name}
		if v, ok := d.Value.(Starter); ok {
			s.start = v.Start
		}
		if v, ok := d.Value.(Stopper); ok {
			s.stop = v.Shutdown
		}
		if s.start == nil && s.stop == nil {
			continue
		}
		steps = append(steps, s)
	}
	return steps
}

// engine records the phases that began to start, in order, and unwinds them
// in reverse. It is the executor under [Coordinator]. The zero value is ready
// to use.
type engine struct {
	phases [][]step
}

// start runs one phase: steps concurrently, each on its own goroutine, under
// ctx, recording each failure on failed labelled "name: err". The first
// failure recorded calls fail, cancelling ctx so the rest of the phase can
// stop early. It pushes the steps as one new phase for unwind, whether their
// starts succeed or not, so a step whose start failed is still stopped.
func (e *engine) start(
	ctx context.Context,
	fail context.CancelCauseFunc,
	steps []step,
	failed *failures,
) {
	e.phases = append(e.phases, steps)

	var wg sync.WaitGroup
	for _, s := range steps {
		if s.start == nil {
			continue
		}
		wg.Go(func() {
			err := s.start(ctx)
			if err == nil {
				return
			}
			err = fmt.Errorf("%s: %w", s.name, err)
			if failed.recordUnlessConsequent(ctx, err) {
				fail(err)
			}
		})
	}
	wg.Wait()
}

// unwind stops every pushed phase, the last first, each phase's steps
// concurrently, under one context derived from context.Background and
// bounded by timeout, so cleanup has its whole budget whatever became of the
// context start ran under. Stop errors are labelled "name: err". The first
// phase that outlives the deadline adds one error wrapping
// context.DeadlineExceeded; its unfinished stops continue on the expired
// context and their late errors are dropped. Each remaining phase still
// starts its stops on the expired context, and unwind does not wait for
// them. unwind leaves the engine empty, so a second call returns nothing.
func (e *engine) unwind(timeout time.Duration) []error {
	phases := e.phases
	e.phases = nil
	if len(phases) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var errs []error
	timedOut := false
	for _, phase := range slices.Backward(phases) {
		var (
			mu       sync.Mutex
			phaseErr []error
			wg       sync.WaitGroup
		)
		for _, s := range phase {
			if s.stop == nil {
				continue
			}
			wg.Go(func() {
				if err := s.stop(ctx); err != nil {
					mu.Lock()
					phaseErr = append(phaseErr, fmt.Errorf("%s: %w", s.name, err))
					mu.Unlock()
				}
			})
		}
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		finished := true
		select {
		case <-done:
		case <-ctx.Done():
			select {
			case <-done:
			default:
				finished = false
			}
		}
		// A phase cut short contributes what it recorded by the deadline;
		// its stragglers write to phaseErr after this snapshot, unread.
		mu.Lock()
		errs = append(errs, phaseErr...)
		mu.Unlock()
		if !finished && !timedOut {
			timedOut = true
			errs = append(errs, fmt.Errorf(
				"drain timeout after %v: %w", timeout, ctx.Err(),
			))
		}
	}
	return errs
}

// failures collects the errors of a phase's concurrent participants.
type failures struct {
	mu   sync.Mutex
	errs []error
}

// record records err, unless it is nil.
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
