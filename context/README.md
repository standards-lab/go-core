# go-core

The base SDK of `go-minimal`, the Standards Lab organization's minimal-dependency Go standard. This
is the standard's base tier: the process-level packages every program in the standard builds on, and
the first place the standard materializes as code. Built with the marathon workflow, so the workflow
itself is exercised in the process.

## What we're building toward

- The lowest practical level of abstraction, no frameworks. Dependencies flow downward only, and
  this repository is the bottom of the graph: it depends on the standard library alone and is
  unaware of what runs above it.
- Process-level and universal. A package belongs in the base SDK only when every type of program in
  the standard uses it; anything specific to one application type or one external technology lives
  in its own repository, above this one.
- One Go module, versioned and released as one artifact on root tags.

## Repository topology

The repository is a single Go module rooted at `github.com/standards-lab/go-core`, with each
capability a package inside it: `config`, `lifecycle`, `logging`. There are no sub-modules and no
providers. The module may depend only on packages as idiomatic and stable as the standard library
itself; today it depends on the standard library alone.

## Capability map

All three packages are built; the code and each package's `doc.go` are authoritative for the API.

- **config** — layered configuration: a base file, environment overlays, and secrets, resolved
  through a merge/finalize contract each subsystem's config implements. See `design/config.md`.
- **lifecycle** — the process lifecycle for long-running programs: concurrent startup, a readiness
  signal that tracks subsystem status through startup and drain, and timeout-bounded graceful
  shutdown. Conceptually, cold start initializes objects from configuration so their state is valid
  (the `/healthz` side), and hot start brings the long-running subsystems up until they are ready to
  receive work (the `/readyz` side).
- **logging** — the `*slog.Logger` a process writes through, built from a configuration that takes
  part in the layered load. It constructs a logger and nothing more; the level vocabulary is
  `slog`'s. See `design/logging.md`.

The map is complete for the tier. The base SDK grows only when a pattern proves process-level and
universal; it never grows toward any one application type or external technology.

## How this repository works

- **Configuration** — the loader, the merge/finalize contract, and the environment-variable
  convention. See `design/config.md`.
- **Logging** — why the vocabulary is `slog`'s and the writer is a parameter. See
  `design/logging.md`.
- **Package conventions** — the dependency rule, process lifecycle and context ownership, co-located
  black-box tests, `doc.go` ownership. See `design/conventions.md`.
- **Releases and CI** — root tags from `CHANGELOG.md`, the CI checks, the `mise` tasks. See
  `design/release-and-ci.md`.
