package logging_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

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
