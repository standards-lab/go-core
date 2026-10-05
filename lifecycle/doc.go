// Package lifecycle hosts process startup, readiness, and graceful shutdown.
//
// The package exports:
//
//   - [Coordinator], which runs one process's lifecycle; the zero value is
//     ready to use, and [New] returns one
//   - [Service], one subsystem's lifecycle declaration
//   - [StageRoot], the stage reserved for the request edge
//   - [ReadinessChecker], the interface a readiness probe consumes
//   - [Check], a readiness check paired with its probe name
//
// A [Coordinator]'s life divides into an inert declaration phase and a single
// blocking call that owns everything after. Before Run, the registration
// calls — [Coordinator.Add], [Coordinator.OnStartup], [Coordinator.OnShutdown],
// [Coordinator.OnReady], [Coordinator.Monitor] — declare what starting,
// draining, becoming ready, and runtime failure mean for this service; nothing
// executes. [Coordinator.Run] then drives the whole sequence and returns one
// joined error. Registration is legal only before Run, and Run is legal
// exactly once and with a positive drain timeout. Each violation panics: it
// is a wiring mistake, not a runtime condition.
//
// # Services and hooks
//
// A subsystem with a name and a place in the process's dependency order is a
// [Service], added with [Coordinator.Add]. Its Stage orders it against the
// other services: numbered stages start ascending with a barrier between
// them, services within a stage start concurrently, and [StageRoot] reserves
// the request edge, started after every numbered stage and drained first. The
// drain reverses the stages, shutting down only services whose Start
// succeeded. Hooks carry no ordering and bracket the stages: OnStartup hooks
// run before the first stage, OnShutdown hooks after the last drain stage. A
// subsystem that must order against the services is itself a Service; a
// process-level callback with no name or order is a hook.
//
// # Context ownership
//
// The caller owns the signal context: the application's entrypoint builds
// one — the process package's SignalContext — and passes it through the
// composition root's run call to Run. Run derives the run context, which is
// cancelled by the signal, by a startup or monitored failure, or when Run
// ends. Run passes the run context to every startup hook and service Start;
// work that outlives its call keeps watching that context. The coordinator
// installs no signal handlers of its own.
//
// # Startup and readiness
//
// Run launches every startup hook concurrently and waits, then starts the
// service stages. The first hook or service to return an error cancels the
// run context, so the rest of its phase can stop early. The coordinator then
// drains what did start, and Run returns the joined failures wrapped
// "startup:", each service failure labeled with its name. Run drops the
// errors that wrap context.Canceled and arrive once the run context is
// cancelled and a failure is on record: they are consequences of the first
// failure.
//
// A signal during startup is the signal-driven exit, not a failure: when
// every startup error wraps context.Canceled and the signal context is
// cancelled, Run drains and returns only the drain's errors.
//
// Readiness never flips during a failed startup, so a probe backed by
// [Coordinator.Ready] cannot report a partially started process. On success
// the coordinator is ready and the OnReady hooks run synchronously, in
// registration order. [Coordinator] satisfies [ReadinessChecker], the
// contract a /readyz endpoint consumes. Readiness is non-monotonic: it is
// false again the moment draining begins. [Coordinator.Checks] exposes the
// services' named checks in start order for a probe aggregate to consume.
//
// # Running and monitors
//
// While running, Run blocks until the signal context is cancelled (the clean
// path) or a monitored channel yields a non-nil error, which ends the run
// and joins Run's return wrapped "run:". Run ignores a nil received error,
// and a channel closing retires its watcher quietly: that is the expected end
// of a source that stopped cleanly.
//
// # Drain
//
// The drain runs the root stage first, the numbered stages descending, and
// finally every shutdown hook. Each phase runs its participants concurrently,
// and every participant receives a fresh drain context bounded by Run's
// timeout and derived from context.Background, so cleanup has its whole
// budget regardless of the cancelled run context. Errors join Run's return
// wrapped "shutdown:", service failures labeled by name.
//
// A drain that outlives the timeout adds one error wrapping
// context.DeadlineExceeded. Unfinished work continues on the expired context
// and its late errors are dropped, because the coordinator cannot stop a
// goroutine. The remaining phases are still attempted, so participants that
// honor their context stop promptly. Run returns nil exactly when a
// signal-driven exit drained cleanly.
package lifecycle
