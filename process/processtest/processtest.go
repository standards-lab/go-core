package processtest

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// Failsafe bounds every wait for an event that should occur, so a broken
// composition fails a test instead of hanging the run.
const Failsafe = 15 * time.Second

// The build Main produced for the run: the binary's path and the module
// root, the working directory the program runs in.
var (
	binary string
	root   string
)

// Main is the suite's TestMain: it builds the main package pkg once, with
// the race detector so the program runs under it too, runs the tests, and
// removes the build. pkg is a package path relative to the module root,
// such as ./cmd/server. A build failure ends the run before any test starts.
func Main(m *testing.M, pkg string) {
	code, err := run(m, pkg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "processtest:", err)
		code = 1
	}
	os.Exit(code)
}

func run(m *testing.M, pkg string) (int, error) {
	var err error
	if root, err = moduleRoot(); err != nil {
		return 0, err
	}
	name := path.Base(pkg)
	dir, err := os.MkdirTemp("", name+"-processtest-")
	if err != nil {
		return 0, err
	}
	defer func() { _ = os.RemoveAll(dir) }()

	binary = filepath.Join(dir, name)
	build := exec.Command("go", "build", "-race", "-o", binary, pkg)
	build.Dir = root
	build.Stdout, build.Stderr = os.Stderr, os.Stderr
	if err := build.Run(); err != nil {
		return 0, fmt.Errorf("build %s: %w", pkg, err)
	}
	return m.Run(), nil
}

// moduleRoot resolves the directory of the module the suite's package
// belongs to, the working directory the program loads its files from. It
// asks for the current package's module rather than listing modules,
// since under a multi-module workspace the list is every module.
func moduleRoot() (string, error) {
	out, err := exec.Command("go", "list", "-f", "{{.Module.Dir}}", ".").Output()
	if err != nil {
		return "", fmt.Errorf("locate module root: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Process is one running program: its captured output and its exit.
type Process struct {
	cmd    *exec.Cmd
	out    *output
	exited chan struct{}
	code   int
}

// Launch starts the binary Main built in the module root, with the parent's
// environment plus env's KEY=VALUE overrides, and returns without waiting.
// At test cleanup, a process still running is interrupted, then killed
// after Failsafe.
func Launch(t testing.TB, env ...string) *Process {
	t.Helper()
	if binary == "" {
		t.Fatal("processtest: no binary; the suite's TestMain must call Main")
	}
	cmd := exec.Command(binary)
	cmd.Dir = root
	// Races report as they happen, so the race runtime's one-second sleep
	// at exit only slows every Stop; the parent's GORACE options stay.
	gorace := strings.TrimSpace(os.Getenv("GORACE") + " atexit_sleep_ms=0")
	cmd.Env = append(os.Environ(), "GORACE="+gorace)
	cmd.Env = append(cmd.Env, env...)
	out := &output{}
	cmd.Stdout, cmd.Stderr = out, out
	if err := cmd.Start(); err != nil {
		t.Fatalf("start %s: %v", filepath.Base(binary), err)
	}

	p := &Process{cmd: cmd, out: out, exited: make(chan struct{})}
	go func() {
		defer close(p.exited)
		_ = cmd.Wait()
		p.code = cmd.ProcessState.ExitCode()
	}()
	t.Cleanup(func() {
		if p.Exited() {
			return
		}
		// Interrupt first, so the process drains its connections the way
		// it would in service; kill only a process that does not.
		_ = cmd.Process.Signal(syscall.SIGINT)
		select {
		case <-p.exited:
		case <-time.After(Failsafe):
			_ = cmd.Process.Kill()
			<-p.exited
		}
	})
	return p
}

// Await polls cond until it holds, failing the test with the captured
// output if the process exits or Failsafe elapses first. A harness waits
// for readiness this way: on the condition a client observes, never on the
// output.
func (p *Process) Await(t testing.TB, what string, cond func() bool) {
	t.Helper()
	var exited bool
	held := poll(func() bool {
		exited = p.Exited()
		return exited || cond()
	})
	switch {
	case exited:
		t.Fatalf("process exited with %d before %s:\n%s", p.code, what, p.Output())
	case !held:
		t.Fatalf("%s not observed within %s:\n%s", what, Failsafe, p.Output())
	}
}

// Output is everything the process has written so far, stdout and stderr
// interleaved as they arrived.
func (p *Process) Output() string { return p.out.String() }

// Stop interrupts the process, the signal a terminal or an orchestrator
// sends, waits for it to exit, and returns its exit code. A process still
// running after Failsafe is killed, and the test fails.
func (p *Process) Stop(t testing.TB) int {
	t.Helper()
	if p.Exited() {
		return p.code
	}
	// A process that exited on its own may be reaped before Exited reports it.
	err := p.cmd.Process.Signal(syscall.SIGINT)
	if err != nil && !errors.Is(err, os.ErrProcessDone) {
		t.Fatalf("interrupt process: %v", err)
	}
	return p.Wait(t)
}

// Wait blocks until the process exits and returns its exit code, failing
// the test if Failsafe elapses first.
func (p *Process) Wait(t testing.TB) int {
	t.Helper()
	select {
	case <-p.exited:
		return p.code
	case <-time.After(Failsafe):
		_ = p.cmd.Process.Kill()
		<-p.exited
		t.Fatalf("process did not exit within %s:\n%s", Failsafe, p.Output())
		return -1
	}
}

// Exited reports whether the process has ended.
func (p *Process) Exited() bool {
	select {
	case <-p.exited:
		return true
	default:
		return false
	}
}

// output is the process's captured writes, read whole on a failure.
type output struct {
	mu  sync.Mutex
	buf strings.Builder
}

func (o *output) Write(b []byte) (int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.buf.Write(b)
}

func (o *output) String() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.buf.String()
}

// FreePort reserves an ephemeral loopback port and releases it for the
// process to bind. The window between release and bind is the usual one of
// a port-based harness; a lost race fails the bind, and Await reports the
// exit with the output.
func FreePort(t testing.TB) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	if err := l.Close(); err != nil {
		t.Fatalf("release port: %v", err)
	}
	return port
}

// WaitFor polls fn until it returns true, failing the test with what if
// Failsafe elapses first. It is Await for a condition no single process
// owns.
func WaitFor(t testing.TB, what string, fn func() bool) {
	t.Helper()
	if !poll(fn) {
		t.Fatalf("timed out waiting for %s", what)
	}
}

// poll calls cond until it holds or Failsafe elapses, reporting which.
func poll(cond func() bool) bool {
	deadline := time.Now().Add(Failsafe)
	for {
		if cond() {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(20 * time.Millisecond)
	}
}
