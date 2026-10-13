// Package event defines the CloudEvents 1.0 envelope and the types a domain
// uses to raise its events.
//
// The package exports:
//
//   - [SpecVersion], the CloudEvents version it implements, and [Event], one
//     event's context attributes and data, with [Event.Validate], which
//     reports every way an event breaks the specification
//   - [Header], a message's headers, and [Encode] and [Decode], which map an
//     Event to and from binary content mode
//   - [EventKind], one event a domain raises, its type paired with the Go
//     type of its data; [Define], which declares one, and [CheckType], the
//     rule its type follows; [EventKind.Type], and [EventKind.Raise], which
//     adds an event of the kind to a [Queue]
//   - [Queue], the events one command raises, with [Queue.Events], a copy of
//     them, and [Queue.Err], every failure a raise held
//   - [Sink], what writes stamped events into a transaction, and [Recorder],
//     which emits a command's events through one, with [NewRecorder], which
//     returns one, and [Recorder.Emit], which wraps a command's body into
//     the function a transaction runs
//   - [Option], which configures a Recorder, with [Clock], the time it
//     stamps, and [IDs], how it mints an event id
//   - [Permanent], which marks a handler's error as one redelivery cannot
//     fix, and [IsPermanent], which reports the mark
//
// # Envelope
//
// An [Event] carries the context attributes a service sets, and [Encode] and
// [Decode] map it to and from binary content mode: each attribute is a
// ce-prefixed header, datacontenttype is the content-type header, and the
// data travels as the message body. [Header] is a plain map, so it converts
// to a NATS or HTTP header without this package importing either.
//
// # Raising and emitting
//
// A domain raises events, and a recorder emits them. A domain declares each
// event it raises as an [EventKind], which pairs the event type with the Go
// type of its data:
//
//	var Started = event.Define[StartedData]("exercise.started")
//
// A command raises its events into a [Queue] while it runs, and a
// [Recorder] emits them through its [Sink] into the command's own
// transaction, so an event is emitted only in the transaction of the command
// that makes it true. [Recorder.Emit] states when it stamps and writes.
//
// The transaction is a type parameter, so this package names no database.
// The composition root fixes the type when it builds the recorder over a
// sink, such as an outbox's, and passing a pool where the sink expects its
// transaction type fails to compile.
//
// # Handling
//
// A handler that must never see an event again wraps its error with
// [Permanent].
//
// The package uses the standard library alone.
package event
