package processtest

import (
	"os"
	"os/exec"
	"testing"
)

// signalled is the test's TB, closing exited at Wait's Helper call: after
// Stop has checked Exited and sent its signal, so the signal meets a
// process already reaped.
type signalled struct {
	*testing.T
	exited  chan struct{}
	helpers int
}

func (s *signalled) Helper() {
	s.T.Helper()
	if s.helpers++; s.helpers == 2 {
		close(s.exited)
	}
}

// Stop on a process that exited on its own and was reaped before Exited
// reported it returns the exit code: the interrupt finds the process done,
// which is not a failure.
func TestStop_ProcessReapedBeforeExitedReports(t *testing.T) {
	cmd := exec.Command(binary)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "TESTPROG_EXIT_AT_ONCE=1", "TESTPROG_EXIT=2")
	_ = cmd.Run() // started and reaped; its exit status is the code below

	p := &Process{cmd: cmd, out: &output{}, exited: make(chan struct{}), code: 2}
	tb := &signalled{T: t, exited: p.exited}
	if code := p.Stop(tb); code != 2 {
		t.Errorf("Stop = %d, want 2", code)
	}
	if tb.helpers < 2 {
		t.Fatal("Stop returned before signalling; the reaped path went untested")
	}
}
