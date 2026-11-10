package graph_test

import (
	"errors"
	"fmt"
	"runtime"
	"slices"
	"testing"

	"github.com/standards-lab/go-core/graph"
)

// refNames returns each ref's name, for a failure message.
func refNames(refs []graph.Ref) []string {
	out := make([]string, len(refs))
	for i, r := range refs {
		out[i] = r.Name()
	}
	return out
}

// layered defines a graph that, built from server, clock, and tier, has
// the layers 0: cfg, clock, seed, gate; 1: cache, db; 2: server, each in
// definition order. gate is a Fresh node the server Uses, seed a Value node
// the db Uses, and clock a Value node built only as a root; the server
// orders After tier, whose members are cache and db. lonely is never
// reached. It returns the graph and its handles.
func layered() (g *graph.Graph, h struct {
	cfg, db, cache, server, lonely *graph.Node[int]
	clock                          *graph.Node[string]
	gate                           *graph.Node[*bool]
	seed                           *graph.Node[int]
	tier                           *graph.Group
}) {
	g = graph.New()
	h.tier = g.Group("tier")
	h.cfg = g.Define("cfg", func(*graph.Scope) (int, error) { return 1, nil })
	h.clock = g.Value("clock", "tick")
	h.seed = g.Value("seed", 7)
	h.gate = g.Fresh[bool]("gate")
	h.lonely = g.Define("lonely", zero)
	h.cache = g.Define("cache", func(s *graph.Scope) (int, error) { return s.Use(h.cfg), nil }, graph.In(h.tier))
	h.db = g.Define("db", func(s *graph.Scope) (int, error) { return s.Use(h.cfg) + s.Use(h.seed), nil }, graph.In(h.tier))
	h.server = g.Define("server", func(s *graph.Scope) (int, error) {
		_ = s.Use(h.gate)
		return s.Use(h.db), nil
	}, graph.After(h.tier))
	return g, h
}

func TestDependency_RefIsTheHandle(t *testing.T) {
	g, h := layered()
	sys := mustBuild(t, g, h.server, h.clock, h.tier)

	want := map[string]graph.Ref{
		"cfg": h.cfg, "clock": h.clock, "seed": h.seed, "gate": h.gate,
		"cache": h.cache, "db": h.db, "server": h.server,
	}
	seen := 0
	for _, layer := range sys.Layers() {
		for _, d := range layer {
			seen++
			ref, ok := want[d.Name]
			if !ok {
				t.Fatalf("unexpected node %q in the System", d.Name)
			}
			// Interface equality: the same *Node[T] Define, Value, or Fresh
			// returned, not a copy carrying the same node.
			if d.Ref != ref {
				t.Errorf("%s: Ref = %#v, want the handle %#v", d.Name, d.Ref, ref)
			}
		}
	}
	if seen != len(want) {
		t.Fatalf("saw %d nodes, want %d", seen, len(want))
	}
	// The Ref asserts back to the typed handle, which Get takes.
	d := dependency(t, sys, h.gate)
	n, ok := d.Ref.(*graph.Node[*bool])
	if !ok {
		t.Fatalf("gate's Ref is %T, want *graph.Node[*bool]", d.Ref)
	}
	if sys.Get(n) != d.Value {
		t.Errorf("gate's Ref does not reach its value through Get")
	}
}

func TestObserve_PassesValueAndFreshHandles(t *testing.T) {
	g, h := layered()
	var seen []graph.Ref
	g.Observe(func(r graph.Ref) { seen = append(seen, r) })
	mustBuild(t, g, h.server)
	want := []graph.Ref{h.server, h.gate, h.db, h.cfg, h.seed}
	if !slices.Equal(seen, want) {
		t.Fatalf("observed %v, want %v", refNames(seen), refNames(want))
	}
}

// An observer is never removed: it sees every later Build of its Graph,
// each node once per Build.
func TestObserve_PersistsAcrossBuilds(t *testing.T) {
	g, h := layered()
	var seen []graph.Ref
	g.Observe(func(r graph.Ref) { seen = append(seen, r) })
	mustBuild(t, g, h.cfg)
	mustBuild(t, g, h.clock, h.cfg)
	want := []graph.Ref{h.cfg, h.clock, h.cfg}
	if !slices.Equal(seen, want) {
		t.Fatalf("observed %v, want %v", refNames(seen), refNames(want))
	}
}

