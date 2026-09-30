# Changelog

All notable changes to `github.com/standards-lab/go-core` are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the module adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.5.0] - 2026-09-30

The closing review of the storage suite: correctness fixes across the lifecycle, the loader,
and the integration toolkit; a stricter configuration load; and documentation cut to the
contracts it states.

### Added

- `config.SetFromEnv` — the generic form of `SetDurationFromEnv`: it applies one environment
  override to a tri-state (pointer) field through a parse function such as `strconv.Atoi`.
  `SetDurationFromEnv` is now its `Duration` form.

### Changed

- **Breaking:** `config.Load` rejects a key the configuration type does not declare, in any of
  the four files, and data after the top-level value. Every file decodes into the same type, so
  a stray or misspelled key in `config.json` or `secrets.json` now fails the load instead of
  being ignored.
- **Breaking:** `config.EnvName` returns `""` when the prefix is empty, or empty once sanitized,
  where it composed the name from the parts alone; an empty name reads as no override, so an
  empty prefix disables every override without a guard, and a whitespace-only `EnvPrefix` no
  longer derives `ENV` as `EnvVar`. `logging.NewEnv` drops its own guard accordingly.
- **Breaking:** `config.Load` fails an environment value containing `/`, `\`, or `..`, and an
  `OverlayPattern` that drops the stem or the environment, which would resolve both overlays,
  or every environment, to one file.
- **Breaking:** `lifecycle.Coordinator.Run` panics on a drain timeout that is not positive,
  which would time out every drain.
- `lifecycle`: the zero `Coordinator` is ready to use; `New` remains.
- `lifecycle`: the first startup failure cancels the run context at once, so the rest of its
  phase can stop early; the errors wrapping `context.Canceled` returned once the run context is
  cancelled and a failure is on record are its consequence and are dropped from Run's error.
- `process/processtest`: `Launch` appends its race option to the parent's `GORACE` instead of
  replacing it.
- Package documentation is trimmed to the contracts it states.

### Fixed

- `lifecycle`: a startup failure drained with the run context still live, so work a started
  service left running on it kept running through the drain.
- `lifecycle`: a signal during startup made `Run` return the cancellations of the Starts that
  honored it; a startup cut short only by the signal context's cancellation now drains as a
  clean exit, and `Run` returns nil when the drain is clean.
- `config`: an empty file fails its load as `empty file` rather than a bare `EOF`, and malformed
  data after the top-level value keeps the decoder's error.
- `process/processtest`: `Stop` failed the test when the process had exited on its own and was
  reaped before `Exited` reported it.
- `process/processtest`: `Forwarder` dialed the backing service in its accept loop, so a target
  that never answered stalled every later connection and `Sever`.
- `process/processtest`: `Forwarder.Restore` without a prior `Sever` failed on the address in
  use; it is now a no-op while listening, and concurrent calls restore once.
- `config`: a number in exponent form for a `Duration` no longer reports itself as a fraction.

### Removed

- **Breaking:** `processtest.Forwarder.Close`, identical to `Sever`; `Forward` registers the
  cleanup, and a test that severs early calls `Sever`.

## [v0.4.1] - 2026-09-07

### Fixed

- `process/processtest`: `Main` resolves the module root as the suite package's own module.
  It listed modules, which under a multi-module `go.work` is every module, so a suite in a
  workspace of sibling checkouts failed to build its program before any test ran.

## [v0.4.0] - 2026-09-07

The integration toolkit beside `process`. A library whose infrastructure is exercised by
integration testing ships its toolkit beside it, and the reference service's harness proved
the runner and the relay process-level: they move here from the service, their API as built.

### Added

- `process/processtest` — runs a program as the binary and drives it only through the seams a
  terminal or an orchestrator uses: `Main` builds one main package per suite run with the race
  detector, `Launch` runs it as a subprocess with a composed environment and captured output,
  `Process.Await` waits on an observable condition and fails with the output if the process
  exits first, `Process.Stop` interrupts it and returns the exit code, and `FreePort` and
  `WaitFor` sit beside them. `Forwarder` is a loopback TCP relay between the process and a
  backing service, the seam a test injects an outage through: `Sever` refuses new connections
  and drops the open ones, `Restore` listens again on the same address. Hermetic: the package
  proves itself on the unit tier against its own test program.

### Fixed

- `config`: the overlay-pattern probe filename typo (`proble.json` → `probe.json`); the probe is
  a throwaway input to the format-verb check, so behavior is unchanged.

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

[Unreleased]: https://github.com/standards-lab/go-core/compare/v0.5.0...HEAD
[v0.5.0]: https://github.com/standards-lab/go-core/compare/v0.4.1...v0.5.0
[v0.4.1]: https://github.com/standards-lab/go-core/compare/v0.4.0...v0.4.1
[v0.4.0]: https://github.com/standards-lab/go-core/compare/v0.3.0...v0.4.0
[v0.3.0]: https://github.com/standards-lab/go-core/compare/v0.2.0...v0.3.0
[v0.2.0]: https://github.com/standards-lab/go-core/compare/v0.1.0...v0.2.0
[v0.1.0]: https://github.com/standards-lab/go-core/releases/tag/v0.1.0
