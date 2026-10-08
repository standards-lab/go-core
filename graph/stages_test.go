package graph_test

import (
	"slices"
	"testing"

	"github.com/standards-lab/go-core/graph"
)

// webService is a web service's composition expressed as a graph, with
// fake values: its startup stages fall out of the edges instead of being
// numbered by hand.
type webService struct {
	graph    *graph.Graph
	config   *graph.Node[string]
	database *graph.Node[string]
	store    *graph.Node[string]
	domain   *graph.Node[string]
	schema   *graph.Node[string]
	storage  *graph.Node[string]
	reactors *graph.Node[string]
	server   *graph.Node[string]
}

func newWebService() *webService {
	g := graph.New()
	w := &webService{graph: g}
	w.config = g.Define("config", func(*graph.Scope) (string, error) { return "config", nil })
	// Infrastructure: the pool and the object store, over the config.
	w.database = g.Define("database", func(s *graph.Scope) (string, error) {
		return "pool(" + s.Use(w.config) + ")", nil
	})
	w.store = g.Define("store", func(s *graph.Scope) (string, error) {
		return "objects(" + s.Use(w.config) + ")", nil
	})
	// The domain runs over the database and the object store.
	w.domain = g.Define("domain", func(s *graph.Scope) (string, error) {
		return "domain(" + s.Use(w.database) + "," + s.Use(w.store) + ")", nil
	})
	// Schema: the admin service over the pool.
	w.schema = g.Define("schema", func(s *graph.Scope) (string, error) {
		return "admin(" + s.Use(w.database) + ")", nil
	})
	// The storage admin domain over the object store.
	w.storage = g.Define("storage", func(s *graph.Scope) (string, error) {
		return "storage(" + s.Use(w.store) + ")", nil
	})
	// Reactors: the sweeper over the pool, once the schema is verified;
	// it takes nothing from the schema, so the edge is order-only.
	w.reactors = g.Define("reactors", func(s *graph.Scope) (string, error) {
		s.After(w.schema)
		return "sweeper(" + s.Use(w.database) + ")", nil
	})
	// The request edge: the server over the domain and admin pieces, started after
	// the reactors and drained before them.
	w.server = g.Define("server", func(s *graph.Scope) (string, error) {
		s.After(w.reactors)
		return "server(" + s.Use(w.domain) + "," + s.Use(w.schema) + "," + s.Use(w.storage) + ")", nil
	})
	return w
}

func TestLayers_WebServiceStageOrder(t *testing.T) {
	w := newWebService()
	// The reactors are a root: nothing uses their value, and the server
	// only orders on them.
	sys := mustBuild(t, w.graph, w.server, w.reactors)

	layer := func(name string) int {
		t.Helper()
		l := layerOf(sys, name)
		if l < 0 {
			t.Fatalf("%s is not in the System: %v", name, names(sys))
		}
		return l
	}
	for _, infra := range []string{"database", "store"} {
		if layer(infra) >= layer("schema") {
			t.Errorf("%s (layer %d) is not below schema (layer %d)", infra, layer(infra), layer("schema"))
		}
	}
	if layer("schema") >= layer("reactors") {
		t.Errorf("schema (layer %d) is not below reactors (layer %d)", layer("schema"), layer("reactors"))
	}
	if layer("reactors") >= layer("server") {
		t.Errorf("reactors (layer %d) is not below server (layer %d)", layer("reactors"), layer("server"))
	}

	want := [][]string{
		{"config"},
		{"database", "store"},
		{"domain", "schema", "storage"},
		{"reactors"},
		{"server"},
	}
	if got := names(sys); !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("layers = %v, want %v", got, want)
	}
	if got := sys.Get(w.server); got != "server(domain(pool(config),objects(config)),admin(pool(config)),storage(objects(config)))" {
		t.Fatalf("server = %q", got)
	}
}

func TestBuild_WebServiceSubset(t *testing.T) {
	w := newWebService()
	sys := mustBuild(t, w.graph, w.schema)
	for _, name := range []string{"config", "database", "schema"} {
		if !contains(sys, name) {
			t.Errorf("%s is missing from the schema-only System", name)
		}
	}
	for _, name := range []string{"store", "domain", "storage", "reactors", "server"} {
		if contains(sys, name) {
			t.Errorf("%s is in the schema-only System", name)
		}
	}
}
