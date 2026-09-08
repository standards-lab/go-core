# go-core

The Core SDK of Go Elemental, the Standards Lab organization's Go implementation of the
Elemental Architecture: the common primitives useful across all Go Elemental application
types, and the first place the
standard becomes code.

The README and each package's `doc.go` document this repository, and the
[Go Elemental](https://github.com/standards-lab/architecture/blob/main/standards/go-elemental/README.md) standard states the principles it follows; this context records only
working knowledge the code and the README do not express.

## Capability map

All five packages are built; the README lists them, and the code and each package's `doc.go`
are authoritative for the API.

- **config** — layered configuration.
- **lifecycle** — the staged process lifecycle. The standard's
  [lifecycle and context ownership](https://github.com/standards-lab/architecture/blob/main/standards/go-elemental/principles/lifecycle-and-context.md)
  principle.
- **logging** — the process logger.
- **process** — the pre-infrastructure main sequence. Documented alongside the lifecycle
  principle.
- **process/processtest** — the integration toolkit beside `process`, built at
  `v1.data.sql.tasks.toolkit` (2026-09-07) from the reference service's harness.

The map is complete for the tier. The Core SDK grows only when a pattern proves process-level
and universal; it never grows toward one application type or one external technology.
