package logging_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/standards-lab/go-core/config"
	"github.com/standards-lab/go-core/logging"
)

// Instantiating Load proves *logging.Config satisfies the config.Config
// contract at compile time; the constraint cannot be written as an ordinary
// interface assertion because it carries a type element.
var _ = config.Load[logging.Config]

// testPrefix is the env prefix override tests finalize with; the names below
// are what Finalize composes from it.
const (
	testPrefix = "test"
	envLevel   = "TEST_LOG_LEVEL"
	envFormat  = "TEST_LOG_FORMAT"
)

func TestConfig_MergeSourceWinsWithoutClearing(t *testing.T) {
	base := logging.Config{Level: logging.LevelDebug, Format: logging.FormatText}
	base.Merge(&logging.Config{Format: logging.FormatJSON})

	if base.Level != logging.LevelDebug {
		t.Errorf("Level = %q, want debug (the source omits it and must not clear it)", base.Level)
	}
	if base.Format != logging.FormatJSON {
		t.Errorf("Format = %q, want json (the source sets it)", base.Format)
	}
}

// An empty Level is what makes the merge contract work: slog.LevelInfo is the
// zero value of slog.Level, so a layer setting the level to info would be
// indistinguishable from a layer that says nothing.
func TestConfig_MergeInfoIsDistinguishableFromUnset(t *testing.T) {
	base := logging.Config{Level: logging.LevelDebug}
	base.Merge(&logging.Config{Level: logging.LevelInfo})

	if base.Level != logging.LevelInfo {
		t.Errorf("Level = %q, want info (an overlay setting info must win over debug)", base.Level)
	}
}

func TestConfig_FinalizeAppliesDefaults(t *testing.T) {
	var cfg logging.Config
	if err := cfg.Finalize(""); err != nil {
		t.Fatalf("Finalize: %v", err)
	}

	if cfg.Level != logging.LevelInfo {
		t.Errorf("Level = %q, want info", cfg.Level)
	}
	if cfg.Format != logging.FormatText {
		t.Errorf("Format = %q, want text", cfg.Format)
	}
}

func TestConfig_FinalizeWhitespaceOnlyTakesDefaults(t *testing.T) {
	cfg := logging.Config{Level: "   ", Format: "   "}
	if err := cfg.Finalize(""); err != nil {
		t.Fatalf("Finalize: %v", err)
	}

	if cfg.Level != logging.LevelInfo {
		t.Errorf("Level = %q, want info (whitespace-only reads as unset)", cfg.Level)
	}
	if cfg.Format != logging.FormatText {
		t.Errorf("Format = %q, want text (whitespace-only reads as unset)", cfg.Format)
	}
}

func TestConfig_FinalizeNormalizesEnvLevel(t *testing.T) {
	t.Setenv(envLevel, " INFO ")

	cfg := logging.Config{Level: logging.LevelDebug}
	if err := cfg.Finalize(testPrefix); err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	if cfg.Level != logging.LevelInfo {
		t.Errorf("Level = %q, want info (env value normalized after the read)", cfg.Level)
	}
}

func TestConfig_FinalizeEnvOverridesFiles(t *testing.T) {
	t.Setenv(envLevel, "error")
	t.Setenv(envFormat, "json")

	cfg := logging.Config{Level: logging.LevelDebug, Format: logging.FormatText}
	if err := cfg.Finalize(testPrefix); err != nil {
		t.Fatalf("Finalize: %v", err)
	}

	if cfg.Level != logging.LevelError {
		t.Errorf("Level = %q, want error (the environment wins over a file value)", cfg.Level)
	}
	if cfg.Format != logging.FormatJSON {
		t.Errorf("Format = %q, want json (the environment wins over a file value)", cfg.Format)
	}
}

// Normalization runs after the environment read, so a shell value in any casing
// lands on the canonical form the constants use.
func TestConfig_FinalizeNormalizesCasingAndSpace(t *testing.T) {
	t.Setenv(envLevel, "  WARN  ")
	t.Setenv(envFormat, "JSON")

	var cfg logging.Config
	if err := cfg.Finalize(testPrefix); err != nil {
		t.Fatalf("Finalize: %v", err)
	}

	if cfg.Level != logging.LevelWarn {
		t.Errorf("Level = %q, want warn", cfg.Level)
	}
	if cfg.Format != logging.FormatJSON {
		t.Errorf("Format = %q, want json", cfg.Format)
	}
}

func TestConfig_FinalizeEmptyEnvValueLeavesConfigured(t *testing.T) {
	// An empty variable reads as unset, not as a request to clear the value.
	t.Setenv(envLevel, "")

	cfg := logging.Config{Level: logging.LevelError}
	if err := cfg.Finalize(testPrefix); err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	if cfg.Level != logging.LevelError {
		t.Errorf("Level = %q, want error", cfg.Level)
	}
}

func TestConfig_FinalizeEmptyPrefixSkipsOverrides(t *testing.T) {
	// An empty prefix composes no variable names, so nothing in the
	// environment applies — the hermetic form.
	t.Setenv(envLevel, "error")
	t.Setenv(envFormat, "json")

	var cfg logging.Config
	if err := cfg.Finalize(""); err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	if cfg.Level != logging.LevelInfo || cfg.Format != logging.FormatText {
		t.Errorf("Level/Format = %q/%q, want the defaults with a zero Env", cfg.Level, cfg.Format)
	}
}

func TestConfig_FinalizeRejectsUnknownLevel(t *testing.T) {
	cfg := logging.Config{Level: "trace"}
	err := cfg.Finalize("")
	if err == nil {
		t.Fatal("Finalize returned nil for an unknown level")
	}
	if !strings.Contains(err.Error(), `invalid level "trace"`) {
		t.Errorf("error = %v, want it to name the rejected level", err)
	}
}

func TestConfig_FinalizeRejectsUnknownFormat(t *testing.T) {
	cfg := logging.Config{Format: "yaml"}
	err := cfg.Finalize("")
	if err == nil {
		t.Fatal("Finalize returned nil for an unknown format")
	}
	if !strings.Contains(err.Error(), `invalid format "yaml"`) {
		t.Errorf("error = %v, want it to name the rejected format", err)
	}
}

// A malformed override stops startup rather than being silently discarded.
func TestConfig_FinalizeRejectsMalformedEnvValue(t *testing.T) {
	t.Setenv(envLevel, "nonsense")

	var cfg logging.Config
	if err := cfg.Finalize(testPrefix); err == nil {
		t.Fatal("Finalize returned nil for a malformed level override")
	}
}

func TestConfig_LoadsThroughConfigLoad(t *testing.T) {
	dir := t.TempDir()
	body := `{"level":"DEBUG"}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(body), 0o600); err != nil {
		t.Fatalf("write config.json: %v", err)
	}

	cfg, err := config.Load[logging.Config](config.Options{Dir: dir})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Level != logging.LevelDebug {
		t.Errorf("Level = %q, want debug (normalized from the file)", cfg.Level)
	}
	if cfg.Format != logging.FormatText {
		t.Errorf("Format = %q, want the default", cfg.Format)
	}
}
