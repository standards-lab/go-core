# Changelog

All notable changes to `github.com/standards-lab/go-core` are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the module adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.1.0] - 2026-08-19

The first release of the base SDK: the `config`, `lifecycle`, and `logging` packages. The module
depends on the standard library alone.

### Added

- `config` — the layered configuration loader: a generic `Load` over the `config.Config[T]`
  merge/finalize contract, reading a base file, an environment overlay, secrets, and a secrets
  overlay; structured environment-variable overrides composed by `config.EnvName` and applied in
  `Finalize`; `config.Duration` and `SetDurationFromEnv` for duration fields.
- `lifecycle` — the process lifecycle coordinator: concurrent startup hooks, a non-monotonic
  readiness signal, monitored runtime failure, and a two-phase timeout-bounded graceful drain;
  `lifecycle.ReadinessChecker` for subsystems that expose their own readiness.
- `logging` — construction of the process `*slog.Logger` from a configuration that takes part in the
  layered load: `Level` delegating its vocabulary to `slog`, `Format` selecting the handler, and the
  writer as a parameter to `New`.
