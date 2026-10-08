// Package lifecycle runs a built [graph.System]: process startup, readiness,
// and graceful shutdown.
//
// The package exports:
//
//   - [Starter] and [Stopper], the single-method interfaces a dependency's
//     value implements to take part in startup and in shutdown, and
//     [Subsystem], which embeds both
//   - [Config], the Coordinator's configuration, with [Config.Merge] and
//     [Config.Finalize], and [Env], the environment-variable name Finalize
//     reads, composed by [NewEnv]
//   - [Coordinator], which runs one System once, and [New], which returns
//     one
//   - [Coordinator.Exec], which starts the System, runs a function, and
//     shuts down: the one-shot form a CLI command uses
//   - [Coordinator.Run], which starts the System, serves until its context
//     ends, and shuts down: the long-running form a service uses
//   - [Coordinator.OnReady], which registers a hook run once the System is
//     ready, and [Coordinator.Monitor], which registers a channel whose
//     failure ends the run
//   - [Coordinator.Ready], whether the System is ready, and
//     [Coordinator.Checks], the System's readiness checks
//   - [ReadinessChecker], the interface a readiness probe consumes
//   - [Check], a readiness check paired with its probe name
//   - [Readiness], a graph node's value that reports its Coordinator's
//     readiness to the nodes that depend on it
//
// A [Coordinator]'s life divides into an inert declaration phase and a single
// blocking call that owns everything after. Nothing is registered for
// startup or shutdown: the participants are the System's dependencies, and
// their order is the System's layers. Before the call, [Coordinator.OnReady]
// and [Coordinator.Monitor] declare what becoming ready and runtime failure
// mean; nothing executes. [Coordinator.Exec] or [Coordinator.Run] then
// drives the whole sequence and returns one joined error. Registration is
// legal only before that call, and the call is legal exactly once. Each
// violation panics, as does [New] on a nil System, a ShutdownTimeout that
// is not positive, or a [Readiness] already bound to another Coordinator:
// they are wiring mistakes, not runtime conditions.
//
// # Participation
//
// A [graph.Dependency] takes part only through its Value's methods, in each
// phase the Value implements: a [Starter] starts, a [Stopper] shuts down,
// and a [Subsystem], which is both, does both. A Value that implements one
// alone is a start-only or stop-only participant, and a Value that
// implements neither takes no part. A node changes its part only by
// changing its value, such as a test's substitute through graph's Replace.
//
// # Context ownership
//
// The caller owns the signal context: the application's entrypoint builds
// one — the process package's SignalContext — and passes it to Exec or Run.
// The Coordinator derives the run context from it, which is cancelled by
// the signal, by a startup or monitored failure, or when shutdown begins.
// Every Start, and Exec's function, receives the run context; work that
// outlives its call keeps watching that context. The coordinator installs no
// signal handlers of its own.
//
// # Startup and readiness
//
// The layers start lowest first, as [graph.System.Layers] orders them; a
// layer's participants start concurrently, and the next layer starts only
// once the whole layer has. The first start failure cancels the run
// context, so the rest of its layer can stop early, and no higher layer
// starts. The errors that wrap context.Canceled and arrive once the run
// context is cancelled and a failure is on record are dropped: they are
// consequences of the first failure. The startup failures are joined,
// each labelled with its dependency's name, and wrapped "startup:".
//
// A context that ends before or during startup stops it: no further layer
// starts. Run treats that as the signal-driven exit, not a failure, when
// every startup error wraps the context's error, and returns only the
// shutdown's errors. Exec returns the cancellation, since its function
// never ran.
//
// Readiness never flips during a failed startup, so a probe backed by
// [Coordinator.Ready] cannot report a partially started process. On success
// the coordinator is ready, and Exec and Run alike invoke the OnReady hooks
// synchronously, in registration order, before Exec's function runs or Run
// serves. [Coordinator] satisfies [ReadinessChecker], the contract a /readyz
// endpoint consumes. Readiness is non-monotonic: it is false again the
// moment shutdown begins.
//
// [Coordinator.Checks] lists the System's own readiness checks: a [Check]
// for each value that implements [ReadinessChecker], named by its node, in
// layer order and, within a layer, definition order.
//
// A node that serves readiness, such as a health handler, reaches the
// Coordinator through a [Readiness]: the application defines a node whose
// value is new(lifecycle.Readiness), the zero value, and the handler's node
// uses it. There is no constructor, since the zero value is complete, and
// the node's value is the pointer, since binding writes to it. [New] binds
// every *Readiness in its System to the Coordinator it returns; a bound
// Readiness reports that Coordinator's Ready and Checks, and is not among
// them. An unbound Readiness is not ready and has no checks. A Readiness
// binds once, so a node whose constructor returns one shared Readiness
// across two Builds, each given to a New, panics in the second.
//
// # Running and monitors
//
// Exec runs its function under the run context once every layer has
// started, and its error joins Exec's return. Run blocks until the signal
// context is cancelled (the clean path) or a monitored channel yields a
// non-nil error, which ends the run and joins Run's return wrapped "run:".
// Run ignores a nil received error, and a channel closing retires its
// watcher quietly: that is the expected end of a source that stopped
// cleanly.
//
// # Shutdown
//
// Shutdown runs on every path: after Exec's function returns, whatever its
// error, after Run's context ends or a monitor fails, and after a startup
// failure or cancellation. It cancels the run context, then shuts down every
// participant of every layer that began to start, the last layer first,
// each layer concurrently. That includes a participant whose Start failed,
// so a dependency constructed but not started leaks nothing; the start
// error stays the error reported.
//
// Shutdown runs under one context derived from context.Background and
// bounded by [Config].ShutdownTimeout, so cleanup has its whole budget
// regardless of the cancelled run context. Errors are labelled with the
// dependency's name, joined, wrapped "shutdown:", and joined with Exec's or
// Run's result, so a failed shutdown fails an otherwise clean run.
//
// The first layer that outlives the deadline adds one error wrapping
// context.DeadlineExceeded. Its unfinished shutdowns continue on the expired
// context and their late errors are dropped, because the coordinator cannot
// stop a goroutine. Each remaining layer still starts its shutdowns on the
// expired context, so participants that honor their context stop promptly,
// and shutdown does not wait for them. Run returns nil exactly when a
// signal-driven exit shut down cleanly.
package lifecycle
