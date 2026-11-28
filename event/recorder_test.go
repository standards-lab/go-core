package event_test

import (
	"context"
	"errors"
	"maps"
	"math"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/standards-lab/go-core/event"
)

type tx struct{ name string }

type sink struct {
	tx     *tx
	events []event.Event
	err    error
}

func (s *sink) Write(_ context.Context, t *tx, es ...event.Event) error {
	if s.err != nil {
		return s.err
	}
	s.tx = t
	s.events = append(s.events, es...)
	return nil
}

type started struct {
	Round int `json:"round"`
}

var (
	kindStarted  = event.Define[started]("lab.exercise.started")
	kindStopped  = event.Define[struct{}]("lab.exercise.stopped")
	kindBadValue = event.Define[float64]("lab.exercise.bad")
)

func fixed(s *sink) *event.Recorder[*tx] {
	at := time.Date(2026, 9, 28, 12, 0, 0, 0, time.FixedZone("x", 3600))
	n := 0
	return event.NewRecorder[*tx](s, "/exercise", event.Options{
		Clock: func() time.Time { return at },
		IDs:   func() string { n++; return strings.Repeat("a", n) },
	})
}

func emitWith(t *testing.T, opts event.Options, raises ...event.RaiseOptions) []event.Event {
	t.Helper()
	s := &sink{}
	_, err := event.NewRecorder[*tx](s, "/exercise", opts).Emit(t.Context(), func(_ *tx, q *event.Queue) (int, error) {
		for _, ro := range raises {
			kindStarted.Raise(q, "ex-1", started{}, ro)
		}
		return 0, nil
	})(&tx{})
	if err != nil {
		t.Fatal(err)
	}
	return s.events
}

func TestEmitStampsAndWritesInRaiseOrder(t *testing.T) {
	s := &sink{}
	on := &tx{name: "t1"}
	run := fixed(s).Emit(t.Context(), func(got *tx, q *event.Queue) (string, error) {
		if got != on {
			t.Errorf("body ran on %v, want %v", got, on)
		}
		kindStarted.Raise(q, "ex-1", started{Round: 0}, event.RaiseOptions{})
		kindStopped.Raise(q, "ex-1", struct{}{}, event.RaiseOptions{})
		return "done", nil
	})
	out, err := run(on)
	if err != nil || out != "done" {
		t.Fatalf("run = %q, %v", out, err)
	}
	if s.tx != on {
		t.Errorf("wrote on %v, want %v", s.tx, on)
	}
	if len(s.events) != 2 {
		t.Fatalf("wrote %d events, want 2", len(s.events))
	}
	first, second := s.events[0], s.events[1]
	if first.Type != "lab.exercise.started" || second.Type != "lab.exercise.stopped" {
		t.Errorf("types = %q, %q; want raise order", first.Type, second.Type)
	}
	if first.ID != "a" || second.ID != "aa" {
		t.Errorf("ids = %q, %q", first.ID, second.ID)
	}
	want := time.Date(2026, 9, 28, 11, 0, 0, 0, time.UTC)
	if first.Source != "/exercise" || !first.Time.Equal(want) || first.Time.Location() != time.UTC {
		t.Errorf("stamp = %q %v", first.Source, first.Time)
	}
	if first.Subject != "ex-1" || first.DataContentType != "application/json" || string(first.Data) != `{"round":0}` {
		t.Errorf("event = %+v", first)
	}
}

func TestEmitStampsOneTimeForTheWholeQueue(t *testing.T) {
	s := &sink{}
	at := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	tick := func() time.Time { at = at.Add(time.Second); return at }
	_, err := event.NewRecorder[*tx](s, "/exercise", event.Options{Clock: tick}).Emit(t.Context(), func(_ *tx, q *event.Queue) (int, error) {
		kindStarted.Raise(q, "ex-1", started{Round: 0}, event.RaiseOptions{})
		kindStarted.Raise(q, "ex-1", started{Round: 1}, event.RaiseOptions{})
		kindStopped.Raise(q, "ex-1", struct{}{}, event.RaiseOptions{})
		return 0, nil
	})(&tx{})
	if err != nil {
		t.Fatal(err)
	}
	if len(s.events) != 3 {
		t.Fatalf("wrote %d events, want 3", len(s.events))
	}
	for i, e := range s.events[1:] {
		if !e.Time.Equal(s.events[0].Time) {
			t.Errorf("event %d time = %v, want %v, the first event's", i+1, e.Time, s.events[0].Time)
		}
	}
}

func TestEmitWritesNothingWhenTheBodyFails(t *testing.T) {
	s := &sink{}
	boom := errors.New("refused")
	out, err := fixed(s).Emit(t.Context(), func(_ *tx, q *event.Queue) (int, error) {
		kindStarted.Raise(q, "ex-1", started{}, event.RaiseOptions{})
		return 7, boom
	})(&tx{})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the body's", err)
	}
	if out != 0 {
		t.Errorf("result = %d, want the zero result", out)
	}
	if len(s.events) != 0 {
		t.Errorf("wrote %d events after a failed body", len(s.events))
	}
}

