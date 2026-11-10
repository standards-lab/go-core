package graph_test

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/standards-lab/go-core/graph"
)

// TestConcurrentBuilds builds one Graph from many goroutines at once, with
// a group, an After, and a shared observer, so the race detector sees every
// read a Build makes of the Graph. Each Build constructs its own values,
// and the observer, called concurrently, sees every node of every Build.
func TestConcurrentBuilds(t *testing.T) {
	const builds = 16
	g := graph.New()
	servers := g.Group("servers")
	config := g.Fresh[int]("config")
	logger := g.Define("logger", func(s *graph.Scope) (string, error) {
		return fmt.Sprintf("logger(%d)", *s.Use(config)), nil
	})
	db := g.Define("db", func(s *graph.Scope) (*int, error) {
		v := *s.Use(config) + 1
		return &v, nil
	}, graph.After(logger))
	g.Define("http", func(s *graph.Scope) (int, error) {
		return *s.Use(db), nil
	}, graph.In(servers))
	var observed atomic.Int64
	g.Observe(func(graph.Ref) { observed.Add(1) })

	systems := make([]*graph.System, builds)
	errs := make([]error, builds)
	var wg sync.WaitGroup
	for i := range builds {
		wg.Go(func() {
			systems[i], errs[i] = g.Build(servers, logger)
		})
	}
	wg.Wait()

	const want = "0: config\n1: logger\n2: db\n3: http"
	for i, sys := range systems {
		if errs[i] != nil {
			t.Fatalf("Build %d: %v", i, errs[i])
		}
		if got := sys.String(); got != want {
			t.Errorf("Build %d layers:\n%s\nwant:\n%s", i, got, want)
		}
		for j, other := range systems[:i] {
			if sys.Get(config) == other.Get(config) {
				t.Errorf("Builds %d and %d share one Fresh value", i, j)
			}
		}
	}
	if got := observed.Load(); got != builds*4 {
		t.Errorf("observer called %d times; want %d, four nodes in each of %d Builds", got, builds*4, builds)
	}
}

// describe runs fn, a describing call made beside a Build, and reports
// whether it completed. It recovers the panic of a call the Build's freeze
// came first for, and fails the test on any other panic.
func describe(t *testing.T, fn func()) (completed bool) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		if msg := fmt.Sprint(r); !strings.HasPrefix(msg, "graph: ") || !strings.Contains(msg, "after Build") {
			t.Errorf("describing beside a Build panicked %q; want an \"after Build\" panic", msg)
		}
	}()
	fn()
	return true
}

// TestDescribeBesideBuild races every describing call, and Observe, against
// Builds of the same Graph. Each describing call either completes before
// the freeze or panics after it, and the lock it holds is released either
// way, so the Graph stays usable; once the Builds are done, describing
// panics as it always has.
func TestDescribeBesideBuild(t *testing.T) {
	for range 8 {
		g := graph.New()
		group := g.Group("group")
		base := g.Define("base", func(*graph.Scope) (int, error) { return 1, nil }, graph.In(group))

		start := make(chan struct{})
		var wg sync.WaitGroup
		var replaced atomic.Bool
		for i := range 4 {
			wg.Go(func() {
				<-start
				sys, err := g.Build(group)
				if err != nil {
					t.Errorf("Build: %v", err)
					return
				}
				if v := sys.Get(base); v != 1 && v != 2 {
					t.Errorf("base = %d; want 1, or 2 from the Replace", v)
				}
			})
			wg.Go(func() {
				<-start
				name := fmt.Sprint("node", i)
				describe(t, func() { g.Define(name, func(*graph.Scope) (int, error) { return i, nil }, graph.In(group)) })
				describe(t, func() { g.Value(name+"-value", i, graph.After(base)) })
				describe(t, func() { g.Fresh[int](name + "-fresh") })
				describe(t, func() { g.Group(name + "-group") })
				if describe(t, func() { g.Replace(base, func(*graph.Scope) (int, error) { return 2, nil }) }) {
					replaced.Store(true)
				}
			})
			wg.Go(func() {
				<-start
				g.Observe(func(graph.Ref) {})
			})
		}
		close(start)
		wg.Wait()

		mustPanic(t, `Define of "late" after Build`, func() {
			g.Define("late", func(*graph.Scope) (int, error) { return 0, nil })
		})
		mustPanic(t, `Replace of "base" after Build`, func() {
			g.Replace(base, func(*graph.Scope) (int, error) { return 3, nil })
		})
		sys, err := g.Build(base)
		if err != nil {
			t.Fatalf("Build after the race: %v", err)
		}
		want := 1
		if replaced.Load() {
			want = 2
		}
		if got := sys.Get(base); got != want {
			t.Errorf("base = %d after the race; want %d", got, want)
		}
	}
}

// TestObserveDuringBuild adds an observer while a Build runs, from another
// goroutine as a constructor waits: the running Build does not call it, and
// the next Build does.
func TestObserveDuringBuild(t *testing.T) {
	g := graph.New()
	running := make(chan struct{})
	resume := make(chan struct{})
	gate := g.Define("gate", func(*graph.Scope) (int, error) {
		if running != nil {
			close(running)
			<-resume
		}
		return 0, nil
	})
	after := g.Define("after", func(s *graph.Scope) (int, error) { return s.Use(gate), nil })

	var seen []graph.Ref
	done := make(chan error)
	go func() {
		_, err := g.Build(after)
		done <- err
	}()
	<-running
	g.Observe(func(r graph.Ref) { seen = append(seen, r) })
	close(resume)
	if err := <-done; err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(seen) != 0 {
		t.Fatalf("the running Build called the observer Observe added during it, with %v", seen)
	}

	running = nil
	if _, err := g.Build(after); err != nil {
		t.Fatalf("next Build: %v", err)
	}
	if want := []graph.Ref{after, gate}; len(seen) != 2 || seen[0] != want[0] || seen[1] != want[1] {
		t.Errorf("the next Build observed %v; want %v", seen, want)
	}
}

// TestObserveFromObserver adds an observer from inside an observer: Build
// holds no lock as it calls one, so the call returns, and the new observer
// applies from the next Build on.
func TestObserveFromObserver(t *testing.T) {
	g := graph.New()
	n := g.Value("n", 1)
	var added, later int
	g.Observe(func(graph.Ref) {
		if added == 0 {
			g.Observe(func(graph.Ref) { later++ })
		}
		added++
	})
	if _, err := g.Build(n); err != nil {
		t.Fatalf("Build: %v", err)
	}
	if later != 0 {
		t.Fatalf("the Build that added the observer called it %d times; want 0", later)
	}
	if _, err := g.Build(n); err != nil {
		t.Fatalf("next Build: %v", err)
	}
	if later != 1 {
		t.Errorf("the next Build called the added observer %d times; want 1", later)
	}
}
