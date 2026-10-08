// Package processtest is the integration toolkit for a program built on the
// process package. It runs the program as the binary and drives it only
// through the interfaces a terminal or an orchestrator uses: its captured
// output, the signals process.SignalContext answers, and its exit code.
// Nothing in the runtime exists for the tests' sake.
//
// The package exports:
//
//   - [Main], a suite's TestMain, which builds the program once per run
//   - [Launch], which starts the built program as a [Process]
//   - [Run], which runs the built program once as a [Cmd] says, with its
//     arguments, standard input, and environment, and returns its [Result],
//     the stdout, stderr, and exit code
//   - [Process], one running program, which a test waits on with
//     [Process.Await] and [Process.Wait], reads with [Process.Output] and
//     [Process.Exited], and interrupts with [Process.Stop]
//   - [Forward], which starts a [Forwarder], a loopback relay to a backing
//     service that [Forwarder.Sever] cuts and [Forwarder.Restore] reopens at
//     [Forwarder.Addr]
//   - [WaitFor], which polls a condition no single process owns
//   - [FreePort], which reserves a loopback port for the process to bind
//   - [Failsafe], the bound on every wait
package processtest
