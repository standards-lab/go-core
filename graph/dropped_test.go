package graph_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/standards-lab/go-core/graph"
)

// dropNames returns each drop as "node after target", for comparison and a
// failure message.
func dropNames(drops []graph.Drop) []string {
	out := make([]string, len(drops))
	for i, d := range drops {
		out[i] = fmt.Sprintf("%s after %s", d.Node.Name(), d.After.Name())
	}
	return out
}

// ordered defines a graph whose server orders After the cache node, the
// workers group, and the db node, in that order, and whose api orders
// After the db node and the workers group; cron and mail are the workers.
// No node Uses another, so a Build holds exactly its roots. It returns the
// graph and its handles.
func ordered() (g *graph.Graph, h struct {
	db, cache, cron, mail, api, server *graph.Node[int]
	workers                            *graph.Group
}) {
	g = graph.New()
	h.workers = g.Group("workers")
	h.db = g.Define("db", zero)
	h.cache = g.Define("cache", zero)
	h.cron = g.Define("cron", zero, graph.In(h.workers))
	h.mail = g.Define("mail", zero, graph.In(h.workers))
	h.api = g.Define("api", zero, graph.After(h.db, h.workers))
	h.server = g.Define("server", zero, graph.After(h.cache), graph.After(h.workers, h.db))
	return g, h
}

func TestDropped_FullBuildReportsNone(t *testing.T) {
	g, h := ordered()
	sys := mustBuild(t, g, h.server, h.api, h.db, h.cache, h.workers)
	if got := sys.Dropped(); len(got) != 0 {
		t.Fatalf("Dropped() = %v, want none; layers %v", dropNames(got), names(sys))
	}
}

func TestDropped_SubsetReportsTheAbsentNodeTarget(t *testing.T) {
	g, h := ordered()
	// db and every worker are present; only the server's cache is not.
	sys := mustBuild(t, g, h.server, h.db, h.cron, h.mail)
	want := []string{"server after cache"}
	if got := dropNames(sys.Dropped()); !slices.Equal(got, want) {
		t.Fatalf("Dropped() = %v, want %v", got, want)
	}
	// The drop is silent at Build: the System still orders what is present.
	if layerOf(sys, "server") != 1 {
		t.Fatalf("server is not above its present targets; layers %v", names(sys))
	}
}

func TestDropped_GroupWithNoMemberPresent(t *testing.T) {
	g, h := ordered()

	t.Run("no member present is a drop", func(t *testing.T) {
		sys := mustBuild(t, g, h.api, h.db)
		want := []string{"api after workers"}
		if got := dropNames(sys.Dropped()); !slices.Equal(got, want) {
			t.Fatalf("Dropped() = %v, want %v", got, want)
		}
		if d := sys.Dropped()[0]; d.After != graph.Ref(h.workers) {
			t.Fatalf("After = %#v, want the group handle %#v as declared", d.After, h.workers)
		}
	})
	t.Run("a member present is not a drop", func(t *testing.T) {
		// mail is absent, but cron orders the api, so the group is no drop.
		sys := mustBuild(t, g, h.api, h.db, h.cron)
		if got := sys.Dropped(); len(got) != 0 {
			t.Fatalf("Dropped() = %v, want none with cron present", dropNames(got))
		}
	})
}

func TestDropped_IsInDefinitionThenAfterOrder(t *testing.T) {
	g, h := ordered()
	// The roots list the server before the api, and the server's Afters
	// span two options; the drops follow the api's definition first, then
	// each node's After refs as declared.
	sys := mustBuild(t, g, h.server, h.api)
	want := []string{
		"api after db", "api after workers",
		"server after cache", "server after workers", "server after db",
	}
	if got := dropNames(sys.Dropped()); !slices.Equal(got, want) {
		t.Fatalf("Dropped() = %v, want %v", got, want)
	}
	// Each Build of the same roots reports the same order, whatever order
	// the Build's map of entries iterates in.
	for range 20 {
		if got := dropNames(mustBuild(t, g, h.server, h.api).Dropped()); !slices.Equal(got, want) {
			t.Fatalf("Dropped() = %v, want %v", got, want)
		}
	}
}

func TestDropped_CarriesTheHandles(t *testing.T) {
	g, h := ordered()
	sys := mustBuild(t, g, h.server, h.api)
	want := []graph.Drop{
		{Node: h.api, After: h.db},
		{Node: h.api, After: h.workers},
		{Node: h.server, After: h.cache},
		{Node: h.server, After: h.workers},
		{Node: h.server, After: h.db},
	}
	got := sys.Dropped()
	// Interface equality: the very handles Define and Group returned.
	if !slices.Equal(got, want) {
		t.Fatalf("Dropped() = %v, want %v", dropNames(got), dropNames(want))
	}
	// The Node asserts back to the typed handle, as a Dependency's Ref does.
	if n, ok := got[2].Node.(*graph.Node[int]); !ok || n != h.server {
		t.Fatalf("server's drop Node is %T, want the *graph.Node[int] handle", got[2].Node)
	}
}

func TestDropped_ReturnsACopy(t *testing.T) {
	g, h := ordered()
	sys := mustBuild(t, g, h.server, h.db, h.workers)
	first := sys.Dropped()
	if len(first) != 1 {
		t.Fatalf("Dropped() = %v, want one drop", dropNames(first))
	}
	first[0] = graph.Drop{Node: h.db, After: h.db}
	if got := dropNames(sys.Dropped()); !slices.Equal(got, []string{"server after cache"}) {
		t.Fatalf("Dropped() after a caller's change = %v", got)
	}
}
