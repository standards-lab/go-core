package logging

import (
	"io"
	"log/slog"
)

// New returns a logger writing to w: a JSON handler for [FormatJSON], a text
// handler otherwise. New returns no error — Finalize is the validation point —
// and an unparseable level falls back to info.
func New(w io.Writer, cfg Config) *slog.Logger {
	level, err := cfg.Level.Slog()
	if err != nil {
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if cfg.Format == FormatJSON {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		handler = slog.NewTextHandler(w, opts)
	}

	return slog.New(handler)
}
