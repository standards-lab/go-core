// Package processtest is the integration toolkit for a program built on the
// process package: it runs the program as the binary and drives it only
// through the seams a terminal or an orchestrator uses, so nothing in the
// runtime exists for the tests' sake.
//
// [Main] is a suite's TestMain: it builds one main package once per run,
// with the race detector. [Launch] runs that binary as a subprocess with the
// environment the test composes, capturing its output; [Process.Await] waits
// on an observable condition, failing with the captured output if the process
// exits or [Failsafe] elapses first; [Process.Stop] interrupts the process,
// the signal the process package's SignalContext answers, and returns its
// exit code. A [Forwarder] is a loopback relay between the process and one of
// its backing services, the seam a test injects an outage through.
//
// The toolkit is hermetic: it needs no service beyond the binary it builds,
// so a consumer's harness type-checks and proves itself on the unit tier,
// and only the suite that needs a running stack carries a build tag.
package processtest
