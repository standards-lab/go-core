# Configuration

How configuration is loaded and finalized. The `config` package holds the machinery; the code and its
`doc.go` are authoritative for the package API.

## A loader in the base SDK, not a per-app convention

Earlier implementations of this layer left configuration as a per-package convention — each subsystem
shipped a `Config`/`Env`/`Merge`/`Finalize` set, but there was no shared interface and no loader, so every
consumer hand-wrote the same file-layering `Load` (base file, environment overlay, secrets, then finalize).
The `config` package makes that loader a real base primitive: a single generic `Load` that every consumer
calls, parameterized by the filenames and the environment-selector variable rather than reimplemented.
Consumers keep owning their own config types and merge/finalize behavior; what moves into the base is only
the orchestration they all shared.

## The contract

A configuration is a type `T` whose pointer implements the `config.Config[T]` constraint: `Merge(*T)`
and `Finalize(envPrefix string) error`. Expressing the contract as a generic constraint keeps the
methods concretely typed — a consumer writes `Merge(*Config)` and `Finalize(envPrefix string) error`
against its own type, with no `any` and no type assertions — while letting `Load` drive any conforming
type.

- **Merge** overlays a source's set fields onto the receiver: a non-zero source field wins, a zero one is
  left alone, and nested sub-configs delegate to their own merge. It is written field by field, without
  reflection. Because a set field always wins over an unset receiver, merging a fully-populated layer onto
  a zero value reproduces that layer — so `Load` treats the base file the same as every overlay rather
  than special-casing it.
- **Finalize** runs once, after every file has been merged, in a fixed order: compose the environment
  override names from the prefix it receives, apply defaults, read the overrides, then validate.
  Deferring validation to the end lets a required value arrive from any layer or from the environment.
  A malformed environment value fails Finalize — a bad input stops startup rather than being silently
  discarded. An empty prefix composes no names, so no overrides apply — the hermetic form tests use.

## Layered load

`Load` reads up to four files from a directory, in precedence order, merging each that exists:

1. base file — `config.json`
2. environment overlay — `config.<env>.json`
3. secrets file — `secrets.json`
4. secrets overlay — `secrets.<env>.json`

then calls `Finalize` once, passing `Options.EnvPrefix`. The active environment is the value of the
selector variable — named by `Options.EnvVar`, or derived from `EnvPrefix` as the prefixed `env` name
when `EnvVar` is unset; when it resolves empty, both overlays are skipped. A single overlay pattern
produces both overlay names from the base and secrets stems, so the two files layer consistently. Every
file is optional — a missing file is skipped, and a directory with none yields a configuration carrying
only what `Finalize` supplies; any other read error, or malformed JSON, stops the load. Later files win
over earlier ones, and the environment overrides `Finalize` reads win over every file.

## Environment-variable names

Environment overrides are structured, not scattered `os.Getenv` calls. A subsystem pairs its config with
an `Env` struct whose fields hold the variable names its `Finalize` reads, composed with
`config.EnvName(prefix, parts...)` — which upper-cases each segment, collapses separators to single
underscores, drops empty segments, and joins with underscores. The struct is uniformly named `Env`
across subsystems.

Finalize composes its own `Env` from the prefix it receives and records it for introspection. The
earlier shape — the composition root seeding `Env` before calling `Finalize()` — made the seed and the
call an inseparable pairing whose forgotten half silently disabled every override; the parameter makes
the compiler enforce the pairing, and the prefix keeps the subsystem free of any one application's
naming just as the seeded struct did. `NewEnv` stays exported as the escape hatch for a
custom-structured `Env`, and `NewEnv("")` is the zero `Env`.

## Configuration is ephemeral

A configuration exists to initialize subsystems, not to be retained. A composition root loads it,
constructs subsystems from its values, and discards it; runtime code holds the values it needs, not the
configuration. This keeps the point at which a setting is read fixed at startup, and lets a
subsystem be constructed in a test from plain values without assembling a whole configuration graph.
