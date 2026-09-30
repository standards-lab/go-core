// Package logging constructs the *slog.Logger a process writes through, from a
// configuration that takes part in the layered load.
//
// # Configuration
//
// [Config] holds a [Level] and a [Format] and implements the config package's
// Merge and Finalize contract, so it loads as part of an application's
// configuration rather than on its own. Finalize runs, in order:
//
//   - composes its override names from the prefix it receives with [NewEnv],
//     recording them on [Env]
//   - normalizes (trims and lower-cases)
//   - applies defaults (info, text)
//   - reads the overrides
//   - normalizes again
//   - validates
//
// A value from a file or a shell therefore validates identically in any
// casing, and a blank or whitespace-only value takes the default. An empty
// prefix composes no names, disabling the overrides.
//
// # Construction
//
// [New] returns a logger writing to the caller's io.Writer, with a JSON
// handler for [FormatJSON] and a text handler otherwise. Finalize is the
// validation point: New returns no error, and a Config that skipped Finalize
// yields an info-level logger.
package logging
