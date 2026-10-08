package processtest_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/standards-lab/go-core/process/processtest"
)

// Run passes its arguments to the program, which echoes them one per line.
func TestRun_PassesArgs(t *testing.T) {
	res := processtest.Run(t, processtest.Cmd{Args: []string{"echo", "one", "two words"}})

	if want := "one\ntwo words\n"; !strings.Contains(res.Stdout, want) {
		t.Errorf("stdout lacks %q:\n%s", want, res.Stdout)
	}
}

// Run feeds Stdin to the program's standard input.
func TestRun_FeedsStdin(t *testing.T) {
	res := processtest.Run(t, processtest.Cmd{
		Args:  []string{"echo"},
		Stdin: strings.NewReader("from stdin\n"),
	})

	if !strings.HasSuffix(res.Stdout, "from stdin\n") {
		t.Errorf("stdout lacks the stdin copy:\n%s", res.Stdout)
	}
}

// Run adds Env to the environment Launch gives the program: the parent's,
// with its GORACE options kept.
func TestRun_AddsEnv(t *testing.T) {
	t.Setenv("GORACE", "halt_on_error=1")
	res := processtest.Run(t, processtest.Cmd{Args: []string{"echo"}, Env: []string{"TESTPROG_EXIT=4"}})

	if res.Code != 4 {
		t.Errorf("Code = %d, want the 4 Env named", res.Code)
	}
	if want := "testprog GORACE=halt_on_error=1 atexit_sleep_ms=0"; !strings.Contains(res.Stdout, want) {
		t.Errorf("stdout lacks %q:\n%s", want, res.Stdout)
	}
}

// Run captures stdout and stderr apart.
func TestRun_SeparatesStdoutAndStderr(t *testing.T) {
	res := processtest.Run(t, processtest.Cmd{Args: []string{"echo", "out"}})

	if res.Stderr != "testprog to stderr\n" {
		t.Errorf("Stderr = %q, want only the stderr line", res.Stderr)
	}
	if strings.Contains(res.Stdout, "testprog to stderr") {
		t.Errorf("stderr leaked into stdout:\n%s", res.Stdout)
	}
	if !strings.Contains(res.Stdout, "out\n") {
		t.Errorf("stdout lacks the argument:\n%s", res.Stdout)
	}
}

// A nonzero exit is returned as the result's code; the test goes on.
func TestRun_ReturnsNonzeroCode(t *testing.T) {
	res := processtest.Run(t, processtest.Cmd{Args: []string{"echo"}, Env: []string{"TESTPROG_EXIT=2"}})

	if res.Code != 2 {
		t.Errorf("Code = %d, want 2", res.Code)
	}
	if t.Failed() {
		t.Error("a nonzero exit failed the test")
	}
}

// The zero Cmd runs the program with no arguments, empty stdin, and the
// parent's environment, which here tells it to exit at once.
func TestRun_ZeroCmd(t *testing.T) {
	t.Setenv("TESTPROG_EXIT_AT_ONCE", "1")
	res := processtest.Run(t, processtest.Cmd{})

	if res.Code != 0 {
		t.Errorf("Code = %d, want 0", res.Code)
	}
	if !strings.HasPrefix(res.Stdout, "testprog started\n") {
		t.Errorf("stdout lacks the start line:\n%s", res.Stdout)
	}
	if res.Stderr != "testprog exiting at once\n" {
		t.Errorf("Stderr = %q, want the exit-at-once line", res.Stderr)
	}
}

// logs records what a test logs, so the transcript Run writes can be read.
type logs struct {
	testing.TB
	lines []string
}

func (l *logs) Log(args ...any) { l.lines = append(l.lines, fmt.Sprint(args...)) }

// Run logs a shell line a reader can paste, its words quoted, the stdin the
// program read piped in, then the output and a nonzero exit code.
func TestRun_LogsTheShellLine(t *testing.T) {
	rec := &logs{TB: t}
	processtest.Run(rec, processtest.Cmd{
		Args:  []string{"echo", "it's", "plain"},
		Stdin: strings.NewReader("a b\n"),
		Env:   []string{"TESTPROG_EXIT=3", "TESTPROG_NOTE=x y"},
	})

	if len(rec.lines) != 1 {
		t.Fatalf("logged %d entries, want 1: %q", len(rec.lines), rec.lines)
	}
	got := rec.lines[0]
	want := `$ printf '%s' 'a b
' | TESTPROG_EXIT=3 TESTPROG_NOTE='x y' testprog echo 'it'\''s' plain`
	if !strings.HasPrefix(got, want+"\n") {
		t.Errorf("transcript does not open with the shell line\n got: %s\nwant: %s", got, want)
	}
	for _, part := range []string{"\nit's\nplain\na b\n", "\nstderr: testprog to stderr", "\nexit 3"} {
		if !strings.Contains(got, part) {
			t.Errorf("transcript lacks %q:\n%s", part, got)
		}
	}
}
