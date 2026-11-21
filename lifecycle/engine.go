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

type step struct {
	name        string
	start, stop func(context.Context) error
}

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

// See docs/graph-design.md, "Implementation invariants".
type engine struct {
	phases [][]step
}

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

func (f *failures) any() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.errs) > 0
}

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
