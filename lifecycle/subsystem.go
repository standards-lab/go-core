package lifecycle

import "context"

// Starter is a value that starts. A [graph.Dependency] whose Value
// implements it takes part in a [Coordinator]'s startup.
type Starter interface {
	Start(ctx context.Context) error
}

// Stopper is a value that shuts down. A [graph.Dependency] whose Value
// implements it takes part in a [Coordinator]'s shutdown.
type Stopper interface {
	Shutdown(ctx context.Context) error
}

// Subsystem is a value with a whole lifecycle, one that starts and shuts
// down: a [graph.Dependency] whose Value implements it takes part in both
// phases.
type Subsystem interface {
	Starter
	Stopper
}

// ReadyStarter is a [Starter] that reports whether what it started is
// ready: the value [Coordinator.Checks] infers a readiness check from. A
// value with Ready and no Start isn't one.
type ReadyStarter interface {
	Starter
	ReadinessChecker
}
