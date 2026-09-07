// Command testprog is the program the processtest package's own tests run:
// it answers on a loopback address, waits for the interrupt, and exits with
// the code its environment names, so the runner is proved against a real
// subprocess without a service behind it.
//
// TESTPROG_ADDR is the address to listen on; empty listens on none.
// TESTPROG_EXIT is the exit code, 0 by default. TESTPROG_EXIT_AT_ONCE=1
// exits with that code before waiting for the signal.
package main

import (
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/standards-lab/go-core/process"
)

func main() {
	os.Exit(run())
}

func run() int {
	code, _ := strconv.Atoi(os.Getenv("TESTPROG_EXIT"))
	fmt.Println("testprog started")
	if os.Getenv("TESTPROG_EXIT_AT_ONCE") == "1" {
		fmt.Fprintln(os.Stderr, "testprog exiting at once")
		return code
	}
	if addr := os.Getenv("TESTPROG_ADDR"); addr != "" {
		l, err := net.Listen("tcp", addr)
		if err != nil {
			return process.Fail(os.Stderr, "listen failed", err)
		}
		defer func() { _ = l.Close() }()
		fmt.Println("testprog listening")
	}
	ctx, stop := process.SignalContext()
	defer stop()
	<-ctx.Done()
	fmt.Println("testprog stopped")
	return code
}
