package lifecycle_test

import (
	"strings"
	"testing"
	"time"

	"github.com/standards-lab/go-core/config"
	"github.com/standards-lab/go-core/lifecycle"
)

// Instantiating Load proves *lifecycle.Config satisfies the config.Config
// contract at compile time.
var _ = config.Load[lifecycle.Config]

// envShutdownTimeout is what Finalize composes from the prefix "app".
const envShutdownTimeout = "APP_SHUTDOWN_TIMEOUT"

func TestConfig_MergeSourceWinsWithoutClearing(t *testing.T) {
	base := lifecycle.Config{ShutdownTimeout: config.Duration(time.Second)}
	base.Merge(&lifecycle.Config{})
	if got := base.ShutdownTimeout.Duration(); got != time.Second {
		t.Errorf("ShutdownTimeout = %v, want 1s (an unset source must not clear it)", got)
	}
	base.Merge(&lifecycle.Config{ShutdownTimeout: config.Duration(3 * time.Second)})
	if got := base.ShutdownTimeout.Duration(); got != 3*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 3s (the source sets it)", got)
	}
}

func TestConfig_FinalizeAppliesTheDefault(t *testing.T) {
	t.Setenv(envShutdownTimeout, "")
	var cfg lifecycle.Config
	if err := cfg.Finalize("app"); err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	if got := cfg.ShutdownTimeout.Duration(); got != 10*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 10s", got)
	}
	if cfg.Env.ShutdownTimeout != envShutdownTimeout {
		t.Errorf("Env.ShutdownTimeout = %q, want %q", cfg.Env.ShutdownTimeout, envShutdownTimeout)
	}
}

func TestConfig_FinalizeKeepsASetValue(t *testing.T) {
	t.Setenv(envShutdownTimeout, "")
	cfg := lifecycle.Config{ShutdownTimeout: config.Duration(time.Second)}
	if err := cfg.Finalize("app"); err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	if got := cfg.ShutdownTimeout.Duration(); got != time.Second {
		t.Errorf("ShutdownTimeout = %v, want 1s", got)
	}
}

func TestConfig_FinalizeEnvironmentOverrides(t *testing.T) {
	t.Setenv(envShutdownTimeout, "3s")
	cfg := lifecycle.Config{ShutdownTimeout: config.Duration(time.Second)}
	if err := cfg.Finalize("app"); err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	if got := cfg.ShutdownTimeout.Duration(); got != 3*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 3s", got)
	}
}

func TestConfig_FinalizeEmptyPrefixReadsNoOverride(t *testing.T) {
	t.Setenv(envShutdownTimeout, "3s")
	t.Setenv("SHUTDOWN_TIMEOUT", "3s")
	var cfg lifecycle.Config
	if err := cfg.Finalize(""); err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	if got := cfg.ShutdownTimeout.Duration(); got != 10*time.Second {
		t.Errorf("ShutdownTimeout = %v, want the 10s default", got)
	}
}

func TestConfig_FinalizeInvalidOverrideNamesTheVariable(t *testing.T) {
	t.Setenv(envShutdownTimeout, "soon")
	var cfg lifecycle.Config
	err := cfg.Finalize("app")
	if err == nil || !strings.Contains(err.Error(), envShutdownTimeout) {
		t.Errorf("Finalize = %v, want an error naming %s", err, envShutdownTimeout)
	}
}

func TestConfig_FinalizeRejectsATimeoutThatIsNotPositive(t *testing.T) {
	t.Setenv(envShutdownTimeout, "-1s")
	var cfg lifecycle.Config
	if err := cfg.Finalize("app"); err == nil {
		t.Error("Finalize = nil, want an error for a negative timeout")
	}
}
