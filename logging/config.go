package logging

import (
	"fmt"
	"os"
	"strings"
)

const (
	defaultLevel  = LevelInfo
	defaultFormat = FormatText
)

// Config selects the level and format the process logger is built with. Env
// records the environment-variable names Finalize composed and read; it is
// excluded from JSON.
type Config struct {
	Level  Level  `json:"level"`
	Format Format `json:"format"`
	Env    Env    `json:"-"`
}

// Merge overlays src's set fields onto the receiver.
func (c *Config) Merge(src *Config) {
	if src.Level != "" {
		c.Level = src.Level
	}
	if src.Format != "" {
		c.Format = src.Format
	}
}

// Finalize normalizes the values (trimming and lower-casing), applies
// defaults (info, text), applies the environment overrides named by Env, and
// validates.
func (c *Config) Finalize(envPrefix string) error {
	c.Env = NewEnv(envPrefix)
	c.normalize()
	c.applyDefaults()
	c.applyEnv()
	c.normalize()
	return c.validate()
}

func (c *Config) applyDefaults() {
	if c.Level == "" {
		c.Level = defaultLevel
	}
	if c.Format == "" {
		c.Format = defaultFormat
	}
}

func (c *Config) applyEnv() {
	if v := os.Getenv(c.Env.Level); v != "" {
		c.Level = Level(v)
	}
	if v := os.Getenv(c.Env.Format); v != "" {
		c.Format = Format(v)
	}
}

func (c *Config) normalize() {
	c.Level = Level(strings.ToLower(strings.TrimSpace(string(c.Level))))
	c.Format = Format(strings.ToLower(strings.TrimSpace(string(c.Format))))
}

func (c *Config) validate() error {
	if _, err := c.Level.Slog(); err != nil {
		return fmt.Errorf("invalid level %q: %w", c.Level, err)
	}
	if !c.Format.Valid() {
		return fmt.Errorf("invalid format %q", c.Format)
	}
	return nil
}