func TestLayerOf_MatchesLayers(t *testing.T) {
	g, h := layered()
	sys := mustBuild(t, g, h.server, h.clock, h.tier)

	for i, layer := range sys.Layers() {
		for _, d := range layer {
			if got := sys.LayerOf(d.Ref); got != i {
				t.Errorf("LayerOf(%s) = %d, want %d, its place in Layers; layers:\n%s", d.Name, got, i, sys)
			}
		}
	}
	// Pinned, so a numbering that agreed with a wrong Layers still fails.
	for ref, want := range map[graph.Ref]int{h.cfg: 0, h.clock: 0, h.seed: 0, h.gate: 0, h.cache: 1, h.db: 1, h.server: 2} {
		if got := sys.LayerOf(ref); got != want {
			t.Errorf("LayerOf(%s) = %d, want %d; layers:\n%s", ref.Name(), got, want, sys)
		}
	}
}

func TestLayerOf_Panics(t *testing.T) {
	g, h := layered()
	sys := mustBuild(t, g, h.server)

	t.Run("unreached node", func(t *testing.T) {
		panicExactly(t, `graph: LayerOf of "lonely", a node not in the System`, func() { sys.LayerOf(h.lonely) })
	})
	t.Run("node not built in this System", func(t *testing.T) {
		panicExactly(t, `graph: LayerOf of "clock", a node not in the System`, func() { sys.LayerOf(h.clock) })
	})
	t.Run("node of another Graph", func(t *testing.T) {
		other := graph.New()
		cfg := other.Define("cfg", zero)
		mustBuild(t, other, cfg)
		panicExactly(t, `graph: LayerOf of "cfg", a node not in the System`, func() { sys.LayerOf(cfg) })
	})
	t.Run("group", func(t *testing.T) {
		panicExactly(t, `graph: LayerOf of "tier", a group, which is not a node`, func() { sys.LayerOf(h.tier) })
	})
	t.Run("nil ref", func(t *testing.T) {
		panicExactly(t, "graph: LayerOf of a nil node", func() { sys.LayerOf(nil) })
	})
	t.Run("nil node handle", func(t *testing.T) {
		var n *graph.Node[int]
		panicExactly(t, "graph: LayerOf of a nil node", func() { sys.LayerOf(n) })
	})
	t.Run("nil group handle", func(t *testing.T) {
		var grp *graph.Group
		panicExactly(t, "graph: LayerOf of a nil group", func() { sys.LayerOf(grp) })
	})
}

func TestHas_ReportsMembership(t *testing.T) {
	g, h := layered()
	sys := mustBuild(t, g, h.server)

	for _, ref := range []graph.Ref{h.server, h.gate, h.db, h.cfg, h.seed} {
		if !sys.Has(ref) {
			t.Errorf("Has(%s) = false, want true; layers:\n%s", ref.Name(), sys)
		}
	}
	// cache is a member of the tier the server orders After, which builds
	// nothing; clock and lonely no root reaches.
	for _, ref := range []graph.Ref{h.cache, h.clock, h.lonely} {
		if sys.Has(ref) {
			t.Errorf("Has(%s) = true, want false; layers:\n%s", ref.Name(), sys)
		}
	}
	// A node of another Graph is not in s, even under a name s holds.
	other := graph.New()
	cfg := other.Define("cfg", zero)
	mustBuild(t, other, cfg)
	if sys.Has(cfg) {
		t.Errorf("Has of another Graph's cfg = true, want false")
	}
}

func TestHas_Panics(t *testing.T) {
	g, h := layered()
	sys := mustBuild(t, g, h.server)

	t.Run("group", func(t *testing.T) {
		panicExactly(t, `graph: Has of "tier", a group, which is not a node`, func() { sys.Has(h.tier) })
	})
	t.Run("nil ref", func(t *testing.T) {
		panicExactly(t, "graph: Has of a nil node", func() { sys.Has(nil) })
	})
	t.Run("nil node handle", func(t *testing.T) {
		var n *graph.Node[int]
		panicExactly(t, "graph: Has of a nil node", func() { sys.Has(n) })
	})
	t.Run("nil group handle", func(t *testing.T) {
		var grp *graph.Group
		panicExactly(t, "graph: Has of a nil group", func() { sys.Has(grp) })
	})
}

