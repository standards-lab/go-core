package processtest_test

import (
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/standards-lab/go-core/process/processtest"
)

// The suite's TestMain builds the test program once for the run, the way a
// service suite builds its cmd/server.
func TestMain(m *testing.M) {
	processtest.Main(m, "./process/processtest/internal/testprog")
}

// dial reports whether addr accepts a connection.
func dial(addr string) bool {
	c, err := net.Dial("tcp", addr)
	if err != nil {
		return false
	}
	_ = c.Close()
	return true
}

// Launch runs the built program on a reserved port; Await observes it
// through the network, not the output; Stop interrupts it and returns the
// exit code its drain chose, with everything it wrote captured.
func TestProcess_LaunchAwaitStop(t *testing.T) {
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(processtest.FreePort(t)))
	p := processtest.Launch(t, "TESTPROG_ADDR="+addr, "TESTPROG_EXIT=3")
	if p.Exited() {
		t.Fatal("exited before it was awaited")
	}

	p.Await(t, "listening", func() bool { return dial(addr) })

	if code := p.Stop(t); code != 3 {
		t.Errorf("Stop = %d, want 3", code)
	}
	if !p.Exited() {
		t.Error("Exited = false after Stop")
	}
	out := p.Output()
	for _, line := range []string{"testprog started", "testprog listening", "testprog stopped"} {
		if !strings.Contains(out, line) {
			t.Errorf("output lacks %q:\n%s", line, out)
		}
	}
	if code := p.Stop(t); code != 3 {
		t.Errorf("second Stop = %d, want the recorded 3", code)
	}
}

// A process that exits on its own reports its code through Wait, and its
// stderr is captured alongside stdout.
func TestProcess_WaitReportsOwnExit(t *testing.T) {
	p := processtest.Launch(t, "TESTPROG_EXIT_AT_ONCE=1", "TESTPROG_EXIT=2")

	if code := p.Wait(t); code != 2 {
		t.Errorf("Wait = %d, want 2", code)
	}
	if !strings.Contains(p.Output(), "testprog exiting at once") {
		t.Errorf("stderr not captured:\n%s", p.Output())
	}
}

// Two launches are two processes, each with its own output.
func TestProcess_LaunchSeveral(t *testing.T) {
	a := processtest.Launch(t, "TESTPROG_EXIT=1")
	b := processtest.Launch(t, "TESTPROG_EXIT=2")
	a.Await(t, "start", func() bool { return strings.Contains(a.Output(), "started") })
	b.Await(t, "start", func() bool { return strings.Contains(b.Output(), "started") })

	if code := a.Stop(t); code != 1 {
		t.Errorf("a = %d, want 1", code)
	}
	if code := b.Stop(t); code != 2 {
		t.Errorf("b = %d, want 2", code)
	}
}

// FreePort hands out a port that is free to bind.
func TestFreePort_IsBindable(t *testing.T) {
	port := processtest.FreePort(t)
	l, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		t.Fatalf("bind reserved port %d: %v", port, err)
	}
	_ = l.Close()
}

// WaitFor returns as soon as the condition holds.
func TestWaitFor_ReturnsOnCondition(t *testing.T) {
	n := 0
	processtest.WaitFor(t, "three polls", func() bool { n++; return n == 3 })
	if n != 3 {
		t.Errorf("polled %d times, want 3", n)
	}
}
