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
//   - [Reactor.Ready], whether the reactor runs and its source is
//     receiving, and [Reactor.Err],
//     the channel that reports a failure while running
//   - [Option], which configures a Reactor, with [CancelAfter], which
//     bounds how long its drain lets the handling in flight run before it
//     cancels the handlers, and [CancelWithin], which derives that bound
//     from a coordinator's drain timeout
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
// A reactor's handling outlives the run context and ends at its layer's
// drain: [Reactor.Start] and [Reactor.Shutdown] state the two phases, and
// [CancelAfter] bounds the first. A handler that runs in several steps,
// such as a loop over bounded passes, reads [Draining] to stop between
// steps once the drain begins.
package reactor
