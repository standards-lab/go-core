package sse_test

import (
	"bytes"
	"errors"
	"io"
	"math"
	"slices"
	"strconv"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/standards-lab/go-core/sse"
)

type event struct {
	Type, Data, ID string
}

func readAll(t *testing.T, d *sse.Decoder) []event {
	t.Helper()
	var got []event
	for {
		e, err := d.Next()
		if errors.Is(err, io.EOF) {
			return got
		}
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		got = append(got, event{e.Type, string(e.Data), e.ID})
	}
}

type splitReader struct{ chunks []string }

func (r *splitReader) Read(p []byte) (int, error) {
	if len(r.chunks) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.chunks[0])
	r.chunks[0] = r.chunks[0][n:]
	if r.chunks[0] == "" {
		r.chunks = r.chunks[1:]
	}
	return n, nil
}

var readers = []struct {
	name string
	wrap func(string) io.Reader
}{
	{"whole", func(s string) io.Reader { return strings.NewReader(s) }},
	{"one byte", func(s string) io.Reader { return iotest.OneByteReader(strings.NewReader(s)) }},
	{"split after CR, every CRLF across two reads", func(s string) io.Reader {
		var chunks []string
		for s != "" {
			i := strings.IndexByte(s, '\r')
			if i < 0 {
				i = len(s) - 1
			}
			chunks = append(chunks, s[:i+1])
			s = s[i+1:]
		}
		return &splitReader{chunks: chunks}
	}},
}

