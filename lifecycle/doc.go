// Package lifecycle hosts process startup, readiness, and graceful shutdown.
//
// A [Coordinator]'s life divides into an inert declaration phase and a single
// blocking call that owns everything after. While waiting, the registration
// calls — [Coordinator.OnStartup], [Coordinator.OnShutdown],
// [Coordinator.OnReady], [Coordinator.Monitor] — declare what starting,
// draining, becoming ready, and runtime failure mean for this service; nothing
// executes. [Coordinator.Run] then drives the whole sequence and returns one
// joined error. Internally the coordinator moves WAITING → STARTING → RUNNING
// → DRAINING → STOPPED; registration is legal only while waiting, Run exactly
// once, and each violation panics — a late registration is a programming
// error, not a runtime condition.
//
// # Context ownership
//
// The caller owns the signal context: a composition root builds one with
// signal.NotifyContext and passes it to Run. Run derives the run context —
// cancelled by the signal, by a monitored failure, or when Run ends — and
// passes it to every startup hook; work that outlives its hook keeps watching
// that context. The coordinator installs no signal handlers of its own.
//
// # Startup and readiness
//
// Run launches every startup hook concurrently and waits. If any hook returns
// an error, the coordinator drains what did start and Run returns the joined
// failures wrapped "startup:" — readiness never flips, so a probe backed by
// [Coordinator.Ready] cannot report a partially started process. On success
// the coordinator is ready and the OnReady hooks run synchronously, in
// registration order. [Coordinator] satisfies [ReadinessChecker], the
// contract a /readyz endpoint consumes; readiness is non-monotonic, false
// again the moment draining begins.
//
// # Running and monitors
//
// While running, Run blocks until the signal context is cancelled — the clean
// path — or a monitored channel yields a non-nil error, which ends the run
// and joins Run's return wrapped "run:". A nil received error is ignored, and
// a channel closing retires its watcher quietly: that is the expected end of
// a source that stopped cleanly.
//
// # Drain
//
// The drain runs every shutdown hook concurrently, each passed a fresh drain
// context bounded by Run's timeout and derived from context.Background, so
// cleanup has its whole budget regardless of the cancelled run context. Hook
// errors join Run's return wrapped "shutdown:"; a drain that outlives the
// timeout adds an error wrapping context.DeadlineExceeded while the
// unfinished hooks continue on the expired context — the coordinator cannot
// stop a goroutine — so hooks that honor their context stop promptly. Run
// returns nil exactly when a signal-driven exit drained cleanly.
package lifecycle
