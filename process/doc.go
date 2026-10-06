// Package process holds the parts of a binary's main sequence that run
// before the program's own infrastructure exists: reporting a failure or a
// usage error when there is no logger yet, the signal-derived root context,
// and the exit-code convention the reporters return. An application's
// entrypoint, the minimal binary above its composition root, composes its run
// function from it, so the convention cannot drift between a program's
// binaries.
//
// The package exports:
//
//   - [ExitOK], [ExitFailure], and [ExitUsage], the exit codes
//   - [Fail], which reports a runtime failure and returns ExitFailure
//   - [Usage], which reports a usage error or help and returns ExitUsage
//   - [SignalContext], which returns the root context SIGINT or SIGTERM
//     cancels
package process
