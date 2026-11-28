// Package sse reads a server-sent event stream, the text/event-stream format
// of the WHATWG HTML standard, as its bytes arrive.
//
// It belongs in go-core because the pattern is process-level and universal:
// clients read event streams and servers write them, so the format serves
// neither alone. It uses only the standard library and knows nothing of HTTP:
// it reads any [io.Reader]. Its events are neutral; nothing maps them to or
// from CloudEvents.
//
// The package exports:
//
//   - [Event], one dispatched event: its type, data, and last event ID
//   - [Decoder], which reads the events of one stream, and [NewDecoder],
//     which returns one
//   - [Decoder.Next], which returns the stream's next event, or io.EOF at
//     its end
//   - [Decoder.Retry], the reconnection time the stream last set
package sse
