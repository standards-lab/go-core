// Package sse reads and writes a server-sent event stream, the
// text/event-stream format of the WHATWG HTML standard: it reads events as
// their bytes arrive and writes each event in one write.
//
// It uses only the standard library and knows nothing of HTTP: it reads any
// [io.Reader] and writes any [io.Writer]. Its events are neutral; nothing
// maps them to or from CloudEvents.
//
// The package exports:
//
//   - [Event], one dispatched event: its type, data, and last event ID
//   - [Decoder], which reads the events of one stream, and [NewDecoder],
//     which returns one
//   - [Decoder.Next], which returns the stream's next event
//   - [Decoder.Retry], the reconnection time the stream last set
//   - [Encoder], which writes the events of one stream, and [NewEncoder],
//     which returns one
//   - [Encoder.Encode], which writes one event
//   - [Encoder.Comment], which writes comment lines no reader dispatches
//   - [Encoder.Retry], which sets the stream's reconnection time
package sse
