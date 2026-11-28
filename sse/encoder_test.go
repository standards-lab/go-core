package sse_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/standards-lab/go-core/sse"
)

type recorder struct{ writes []string }

func (r *recorder) Write(p []byte) (int, error) {
	r.writes = append(r.writes, string(p))
	return len(p), nil
}

type failWriter struct{ err error }

func (w failWriter) Write([]byte) (int, error) { return 0, w.err }

func assertOneWrite(t *testing.T, r *recorder, want string) {
	t.Helper()
	if len(r.writes) != 1 {
		t.Fatalf("writes = %q, want exactly one write of %q", r.writes, want)
	}
	if r.writes[0] != want {
		t.Errorf("wrote %q, want %q", r.writes[0], want)
	}
}

var encodeTests = []struct {
	name  string
	event sse.Event
	want  string
}{
	{"data only", sse.Event{Data: []byte("a")}, "data: a\n\n"},
	{"type, id, and data", sse.Event{Type: "t", ID: "1", Data: []byte("a")}, "event: t\nid: 1\ndata: a\n\n"},
	{"empty data is one empty data line", sse.Event{}, "data: \n\n"},
	{"LF splits data", sse.Event{Data: []byte("a\nb")}, "data: a\ndata: b\n\n"},
	{"CR splits data", sse.Event{Data: []byte("a\rb")}, "data: a\ndata: b\n\n"},
	{"CRLF splits data once", sse.Event{Data: []byte("a\r\nb")}, "data: a\ndata: b\n\n"},
	{"LF CR splits data twice", sse.Event{Data: []byte("a\n\rb")}, "data: a\ndata: \ndata: b\n\n"},
	{"trailing line break is an empty last line", sse.Event{Data: []byte("a\n")}, "data: a\ndata: \n\n"},
	{"leading space kept", sse.Event{Type: " t", ID: " 1", Data: []byte(" a")}, "event:  t\nid:  1\ndata:  a\n\n"},
	{"colon in values", sse.Event{Type: "a:b", Data: []byte(":x")}, "event: a:b\ndata: :x\n\n"},
}

func TestEncode(t *testing.T) {
	for _, tt := range encodeTests {
		t.Run(tt.name, func(t *testing.T) {
			r := &recorder{}
			if err := sse.NewEncoder(r).Encode(tt.event); err != nil {
				t.Fatalf("Encode: %v", err)
			}
			assertOneWrite(t, r, tt.want)
		})
	}
}

func TestEncodeRejects(t *testing.T) {
	for _, tt := range []struct {
		name  string
		event sse.Event
	}{
		{"LF in type", sse.Event{Type: "a\nb"}},
		{"CR in type", sse.Event{Type: "a\rb"}},
		{"LF in id", sse.Event{ID: "a\nb"}},
		{"CR in id", sse.Event{ID: "a\rb"}},
		{"NUL in id", sse.Event{ID: "a\x00b"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := &recorder{}
			if err := sse.NewEncoder(r).Encode(tt.event); err == nil {
				t.Errorf("Encode(%q) = nil, want an error", tt.event)
			}
			if len(r.writes) != 0 {
				t.Errorf("writes = %q, want none", r.writes)
			}
		})
	}
}

func TestEncodeAllowsNULOutsideID(t *testing.T) {
	r := &recorder{}
	if err := sse.NewEncoder(r).Encode(sse.Event{Type: "a\x00", Data: []byte("b\x00")}); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	assertOneWrite(t, r, "event: a\x00\ndata: b\x00\n\n")
}

func TestComment(t *testing.T) {
	for _, tt := range []struct {
		name, text, want string
	}{
		{"one line", "keep-alive", ": keep-alive\n"},
		{"empty", "", ": \n"},
		{"LF, CR, and CRLF split", "a\nb\rc\r\nd", ": a\n: b\n: c\n: d\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := &recorder{}
			if err := sse.NewEncoder(r).Comment(tt.text); err != nil {
				t.Fatalf("Comment: %v", err)
			}
			assertOneWrite(t, r, tt.want)
			if got := readAll(t, sse.NewDecoder(strings.NewReader(r.writes[0]+"\n"))); len(got) != 0 {
				t.Errorf("decoded %q from a comment, want no events", got)
			}
		})
	}
}

func TestEncoderRetry(t *testing.T) {
	for _, tt := range []struct {
		name string
		d    time.Duration
		want string
	}{
		{"whole milliseconds", 1500 * time.Millisecond, "retry: 1500\n\n"},
		{"zero", 0, "retry: 0\n\n"},
		{"fraction dropped", 2*time.Millisecond + 999*time.Microsecond, "retry: 2\n\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := &recorder{}
			if err := sse.NewEncoder(r).Retry(tt.d); err != nil {
				t.Fatalf("Retry: %v", err)
			}
			assertOneWrite(t, r, tt.want)
		})
	}
}

func TestEncoderRetryRejectsNegative(t *testing.T) {
	r := &recorder{}
	if err := sse.NewEncoder(r).Retry(-time.Millisecond); err == nil {
		t.Error("Retry(-1ms) = nil, want an error")
	}
	if len(r.writes) != 0 {
		t.Errorf("writes = %q, want none", r.writes)
	}
}

func TestEncoderWriteError(t *testing.T) {
	boom := errors.New("boom")
	enc := sse.NewEncoder(failWriter{boom})
	if err := enc.Encode(sse.Event{Data: []byte("a")}); err != boom {
		t.Errorf("Encode error = %v, want %v unchanged", err, boom)
	}
	if err := enc.Comment("a"); err != boom {
		t.Errorf("Comment error = %v, want %v unchanged", err, boom)
	}
	if err := enc.Retry(time.Second); err != boom {
		t.Errorf("Retry error = %v, want %v unchanged", err, boom)
	}
}

func TestRoundTrip(t *testing.T) {
	r := &recorder{}
	enc := sse.NewEncoder(r)
	steps := []func() error{
		func() error { return enc.Comment("open\r\nstream") },
		func() error { return enc.Encode(sse.Event{Type: "greet", ID: "1", Data: []byte("hello\nworld")}) },
		func() error { return enc.Retry(2500 * time.Millisecond) },
		func() error { return enc.Encode(sse.Event{ID: "2", Data: []byte("a\rb\r\nc\n")}) },
		func() error { return enc.Comment("ping") },
		func() error { return enc.Encode(sse.Event{Type: "empty", ID: "3"}) },
		func() error { return enc.Encode(sse.Event{Data: []byte("keeps id 3")}) },
		func() error { return enc.Encode(sse.Event{Type: "end", ID: "4", Data: []byte(" spaced: ✓ ")}) },
	}
	for _, step := range steps {
		if err := step(); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}
	if len(r.writes) != len(steps) {
		t.Fatalf("writes = %d, want one per call (%d)", len(r.writes), len(steps))
	}
	d := sse.NewDecoder(strings.NewReader(strings.Join(r.writes, "")))
	got := readAll(t, d)
	want := []event{
		{Type: "greet", Data: "hello\nworld", ID: "1"},
		{Data: "a\nb\nc\n", ID: "2"},
		{Type: "empty", ID: "3"},
		{Data: "keeps id 3", ID: "3"},
		{Type: "end", Data: " spaced: ✓ ", ID: "4"},
	}
	if len(got) != len(want) {
		t.Fatalf("events = %q\nwant     %q", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("events = %q\nwant     %q", got, want)
			break
		}
	}
	if retry, ok := d.Retry(); retry != 2500*time.Millisecond || !ok {
		t.Errorf("Retry() = %v, %v; want 2.5s, true", retry, ok)
	}
}
