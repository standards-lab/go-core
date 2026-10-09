package logging_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/standards-lab/go-core/logging"
)

func TestNew_TextFormatWritesToTheCallersWriter(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.New(&buf, logging.Config{Level: logging.LevelInfo, Format: logging.FormatText})

	logger.Info("hello", "count", 2)

	out := buf.String()
	if !strings.Contains(out, "msg=hello") || !strings.Contains(out, "count=2") {
		t.Errorf("output = %q, want a text record carrying the message and attribute", out)
	}
}

func TestNew_JSONFormatEmitsOneObjectPerRecord(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.New(&buf, logging.Config{Level: logging.LevelInfo, Format: logging.FormatJSON})

	logger.Info("hello", "count", 2)

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("unmarshal %q: %v", buf.String(), err)
	}
	if record["msg"] != "hello" {
		t.Errorf("msg = %v, want hello", record["msg"])
	}
	if record["level"] != "INFO" {
		t.Errorf("level = %v, want INFO", record["level"])
	}
}

func TestNew_LevelGatesRecords(t *testing.T) {
	for _, tc := range []struct {
		level   logging.Level
		at      slog.Level
		enabled bool
	}{
		{logging.LevelDebug, slog.LevelDebug, true},
		{logging.LevelInfo, slog.LevelDebug, false},
		{logging.LevelInfo, slog.LevelInfo, true},
		{logging.LevelWarn, slog.LevelInfo, false},
		{logging.LevelError, slog.LevelWarn, false},
		{logging.LevelError, slog.LevelError, true},
	} {
		logger := logging.New(&bytes.Buffer{}, logging.Config{Level: tc.level})
		if got := logger.Enabled(context.Background(), tc.at); got != tc.enabled {
			t.Errorf("Config{Level: %q}: Enabled(%v) = %v, want %v", tc.level, tc.at, got, tc.enabled)
		}
	}
}

// The offset syntax is slog's, and it reaches the handler untouched.
func TestNew_LevelCarriesSlogOffsets(t *testing.T) {
	logger := logging.New(&bytes.Buffer{}, logging.Config{Level: "warn+2"})

	if logger.Enabled(context.Background(), slog.LevelWarn) {
		t.Error("Enabled(warn) = true, want false at warn+2")
	}
	if !logger.Enabled(context.Background(), slog.LevelError) {
		t.Error("Enabled(error) = false, want true at warn+2")
	}
}

// Finalize is the validation point; a literal that skipped it still logs.
func TestNew_UnvalidatedLevelFallsBackToInfo(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.New(&buf, logging.Config{Level: "trace"})

	logger.Info("hello")
	if buf.Len() == 0 {
		t.Error("no output for an unvalidated level, want a fallback to info")
	}
	if logger.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("Enabled(debug) = true, want false at the info fallback")
	}
}

// A zero Config has never been finalized, so it names no format either; text is
// what New writes when the format is not json.
func TestNew_ZeroConfigWritesText(t *testing.T) {
	var buf bytes.Buffer
	logging.New(&buf, logging.Config{}).Info("hello")

	if !strings.Contains(buf.String(), "msg=hello") {
		t.Errorf("output = %q, want a text record", buf.String())
	}
}

// recordTime returns the top-level time a record carries, as the handler wrote
// it: the JSON "time" field or the text time= value.
func recordTime(t *testing.T, format logging.Format, out string) string {
	t.Helper()
	if format == logging.FormatJSON {
		var record map[string]any
		if err := json.Unmarshal([]byte(out), &record); err != nil {
			t.Fatalf("unmarshal %q: %v", out, err)
		}
		s, ok := record[slog.TimeKey].(string)
		if !ok {
			t.Fatalf("record %q carries no string time", out)
		}
		return s
	}
	for field := range strings.FieldsSeq(out) {
		if v, ok := strings.CutPrefix(field, slog.TimeKey+"="); ok {
			return v
		}
	}
	t.Fatalf("record %q carries no time= field", out)
	return ""
}

// slog stamps a record with time.Now in time.Local, so each handler converts
// the record's time. The output shows only the offset, and Europe/London is
// GMT in winter, where an unconverted time already prints as Z; the record
// times here are in summer (BST, +01:00) and in zones never at a zero offset,
// so only the conversion makes them Z.
func TestNew_RecordTimeIsUTC(t *testing.T) {
	instant := time.Date(2026, time.July, 1, 12, 30, 45, 123_000_000, time.UTC)
	newYork, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	kolkata, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		t.Fatal(err)
	}

	for _, loc := range []*time.Location{time.Local, newYork, kolkata} {
		for _, format := range []logging.Format{logging.FormatJSON, logging.FormatText} {
			t.Run(loc.String()+"/"+format.String(), func(t *testing.T) {
				var buf bytes.Buffer
				logger := logging.New(&buf, logging.Config{Level: logging.LevelInfo, Format: format})

				record := slog.NewRecord(instant.In(loc), slog.LevelInfo, "hello", 0)
				if err := logger.Handler().Handle(context.Background(), record); err != nil {
					t.Fatalf("Handle: %v", err)
				}

				got := recordTime(t, format, buf.String())
				if !strings.HasSuffix(got, "Z") {
					t.Errorf("time = %q, want UTC (a Z suffix)", got)
				}
				parsed, err := time.Parse(time.RFC3339Nano, got)
				if err != nil {
					t.Fatalf("parse time %q: %v", got, err)
				}
				if !parsed.Equal(instant) {
					t.Errorf("time = %v, want the record's instant %v", parsed, instant)
				}
			})
		}
	}
}

// Only the record's own time is converted: a time the caller logs, at the top
// level or as a "time" key inside a group, is written as the caller built it.
func TestNew_CallerTimeAttrsAreUntouched(t *testing.T) {
	at := time.Date(2026, time.July, 1, 8, 0, 0, 0, time.Local) // BST, +01:00

	var buf bytes.Buffer
	logging.New(&buf, logging.Config{Format: logging.FormatJSON}).
		Info("hello", "at", at, slog.Group("req", slog.Time(slog.TimeKey, at)))

	var record struct {
		At  string `json:"at"`
		Req struct {
			Time string `json:"time"`
		} `json:"req"`
	}
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("unmarshal %q: %v", buf.String(), err)
	}
	want := at.Format(time.RFC3339Nano)
	if record.At != want {
		t.Errorf("at = %q, want %q (the caller's zone kept)", record.At, want)
	}
	if record.Req.Time != want {
		t.Errorf("req.time = %q, want %q (a grouped time key is the caller's)", record.Req.Time, want)
	}
}
