// Package config loads layered JSON configuration and defines the contract a
// configuration type implements to take part in that load. A configuration is
// loaded, used to construct the subsystems, and discarded.
//
// # The contract
//
// A configuration is a type T whose pointer implements [Config]: a Merge that
// overlays another instance's set fields onto the receiver, and a Finalize
// that runs, in order:
//
//   - composes its environment override names from a prefix
//   - applies defaults
//   - reads the overrides
//   - validates
//
// # Layered load
//
// [Load] reads up to four files from a directory, in a fixed precedence, and
// merges each one that exists onto a zero value of T:
//
//   - a base file (config.json),
//   - an environment overlay (config.<env>.json),
//   - a secrets file (secrets.json), and
//   - a secrets overlay (secrets.<env>.json).
//
// The active environment is the value of the [Options.EnvVar] variable,
// named explicitly or derived from [Options.EnvPrefix] as the prefixed "env"
// name. When it resolves empty, both overlays are skipped; a value containing
// a path separator or ".." fails the load. A single [Options.OverlayPattern]
// produces both overlay names from the base and secrets stems; Load validates
// the pattern and fails one that misuses its verbs or drops the stem or the
// environment, rather than silently misnaming the overlays. Every file is
// optional: a missing file is skipped, so a load with no files present yields
// a configuration carrying only what Finalize supplies. Any other read error,
// malformed JSON, or a key T does not declare stops the load; every file
// decodes into the same T, so a secrets file carries only keys T declares.
//
// Later sources win: the secrets overlay overrides the secrets file, which
// overrides the environment overlay, which overrides the base. A set source
// field always wins over an unset receiver. Finalize runs once, after every
// file has been merged, receiving [Options.EnvPrefix].
//
// # Environment overrides
//
// A capability pairs its configuration with an Env struct naming the
// variables its Finalize reads. Env fields are excluded from JSON and are
// output, not input: each Finalize composes its own names from the prefix it
// receives and records them on Env for introspection. An empty prefix
// composes no names, so nothing in the environment applies. Hermetic tests
// rely on that. Once a prefix is supplied, the overrides Finalize reads take
// precedence over every file.
//
// [EnvName] composes the names. [SetFromEnv] applies one override to a
// tri-state (pointer) field through a parse function, and
// [SetDurationFromEnv] is its [Duration] form.
//
// # Durations
//
// [Duration] is a time.Duration that reads from JSON as "1m30s" or integer
// nanoseconds.
package config
