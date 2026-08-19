package logging_test

import (
	"testing"

	"github.com/standards-lab/go-core/logging"
)

func TestFormat_ValidAcceptsTheSupportedSet(t *testing.T) {
	for _, format := range []logging.Format{logging.FormatText, logging.FormatJSON} {
		if !format.Valid() {
			t.Errorf("Valid(%q) = false, want true", format)
		}
	}
}

// Valid compares exactly; Config.Finalize is what lower-cases a value before it
// gets here.
func TestFormat_ValidRejectsUnknownAndUnnormalized(t *testing.T) {
	for _, format := range []logging.Format{"", "yaml", "logfmt", "JSON", "Text"} {
		if format.Valid() {
			t.Errorf("Valid(%q) = true, want false", format)
		}
	}
}

func TestFormat_StringIsTheConfiguredText(t *testing.T) {
	if got := logging.FormatJSON.String(); got != "json" {
		t.Errorf("String() = %q, want json", got)
	}
}
