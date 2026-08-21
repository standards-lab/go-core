package lifecycle

import (
	"context"
	"fmt"
)

// Service is one subsystem's lifecycle declaration: a name, the stage it
// belongs to, and up to three optional members — at least one must be set.
// A nil Start or Shutdown is a no-op within its stage, and a nil Check
// contributes no readiness. Shutdown runs only when the service's stage was
// reached and its Start succeeded, so it never has to tolerate a subsystem
// that never came up. Name labels startup errors, shutdown errors, and the
// readiness check, and must be unique on its coordinator.
type Service struct {
	Name     string
	Stage    int
	Start    func(context.Context) error
	Shutdown func(context.Context) error
	Check    ReadinessChecker
}

// service wraps a declaration with its runtime state and rules: whether
// Start succeeded, error naming, and nil-member skipping — so the
// coordinator's phases launch services without inspecting them.
type service struct {
	Service
	started bool
}

// start runs the declared Start, if any, and marks the service started so
// the drain unwinds it. started is written by one goroutine during the
// service's stage and read only after the stage's join, so it needs no lock.
func (s *service) start(ctx context.Context) error {
	if s.Start != nil {
		if err := s.Start(ctx); err != nil {
			return fmt.Errorf("%s: %w", s.Name, err)
		}
	}
	s.started = true
	return nil
}

// shutdown runs the declared Shutdown when the service actually started.
func (s *service) shutdown(ctx context.Context) error {
	if !s.started || s.Shutdown == nil {
		return nil
	}
	if err := s.Shutdown(ctx); err != nil {
		return fmt.Errorf("%s: %w", s.Name, err)
	}
	return nil
}
