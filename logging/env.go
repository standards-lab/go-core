package logging

import "github.com/standards-lab/go-core/config"

// Env names the environment variables [Config.Finalize] reads. An empty name
// disables that one override; the zero value disables both.
type Env struct {
	Level  string
	Format string
}

// NewEnv composes <PREFIX>_LOG_LEVEL and <PREFIX>_LOG_FORMAT with
// [config.EnvName]; an empty prefix returns the zero Env.
func NewEnv(prefix string) Env {
	return Env{
		Level:  config.EnvName(prefix, "log", "level"),
		Format: config.EnvName(prefix, "log", "format"),
	}
}
