package processtest_test

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/standards-lab/go-core/process/processtest"
)

// stalling is a loopback target whose connects stall until it frees its
// one queue slot: a listener with a zero backlog that never accepts on its
// own drops every SYN once the filler connection holds that slot.
type stalling struct {
	fd   int
	port int
}

func stall(t *testing.T) *stalling {
	t.Helper()
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM|syscall.SOCK_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		// Shutdown wakes an Accept still blocked on the socket; Close alone
		// does not.
		_ = syscall.Shutdown(fd, syscall.SHUT_RDWR)
		_ = syscall.Close(fd)
	})
	if err := syscall.Bind(fd, &syscall.SockaddrInet4{Addr: [4]byte{127, 0, 0, 1}}); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Listen(fd, 0); err != nil {
		t.Fatal(err)
	}
	sa, err := syscall.Getsockname(fd)
	if err != nil {
		t.Fatal(err)
	}
	s := &stalling{fd: fd, port: sa.(*syscall.SockaddrInet4).Port}
	filler, err := net.Dial("tcp", s.addr())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = filler.Close() })
	return s
}

func (s *stalling) addr() string {
	return net.JoinHostPort("127.0.0.1", strconv.Itoa(s.port))
}

// accept takes the next connection off the queue, freeing its slot.
func (s *stalling) accept() (net.Conn, error) {
	nfd, _, err := syscall.Accept4(s.fd, syscall.SOCK_CLOEXEC)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(nfd), "stalling")
	defer func() { _ = f.Close() }()
	return net.FileConn(f)
}

// dialing reports whether a connect to the target is stalled: a socket in
// SYN_SENT whose remote port is the target's.
func (s *stalling) dialing() bool {
	data, err := os.ReadFile("/proc/net/tcp")
	if err != nil {
		return false
	}
	remote := fmt.Sprintf(":%04X", s.port)
	for _, line := range strings.Split(string(data), "\n")[1:] {
		fields := strings.Fields(line)
		if len(fields) > 3 && strings.HasSuffix(fields[2], remote) && fields[3] == "02" {
			return true
		}
	}
	return false
}

// A target that never answers a connect holds only its own connection:
// while the first relay's dial is stalled, the forwarder accepts a second
// connection and relays it, and Sever returns at once. A forwarder that
// dialed in its accept loop would leave the second connection unaccepted
// until the first dial's SYN retransmit, which the one accept below takes.
func TestForwarder_StalledTargetBlocksNothing(t *testing.T) {
	target := stall(t)
	f := processtest.Forward(t, target.addr())

	first, err := net.Dial("tcp", f.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = first.Close() }()
	processtest.WaitFor(t, "the first relay's stalled dial", target.dialing)

	// Free the slot: the first dial stays stalled until its SYN
	// retransmit, while a fresh connect completes at once.
	filler, err := target.accept()
	if err != nil {
		t.Fatal(err)
	}
	_ = filler.Close()

	second, err := net.Dial("tcp", f.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = second.Close() }()
	go func() {
		up, err := target.accept()
		if err != nil {
			return
		}
		defer func() { _ = up.Close() }()
		line, err := bufio.NewReader(up).ReadString('\n')
		if err == nil {
			_, _ = up.Write([]byte(line))
		}
	}()
	_ = second.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err := second.Write([]byte("hello\n")); err != nil {
		t.Fatal(err)
	}
	if got, err := bufio.NewReader(second).ReadString('\n'); err != nil || got != "hello\n" {
		t.Fatalf("second connection relayed %q, %v; want it relayed past the stalled dial", got, err)
	}

	severed := make(chan struct{})
	go func() {
		f.Sever()
		close(severed)
	}()
	select {
	case <-severed:
	case <-time.After(2 * time.Second):
		t.Fatal("Sever blocked on a connection whose target dial is in flight")
	}
}
