package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

const envSeparatorPattern = `[^A-Z0-9]+`

var envSeparators = regexp.MustCompile(envSeparatorPattern)

// EnvName composes an environment-variable name from a prefix and parts: each
// segment upper-cases, runs of characters outside A-Z and 0-9 collapse to
// single underscores, and empty segments drop out, so EnvName("app", "db",
// "host") is "APP_DB_HOST" and a caller can vary the prefix freely.
func EnvName(prefix string, parts ...string) string {
	segments := make([]string, 0, len(parts)+1)
	if s := sanitize(prefix); s != "" {
		segments = append(segments, s)
	}
	for _, p := range parts {
		if s := sanitize(p); s != "" {
			segments = append(segments, s)
		}
	}
	return strings.Join(segments, "_")
}

// SetDurationFromEnv reads the environment variable name and, when it is set,
// parses it as a [Duration] and points dest at the result. An unset variable
// or an empty name leaves dest untouched; a value [Duration.Set] rejects
// fails with an error naming the variable. Capability configurations call it
// from their Finalize env pass for each tri-state duration field.
func SetDurationFromEnv(dest **Duration, name string) error {
	v := os.Getenv(name)
	if v == "" {
		return nil
	}
	var d Duration
	if err := d.Set(v); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	*dest = &d
	return nil
}

func sanitize(s string) string {
	s = envSeparators.ReplaceAllString(
		strings.ToUpper(strings.TrimSpace(s)),
		"_",
	)
	return strings.Trim(s, "_")
}