func TestEmitWritesNothingWhenNothingIsRaised(t *testing.T) {
	s := &sink{err: errors.New("must not be called")}
	if _, err := fixed(s).Emit(t.Context(), func(*tx, *event.Queue) (int, error) { return 1, nil })(&tx{}); err != nil {
		t.Fatalf("err = %v", err)
	}
}

func TestEmitFailsTheCommandWhenTheSinkFails(t *testing.T) {
	boom := errors.New("insert failed")
	out, err := fixed(&sink{err: boom}).Emit(t.Context(), func(_ *tx, q *event.Queue) (string, error) {
		kindStarted.Raise(q, "ex-1", started{}, event.RaiseOptions{})
		return "done", nil
	})(&tx{})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the sink's", err)
	}
	if out != "" {
		t.Errorf("out = %q, want the zero result on failure", out)
	}
}

func TestEmitFailsTheCommandWhenARaiseFails(t *testing.T) {
	s := &sink{}
	_, err := fixed(s).Emit(t.Context(), func(_ *tx, q *event.Queue) (int, error) {
		kindStarted.Raise(q, "ex-1", started{}, event.RaiseOptions{})
		kindBadValue.Raise(q, "ex-1", math.NaN(), event.RaiseOptions{})
		if q.Err() == nil {
			t.Error("the queue holds no error after an unencodable raise")
		}
		if n := len(q.Events()); n != 1 {
			t.Errorf("queued %d events, want the one that encoded", n)
		}
		return 0, nil
	})(&tx{})
	if err == nil || !strings.Contains(err.Error(), "lab.exercise.bad") {
		t.Fatalf("err = %v, want the raise's", err)
	}
	if len(s.events) != 0 {
		t.Errorf("wrote %d events after a failed raise", len(s.events))
	}
}

func TestEmitMintsVersion7IDs(t *testing.T) {
	s := &sink{}
	_, err := event.NewRecorder[*tx](s, "/exercise", event.Options{}).Emit(t.Context(), func(_ *tx, q *event.Queue) (int, error) {
		kindStarted.Raise(q, "", started{}, event.RaiseOptions{})
		kindStarted.Raise(q, "", started{}, event.RaiseOptions{})
		return 0, nil
	})(&tx{})
	if err != nil {
		t.Fatal(err)
	}
	v7 := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	for _, e := range s.events {
		if !v7.MatchString(e.ID) {
			t.Errorf("id %q is not a version 7 UUID", e.ID)
		}
		if e.Subject != "" {
			t.Errorf("subject = %q, want absent", e.Subject)
		}
	}
	if s.events[0].ID == s.events[1].ID {
		t.Error("two events share an id")
	}
}

func TestEmitStampsTheDefaultClockInUTC(t *testing.T) {
	s := &sink{}
	_, err := event.NewRecorder[*tx](s, "/exercise", event.Options{}).Emit(t.Context(), func(_ *tx, q *event.Queue) (int, error) {
		kindStarted.Raise(q, "", started{}, event.RaiseOptions{})
		return 0, nil
	})(&tx{})
	if err != nil {
		t.Fatal(err)
	}
	if loc := s.events[0].Time.Location(); loc != time.UTC {
		t.Errorf("time location = %v, want UTC", loc)
	}
}

func TestNewRecorderKeepsTheDefaultsLeftUnset(t *testing.T) {
	at := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	raise := func(_ *tx, q *event.Queue) (int, error) {
		kindStarted.Raise(q, "", started{}, event.RaiseOptions{})
		return 0, nil
	}

	clocked := &sink{}
	if _, err := event.NewRecorder[*tx](clocked, "/exercise", event.Options{
		Clock: func() time.Time { return at },
	}).Emit(t.Context(), raise)(&tx{}); err != nil {
		t.Fatal(err)
	}
	v7 := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if e := clocked.events[0]; !e.Time.Equal(at) || !v7.MatchString(e.ID) {
		t.Errorf("with Clock set: time %v, id %q; want the set time and a version 7 UUID", e.Time, e.ID)
	}

	ided := &sink{}
	before := time.Now()
	if _, err := event.NewRecorder[*tx](ided, "/exercise", event.Options{
		IDs: func() string { return "fixed" },
	}).Emit(t.Context(), raise)(&tx{}); err != nil {
		t.Fatal(err)
	}
	after := time.Now()
	e := ided.events[0]
	if e.ID != "fixed" || e.Time.Before(before) || e.Time.After(after) || e.Time.Location() != time.UTC {
		t.Errorf("with IDs set: id %q, time %v; want the set id and now in UTC", e.ID, e.Time)
	}
}

func TestNewRecorderRejectsAWiringDefect(t *testing.T) {
	for name, build := range map[string]func(){
		"nil sink":     func() { event.NewRecorder[*tx](nil, "/exercise", event.Options{}) },
		"empty source": func() { event.NewRecorder[*tx](&sink{}, "", event.Options{}) },
	} {
		func() {
			defer func() {
				msg, _ := recover().(string)
				if !strings.HasPrefix(msg, "event: ") {
					t.Errorf("%s panicked with %q, want an \"event: \" message", name, msg)
				}
			}()
			build()
		}()
	}
}

