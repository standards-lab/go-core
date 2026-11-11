# go-core

go-core is the Core SDK of Go Elemental, the Standards Lab organization's Go implementation of
the Elemental Architecture. It provides the common primitives every Go Elemental application
type uses, and is the first place the standard becomes code.

The README and each package's `doc.go` document this repository. The
[Go Elemental](https://github.com/standards-lab/architecture/blob/main/standards/go-elemental/README.md) standard states the principles it follows.
This context records only working knowledge the code and the README do not express.

- [`graph-ergonomics.md`](graph-ergonomics.md) holds what the code does not express about
  v0.11.0's `graph` and `lifecycle` API: the pain points it answers, the behavior it kept, the
  reasons behind its decisions, the rejected alternatives, and what the consumers' migrations
  need, with sketches of their target.

## Capability map

go-core has nine packages; the README lists them, and the code and each package's `doc.go`
are authoritative for the API.

- **config** — layered configuration.
- **event** — the CloudEvents 1.0 envelope and its codec, and the event kinds, queue, and
  recorder a domain raises and emits its events with.
- **graph** — the typed dependency graph the lifecycle runs.
- **graph/graphtest** — the test toolkit beside `graph`: the check of a struct of handles
  against its nodes' names.
- **lifecycle** — the process lifecycle that runs the graph's layers, governed by the standard's
  [lifecycle and context ownership](https://github.com/standards-lab/architecture/blob/main/standards/go-elemental/principles/lifecycle-and-context.md)
  principle.
- **logging** — the process logger.
- **process** — the pre-infrastructure main sequence, documented alongside the lifecycle
  principle.
- **process/processtest** — the integration toolkit beside `process`, promoted from the
  reference service's harness.
- **reactor** — the lifecycle component that runs one source of occurrences into one handler
  and drains at shutdown, its interval sources `Every` and `Wake`, and the quiesce gate `Gate`.

The map is complete for the tier.
