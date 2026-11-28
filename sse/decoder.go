package sse

import (
	"bufio"
	"bytes"
	"io"
	"math"
	"time"
	"unicode/utf8"
)

// Event is one event a stream dispatched.
type Event struct {
	// Type is the event's type, from its block's last event field. Empty
	// means the stream named none, which the standard reads as "message".
	Type string
	// Data is the event's data: its data fields' values joined with "\n".
	// It is empty for an event whose only data field had an empty value.
	// Each event's Data is its own slice, which no later read touches.
	Data []byte
	// ID is the stream's last event ID when the event was dispatched: the
	// last id field's value, in this block or an earlier one, that held no
	// NUL. Empty means none was set, or an empty id field cleared it.
	ID string
}

// Decoder reads the events of one stream. It isn't safe for concurrent use.
//
// One leading byte order mark is stripped, and no other. A line ends at CR,
// LF, or CRLF. A CR is remembered rather than settled by reading past it, so
// an LF that starts the next read completes the CRLF, and an event
// dispatches as soon as its blank line arrives. A line with no colon is a
// field name with an empty value, and one space after the colon is removed
// from the value. A line or event the stream's end cuts off is discarded.
//
// Each line is decoded as UTF-8 with replacement, as the standard requires:
// each invalid byte sequence becomes one U+FFFD per maximal subpart, as the
// Encoding Standard's UTF-8 decoder yields, so every Type, Data, and ID is
// valid UTF-8.
//
// A Decoder holds only the line and the event it is building, with no size
// limit: a stream's longest line or event is the most it buffers. The caller
// bounds a stream it doesn't trust through its context or transport.
type Decoder struct {
	r          *bufio.Reader
	bomPending bool
	afterCR    bool
	line       []byte

	typ    string
	data   bytes.Buffer
	lastID string

	retry    time.Duration
	retrySet bool
}

// NewDecoder returns a Decoder reading the stream r.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: bufio.NewReader(r), bomPending: true}
}

// Next reads the stream until a blank line ends a block that holds a data
// field, and returns that block's event. Comments, unknown fields, retry
// fields, and blocks without data dispatch nothing, and Next reads on past
// them. At the stream's end it returns io.EOF, and a read error is returned
// unchanged.
func (d *Decoder) Next() (Event, error) {
	for {
		line, err := d.readLine()
		if err != nil {
			return Event{}, err
		}
		if len(line) == 0 {
			if e, ok := d.dispatch(); ok {
				return e, nil
			}
			continue
		}
		d.field(decodeUTF8(line))
	}
}

// Retry reports the reconnection time the stream's last valid retry field
// set, in milliseconds. A valid value is one or more ASCII digits whose
// duration fits in a [time.Duration]; any other value, including one that
// overflows, is ignored. Retry reports false until a valid retry field
// arrives.
func (d *Decoder) Retry() (time.Duration, bool) {
	return d.retry, d.retrySet
}

func (d *Decoder) readLine() ([]byte, error) {
	d.line = d.line[:0]
	for {
		b, err := d.r.ReadByte()
		if err != nil {
			return nil, err
		}
		if d.afterCR {
			d.afterCR = false
			if b == '\n' {
				continue
			}
		}
		switch b {
		case '\r':
			d.afterCR = true
			return d.stripBOM(d.line), nil
		case '\n':
			return d.stripBOM(d.line), nil
		}
		d.line = append(d.line, b)
	}
}

var bom = []byte("\uFEFF")

func (d *Decoder) stripBOM(line []byte) []byte {
	if d.bomPending {
		d.bomPending = false
		return bytes.TrimPrefix(line, bom)
	}
	return line
}

func decodeUTF8(line []byte) []byte {
	if utf8.Valid(line) {
		return line
	}
	out := make([]byte, 0, len(line)+2*utf8.UTFMax)
	var (
		cp           rune
		needed, seen int
		lower, upper byte = 0x80, 0xBF
	)
	for i := 0; i < len(line); {
		b := line[i]
		if needed == 0 {
			i++
			switch {
			case b <= 0x7F:
				out = append(out, b)
			case 0xC2 <= b && b <= 0xDF:
				needed, cp = 1, rune(b&0x1F)
			case 0xE0 <= b && b <= 0xEF:
				switch b {
				case 0xE0:
					lower = 0xA0
				case 0xED:
					upper = 0x9F
				}
				needed, cp = 2, rune(b&0x0F)
			case 0xF0 <= b && b <= 0xF4:
				switch b {
				case 0xF0:
					lower = 0x90
				case 0xF4:
					upper = 0x8F
				}
				needed, cp = 3, rune(b&0x07)
			default:
				out = utf8.AppendRune(out, utf8.RuneError)
			}
			continue
		}
		if b < lower || b > upper {
			out = utf8.AppendRune(out, utf8.RuneError)
			cp, needed, seen, lower, upper = 0, 0, 0, 0x80, 0xBF
			continue
		}
		i++
		lower, upper = 0x80, 0xBF
		cp = cp<<6 | rune(b&0x3F)
		if seen++; seen == needed {
			out = utf8.AppendRune(out, cp)
			cp, needed, seen = 0, 0, 0
		}
	}
	if needed != 0 {
		out = utf8.AppendRune(out, utf8.RuneError)
	}
	return out
}

func (d *Decoder) field(line []byte) {
	if line[0] == ':' {
		return
	}
	name, value, _ := bytes.Cut(line, []byte(":"))
	value = bytes.TrimPrefix(value, []byte(" "))
	switch string(name) {
	case "event":
		d.typ = string(value)
	case "data":
		d.data.Write(value)
		d.data.WriteByte('\n')
	case "id":
		if bytes.IndexByte(value, 0) < 0 {
			d.lastID = string(value)
		}
	case "retry":
		if ms, ok := parseRetry(value); ok {
			d.retry, d.retrySet = ms, true
		}
	}
}

const maxRetryMS = math.MaxInt64 / int64(time.Millisecond)

func parseRetry(value []byte) (time.Duration, bool) {
	if len(value) == 0 {
		return 0, false
	}
	var ms int64
	for _, b := range value {
		if b < '0' || b > '9' {
			return 0, false
		}
		digit := int64(b - '0')
		if ms > (maxRetryMS-digit)/10 {
			return 0, false
		}
		ms = ms*10 + digit
	}
	return time.Duration(ms) * time.Millisecond, true
}

func (d *Decoder) dispatch() (Event, bool) {
	typ := d.typ
	d.typ = ""
	if d.data.Len() == 0 {
		return Event{}, false
	}
	data := bytes.Clone(bytes.TrimSuffix(d.data.Bytes(), []byte("\n")))
	d.data.Reset()
	return Event{Type: typ, Data: data, ID: d.lastID}, true
}
