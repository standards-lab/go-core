package sse

import (
	"errors"
	"io"
	"strconv"
	"strings"
	"time"
)

// Encoder writes events, comments, and retry fields to one stream. It isn't
// safe for concurrent use.
//
// Each call that writes sends all of its bytes to the writer in exactly one
// Write call, so a writer that flushes once per Write sends each event
// whole. A write error is returned unchanged. Every line ends with LF.
type Encoder struct {
	w   io.Writer
	buf []byte
}

// NewEncoder returns an Encoder writing to w.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

// Encode writes e as one event: an event line when Type is set, an id line
// when ID is set, one data line for each line of Data, and a blank line that
// dispatches it. Data is split into lines at CR, LF, and CRLF; an empty Data
// is written as one empty data line, so the event still dispatches.
//
// An empty ID writes no id line, so a reader keeps the last event ID it
// read. A Decoder reading the output returns e, except that CR and CRLF in
// Data come back as LF, invalid UTF-8 comes back replaced as the Decoder's
// doc describes, and an empty ID comes back as the last ID the stream set.
//
// Encode returns an error and writes nothing when Type or ID holds a CR or
// LF, or ID holds a NUL.
func (enc *Encoder) Encode(e Event) error {
	if strings.ContainsAny(e.Type, "\r\n") {
		return errors.New("sse: event type holds a line break")
	}
	if strings.ContainsAny(e.ID, "\r\n\x00") {
		return errors.New("sse: event id holds a line break or NUL")
	}
	b := enc.buf[:0]
	if e.Type != "" {
		b = appendField(b, "event", e.Type)
	}
	if e.ID != "" {
		b = appendField(b, "id", e.ID)
	}
	for _, line := range splitLines(string(e.Data)) {
		b = appendField(b, "data", line)
	}
	b = append(b, '\n')
	return enc.write(b)
}

// Comment writes text as one comment line for each of its lines, split at
// CR, LF, and CRLF. A reader skips comment lines, so no event dispatches.
func (enc *Encoder) Comment(text string) error {
	b := enc.buf[:0]
	for _, line := range splitLines(text) {
		b = appendField(b, "", line)
	}
	return enc.write(b)
}

// Retry writes a retry field holding d in whole milliseconds, with any
// fraction of a millisecond dropped, followed by a blank line. A reader sets
// its reconnection time from it and dispatches no event. A negative d is an
// error, and nothing is written.
func (enc *Encoder) Retry(d time.Duration) error {
	if d < 0 {
		return errors.New("sse: negative retry duration")
	}
	b := enc.buf[:0]
	b = append(b, "retry: "...)
	b = strconv.AppendInt(b, d.Milliseconds(), 10)
	b = append(b, "\n\n"...)
	return enc.write(b)
}

func (enc *Encoder) write(b []byte) error {
	enc.buf = b
	_, err := enc.w.Write(b)
	return err
}

func appendField(b []byte, name, value string) []byte {
	b = append(b, name...)
	b = append(b, ": "...)
	b = append(b, value...)
	return append(b, '\n')
}

func splitLines(s string) []string {
	return strings.Split(strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n"), "\n")
}
