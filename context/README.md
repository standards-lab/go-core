# go-core

go-core is the Core SDK of Go Elemental, the Standards Lab organization's Go implementation of
the Elemental Architecture. It provides the common primitives every Go Elemental application
type uses, and is the first place the standard becomes code.

The README, [`docs/graph-design.md`](../docs/graph-design.md), and each package's `doc.go`
document this repository. The
[Go Elemental](https://github.com/standards-lab/architecture/blob/main/standards/go-elemental/README.md)
standard states the principles it follows.

## Capability map

- **config** (built): layered configuration ([doc.go](../config/doc.go)).
- **event** (built): the CloudEvents 1.0 envelope and its codec, and the event kinds, queue, and
  recorder a domain raises and emits its events with ([doc.go](../event/doc.go)).
- **graph** (built): the typed dependency graph the lifecycle runs
  ([doc.go](../graph/doc.go)).
- **graph/graphtest** (built): the check of a struct of handles against its nodes' names
  ([doc.go](../graph/graphtest/doc.go)).
- **lifecycle** (built): the process lifecycle that runs the graph's layers
  ([doc.go](../lifecycle/doc.go)).
- **logging** (built): the process logger ([doc.go](../logging/doc.go)).
- **process** (built): the pre-infrastructure main sequence ([doc.go](../process/doc.go)).
- **process/processtest** (built): the integration toolkit beside `process`
  ([doc.go](../process/processtest/doc.go)).
- **reactor** (built): the lifecycle component that runs one source of occurrences into one
  handler and drains at shutdown, its interval sources, and the quiesce gate
  ([doc.go](../reactor/doc.go)).
- **sse** (built): the server-sent event stream reader ([doc.go](../sse/doc.go)).
