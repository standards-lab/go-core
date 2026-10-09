package logging

import (
	"io"
	"log/slog"
)

// New returns a logger writing to w, with a JSON handler for [FormatJSON]
// and a text handler otherwise. A level Finalize would reject falls back to
// info. Either handler writes each record's time in time.UTC.
func New(w io.Writer, cfg Config) *slog.Logger {
	level, err := cfg.Level.Slog()
	if err != nil {
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level, ReplaceAttr: recordTimeUTC}

	var handler slog.Handler
	if cfg.Format == FormatJSON {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		handler = slog.NewTextHandler(w, opts)
	}

	return slog.New(handler)
}

// recordTimeUTC converts the record's built-in time to UTC, so the output does
// not depend on the host's zone. Only the top-level slog.TimeKey is the
// record's own time; an attribute of that name inside a group, and any other
// time-valued attribute, is the caller's and passes through untouched.
func recordTimeUTC(groups []string, a slog.Attr) slog.Attr {
	if len(groups) == 0 && a.Key == slog.TimeKey && a.Value.Kind() == slog.KindTime {
		a.Value = slog.TimeValue(a.Value.Time().UTC())
	}
	return a
}
