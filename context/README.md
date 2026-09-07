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
standard.

## Capability map

All five packages are built; the README lists them, the code and each package's `doc.go` are
authoritative for the API, and the landing zone documents the design.

- **config** — layered configuration.
  [Configuration](https://github.com/standards-lab/docs/blob/main/standards/go-elemental/go-core/config.md).
- **lifecycle** — the staged process lifecycle. The standard's
  [lifecycle and context ownership](https://github.com/standards-lab/docs/blob/main/standards/go-elemental/principles/lifecycle-and-context.md)
  principle.
- **logging** — the process logger.
  [Logging](https://github.com/standards-lab/docs/blob/main/standards/go-elemental/go-core/logging.md).
- **process** — the pre-infrastructure main sequence. Documented alongside the lifecycle
  principle.
- **process/processtest** — the integration toolkit beside `process`, built at
  `v1.data.sql.tasks.toolkit` (2026-09-07) from the reference service's harness; its landing
  zone page is due in `v1.alignment.docs`.

The map is complete for the tier. The Core SDK grows only when a pattern proves process-level
and universal; it never grows toward one application type or one external technology.
