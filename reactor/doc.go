// Package reactor joins one source of occurrences to one function for the
// process lifetime.
//
// The package exports:
//
//   - [Source], what delivers occurrences, and [Func], what handles one
//   - [Reactor], which runs one Source into one Func, and [New], which
//     returns one
//   - [Reactor.Start], which starts the source, and [Reactor.Shutdown],
//     which drains it
//   - [Reactor.Ready], whether the source is receiving, and [Reactor.Err],
//     the channel that reports a failure while running
//   - [Option], which configures a Reactor, with [Grace], which bounds how
//     long its drain lets the handling in flight run, and [GraceWithin],
//     which derives that bound from a coordinator's drain timeout
//   - [Draining], which returns a handler's drain signal
//   - [Every], a Source that delivers the time on each tick of an interval
//   - [Wake], which returns a [Waker]: a Source that delivers the time on
//     each tick of an interval and on each [Waker.Nudge], coalescing the
//     nudges that come before a delivery into one
//   - [Gate], a quiesce gate: background work holds it with [Gate.Shared],
//     and an operation that must run with that work paused holds it with
//     [Gate.Exclusive], each acquisition honoring a context
//
// # Lifecycle
//
// A [Reactor] is a lifecycle component, with the Start, Shutdown, and Ready
// methods other infrastructure exposes, plus [Reactor.Err] for a failure
// while running. The package imports only the standard library and knows
// nothing of the coordinator: the composition root makes a reactor a graph
// node's value, and the lifecycle package's Coordinator infers from those
// methods that it starts with its layer, drains before the nodes it uses,
// reports readiness under its node's name, and ends the run on its failure.
//
// # Drain
//
// Start detaches the reactor from the context it is given, so cancelling the
// run context at a signal does not interrupt handling before the reactor's
// layer drains. Shutdown drains in two phases: it stops the source from
// taking new occurrences and waits for the handling in flight, then, once
// the [Grace] period passes, cancels the handlers' contexts and waits for
// them to unwind. Without Grace, a handler's context is cancelled only when
// Shutdown's own context ends, which is the drain deadline. A handler that
// runs in several steps, such as a loop over bounded passes, reads
// [Draining] from its context to stop between steps once the drain begins,
// rather than run until the grace cancels it.
package reactor
