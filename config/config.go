package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultBaseName       = "config.json"
	defaultSecretsName    = "secrets.json"
	defaultOverlayPattern = "%s.%s.json"
)

// Config is the constraint [Load] places on *T: Merge overlays src's set
// fields onto the receiver, and Finalize applies defaults and
// envPrefix-named overrides, then validates.
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
	// EnvPrefix is passed to Finalize to name the environment overrides, and
	// derives EnvVar when that is empty; empty disables both.
	EnvPrefix string
	// EnvVar names the variable whose value selects the overlay environment;
	// empty derives <PREFIX>_ENV from EnvPrefix. An unset or empty value
	// skips both overlays.
	EnvVar string
	// BaseName is the base file name; "config.json" when empty.
	BaseName string
	// SecretsName is the secrets file name; "secrets.json" when empty.
	SecretsName string
	// OverlayPattern produces an overlay name from a file's stem and the
	// environment, using both; "%s.%s.json" when empty.
	OverlayPattern string
}

func (o *Options) withDefaults() {
	if o.Dir == "" {
		o.Dir = "."
	}
	if o.EnvVar == "" {
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

// validate probes the overlay pattern with distinct markers: a pattern
// that misuses its verbs, or drops the stem or the environment, would name
// one file for both overlays or for every environment.
func (o Options) validate() error {
	const stem, env = "stemprobe", "envprobe"
	probe := o.overlay(stem+".json", env)
	if strings.Contains(probe, "%!") ||
		!strings.Contains(probe, stem) || !strings.Contains(probe, env) {
		return fmt.Errorf("invalid overlay pattern %q", o.OverlayPattern)
	}
	return nil
}

// validEnv rejects an environment value that would reach outside Dir
// through the overlay name.
func validEnv(env string) error {
	if strings.ContainsAny(env, `/\`) || strings.Contains(env, "..") {
		return fmt.Errorf("invalid environment %q: it must not contain a path separator or \"..\"", env)
	}
	return nil
}

// Load reads the layered configuration files opts names — base, environment
// overlay, secrets, secrets overlay, later files winning — merges each one
// that exists onto a zero value of T, finalizes the result, and returns it. A
// missing file is skipped; a read failure, malformed JSON, a key T does not
// declare, a malformed overlay pattern, an environment value containing a
// path separator or "..", or a Finalize error stops the load.
func Load[T any, PT Config[T]](opts Options) (PT, error) {
	opts.withDefaults()

	if err := opts.validate(); err != nil {
		return nil, err
	}

	env := os.Getenv(opts.EnvVar)
	if err := validEnv(env); err != nil {
		return nil, fmt.Errorf("%s: %w", opts.EnvVar, err)
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
		if err := decode(data, layer); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		cfg.Merge(layer)
	}

	if err := cfg.Finalize(opts.EnvPrefix); err != nil {
		return nil, fmt.Errorf("finalize config: %w", err)
	}
	return cfg, nil
}

// decode unmarshals one JSON value into v, rejecting keys v does not declare
// and anything after the value.
func decode(data []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return errors.New("invalid character after top-level value")
	}
	return nil
}
