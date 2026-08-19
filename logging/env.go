package logging

import "github.com/standards-lab/go-core/config"

// Env names the environment variables [Config.Finalize] reads. An empty name
// disables that one override; the zero value disables both.
type Env struct {
	Level  string
	Format string
}

// NewEnv composes the standard override names from a prefix: LOG_LEVEL and
// LOG_FORMAT under whatever prefix [config.EnvName] produces.
func NewEnv(prefix string) Env {
	if prefix == "" {
		return Env{}
	}
	return Env{
		Level: config.EnvName(
			prefix, "log", "level",
		),
		Format: config.EnvName(
			prefix, "log", "format",
		),
	}
}
