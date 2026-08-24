// Package lifecycle hosts process startup, readiness, and graceful shutdown.
//
// A [Coordinator]'s life divides into an inert declaration phase and a single
// blocking call that owns everything after. While waiting, the registration
// calls — [Coordinator.Add], [Coordinator.OnStartup], [Coordinator.OnShutdown],
// [Coordinator.OnReady], [Coordinator.Monitor] — declare what starting,
// draining, becoming ready, and runtime failure mean for this service; nothing
// executes. [Coordinator.Run] then drives the whole sequence and returns one
// joined error. Internally the coordinator moves WAITING → STARTING → RUNNING
// → DRAINING → STOPPED; registration is legal only while waiting, Run exactly
// once, and each violation panics — a late registration is a programming
// error, not a runtime condition.
//
// # Services and hooks
//
// A subsystem with a name and a place in the process's dependency order is a
// [Service], added with [Coordinator.Add]. Its Stage orders it against the
// other services: numbered stages start ascending with a barrier between
// them, services within a stage start concurrently, and [StageRoot] reserves
// the request edge — started after every numbered stage, drained first. The
// drain reverses the stages, shutting down only services whose Start
// succeeded. Hooks carry no ordering and bracket the stages: OnStartup hooks
// run before the first stage, OnShutdown hooks after the last drain stage. A
// subsystem that must order against the services is itself a Service; a
// process-level callback with no name or order is a hook.
//
// # Context ownership
//
// The caller owns the signal context: a composition root builds one — the
// process package's SignalContext — and passes it to Run. Run derives the run context —
// cancelled by the signal, by a monitored failure, or when Run ends — and
// passes it to every startup hook; work that outlives its hook keeps watching
// that context. The coordinator installs no signal handlers of its own.
//
// # Startup and readiness
//
// Run launches every startup hook concurrently and waits, then starts the
// service stages. If a hook or a service returns an error, the coordinator
// drains what did start and Run returns the joined failures wrapped
// "startup:", each service failure labeled with its name — readiness never
// flips, so a probe backed by [Coordinator.Ready] cannot report a partially
// started process. On success the coordinator is ready and the OnReady hooks
// run synchronously, in registration order. [Coordinator] satisfies
// [ReadinessChecker], the contract a /readyz endpoint consumes; readiness is
// non-monotonic, false again the moment draining begins. [Coordinator.Checks]
// exposes the services' named checks in start order for a probe aggregate to
// consume.
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
// The drain runs the root stage first, the numbered stages descending, and
// finally every shutdown hook — each phase concurrent within itself, every
// participant passed a fresh drain context bounded by Run's timeout and
// derived from context.Background, so cleanup has its whole budget
// regardless of the cancelled run context. Errors join Run's return wrapped
// "shutdown:", service failures labeled by name. A drain that outlives the
// timeout adds one error wrapping context.DeadlineExceeded while unfinished
// work continues on the expired context — the coordinator cannot stop a
// goroutine — and the remaining phases are still attempted, so participants
// that honor their context stop promptly. Run returns nil exactly when a
// signal-driven exit drained cleanly.
package lifecycle
