// Package event defines the CloudEvents 1.0 envelope.
//
// An [Event] carries the context attributes a service sets, and [Encode] and
// [Decode] map it to and from binary content mode: each attribute is a
// ce-prefixed header, datacontenttype is the content-type header, and the
// data travels as the message body. [Header] is a plain map, so it converts
// to a NATS or HTTP header without this package importing either. The
// package uses the standard library alone.
package event
