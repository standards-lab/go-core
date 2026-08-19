package config_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/standards-lab/go-core/config"
)

// holder exercises Duration as a struct field, which is how a configuration
// type carries one.
type holder struct {
	Timeout config.Duration `json:"timeout"`
}

func TestDuration_UnmarshalString(t *testing.T) {
	var h holder
	if err := json.Unmarshal([]byte(`{"timeout":"1m30s"}`), &h); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if want := 90 * time.Second; time.Duration(h.Timeout) != want {
		t.Errorf("Timeout = %s, want %s", h.Timeout, want)
	}
}

func TestDuration_UnmarshalNumberIsNanoseconds(t *testing.T) {
	var h holder
	if err := json.Unmarshal([]byte(`{"timeout":1500000000}`), &h); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if want := 1500 * time.Millisecond; time.Duration(h.Timeout) != want {
		t.Errorf("Timeout = %s, want %s", h.Timeout, want)
	}
}

func TestDuration_UnmarshalEmptyStringLeavesUnset(t *testing.T) {
	// An empty string reads as unset rather than as an error, so the zero value
	// survives for Finalize to fill with a default.
	var h holder
	if err := json.Unmarshal([]byte(`{"timeout":""}`), &h); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if h.Timeout != 0 {
		t.Errorf("Timeout = %s, want the zero value", h.Timeout)
	}
}

func TestDuration_UnmarshalMalformed(t *testing.T) {
	var h holder
	err := json.Unmarshal([]byte(`{"timeout":"1hour"}`), &h)
	if err == nil {
		t.Fatal("Unmarshal returned nil for a malformed duration")
	}
	if !strings.Contains(err.Error(), "1hour") {
		t.Errorf("error = %v, want it to quote the offending value", err)
	}
}

func TestDuration_UnmarshalWrongType(t *testing.T) {
	var h holder
	if err := json.Unmarshal([]byte(`{"timeout":true}`), &h); err == nil {
		t.Fatal("Unmarshal returned nil for a boolean duration")
	}
}

func TestDuration_MarshalRoundTrip(t *testing.T) {
	data, err := json.Marshal(holder{Timeout: config.Duration(90 * time.Second)})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if got, want := string(data), `{"timeout":"1m30s"}`; got != want {
		t.Errorf("Marshal = %s, want %s", got, want)
	}

	var back holder
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if time.Duration(back.Timeout) != 90*time.Second {
		t.Errorf("round trip = %s, want 1m30s", back.Timeout)
	}
}

func TestDuration_SetEmptyLeavesValue(t *testing.T) {
	// Set treats an empty string as "nothing to apply", which is what lets an
	// unset environment variable and a disabled override share one path.
	d := config.Duration(30 * time.Second)
	if err := d.Set(""); err != nil {
		t.Fatalf("Set(\"\"): %v", err)
	}
	if time.Duration(d) != 30*time.Second {
		t.Errorf("Timeout = %s, want the value to be left alone", d)
	}
}

func TestDuration_SetMalformed(t *testing.T) {
	var d config.Duration
	if err := d.Set("nope"); err == nil {
		t.Fatal("Set returned nil for a malformed duration")
	}
}

func TestDuration_UnmarshalNullLeavesValue(t *testing.T) {
	h := holder{Timeout: config.Duration(5 * time.Second)}
	if err := json.Unmarshal([]byte(`{"timeout":null}`), &h); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if want := 5 * time.Second; time.Duration(h.Timeout) != want {
		t.Errorf("Timeout = %s, want %s (null must leave the value alone)", h.Timeout, want)
	}

	var zero holder
	if err := json.Unmarshal([]byte(`{"timeout":null}`), &zero); err != nil {
		t.Fatalf("Unmarshal into zero value: %v", err)
	}
	if zero.Timeout != 0 {
		t.Errorf("Timeout = %s, want the zero value", zero.Timeout)
	}
}

func TestDuration_UnmarshalLargeIntegerExact(t *testing.T) {
	// 9007199254740993 = 2^53 + 1: representable as int64 but not as float64,
	// so an exact result proves the decode does not round-trip through float64.
	var h holder
	if err := json.Unmarshal([]byte(`{"timeout":9007199254740993}`), &h); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got := int64(h.Timeout); got != 9007199254740993 {
		t.Errorf("Timeout = %d ns, want 9007199254740993 exactly", got)
	}
}

func TestDuration_UnmarshalFractionalNumberRejected(t *testing.T) {
	var h holder
	err := json.Unmarshal([]byte(`{"timeout":1.5}`), &h)
	if err == nil {
		t.Fatal("Unmarshal returned nil for a fractional bare number")
	}
	if !strings.Contains(err.Error(), "string form") {
		t.Errorf("error = %v, want it to point at the string form", err)
	}
}

func TestDuration_UnmarshalNegativeString(t *testing.T) {
	var h holder
	if err := json.Unmarshal([]byte(`{"timeout":"-5s"}`), &h); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if want := -5 * time.Second; time.Duration(h.Timeout) != want {
		t.Errorf("Timeout = %s, want %s (negative durations parse)", h.Timeout, want)
	}
}

func TestDuration_String(t *testing.T) {
	if got := config.Duration(90 * time.Second).String(); got != "1m30s" {
		t.Errorf("String() = %q, want 1m30s", got)
	}
}

func TestDuration_Duration(t *testing.T) {
	if got, want := config.Duration(90*time.Second).Duration(), 90*time.Second; got != want {
		t.Errorf("Duration() = %s, want %s", got, want)
	}
}
