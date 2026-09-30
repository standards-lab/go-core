package lifecycle

// ReadinessChecker reports whether a subsystem is ready to serve.
// [Coordinator] satisfies it, as does any subsystem a readiness probe
// aggregates.
type ReadinessChecker interface {
	Ready() bool
}

// Check pairs a readiness check with the name a probe reports for it; a
// probe treats a nil Checker as not ready.
type Check struct {
	Name    string
	Checker ReadinessChecker
}
