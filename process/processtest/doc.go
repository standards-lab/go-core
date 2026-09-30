// Package processtest is the integration toolkit for a program built on the
// process package: it runs the program as the binary and drives it only
// through the interfaces a terminal or an orchestrator uses: its captured
// output, the signals process.SignalContext answers, and its exit code.
// Nothing in the runtime exists for the tests' sake.
package processtest
