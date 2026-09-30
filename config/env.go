package config

import (
	"fmt"
	"os"
	"strings"
)

// EnvName composes an environment-variable name from a prefix and parts:
// each segment upper-cases, runs of characters outside A-Z and 0-9 collapse
// to single underscores, and empty segments drop out, so EnvName("app",
// "db", "host") is "APP_DB_HOST". A prefix empty once sanitized composes no
// name: EnvName returns "", which every reader in this module treats as no
// override.
func EnvName(prefix string, parts ...string) string {
	p := sanitize(prefix)
	if p == "" {
		return ""
	}
	segments := []string{p}
	for _, part := range parts {
		if s := sanitize(part); s != "" {
			segments = append(segments, s)
		}
	}
	return strings.Join(segments, "_")
}

// SetFromEnv parses the environment variable name with parse and points dest
// at the result. An unset or empty variable, or an empty name, leaves dest
// untouched; a parse failure returns an error naming the variable.
func SetFromEnv[T any](dest **T, name string, parse func(string) (T, error)) error {
	v := os.Getenv(name)
	if v == "" {
		return nil
	}
	parsed, err := parse(v)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	*dest = &parsed
	return nil
}

// SetDurationFromEnv is [SetFromEnv] parsing with [Duration.Set].
func SetDurationFromEnv(dest **Duration, name string) error {
	return SetFromEnv(dest, name, func(v string) (Duration, error) {
		var d Duration
		err := d.Set(v)
		return d, err
	})
}

func sanitize(s string) string {
	return strings.Join(strings.FieldsFunc(strings.ToUpper(s), isSeparator), "_")
}

// isSeparator reports whether r falls outside A-Z and 0-9.
func isSeparator(r rune) bool {
	return (r < 'A' || r > 'Z') && (r < '0' || r > '9')
}
