// Package logging constructs the *slog.Logger a process writes through, from a
// configuration that takes part in the layered load.
//
// # Configuration
//
// [Config] holds a [Level] and a [Format] and implements the config package's
// Merge and Finalize contract, so it loads as part of an application's
// configuration rather than on its own. Finalize composes its environment
// override names from the prefix it receives (via [NewEnv], recorded on [Env]
// for introspection), normalizes (trims and lower-cases), applies defaults
// (info, text), reads the overrides, normalizes again, and validates. A value
// from a file or a shell therefore validates identically in any casing, and a
// blank or whitespace-only value takes the default. An empty prefix composes
// no names, disabling the overrides.
//
// [Level] is a string. slog.Level parses it — "debug", "INFO", and offsets such
// as "warn+2", case-insensitively — through [Level.Slog]. An empty Level is
// unset, distinct from "set to info"; Finalize has replaced it with a default
// by the time validation runs.
//
// # Construction
//
// [New] returns a logger writing to the caller's io.Writer, with
// slog.NewJSONHandler selected by [FormatJSON] and slog.NewTextHandler
// otherwise. The caller supplies the writer; a test passes a buffer.
//
// New returns no error; Finalize is the validation point. A Config literal that
// skipped Finalize yields an info-level logger.
//
// # Scope
//
// Subsystems take a *slog.Logger as an ordinary dependency. The HTTP request
// logger is in the web package.
package logging
