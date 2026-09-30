package processtest_test

import (
	"net"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/standards-lab/go-core/process/processtest"
)

// stalled returns a loopback address whose connect never completes: a
// listener with a zero backlog that never accepts drops every SYN once its
// one queue slot is taken.
func stalled(t *testing.T) string {
	t.Helper()
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = syscall.Close(fd) })
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
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(sa.(*syscall.SockaddrInet4).Port))
	filler, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = filler.Close() })
	return addr
}

// A target that never answers a connect holds only its own connection:
// the forwarder keeps accepting, and Sever returns at once.
func TestForwarder_StalledTargetBlocksNothing(t *testing.T) {
	f := processtest.Forward(t, stalled(t))

	c, err := net.Dial("tcp", f.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	time.Sleep(50 * time.Millisecond) // let the relay reach its dial

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
