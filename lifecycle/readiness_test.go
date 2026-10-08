package lifecycle_test

import (
	"context"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/standards-lab/go-core/config"
	"github.com/standards-lab/go-core/graph"
	"github.com/standards-lab/go-core/lifecycle"
)

var _ lifecycle.ReadinessChecker = (*lifecycle.Readiness)(nil)

// checker is a value that implements ReadinessChecker and takes no part in
// startup or shutdown.
type checker func() bool

func (c checker) Ready() bool { return c() }

// readiness defines a *Readiness node on g under name, as an application
// does for the nodes that report readiness.
func readiness(g *graph.Graph, name string) *graph.Node[*lifecycle.Readiness] {
	return g.Define(name, func(*graph.Scope) (*lifecycle.Readiness, error) {
		return new(lifecycle.Readiness), nil
	})
}

// checkNames returns the names of checks, in order.
func checkNames(checks []lifecycle.Check) []string {
	var names []string
	for _, c := range checks {
		names = append(names, c.Name)
	}
	return names
}

// Exec carries Run's runtime contract: the flip and the OnReady hooks, in
// registration order, precede fn, and readiness ends once shutdown begins.
func TestExec_ReadyDuringFn(t *testing.T) {
	var r recorder
	var lc *lifecycle.Coordinator
	g := graph.New()
	svc := value(g, "svc", stopper(func(context.Context) error {
		r.record("stop ready=" + strconv.FormatBool(lc.Ready()))
		return nil
	}))
	lc = coordinator(t, failsafe, g, svc)
	lc.OnReady(func() { r.record("hook 1 ready=" + strconv.FormatBool(lc.Ready())) })
	lc.OnReady(func() { r.record("hook 2") })

	if lc.Ready() {
		t.Fatal("Ready() is true before Exec")
	}
	err := lc.Exec(context.Background(), func(context.Context) error {
		r.record("fn ready=" + strconv.FormatBool(lc.Ready()))
		return nil
	})
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	want := []string{"hook 1 ready=true", "hook 2", "fn ready=true", "stop ready=false"}
	if got := r.list(); !slices.Equal(got, want) {
		t.Errorf("events = %q, want %q", got, want)
	}
	if lc.Ready() {
		t.Error("Ready() is true after Exec returned")
	}
}

// Readiness is false once shutdown begins; TestExec_ReadyDuringFn covers
// Exec.
func TestRun_NotReadyOnceDrainingBegins(t *testing.T) {
	ready := make(chan bool, 1)
	var lc *lifecycle.Coordinator
	g := graph.New()
	svc := value(g, "svc", stopper(func(context.Context) error {
		ready <- lc.Ready()
		return nil
	}))
	lc = coordinator(t, failsafe, g, svc)
	if err := runThrough(lc); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if recvOrFail(t, ready, "the participant's Shutdown") {
		t.Error("Ready() is true during shutdown")
	}
}

// The OnReady hooks run in registration order, each after the flip.
func TestRun_OnReadyRunsInRegistrationOrderAfterTheFlip(t *testing.T) {
	var r recorder
	g := graph.New()
	lc := coordinator(t, failsafe, g, value(g, "svc", stopper(noop)))
	for _, name := range []string{"a", "b", "c"} {
		lc.OnReady(func() { r.record(name + " ready=" + strconv.FormatBool(lc.Ready())) })
	}
	if err := runThrough(lc); err != nil {
		t.Fatalf("Run: %v", err)
	}
	want := []string{"a ready=true", "b ready=true", "c ready=true"}
	if got := r.list(); !slices.Equal(got, want) {
		t.Errorf("events = %q, want %q", got, want)
	}
}

