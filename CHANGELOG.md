# Changelog

All notable changes to `github.com/standards-lab/go-core` are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the module adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.3.0] - 2026-08-24

The pre-infrastructure main sequence is now a package. Every composition root in the standard
carried the same signal wiring and exit-code convention inline; `process` holds it once, so the
convention cannot drift between a program's binaries.

### Added

- `process` — the parts of a binary's main sequence that run before the program's own
  infrastructure exists: `SignalContext` builds the signal-derived root context, `Fail` and
  `Usage` report to a writer when no logger exists yet, and `ExitOK`/`ExitFailure`/`ExitUsage`
  fix the exit-code convention the reporters return.

### Changed

- The module builds on Go 1.27 (from 1.26), aligning it with the rest of the standard's
  modules.

## [v0.2.0] - 2026-08-21

The lifecycle coordinator now starts and stops services in stages: a subsystem declares its
name, its stage, and its lifecycle members once, and the coordinator owns staged startup,
reverse-stage drain, and named readiness.

### Added

- `lifecycle.Service` and `Coordinator.Add` — a service declares a `Name`, a `Stage`, and
  optional `Start`, `Shutdown`, and `Check` members. Numbered stages start ascending with a
  barrier between stages, and services within a stage start concurrently; `StageRoot` reserves
  the request edge, started after every numbered stage and drained first. The drain runs the
  stages in reverse and calls `Shutdown` only on services whose `Start` succeeded. `Add` panics
  on an empty or duplicate name, a negative stage, a service declaring none of its three
  members, and registration after `Run`.
- `lifecycle.Check`, pairing a readiness check with the name the probe reports for it, and
  `Coordinator.Checks`, returning the services' checks in start order. `Check` moves here from
  `go-web-sdk`, so application-generic code can name its readiness checks without importing the
  web SDK.

### Changed

- `lifecycle`: the hooks bracket the service stages — `OnStartup` hooks complete before the
  first stage starts, and `OnShutdown` hooks run after the last stage drains. A coordinator
  with no services behaves as in v0.1.0; the hook contracts are unchanged.

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
