# go-core

Go base SDK for Standards Lab: layered configuration, process lifecycle, and logging.

`github.com/standards-lab/go-core` is a single Go module with the process-level packages every Go
program in the organization builds on. It depends on the standard library alone.

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
