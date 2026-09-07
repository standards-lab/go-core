# go-core

The Core SDK of Go Elemental, the Standards Lab organization's Go implementation of the
Elemental Architecture: the common primitives useful across all Go Elemental application
types, and the first place the
standard becomes code.

The design and conventions of this repository are documented in the organization's
[documentation landing zone](https://github.com/standards-lab/docs); this context records only
working knowledge the landing zone and the code do not express. The repository page is
[go-core](https://github.com/standards-lab/docs/blob/main/standards/go-elemental/go-core/index.md),
under the [Go Elemental](https://github.com/standards-lab/docs/blob/main/standards/go-elemental/index.md)
standard. This repository enhances the standard's dependency line to the standard library alone.

## Capability map

All five packages are built; the code and each package's `doc.go` are authoritative for the
API, and the landing zone documents the design.

- **config** — layered configuration through the merge/finalize contract. Documented in
  [Configuration](https://github.com/standards-lab/docs/blob/main/standards/go-elemental/go-core/config.md).
- **lifecycle** — the process lifecycle: staged services (ordered startup, reverse-stage
  drain, named readiness checks), bracketing hooks, live readiness, timeout-bounded drain.
  Documented in the standard's
  [lifecycle and context ownership](https://github.com/standards-lab/docs/blob/main/standards/go-elemental/principles/lifecycle-and-context.md)
  principle.
- **logging** — the `*slog.Logger` a process writes through. Documented in
  [Logging](https://github.com/standards-lab/docs/blob/main/standards/go-elemental/go-core/logging.md).
- **process** — the pre-infrastructure main sequence: the signal-derived root context,
  pre-logger failure and usage reporting, and the exit-code convention the reporters return.
  Documented alongside the
  [lifecycle and context ownership](https://github.com/standards-lab/docs/blob/main/standards/go-elemental/principles/lifecycle-and-context.md)
  principle.
- **process/processtest** — the integration toolkit beside `process`: the runner that builds a
  program once per suite run and drives it as a subprocess through signals, its exit code, and
  what a client observes, and the loopback relay a test severs to inject an outage. Built at
  `v1.data.sql.tasks.toolkit` (2026-09-07) from the reference service's harness; the landing
  zone page is due in the docs pass.

The map is complete for the tier. The Core SDK grows only when a pattern proves process-level
and universal; it never grows toward one application type or one external technology.
