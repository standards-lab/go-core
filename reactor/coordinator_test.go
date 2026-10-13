package reactor_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/standards-lab/go-core/config"
	"github.com/standards-lab/go-core/graph"
	"github.com/standards-lab/go-core/lifecycle"
	"github.com/standards-lab/go-core/reactor"
)

// A reactor joins a graph-backed coordinator as a node's value with no
// adapter: the coordinator infers from its methods that it starts, drains,
// reports readiness, and is monitored.
var _ interface {
	lifecycle.Subsystem
	lifecycle.ReadyStarter
	lifecycle.Monitored
} = (*reactor.Reactor[time.Time])(nil)

// events collects what happens, in order, across goroutines.
type events struct {
	mu   sync.Mutex
	list []string
}

func (e *events) record(event string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.list = append(e.list, event)
}

func (e *events) all() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Clone(e.list)
}

// store is a Subsystem a reactor uses, which records its start and stop.
type store struct{ events *events }

func (s store) Start(context.Context) error {
	s.events.record("store started")
	return nil
}

func (s store) Shutdown(context.Context) error {
	s.events.record("store stopped")
	return nil
}

// coordinate builds a graph whose "ticker" node's value is r, over a "store"
// node it uses, and returns a coordinator over it that shuts down within
// failsafe.
func coordinate(t *testing.T, r *reactor.Reactor[time.Time], ev *events) *lifecycle.Coordinator {
	t.Helper()
	g := graph.New()
	st := g.Define("store", func(*graph.Scope) (store, error) {
		return store{ev}, nil
	})
	ticker := g.Define("ticker", func(s *graph.Scope) (*reactor.Reactor[time.Time], error) {
		s.Use(st)
		return r, nil
	})
	sys, err := g.Build(ticker)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return lifecycle.New(sys, lifecycle.Config{ShutdownTimeout: config.Duration(failsafe)})
}

// An Every reactor that is a graph node's value, registered nowhere, starts
// with its layer, reports ready under its node's name, and on a signal
// drains its handler in flight before the node it uses stops, to a nil Run
// result.
func TestReactor_CoordinatorStartsReportsReadyAndDrains(t *testing.T) {
	ev := &events{}
	var first sync.Once
	inFlight := make(chan struct{})
	r := reactor.New(reactor.Every(time.Millisecond), func(ctx context.Context, _ time.Time) error {
		first.Do(func() { close(inFlight) })
		<-reactor.Draining(ctx)
		time.Sleep(10 * time.Millisecond) // finish the step in hand
		ev.record("handler drained")
		return nil
	}, reactor.GraceWithin(failsafe))
	lc := coordinate(t, r, ev)

	checks := lc.Checks()
	if len(checks) != 1 || checks[0].Name != "ticker" {
		t.Fatalf("Checks = %+v, want one named ticker", checks)
	}
	ready := make(chan struct{})
	lc.OnReady(func() { close(ready) })
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- lc.Run(ctx) }()
	recvOrFail(t, ready, "the coordinator's readiness")
	eventually(t, checks[0].Checker.Ready, "the ticker's readiness check")
	recvOrFail(t, inFlight, "a tick to be handled")

	cancel()
	if err := recvOrFail(t, done, "Run to drain"); err != nil {
		t.Fatalf("Run = %v, want a clean drain", err)
	}
	want := []string{"store started", "handler drained", "store stopped"}
	if got := ev.all(); !slices.Equal(got, want) {
		t.Errorf("events = %q, want %q", got, want)
	}
	if checks[0].Checker.Ready() {
		t.Error("the ticker reports ready after the drain")
	}
}

// As a graph node's value, registered nowhere, a handler's failure while
// running ends the run with the failure in Run's result, labelled run:.
func TestReactor_CoordinatorEndsRunOnFailure(t *testing.T) {
	boom := errors.New("boom")
	r := reactor.New(reactor.Every(time.Millisecond), func(context.Context, time.Time) error {
		return boom
	}, reactor.Grace(10*time.Millisecond))
	lc := coordinate(t, r, &events{})

	done := make(chan error, 1)
	go func() { done <- lc.Run(context.Background()) }()
	err := recvOrFail(t, done, "Run to end on the reactor's failure")
	if !errors.Is(err, boom) || !strings.HasPrefix(err.Error(), "run: ") {
		t.Fatalf("Run = %v, want the handler's failure labelled run:", err)
	}
}
