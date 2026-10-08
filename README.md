# go-core

go-core is the Core SDK for Standards Lab's Go Elemental standard. It provides layered
configuration, a dependency graph and the process lifecycle that runs it, and logging.

`github.com/standards-lab/go-core` is a single Go module with the process-level packages every
program in the standard builds on.

## Standard

`go-core` is the Core SDK of
[Go Elemental](https://github.com/standards-lab/architecture/blob/main/standards/go-elemental/README.md), the
minimal-dependency Go standard. This README and each package's `doc.go` document the
repository; the standard's principles it enhances are stated below.
It enhances the standard's dependency line: where Go Elemental admits packages as idiomatic and
stable as the standard library, this repository depends on the standard library alone.

## Packages

- `config` — loads configuration in layers: a base file, an environment overlay, and secrets.
- `graph` — a typed dependency graph: nodes described inertly, each a name and a constructor,
  and a Build that constructs what the roots reach into a System of computed layers.
- `lifecycle` — runs a built `graph` System: starts its layers in order, tracks and reports
  readiness, watches for runtime failures, and shuts the layers down in reverse within a
  timeout. A value takes part through the interfaces it implements. `Exec` is the one-shot
  form a CLI command uses; `Run` is the long-running form a service uses.
- `logging` — builds an `*slog.Logger` from a configuration that `config` loads.
- `process` — the parts of a binary's main sequence that run before the program's own
  infrastructure exists: the signal-derived root context, pre-logger failure reporting, and the
  exit-code convention.
- `process/processtest` — the integration toolkit: it builds a program once per suite run and
  runs it as a subprocess. It then waits on what a client observes, interrupts the process and
  reads its exit code, and relays a backing service's connection so a test can sever it. `Run`
  runs the program once and returns its stdout, stderr, and exit code.

## Development

Tasks run through [mise](https://mise.jdx.dev):

```
mise run check      # build, vet, format, fix, tidy, test, and lint; writes nothing
mise run currency   # report requirements, Go, tools, and actions behind their latest
mise run upgrade    # upgrade the go directive, requirements, and tools to their latest
```

## License

[Apache License 2.0](LICENSE).
