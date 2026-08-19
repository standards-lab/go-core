package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

// Duration is a time.Duration that takes part in JSON configuration: it
// unmarshals from the string form time.ParseDuration accepts ("1m30s") or
// from a bare integer number of nanoseconds, and marshals as the string form.
type Duration time.Duration

// Duration returns the value as a time.Duration.
func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}

// String returns the value in time.Duration's string form.
func (d Duration) String() string {
	return time.Duration(d).String()
}

// Set parses value with time.ParseDuration and stores the result. An empty
// value is a no-op, so an unset environment variable leaves the receiver
// unchanged.
func (d *Duration) Set(value string) error {
	if value == "" {
		return nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fmt.Errorf("parse duration %q: %w", value, err)
	}
	*d = Duration(parsed)
	return nil
}

// MarshalJSON encodes the duration as its string form.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// UnmarshalJSON decodes a duration string, a bare integer number of
// nanoseconds, or null, which leaves the value unchanged. A fractional number
// is rejected toward the string form.
func (d *Duration) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var value any
	if err := dec.Decode(&value); err != nil {
		return err
	}
	switch v := value.(type) {
	case nil:
		return nil
	case string:
		return d.Set(v)
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			return fmt.Errorf("invalid duration %s: a bare number is integer nanoseconds; use the string form (e.g. %q) for fractions", data, "1.5s")
		}
		*d = Duration(n)
		return nil
	default:
		return fmt.Errorf("invalid duration: %s", data)
	}
}
