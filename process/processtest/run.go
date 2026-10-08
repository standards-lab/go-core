package processtest

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"testing"
	"time"
)

// Cmd is one invocation of the program for [Run]. Its zero value runs the
// program with no arguments, an empty standard input, and the parent's
// environment.
type Cmd struct {
	// Args are the program's arguments, without the program name.
	Args []string
	// Stdin is the program's standard input; nil is empty.
	Stdin io.Reader
	// Env holds KEY=VALUE overrides added to the parent's environment.
	Env []string
}

// Result is what one run of the program produced.
type Result struct {
	// Stdout and Stderr are the program's two output streams, each
	// captured whole and apart from the other.
	Stdout, Stderr string
	// Code is the program's exit code.
	Code int
}

// Run executes the binary Main built once, in the module root, with the
// environment Launch gives it plus cmd.Env, waits for it to exit, and
// returns its output and exit code. A nonzero code is a result, not a
// failure. A run still going after Failsafe is a stall: it is interrupted,
// killed if it has not exited within Failsafe more, and fails the test with
// its output. Run logs the run through t.Log as a shell line a reader can
// paste, with the output after it.
func Run(t testing.TB, cmd Cmd) Result {
	t.Helper()
	c := command(t, cmd.Args, cmd.Env)
	// The tee keeps what the program read, so the logged line replays it.
	var stdin bytes.Buffer
	if cmd.Stdin != nil {
		c.Stdin = io.TeeReader(cmd.Stdin, &stdin)
	}
	var stdout, stderr bytes.Buffer
	c.Stdout, c.Stderr = &stdout, &stderr
	// A child the program leaves holding its output open would keep Wait
	// copying forever; WaitDelay bounds that too.
	c.WaitDelay = Failsafe
	if err := c.Start(); err != nil {
		t.Fatalf("start %s: %v", filepath.Base(binary), err)
	}
	done := make(chan error, 1)
	go func() { done <- c.Wait() }()

	exited, err := within(done, Failsafe)
	if !exited {
		// Interrupt first, as Launch's cleanup does, so a program that
		// drains on the signal shows what it was doing.
		_ = c.Process.Signal(syscall.SIGINT)
		if exited, _ = within(done, Failsafe); !exited {
			_ = c.Process.Kill()
			<-done
		}
		t.Fatalf("%s did not exit within %s:\nstdout: %s\nstderr: %s",
			shellLine(cmd, stdin.String()), Failsafe, stdout.String(), stderr.String())
	}

	res := Result{Stdout: stdout.String(), Stderr: stderr.String()}
	var exit *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exit):
		res.Code = exit.ExitCode()
	default:
		t.Fatalf("%s: %v", shellLine(cmd, stdin.String()), err)
	}
	t.Log(transcript(shellLine(cmd, stdin.String()), res))
	return res
}

// within waits up to d for the run's Wait to return, reporting whether it
// returned and its error.
func within(done <-chan error, d time.Duration) (bool, error) {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case err := <-done:
		return true, err
	case <-timer.C:
		return false, nil
	}
}

// shellLine renders cmd as the command a reader would type: the
// environment overrides, the program's name, and its arguments, each
// quoted for a POSIX shell, with the standard input the program read piped
// in by printf. The program is named, not pathed, since Main removes its
// build when the run ends.
func shellLine(cmd Cmd, stdin string) string {
	var words []string
	for _, kv := range cmd.Env {
		// Only the value is quoted: a quoted name is no assignment.
		if name, value, ok := strings.Cut(kv, "="); ok {
			words = append(words, name+"="+quote(value))
		} else {
			words = append(words, quote(kv))
		}
	}
	words = append(words, quote(filepath.Base(binary)))
	for _, arg := range cmd.Args {
		words = append(words, quote(arg))
	}
	line := "$ " + strings.Join(words, " ")
	if stdin != "" {
		line = "$ printf '%s' " + quote(stdin) + " | " + strings.TrimPrefix(line, "$ ")
	}
	return line
}

// plain matches a word a POSIX shell reads as itself.
var plain = regexp.MustCompile(`^[A-Za-z0-9_@%+=:,./-]+$`)

// quote returns s as one POSIX shell word: as it is when the shell reads
// it literally, otherwise single-quoted, with each single quote closed,
// escaped, and reopened.
func quote(s string) string {
	if plain.MatchString(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// transcript is the run as a terminal would show it: the shell line, its
// stdout, its stderr when it wrote any, and its exit code when nonzero.
func transcript(line string, res Result) string {
	var b strings.Builder
	b.WriteString(line)
	if res.Stdout != "" {
		b.WriteString("\n" + strings.TrimRight(res.Stdout, "\n"))
	}
	if res.Stderr != "" {
		b.WriteString("\nstderr: " + strings.TrimRight(res.Stderr, "\n"))
	}
	if res.Code != 0 {
		fmt.Fprintf(&b, "\nexit %d", res.Code)
	}
	return b.String()
}