var parseTests = []struct {
	name   string
	stream string
	want   []event
}{
	{"LF endings", "data: a\n\ndata: b\n\n", []event{{Data: "a"}, {Data: "b"}}},
	{"CR endings", "data: a\r\rdata: b\r\r", []event{{Data: "a"}, {Data: "b"}}},
	{"CRLF endings", "data: a\r\n\r\ndata: b\r\n\r\n", []event{{Data: "a"}, {Data: "b"}}},
	{"mixed endings", "data: a\r\n\ndata: b\r\r\ndata: c\n\r", []event{{Data: "a"}, {Data: "b"}, {Data: "c"}}},
	{"CR CR is two line endings", "data: a\r\rdata: b\n\n", []event{{Data: "a"}, {Data: "b"}}},
	{"leading BOM stripped", "\uFEFFdata: a\n\n", []event{{Data: "a"}}},
	{"leading BOM before a blank line", "\uFEFF\ndata: a\n\n", []event{{Data: "a"}}},
	{"only one BOM stripped", "\uFEFF\uFEFFdata: a\n\n", nil},
	{"BOM after the start kept", "data: a\n\n\uFEFFdata: b\n\n", []event{{Data: "a"}}},
	{"comment ignored", ": ping\ndata: a\n: more\n\n", []event{{Data: "a"}}},
	{"comment-only block dispatches nothing", ": ping\n\n: ping\n\n", nil},
	{"no colon is a name with empty value", "data\n\n", []event{{Data: ""}}},
	{"no colon data lines joined", "data\ndata\n\n", []event{{Data: "\n"}}},
	{"one leading space removed", "data:  a\n\n", []event{{Data: " a"}}},
	{"no space after the colon", "data:a\n\n", []event{{Data: "a"}}},
	{"value split at the first colon", "data: a: b\n\n", []event{{Data: "a: b"}}},
	{"trailing space kept", "data: a \n\n", []event{{Data: "a "}}},
	{"data lines joined", "data: a\ndata: b\ndata\ndata: c\n\n", []event{{Data: "a\nb\n\nc"}}},
	{"empty data dispatched", "data:\n\n", []event{{Data: ""}}},
	{"event names the type", "event: response.created\ndata: {}\n\n", []event{{Type: "response.created", Data: "{}"}}},
	{"last event field wins", "event: a\nevent: b\ndata: x\n\n", []event{{Type: "b", Data: "x"}}},
	{"type resets after dispatch", "event: a\ndata: x\n\ndata: y\n\n", []event{{Type: "a", Data: "x"}, {Data: "y"}}},
	{"type resets after a block without data", "event: a\n\ndata: y\n\n", []event{{Data: "y"}}},
	{"id set", "id: 7\ndata: x\n\n", []event{{Data: "x", ID: "7"}}},
	{"id persists", "id: 7\ndata: x\n\ndata: y\n\n", []event{{Data: "x", ID: "7"}, {Data: "y", ID: "7"}}},
	{"id set by a block without data", "id: 7\n\ndata: y\n\n", []event{{Data: "y", ID: "7"}}},
	{"empty id resets", "id: 7\ndata: x\n\nid\ndata: y\n\n", []event{{Data: "x", ID: "7"}, {Data: "y"}}},
	{"id with NUL ignored", "id: 7\ndata: x\n\nid: 8\x009\ndata: y\n\n", []event{{Data: "x", ID: "7"}, {Data: "y", ID: "7"}}},
	{"unknown fields ignored", "foo: bar\nData: no\nRetry: 5\ndata: x\n\n", []event{{Data: "x"}}},
	{"retry field dispatches nothing", "retry: 1500\ndata: x\n\n", []event{{Data: "x"}}},
	{"retry-only block dispatches nothing", "retry: 1500\n\ndata: y\n\n", []event{{Data: "y"}}},
	{"blank line without data dispatches nothing", "\n\n\nevent: a\n\n", nil},
	{"pending event at end discarded", "data: a\n\ndata: b\n", []event{{Data: "a"}}},
	{"pending line at end discarded", "data: a\n\ndata: b", []event{{Data: "a"}}},
	{"empty stream", "", nil},
	{"multi-byte text kept", "event: é\nid: €\ndata: héllo € \U0001D11E\n\n", []event{{Type: "é", Data: "héllo € \U0001D11E", ID: "€"}}},
	{"invalid byte in data replaced", "data: a\xffb\n\n", []event{{Data: "a�b"}}},
	{"invalid byte in event type replaced", "event: a\x80\ndata: x\n\n", []event{{Type: "a�", Data: "x"}}},
	{"invalid byte in id replaced", "id: 7\xfe\ndata: x\n\n", []event{{Data: "x", ID: "7�"}}},
	{"truncated sequence is one replacement", "data: a\xe2\x82b\n\n", []event{{Data: "a�b"}}},
	{"truncated sequence at line end", "data: a\xf0\x9f\x98\ndata: b\xe2\r\n\r\n", []event{{Data: "a�\nb�"}}},
	{"overlong and surrogate forms replaced per byte", "data: \xc0\x80\xed\xa0\x80\n\n", []event{{Data: "�����"}}},
	{"maximal subparts replaced as Unicode Table 3-8 does", "data: a\xf1\x80\x80\xe1\x80\xc2b\x80c\x80\xbfd\n\n", []event{{Data: "a���b�c��d"}}},
}

func TestNext(t *testing.T) {
	for _, tt := range parseTests {
		for _, r := range readers {
			t.Run(tt.name+"/"+r.name, func(t *testing.T) {
				got := readAll(t, sse.NewDecoder(r.wrap(tt.stream)))
				if !slices.Equal(got, tt.want) {
					t.Errorf("events = %q\nwant     %q", got, tt.want)
				}
			})
		}
	}
}

var maxRetryMS = math.MaxInt64 / int64(time.Millisecond)