func TestDependencies_FlattensLayers(t *testing.T) {
	g, h := layered()
	sys := mustBuild(t, g, h.server, h.clock, h.tier)

	var flat []graph.Dependency
	for _, layer := range sys.Layers() {
		flat = append(flat, layer...)
	}
	got := sys.Dependencies()
	if !slices.Equal(got, flat) {
		t.Fatalf("Dependencies() = %v, want Layers() flattened, %v", got, flat)
	}
	// Pinned: layer order, then definition order within a layer.
	refs := make([]graph.Ref, len(got))
	for i, d := range got {
		refs[i] = d.Ref
	}
	want := []graph.Ref{h.cfg, h.clock, h.seed, h.gate, h.cache, h.db, h.server}
	if !slices.Equal(refs, want) {
		t.Fatalf("Dependencies() refs = %v, want %v", refNames(refs), refNames(want))
	}

	// Each call returns a fresh copy.
	got[0] = graph.Dependency{Name: "changed"}
	if again := sys.Dependencies(); again[0].Ref != h.cfg {
		t.Fatalf("a change to one result reached the next: %v", again[0])
	}

	if got := mustBuild(t, graph.New()).Dependencies(); len(got) != 0 {
		t.Fatalf("Dependencies() of an empty System = %v, want none", got)
	}
}

func TestString_PrintsTheLayerTable(t *testing.T) {
	g, h := layered()
	// The roots list the server first, so a table in build order would put
	// gate before cfg; the table is in definition order within a layer.
	sys := mustBuild(t, g, h.server, h.clock, h.tier)
	want := "0: cfg, clock, seed, gate\n1: cache, db\n2: server"
	if got := sys.String(); got != want {
		t.Fatalf("String() =\n%s\nwant\n%s", got, want)
	}

	if got := mustBuild(t, graph.New()).String(); got != "" {
		t.Fatalf("String() of an empty System = %q, want empty", got)
	}
}

// observerPanic builds roots on g and returns what the Build panicked
// with, failing the test when it did not panic.
func observerPanic(t *testing.T, g *graph.Graph, roots ...graph.Ref) (r any) {
	t.Helper()
	defer func() { r = recover() }()
	_, _ = g.Build(roots...)
	t.Fatal("Build returned, want the observer's panic")
	return nil
}

// An observer's panic leaves Build as a "graph: " panic naming the node,
// with the original value wrapped when it is an error, whether the node is
// a root or one a Use reached.
func TestObserve_PanicNamesTheNode(t *testing.T) {
	boom := errors.New("boom")
	cases := []struct {
		name  string
		value any
		want  string
	}{
		{"error", boom, "boom"},
		{"string", "snap", "snap"},
		{"other value", 42, "42"},
	}
	for _, c := range cases {
		for _, at := range []string{"root", "used"} {
			t.Run(c.name+" at "+at, func(t *testing.T) {
				g, h := layered()
				target := graph.Ref(h.server)
				if at == "used" {
					target = h.seed
				}
				g.Observe(func(r graph.Ref) {
					if r == target {
						panic(c.value)
					}
				})
				r := observerPanic(t, g, h.server)

				want := fmt.Sprintf("graph: observer of %q panicked: %s", target.Name(), c.want)
				if msg := fmt.Sprint(r); msg != want {
					t.Fatalf("panic %q; want %q", msg, want)
				}
				if c.value == boom {
					err, ok := r.(error)
					if !ok || !errors.Is(err, boom) {
						t.Fatalf("panic %#v does not wrap the observer's error", r)
					}
				}
			})
		}
	}
}

// A runtime.Goexit in an observer, which t.FailNow calls, is not a panic:
// it ends the Build's goroutine without a "graph: " panic in its way.
func TestObserve_GoexitPassesThrough(t *testing.T) {
	g, h := layered()
	g.Observe(func(r graph.Ref) {
		if r == graph.Ref(h.db) {
			runtime.Goexit()
		}
	})
	done := make(chan struct{})
	var returned bool
	var panicked any
	go func() {
		defer close(done)
		defer func() { panicked = recover() }()
		_, _ = g.Build(h.server)
		returned = true
	}()
	<-done
	if returned || panicked != nil {
		t.Fatalf("Build returned %v, panicked %v; want the goroutine to exit", returned, panicked)
	}
}
