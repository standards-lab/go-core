package graph_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/standards-lab/go-core/graph"
)

// zero is a constructor that returns 0 and does nothing else.
func zero(*graph.Scope) (int, error) { return 0, nil }

func TestGroup_IsARef(t *testing.T) {
	g := graph.New()
	refs := []graph.Ref{
		g.Define("count", zero),
		g.Group("workers"),
	}
	var got []string
	for _, r := range refs {
		got = append(got, r.Name())
	}
	if want := []string{"count", "workers"}; !slices.Equal(got, want) {
		t.Fatalf("names = %v, want %v", got, want)
	}
}

func TestGroup_RootBuildsEveryMemberInDefinitionOrder(t *testing.T) {
	c := counter{}
	g := graph.New()
	var seen []string
	g.Observe(func(name string) { seen = append(seen, name) })
	workers := g.Group("workers")
	g.Define("first", constant(c, "first", 1), graph.In(workers))
	g.Define("outsider", constant(c, "outsider", 2))
	g.Define("second", constant(c, "second", 3), graph.In(workers))
	g.Define("third", constant(c, "third", 4), graph.In(workers))

	sys := mustBuild(t, g, workers)
	if want := []string{"first", "second", "third"}; !slices.Equal(seen, want) {
		t.Fatalf("built %v, want %v in that order", seen, want)
	}
	if c["outsider"] != 0 || contains(sys, "outsider") {
		t.Fatalf("a node outside the group was built: runs %v, layers %v", c, names(sys))
	}
	want := [][]string{{"first", "second", "third"}}
	if got := names(sys); !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("layers = %v, want %v", got, want)
	}
}

func TestAfter_GroupOrdersAfterEachMemberInTheSystem(t *testing.T) {
	c := counter{}
	g := graph.New()
	workers := g.Group("workers")
	base := g.Define("base", constant(c, "base", 0))
	low := g.Define("low", constant(c, "low", 0), graph.In(workers))
	high := g.Define("high", func(s *graph.Scope) (int, error) {
		c["high"]++
		return s.Use(base), nil
	}, graph.In(workers))
	server := g.Define("server", constant(c, "server", 0), graph.After(workers))

	// Both members are in the System, so the server sits above the higher.
	sys := mustBuild(t, g, server, low, high)
	want := [][]string{{"base", "low"}, {"high"}, {"server"}}
	if got := names(sys); !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("layers = %v, want %v", got, want)
	}

	// After a group never builds its members, and orders only on the ones
	// in the System.
	clear(c)
	sys = mustBuild(t, g, server, low)
	if c["high"] != 0 || contains(sys, "high") {
		t.Fatalf("After of the group pulled high in: runs %v, layers %v", c, names(sys))
	}
	want = [][]string{{"low"}, {"server"}}
	if got := names(sys); !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("layers = %v, want %v", got, want)
	}
}

func TestGroup_MembershipDefinedAfterItsUse(t *testing.T) {
	g := graph.New()
	workers := g.Group("workers")
	// The server orders after the group before it has any members.
	server := g.Define("server", zero, graph.After(workers))
	base := g.Define("base", zero)
	g.Define("sweeper", func(s *graph.Scope) (int, error) { return s.Use(base), nil }, graph.In(workers))
	g.Define("relay", zero, graph.In(workers))

	sys := mustBuild(t, g, server, workers)
	want := [][]string{{"base", "relay"}, {"sweeper"}, {"server"}}
	if got := names(sys); !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("layers = %v, want %v", got, want)
	}
}

func TestGroup_EmptyBuildsAndOrdersNothing(t *testing.T) {
	g := graph.New()
	var seen []string
	g.Observe(func(name string) { seen = append(seen, name) })
	empty := g.Group("empty")
	g.Define("unreached", zero)
	base := g.Define("base", zero)
	server := g.Define("server", func(s *graph.Scope) (int, error) { return s.Use(base), nil }, graph.After(empty))

	sys := mustBuild(t, g, empty)
	if len(seen) != 0 || len(sys.Layers()) != 0 {
		t.Fatalf("an empty group root built %v, layers %v", seen, names(sys))
	}

	sys = mustBuild(t, g, server, empty)
	want := [][]string{{"base"}, {"server"}}
	if got := names(sys); !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("layers = %v, want %v", got, want)
	}
}

func TestReplace_KeepsAfterAndIn(t *testing.T) {
	g := graph.New()
	workers := g.Group("workers")
	gate := g.Define("gate", func(*graph.Scope) (string, error) { return "gate", nil })
	worker := g.Define("worker", func(*graph.Scope) (string, error) { return "real", nil },
		graph.In(workers), graph.After(gate))
	g.Replace(worker, func(*graph.Scope) (string, error) { return "fake", nil })

	// The worker is reached only through its group, and sits above gate
	// only through its After.
	sys := mustBuild(t, g, workers, gate)
	if got := sys.Get(worker); got != "fake" {
		t.Fatalf("worker = %q, want the substitute's fake", got)
	}
	want := [][]string{{"gate"}, {"worker"}}
	if got := names(sys); !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("layers = %v, want %v", got, want)
	}
}

func TestOrderOnlyCycle_Panics(t *testing.T) {
	t.Run("a member after its own group", func(t *testing.T) {
		g := graph.New()
		workers := g.Group("workers")
		a := g.Define("a", zero, graph.In(workers), graph.After(workers))
		mustPanic(t, "dependency cycle: a -> workers -> a", func() { _, _ = g.Build(a) })
	})
	t.Run("two groups after each other", func(t *testing.T) {
		g := graph.New()
		first, second := g.Group("first"), g.Group("second")
		g.Define("a", zero, graph.In(first), graph.After(second))
		g.Define("b", zero, graph.In(second), graph.After(first))
		mustPanic(t, "dependency cycle: a -> second -> b -> first -> a", func() { _, _ = g.Build(first, second) })
	})
	t.Run("closed by a Use through a group", func(t *testing.T) {
		g := graph.New()
		workers := g.Group("workers")
		var b *graph.Node[int]
		a := g.Define("a", func(s *graph.Scope) (int, error) { return s.Use(b), nil }, graph.In(workers))
		b = g.Define("b", zero, graph.After(workers))
		mustPanic(t, "dependency cycle: a -> b -> workers -> a", func() { _, _ = g.Build(a) })
	})
}