func TestEmitWritesARaisesExtensions(t *testing.T) {
	want := map[string]string{"correlationid": "c-1"}
	es := emitWith(t, event.Options{}, event.RaiseOptions{Extensions: want})
	if got := es[0].Extensions; !maps.Equal(got, want) {
		t.Errorf("Extensions = %v, want %v", got, want)
	}
}

func TestEmitFailsTheCommandOnARejectedExtensionName(t *testing.T) {
	for _, name := range []string{"Layer", "id", "trace-id", ""} {
		s := &sink{}
		_, err := fixed(s).Emit(t.Context(), func(_ *tx, q *event.Queue) (int, error) {
			kindStarted.Raise(q, "ex-1", started{}, event.RaiseOptions{})
			kindStarted.Raise(q, "ex-1", started{}, event.RaiseOptions{Extensions: map[string]string{name: "v"}})
			return 0, nil
		})(&tx{})
		if err == nil || !strings.Contains(err.Error(), "extension") {
			t.Errorf("extension %q: err = %v, want the invalid name's", name, err)
		}
		if len(s.events) != 0 {
			t.Errorf("extension %q: wrote %d events, want none", name, len(s.events))
		}
	}
}

func TestEmitStampsTheRecordersExtensionsOnEveryEvent(t *testing.T) {
	want := map[string]string{"layer": "domain"}
	es := emitWith(t, event.Options{Extensions: want}, event.RaiseOptions{}, event.RaiseOptions{})
	if len(es) != 2 {
		t.Fatalf("wrote %d events, want 2", len(es))
	}
	for i, e := range es {
		if !maps.Equal(e.Extensions, want) {
			t.Errorf("event %d Extensions = %v, want %v", i, e.Extensions, want)
		}
	}
	es[0].Extensions["layer"] = "changed"
	if es[1].Extensions["layer"] != "domain" {
		t.Errorf("events share an extensions map: second event's layer = %q", es[1].Extensions["layer"])
	}
	if want["layer"] != "domain" {
		t.Errorf("stamping changed the options' map: layer = %q", want["layer"])
	}
}

func TestEmitStampsNoExtensionsFromTheZeroOptions(t *testing.T) {
	for _, opts := range []event.Options{{}, {Extensions: map[string]string{}}} {
		es := emitWith(t, opts, event.RaiseOptions{})
		if es[0].Extensions != nil {
			t.Errorf("Options%+v: Extensions = %v, want nil", opts, es[0].Extensions)
		}
	}
}

func TestEmitKeepsTheRaisesValueOverTheRecorders(t *testing.T) {
	raised := map[string]string{"layer": "raise", "correlationid": "c-1"}
	es := emitWith(t, event.Options{Extensions: map[string]string{"layer": "recorder", "causationid": "e-0"}},
		event.RaiseOptions{Extensions: raised})
	want := map[string]string{"layer": "raise", "correlationid": "c-1", "causationid": "e-0"}
	if !maps.Equal(es[0].Extensions, want) {
		t.Errorf("Extensions = %v, want %v", es[0].Extensions, want)
	}
	if len(raised) != 2 {
		t.Errorf("stamping changed the raise's map: %v", raised)
	}
}

func TestNewRecorderCopiesTheOptionsExtensions(t *testing.T) {
	s := &sink{}
	ext := map[string]string{"layer": "domain"}
	r := event.NewRecorder[*tx](s, "/exercise", event.Options{Extensions: ext})
	ext["layer"] = "changed"
	ext["extra"] = "x"
	_, err := r.Emit(t.Context(), func(_ *tx, q *event.Queue) (int, error) {
		kindStarted.Raise(q, "ex-1", started{}, event.RaiseOptions{})
		return 0, nil
	})(&tx{})
	if err != nil {
		t.Fatal(err)
	}
	if want := map[string]string{"layer": "domain"}; !maps.Equal(s.events[0].Extensions, want) {
		t.Errorf("Extensions = %v, want %v", s.events[0].Extensions, want)
	}
}

func TestNewRecorderRejectsABadExtensionName(t *testing.T) {
	for name, want := range map[string]string{
		"Layer":    `"Layer"`,
		"id":       `"id"`,
		"trace-id": `"trace-id"`,
		"":         "empty",
	} {
		func() {
			defer func() {
				msg, _ := recover().(string)
				if !strings.HasPrefix(msg, "event: ") || !strings.Contains(msg, "Options.Extensions") || !strings.Contains(msg, want) {
					t.Errorf("extension %q: panicked with %q, want an \"event: \" message naming Options.Extensions and %s", name, msg, want)
				}
			}()
			event.NewRecorder[*tx](&sink{}, "/exercise", event.Options{Extensions: map[string]string{name: "v"}})
		}()
	}
}
