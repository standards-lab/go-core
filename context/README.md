# go-core

The Core SDK of Go Minimal, the Standards Lab organization's minimal-dependency Go standard:
the common primitives useful across all Go Minimal application types, and the first place the
standard becomes code.

The design and conventions of this repository are documented in the organization's
[documentation landing zone](https://github.com/standards-lab/docs); this context records only
working knowledge the landing zone and the code do not express. The repository page is
[go-core](https://github.com/standards-lab/docs/blob/main/standards/go-minimal/go-core/index.md),
under the [Go Minimal](https://github.com/standards-lab/docs/blob/main/standards/go-minimal/index.md)
standard. This repository enhances the standard's dependency line to the standard library alone.

## Capability map

All four packages are built; the code and each package's `doc.go` are authoritative for the
API, and the landing zone documents the design.

- **config** — layered configuration through the merge/finalize contract. Documented in
  [Configuration](https://github.com/standards-lab/docs/blob/main/standards/go-minimal/go-core/config.md).
- **lifecycle** — the process lifecycle: staged services (ordered startup, reverse-stage
  drain, named readiness checks), bracketing hooks, live readiness, timeout-bounded drain.
  Documented in the standard's
  [lifecycle and context ownership](https://github.com/standards-lab/docs/blob/main/standards/go-minimal/principles/lifecycle-and-context.md)
  principle.
- **logging** — the `*slog.Logger` a process writes through. Documented in
  [Logging](https://github.com/standards-lab/docs/blob/main/standards/go-minimal/go-core/logging.md).
- **process** — the pre-infrastructure main sequence: the signal-derived root context,
  pre-logger failure and usage reporting, and the exit-code convention the reporters return.
  Documented alongside the
  [lifecycle and context ownership](https://github.com/standards-lab/docs/blob/main/standards/go-minimal/principles/lifecycle-and-context.md)
  principle.

The map is complete for the tier. The Core SDK grows only when a pattern proves process-level
and universal; it never grows toward one application type or one external technology.
