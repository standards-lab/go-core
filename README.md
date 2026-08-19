# go-core

Go base SDK for Standards Lab: layered configuration, process lifecycle, and logging.

`github.com/standards-lab/go-core` is a single Go module with the process-level packages every
program in the standard builds on.

## Target standard

`go-core` is the base SDK of `go-minimal`, the minimal-dependency Go standard. Its dependency rule
is the standard's line — at most packages as idiomatic and stable as the standard library
(`golang.org/x/…`, `google/uuid`), with vendor SDKs never entering — and today it depends on the
standard library alone.

## Packages

- `config` — loads configuration in layers: a base file, an environment overlay, and secrets.
- `lifecycle` — starts subsystems concurrently, tracks and reports readiness as subsystem status
  changes, and shuts down within a timeout.
- `logging` — builds an `*slog.Logger` from a configuration that `config` loads.

## Development

Tasks run through [mise](https://mise.jdx.dev):

```
mise run test
```

## License

[Apache License 2.0](LICENSE).
