package processtest

import (
	"io"
	"net"
	"sync"
	"testing"
)

// Forwarder is a loopback TCP relay between the process and a backing
// service, the connection a test severs to inject an outage.
type Forwarder struct {
	target string

	// restoring serializes Restore's check and listen, so two concurrent
	// Restores cannot both find the forwarder severed.
	restoring sync.Mutex

	mu        sync.Mutex
	listener  net.Listener
	listening bool
	conns     map[net.Conn]struct{}
	wg        sync.WaitGroup
}

// Forward starts relaying to target, host:port, on an ephemeral loopback
// port; the relay is severed at test cleanup.
func Forward(t testing.TB, target string) *Forwarder {
	t.Helper()
	f := &Forwarder{target: target, conns: map[net.Conn]struct{}{}}
	if err := f.listen(""); err != nil {
		t.Fatalf("forwarder: %v", err)
	}
	t.Cleanup(f.Sever)
	return f
}

// Addr is the address the process connects to, host:port. It is stable
// across Sever and Restore.
func (f *Forwarder) Addr() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.listener.Addr().String()
}

func (f *Forwarder) listen(addr string) error {
	if addr == "" {
		addr = "127.0.0.1:0"
	}
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	f.mu.Lock()
	f.listener = l
	f.listening = true
	f.mu.Unlock()
	f.wg.Add(1)
	go f.accept(l)
	return nil
}

// accept tracks each connection and hands it to a relay, which dials the
// target, so an unresponsive target never stalls the loop or Sever.
func (f *Forwarder) accept(l net.Listener) {
	defer f.wg.Done()
	for {
		c, err := l.Accept()
		if err != nil {
			return
		}
		f.track(c)
		go f.relay(c)
	}
}

func (f *Forwarder) track(conns ...net.Conn) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range conns {
		f.conns[c] = struct{}{}
	}
}

func (f *Forwarder) untrack(conns ...net.Conn) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range conns {
		delete(f.conns, c)
		_ = c.Close()
	}
}

// relay dials the target and copies both directions until either side
// closes. A connection Sever drops while the dial is in flight fails the
// copy at once, which closes the upstream too.
func (f *Forwarder) relay(down net.Conn) {
	up, err := net.DialTimeout("tcp", f.target, Failsafe)
	if err != nil {
		f.untrack(down)
		return
	}
	f.track(up)
	done := make(chan struct{}, 2)
	pipe := func(dst, src net.Conn) {
		_, _ = io.Copy(dst, src)
		done <- struct{}{}
	}
	go pipe(up, down)
	go pipe(down, up)
	<-done
	f.untrack(down, up)
	<-done
}

// Sever refuses new connections and drops the relayed ones; a relay still
// dialing the target finishes within Failsafe.
func (f *Forwarder) Sever() {
	f.mu.Lock()
	l := f.listener
	f.listening = false
	f.mu.Unlock()
	_ = l.Close()
	// Wait for the accept loop, so a connection accepted as the listener
	// closed is tracked and dropped with the rest.
	f.wg.Wait()

	f.mu.Lock()
	conns := make([]net.Conn, 0, len(f.conns))
	for c := range f.conns {
		conns = append(conns, c)
	}
	f.mu.Unlock()
	for _, c := range conns {
		_ = c.Close()
	}
}

// Restore listens again on the forwarder's address; while listening, it is
// a no-op.
func (f *Forwarder) Restore(t testing.TB) {
	t.Helper()
	f.restoring.Lock()
	defer f.restoring.Unlock()
	f.mu.Lock()
	listening := f.listening
	f.mu.Unlock()
	if listening {
		return
	}
	addr := f.Addr()
	if err := f.listen(addr); err != nil {
		t.Fatalf("forwarder restore on %s: %v", addr, err)
	}
}
