package lifecycle

// ReadinessChecker reports whether a subsystem is ready to serve.
// [Coordinator] satisfies it, as does any subsystem a readiness probe
// aggregates.
type ReadinessChecker interface {
	Ready() bool
}

// Check pairs a readiness check with the name a probe reports for it. A
// Check with a nil Checker reports not ready, so a subsystem that failed to
// construct fails the probe instead of vanishing from it.
type Check struct {
	Name    string
	Checker ReadinessChecker
}
