package logging_test

import (
	"log/slog"
	"testing"

	"github.com/standards-lab/go-core/logging"
)

func TestLevel_SlogParsesTheStandardVocabulary(t *testing.T) {
	for _, tc := range []struct {
		level logging.Level
		want  slog.Level
	}{
		{logging.LevelDebug, slog.LevelDebug},
		{logging.LevelInfo, slog.LevelInfo},
		{logging.LevelWarn, slog.LevelWarn},
		{logging.LevelError, slog.LevelError},
	} {
		got, err := tc.level.Slog()
		if err != nil {
			t.Errorf("Slog(%q): %v", tc.level, err)
			continue
		}
		if got != tc.want {
			t.Errorf("Slog(%q) = %v, want %v", tc.level, got, tc.want)
		}
	}
}

// The level vocabulary is slog's, so casing and offsets come for free and this
// package never matches on the string itself.
func TestLevel_SlogAcceptsWhatSlogAccepts(t *testing.T) {
	for _, tc := range []struct {
		level logging.Level
		want  slog.Level
	}{
		{"INFO", slog.LevelInfo},
		{"Warn", slog.LevelWarn},
		{"warn+2", slog.LevelWarn + 2},
		{"error-1", slog.LevelError - 1},
	} {
		got, err := tc.level.Slog()
		if err != nil {
			t.Errorf("Slog(%q): %v", tc.level, err)
			continue
		}
		if got != tc.want {
			t.Errorf("Slog(%q) = %v, want %v", tc.level, got, tc.want)
		}
	}
}

func TestLevel_SlogRejectsUnknownAndEmpty(t *testing.T) {
	for _, level := range []logging.Level{"", "trace", "verbose", "  "} {
		if _, err := level.Slog(); err == nil {
			t.Errorf("Slog(%q) = nil error, want a rejection", level)
		}
	}
}

func TestLevel_StringIsTheConfiguredText(t *testing.T) {
	if got := logging.LevelWarn.String(); got != "warn" {
		t.Errorf("String() = %q, want warn", got)
	}
}