var retryTests = []struct {
	name   string
	stream string
	want   time.Duration
	ok     bool
}{
	{"none", "data: x\n\n", 0, false},
	{"set", "retry: 1500\n\n", 1500 * time.Millisecond, true},
	{"zero", "retry: 0\n\n", 0, true},
	{"leading zeros", "retry: 0010\n\n", 10 * time.Millisecond, true},
	{"no space after the colon", "retry:25\n\n", 25 * time.Millisecond, true},
	{"last valid wins", "retry: 100\nretry: 200\n\n", 200 * time.Millisecond, true},
	{"set within an event block", "retry: 100\ndata: x\n\n", 100 * time.Millisecond, true},
	{"set by a block the end cuts off", "retry: 100\n", 100 * time.Millisecond, true},
	{"cut-off line ignored", "retry: 100", 0, false},
	{"empty ignored", "retry\n\n", 0, false},
	{"sign ignored", "retry: +5\n\n", 0, false},
	{"negative ignored", "retry: -5\n\n", 0, false},
	{"second space ignored", "retry:  5\n\n", 0, false},
	{"fraction ignored", "retry: 1.5\n\n", 0, false},
	{"unit ignored", "retry: 5ms\n\n", 0, false},
	{"non-ASCII digit ignored", "retry: ٥\n\n", 0, false},
	{"invalid keeps the last valid", "retry: 100\nretry: x\n\n", 100 * time.Millisecond, true},
	{"largest that fits", "retry: " + strconv.FormatInt(maxRetryMS, 10) + "\n\n", time.Duration(maxRetryMS) * time.Millisecond, true},
	{"overflow ignored", "retry: 100\nretry: " + strconv.FormatInt(maxRetryMS+1, 10) + "\n\n", 100 * time.Millisecond, true},
	{"long overflow ignored", "retry: 99999999999999999999999999\n\n", 0, false},
}

func TestRetry(t *testing.T) {
	for _, tt := range retryTests {
		for _, r := range readers {
			t.Run(tt.name+"/"+r.name, func(t *testing.T) {
				d := sse.NewDecoder(r.wrap(tt.stream))
				if got, ok := d.Retry(); ok {
					t.Fatalf("Retry() before reading = %v, true; want false", got)
				}
				readAll(t, d)
				got, ok := d.Retry()
				if got != tt.want || ok != tt.ok {
					t.Errorf("Retry() = %v, %v; want %v, %v", got, ok, tt.want, tt.ok)
				}
			})
		}
	}
}

func TestNextDataIsItsOwn(t *testing.T) {
	d := sse.NewDecoder(strings.NewReader("data: aaa\n\ndata: bbb\n\ndata: ccc\n\n"))
	first, err := d.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	second, err := d.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if _, err := d.Next(); err != nil {
		t.Fatalf("Next: %v", err)
	}
	if string(first.Data) != "aaa" {
		t.Errorf("first Data after later reads = %q, want %q", first.Data, "aaa")
	}
	copy(first.Data, "XXX")
	if string(second.Data) != "bbb" {
		t.Errorf("second Data after changing the first = %q, want %q", second.Data, "bbb")
	}
}

func TestNextDispatchesAsBytesArrive(t *testing.T) {
	pr, pw := io.Pipe()
	d := sse.NewDecoder(pr)
	writes := []struct {
		chunk string
		want  event
	}{
		{"data: a\r\r", event{Data: "a"}},
		{"\ndata: b\r\n\r", event{Data: "b"}},
		{"\nevent: c\ndata: c\n\n", event{Type: "c", Data: "c"}},
	}
	type result struct {
		e   sse.Event
		err error
	}
	for _, w := range writes {
		go func() { _, _ = io.WriteString(pw, w.chunk) }()
		done := make(chan result, 1)
		go func() {
			e, err := d.Next()
			done <- result{e, err}
		}()
		select {
		case r := <-done:
			if r.err != nil {
				t.Fatalf("Next: %v", r.err)
			}
			if got := (event{r.e.Type, string(r.e.Data), r.e.ID}); got != w.want {
				t.Errorf("Next() = %q, want %q", got, w.want)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("Next waited past the blank line ending %q", w.chunk)
		}
	}
	_ = pw.Close()
	if _, err := d.Next(); !errors.Is(err, io.EOF) {
		t.Errorf("Next at end = %v, want io.EOF", err)
	}
}

func TestNextReadError(t *testing.T) {
	boom := errors.New("boom")
	d := sse.NewDecoder(io.MultiReader(strings.NewReader("data: a\n\ndata: b\n"), iotest.ErrReader(boom)))
	if e, err := d.Next(); err != nil || !bytes.Equal(e.Data, []byte("a")) {
		t.Fatalf("Next() = %q, %v; want a, nil", e.Data, err)
	}
	if _, err := d.Next(); err != boom {
		t.Errorf("Next() error = %v, want %v unchanged", err, boom)
	}
}
