package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultBaseName       = "config.json"
	defaultSecretsName    = "secrets.json"
	defaultOverlayPattern = "%s.%s.json"
)

// Config is the contract a configuration type's pointer implements to take
// part in [Load]: Merge overlays another instance's set fields onto the
// receiver, and Finalize composes its environment override names from the
// prefix, applies defaults, reads the overrides, and validates.
type Config[T any] interface {
	*T
	Merge(src *T)
	Finalize(envPrefix string) error
}

// Options locates and names the files [Load] reads. The zero value reads
// config.json and secrets.json from the current directory, with no
// environment overlays and no environment overrides.
type Options struct {
	// Dir is the directory the files are read from; "." when empty.
	Dir string
	// EnvPrefix is the prefix the loaded configuration's Finalize composes
	// its environment-variable names from; empty disables environment
	// overrides.
	EnvPrefix string
	// EnvVar names the environment variable whose value selects the overlay
	// environment. Empty, or naming an unset variable, skips both overlays.
	EnvVar string
	// BaseName is the base file name; "config.json" when empty.
	BaseName string
	// SecretsName is the secrets file name; "secrets.json" when empty.
	SecretsName string
	// OverlayPattern produces an overlay name from a file's stem and the
	// environment; "%s.%s.json" when empty.
	OverlayPattern string
}

func (o *Options) withDefaults() {
	if o.Dir == "" {
		o.Dir = "."
	}
	if o.EnvVar == "" && o.EnvPrefix != "" {
		o.EnvVar = EnvName(o.EnvPrefix, "env")
	}
	if o.BaseName == "" {
		o.BaseName = defaultBaseName
	}
	if o.SecretsName == "" {
		o.SecretsName = defaultSecretsName
	}
	if o.OverlayPattern == "" {
		o.OverlayPattern = defaultOverlayPattern
	}
}

func (o Options) overlay(name, env string) string {
	stem := strings.TrimSuffix(name, filepath.Ext(name))
	return fmt.Sprintf(o.OverlayPattern, stem, env)
}

func (o Options) validate() error {
	if probe := o.overlay("probe.json", "env"); strings.Contains(probe, "%!") {
		return fmt.Errorf("invalid overlay pattern %q", o.OverlayPattern)
	}
	return nil
}

// Load reads the layered configuration files opts names — base, environment
// overlay, secrets, secrets overlay, later files winning — merges each one
// that exists onto a zero value of T, finalizes the result, and returns it. A
// missing file is skipped; a read failure, malformed JSON, a malformed overlay
// pattern, or a Finalize error stops the load.
func Load[T any, PT Config[T]](opts Options) (PT, error) {
	opts.withDefaults()

	if err := opts.validate(); err != nil {
		return nil, err
	}

	var env string
	if opts.EnvVar != "" {
		env = os.Getenv(opts.EnvVar)
	}

	names := []string{opts.BaseName}
	if env != "" {
		names = append(names, opts.overlay(opts.BaseName, env))
	}
	names = append(names, opts.SecretsName)
	if env != "" {
		names = append(names, opts.overlay(opts.SecretsName, env))
	}

	cfg := PT(new(T))
	for _, name := range names {
		path := filepath.Join(opts.Dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		layer := new(T)
		if err := json.Unmarshal(data, layer); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		cfg.Merge(layer)
	}

	if err := cfg.Finalize(opts.EnvPrefix); err != nil {
		return nil, fmt.Errorf("finalize config: %w", err)
	}
	return cfg, nil
}