func TestOptions_WiringPanics(t *testing.T) {
	t.Run("after of a nil ref", func(t *testing.T) {
		g := graph.New()
		mustPanic(t, `Define "top" with After of a nil node (is its Define later?)`, func() {
			g.Define("top", zero, graph.After(nil))
		})
	})
	t.Run("after of a nil node", func(t *testing.T) {
		g := graph.New()
		var n *graph.Node[int]
		mustPanic(t, `Define "top" with After of a nil node (is its Define later?)`, func() {
			g.Define("top", zero, graph.After(n))
		})
	})
	t.Run("after of a nil group", func(t *testing.T) {
		g := graph.New()
		var grp *graph.Group
		mustPanic(t, `Define "top" with After of a nil group (is its Group later?)`, func() {
			g.Define("top", zero, graph.After(grp))
		})
	})
	t.Run("after of a foreign group", func(t *testing.T) {
		g, h := graph.New(), graph.New()
		foreign := h.Group("workers")
		mustPanic(t, `Define "top" with After of "workers", a group defined on another Graph`, func() {
			g.Define("top", zero, graph.After(foreign))
		})
	})
	t.Run("in a nil group", func(t *testing.T) {
		g := graph.New()
		mustPanic(t, `Define "top" with In of a nil group (is its Group later?)`, func() {
			g.Define("top", zero, graph.In(nil))
		})
	})
	t.Run("in a foreign group", func(t *testing.T) {
		g, h := graph.New(), graph.New()
		foreign := h.Group("workers")
		mustPanic(t, `Define "top" with In of "workers", a group defined on another Graph`, func() {
			g.Define("top", zero, graph.In(foreign))
		})
	})
	t.Run("nil option", func(t *testing.T) {
		g := graph.New()
		mustPanic(t, `Define "top" with a nil option`, func() {
			g.Define("top", zero, nil)
		})
	})
	t.Run("a panicking Define changes nothing", func(t *testing.T) {
		g := graph.New()
		workers := g.Group("workers")
		mustPanic(t, `Define "top" with After of a nil node (is its Define later?)`, func() {
			g.Define("top", zero, graph.In(workers), graph.After(nil))
		})
		// The name is still free, and the group gained no member.
		top := g.Define("top", zero)
		sys := mustBuild(t, g, workers)
		if len(sys.Layers()) != 0 {
			t.Fatalf("the failed Define joined the group: layers %v", names(sys))
		}
		mustBuild(t, g, top)
	})
	t.Run("build of a nil group", func(t *testing.T) {
		g := graph.New()
		var grp *graph.Group
		mustPanic(t, "Build of a nil group", func() { _, _ = g.Build(grp) })
	})
	t.Run("build of a foreign group", func(t *testing.T) {
		g, h, c := graph.New(), graph.New(), counter{}
		real := g.Define("real", constant(c, "real", 1))
		foreign := h.Group("workers")
		h.Define("member", constant(c, "member", 2), graph.In(foreign))
		mustPanic(t, `Build of "workers", a group defined on another Graph`, func() { _, _ = g.Build(real, foreign) })
		if len(c) != 0 {
			t.Errorf("constructors ran %v before the panic, want none", c)
		}
	})
	t.Run("group after build", func(t *testing.T) {
		g := graph.New()
		n := g.Define("n", func(*graph.Scope) (int, error) { return 0, errors.New("fails") })
		_, _ = g.Build(n)
		mustPanic(t, `Group of "late" after Build`, func() { g.Group("late") })
	})
	t.Run("group with an empty name", func(t *testing.T) {
		g := graph.New()
		mustPanic(t, "Group with an empty name", func() { g.Group("") })
	})
	t.Run("group of a group's name", func(t *testing.T) {
		g := graph.New()
		g.Group("workers")
		mustPanic(t, `Group of duplicate name "workers"`, func() { g.Group("workers") })
	})
	t.Run("group of a node's name", func(t *testing.T) {
		g := graph.New()
		g.Define("db", zero)
		mustPanic(t, `Group of duplicate name "db"`, func() { g.Group("db") })
	})
	t.Run("group of a name with a comma", func(t *testing.T) {
		g := graph.New()
		mustPanic(t, `Group of "a,b", a name with a comma or a newline`, func() { g.Group("a,b") })
	})
	t.Run("group of a name with a newline", func(t *testing.T) {
		g := graph.New()
		mustPanic(t, `Group of "a\nb", a name with a comma or a newline`, func() { g.Group("a\nb") })
		// The refused name left nothing behind.
		if g.Group("ok").Name() != "ok" {
			t.Fatal("Group after the refused name failed")
		}
	})
	t.Run("a panicking Group changes nothing", func(t *testing.T) {
		g := graph.New()
		g.Define("db", zero)
		mustPanic(t, `Group of duplicate name "db"`, func() { g.Group("db") })
		if g.Group("workers").Name() != "workers" {
			t.Fatal("Group after the refused name failed")
		}
	})
	t.Run("define of a group's name", func(t *testing.T) {
		g := graph.New()
		g.Group("workers")
		mustPanic(t, `Define of duplicate name "workers"`, func() { g.Define("workers", zero) })
	})
}
