package logging

import "log/slog"

// Level is the minimum level the logger emits, in the vocabulary slog.Level
// parses: "debug", "INFO", and offsets such as "warn+2", case-insensitively.
// An empty Level is unset, distinct from "set to info".
type Level string

// The standard levels; any string slog.Level parses is valid.
const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

// String returns the level's string value.
func (l Level) String() string {
	return string(l)
}

// Slog parses the level into a slog.Level.
func (l Level) Slog() (slog.Level, error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(l)); err != nil {
		return 0, err
	}
	return level, nil
}
