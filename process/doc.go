// Package process holds the parts of a binary's main sequence that run
// before the program's own infrastructure exists: reporting a failure or a
// usage error when there is no logger yet, the signal-derived root context,
// and the exit-code convention the reporters return. A composition root
// composes its run function from it, so the convention cannot drift between
// a program's binaries.
package process