func TestCoordinator_ChecksListsCheckersInLayerThenDefinitionOrder(t *testing.T) {
	g := graph.New()
	base := node(g, &fake{name: "base"})
	db := g.Define("db", func(s *graph.Scope) (checker, error) {
		s.Use(base)
		return func() bool { return true }, nil
	})
	cache := value(g, "cache", checker(func() bool { return false }))
	probe := readiness(g, "readiness")
	plain := value(g, "plain", 2)

	lc := coordinator(t, failsafe, g, db, cache, probe, plain)
	checks := lc.Checks()
	if got, want := checkNames(checks), []string{"cache", "db"}; !slices.Equal(got, want) {
		t.Fatalf("Checks names = %q, want %q: layer 0 before layer 1, no Readiness", got, want)
	}
	if checks[0].Checker.Ready() || !checks[1].Checker.Ready() {
		t.Error("Checks' Checkers are not the System's values")
	}

	checks[0].Name = "changed"
	if got := lc.Checks()[0].Name; got != "cache" {
		t.Errorf("Checks()[0].Name = %q after changing an earlier result, want a fresh copy", got)
	}
}

func TestCoordinator_ChecksIsEmptyWithoutCheckers(t *testing.T) {
	g := graph.New()
	lc := coordinator(t, failsafe, g, node(g, &fake{name: "svc"}), readiness(g, "readiness"))
	if got := lc.Checks(); len(got) != 0 {
		t.Errorf("Checks = %v, want none", got)
	}
}

func TestReadiness_UnboundIsNotReady(t *testing.T) {
	var r lifecycle.Readiness
	if r.Ready() {
		t.Error("Ready() is true on an unbound Readiness")
	}
	if got := r.Checks(); len(got) != 0 {
		t.Errorf("Checks = %v on an unbound Readiness, want none", got)
	}
}

// A node that depends on the Readiness, as a health handler does, sees the
// Coordinator's readiness through it.
func TestReadiness_ReportsTheBoundCoordinator(t *testing.T) {
	var r recorder
	g := graph.New()
	probe := readiness(g, "readiness")
	db := value(g, "db", checker(func() bool { return true }))
	health := g.Define("health", func(s *graph.Scope) (starter, error) {
		rd := s.Use(probe)
		return func(context.Context) error {
			r.record("health start ready=" + strconv.FormatBool(rd.Ready()))
			return nil
		}, nil
	})
	sys, err := g.Build(health, db)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	rd := sys.Get(probe)
	lc := lifecycle.New(sys, lifecycle.Config{ShutdownTimeout: config.Duration(failsafe)})

	if rd.Ready() {
		t.Error("Readiness is ready before Exec")
	}
	if got, want := checkNames(rd.Checks()), checkNames(lc.Checks()); !slices.Equal(got, want) || len(got) != 1 {
		t.Errorf("Readiness Checks = %q, want the Coordinator's %q", got, want)
	}
	err = lc.Exec(context.Background(), func(context.Context) error {
		r.record("fn ready=" + strconv.FormatBool(rd.Ready()))
		return nil
	})
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	want := []string{"health start ready=false", "fn ready=true"}
	if got := r.list(); !slices.Equal(got, want) {
		t.Errorf("events = %q, want %q", got, want)
	}
	if rd.Ready() {
		t.Error("Readiness is ready after Exec returned")
	}
}

// A Readiness binds to one Coordinator: a second New over a System holding
// the same Readiness is a wiring mistake.
func TestNew_PanicsOnAReadinessBoundTwice(t *testing.T) {
	shared := new(lifecycle.Readiness)
	g := graph.New()
	probe := g.Define("readiness", func(*graph.Scope) (*lifecycle.Readiness, error) {
		return shared, nil
	})
	cfg := lifecycle.Config{ShutdownTimeout: config.Duration(time.Second)}
	build := func() *graph.System {
		sys, err := g.Build(probe)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		return sys
	}
	first := build()
	second := build()
	lifecycle.New(first, cfg)
	mustPanic(t, `lifecycle: New: Readiness "readiness" is bound to another Coordinator`, func() {
		lifecycle.New(second, cfg)
	})
}
