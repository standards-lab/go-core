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

var _ interface {
	lifecycle.Subsystem
	lifecycle.ReadyStarter
	lifecycle.Monitored
} = (*reactor.Reactor[time.Time])(nil)

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

type store struct{ events *events }

func (s store) Start(context.Context) error {
	s.events.record("store started")
	return nil
}

func (s store) Shutdown(context.Context) error {
	s.events.record("store stopped")
	return nil
}

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

func TestReactor_CoordinatorStartsReportsReadyAndDrains(t *testing.T) {
	ev := &events{}
	var first sync.Once
	inFlight := make(chan struct{})
	r := reactor.New(reactor.Every(time.Millisecond), func(ctx context.Context, _ time.Time) error {
		first.Do(func() { close(inFlight) })
		<-reactor.Draining(ctx)
		time.Sleep(10 * time.Millisecond)
		ev.record("handler drained")
		return nil
	}, reactor.Options{CancelAfter: reactor.CancelWithin(failsafe)})
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

func TestReactor_CoordinatorEndsRunOnFailure(t *testing.T) {
	boom := errors.New("boom")
	r := reactor.New(reactor.Every(time.Millisecond), func(context.Context, time.Time) error {
		return boom
	}, reactor.Options{CancelAfter: 10 * time.Millisecond})
	lc := coordinate(t, r, &events{})

	done := make(chan error, 1)
	go func() { done <- lc.Run(context.Background()) }()
	err := recvOrFail(t, done, "Run to end on the reactor's failure")
	if !errors.Is(err, boom) || !strings.HasPrefix(err.Error(), "run: ") {
		t.Fatalf("Run = %v, want the handler's failure labelled run:", err)
	}
}
