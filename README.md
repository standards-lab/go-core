# go-core

Core SDK for Standards Lab's Go Elemental standard: layered configuration, process lifecycle, and
logging.

`github.com/standards-lab/go-core` is a single Go module with the process-level packages every
program in the standard builds on.

## Standard

`go-core` is the Core SDK of
[Go Elemental](https://github.com/standards-lab/docs/blob/main/standards/go-elemental/index.md), the
minimal-dependency Go standard, and its design is documented on the standard's
[go-core page](https://github.com/standards-lab/docs/blob/main/standards/go-elemental/go-core/index.md).
It enhances the standard's dependency line: where Go Elemental admits packages as idiomatic and
stable as the standard library, this repository depends on the standard library alone.

## Packages

- `config` — loads configuration in layers: a base file, an environment overlay, and secrets.
- `lifecycle` — starts subsystems concurrently, tracks and reports readiness as subsystem status
  changes, and shuts down within a timeout.
- `logging` — builds an `*slog.Logger` from a configuration that `config` loads.
- `process` — the parts of a binary's main sequence that run before the program's own
  infrastructure exists: the signal-derived root context, pre-logger failure reporting, and the
  exit-code convention.
- `process/processtest` — the integration toolkit: builds a program once per suite run, runs it
  as a subprocess, waits on what a client observes, interrupts it and reads its exit code, and
  relays a backing service's connection so a test can sever it.

## Development

Tasks run through [mise](https://mise.jdx.dev):

```
mise run test
```

## License

[Apache License 2.0](LICENSE).
