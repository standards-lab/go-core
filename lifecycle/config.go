package lifecycle

import (
	"fmt"
	"os"
	"time"

	"github.com/standards-lab/go-core/config"
)

// defaultShutdownTimeout is the shutdown budget a zero [Config] finalizes
// to.
const defaultShutdownTimeout = 10 * time.Second

// Config is a [Coordinator]'s configuration. Env records the
// environment-variable names Finalize composed and read; it is excluded from
// JSON.
type Config struct {
	// ShutdownTimeout bounds the whole shutdown, every layer together.
	ShutdownTimeout config.Duration `json:"shutdown_timeout"`
	Env             Env             `json:"-"`
}

// Env names the environment variable [Config.Finalize] reads. An empty name
// disables the override.
type Env struct {
	ShutdownTimeout string
}

// NewEnv composes <PREFIX>_SHUTDOWN_TIMEOUT with [config.EnvName]; an empty
// prefix returns the zero Env.
func NewEnv(prefix string) Env {
	return Env{ShutdownTimeout: config.EnvName(prefix, "shutdown", "timeout")}
}

// Merge overlays src's set fields onto the receiver.
func (c *Config) Merge(src *Config) {
	if src.ShutdownTimeout != 0 {
		c.ShutdownTimeout = src.ShutdownTimeout
	}
}

// Finalize applies the default ShutdownTimeout of 10s when it is unset, reads
// the envPrefix-named override, then validates. An unparsable override is an
// error naming the variable, and a ShutdownTimeout that is not positive is an
// error.
func (c *Config) Finalize(envPrefix string) error {
	c.Env = NewEnv(envPrefix)
	if c.ShutdownTimeout == 0 {
		c.ShutdownTimeout = config.Duration(defaultShutdownTimeout)
	}
	if err := c.ShutdownTimeout.Set(os.Getenv(c.Env.ShutdownTimeout)); err != nil {
		return fmt.Errorf("%s: %w", c.Env.ShutdownTimeout, err)
	}
	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("shutdown_timeout must be positive, got %s", c.ShutdownTimeout)
	}
	return nil
}
