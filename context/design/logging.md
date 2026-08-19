# Logging

The design of the `logging` package. The code and its `doc.go` are authoritative for the package API;
this note holds the reasoning behind it.

## The standard library owns the vocabulary

Earlier implementations of this layer hand-rolled the same level enum: a string type, a `Valid()` switch
over four constants, and a `Slog()` switch converting it to `slog.Level`. None of that was needed.
`slog.Level` already implements `UnmarshalText`, which upper-cases the name before matching and accepts
offsets such as `warn+2`, so the whole vocabulary — including its case-insensitivity — is a delegation
rather than a switch.

So `logging` defines no level vocabulary. `Level.Slog` is a call into `slog`, and validation is that call
returning no error. Adding a level, or accepting a spelling `slog` accepts, costs this package nothing.
`Format` stays hand-written because the handler choice belongs to this package and `slog` has no opinion
about it.

This is the same finding that produced `config.Duration`: the ceremony re-implemented a type the standard
library already had.

## Level is a string because slog.Level's zero value is info

The one place the standard library's type could not be used directly is the configuration field.
`slog.LevelInfo` is `0`, so a `slog.Level` field cannot distinguish "set to info" from "unset" — and
`design/config.md`'s merge contract runs on exactly that distinction, where a non-zero source field wins
and a zero one is left alone. A layer setting the level to info would be silently ignored.

`Level` is therefore a string whose `""` means unset. The alternative considered was `*slog.Level`, where
nil means unset; it was rejected for putting a pointer field in an otherwise value-typed configuration,
and for moving the rejection of a bad value from `Finalize` to JSON parsing.

`Finalize` normalizes before applying defaults — so a blank or whitespace-only value takes the default
rather than failing validation — and again after reading the environment, so a value arriving from a file
or a shell in any casing lands on the canonical form the constants use.

## The writer is a parameter, not configuration

A configuration carries values and is discarded (`design/config.md`), so the destination is an argument to
`New` rather than a field. The composition root passes `os.Stdout`; a test passes a buffer. This is also
what makes the package testable on its output rather than on the handler's concrete type.

`New` returns no error. `Finalize` is the validation point, and a caller that built a `Config` literal and
skipped it falls back to info — a misconfiguration costs the process a level, not its logs.

## The package constructs a logger; it does not hold one

`logging` is transport-agnostic. It does not define any transport's request logger: a middleware belongs
to the transport that consumes it, so a command-line program or a worker that wants a `*slog.Logger` does
not compile `net/http` to get one. Subsystems take a `*slog.Logger` as an ordinary dependency; only the
composition root imports `logging`.
