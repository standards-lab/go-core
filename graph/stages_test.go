package graph_test

import (
	"testing"

	"github.com/standards-lab/go-core/graph"
)

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
	w.database = g.Define("database", func(s *graph.Scope) (string, error) {
		return "pool(" + s.Use(w.config) + ")", nil
	})
	w.store = g.Define("store", func(s *graph.Scope) (string, error) {
		return "objects(" + s.Use(w.config) + ")", nil
	})
	w.domain = g.Define("domain", func(s *graph.Scope) (string, error) {
		return "domain(" + s.Use(w.database) + "," + s.Use(w.store) + ")", nil
	})
	w.schema = g.Define("schema", func(s *graph.Scope) (string, error) {
		return "admin(" + s.Use(w.database) + ")", nil
	})
	w.storage = g.Define("storage", func(s *graph.Scope) (string, error) {
		return "storage(" + s.Use(w.store) + ")", nil
	})
	w.reactors = g.Define("reactors", func(s *graph.Scope) (string, error) {
		return "sweeper(" + s.Use(w.database) + ")", nil
	}, graph.After(w.schema))
	w.server = g.Define("server", func(s *graph.Scope) (string, error) {
		return "server(" + s.Use(w.domain) + "," + s.Use(w.schema) + "," + s.Use(w.storage) + ")", nil
	}, graph.After(w.reactors))
	return w
}

func TestLayers_WebServiceStageOrder(t *testing.T) {
	w := newWebService()
	sys := mustBuild(t, w.graph, w.server, w.reactors)

	below := func(low, high graph.Ref) {
		t.Helper()
		if l, h := sys.LayerOf(low), sys.LayerOf(high); l >= h {
			t.Errorf("%s (layer %d) is not below %s (layer %d); layers:\n%s", low.Name(), l, high.Name(), h, sys)
		}
	}
	below(w.database, w.schema)
	below(w.store, w.schema)
	below(w.schema, w.reactors)
	below(w.reactors, w.server)

	want := "0: config\n1: database, store\n2: domain, schema, storage\n3: reactors\n4: server"
	if got := sys.String(); got != want {
		t.Fatalf("layers:\n%s\nwant:\n%s", got, want)
	}
	if got := sys.Dropped(); len(got) != 0 {
		t.Fatalf("Dropped() = %v, want none", got)
	}
	if got := sys.Get(w.server); got != "server(domain(pool(config),objects(config)),admin(pool(config)),storage(objects(config)))" {
		t.Fatalf("server = %q", got)
	}
}

func TestBuild_WebServiceSubset(t *testing.T) {
	w := newWebService()
	sys := mustBuild(t, w.graph, w.schema)
	for _, n := range []graph.Ref{w.config, w.database, w.schema} {
		if !sys.Has(n) {
			t.Errorf("%s is missing from the schema-only System", n.Name())
		}
	}
	for _, n := range []graph.Ref{w.store, w.domain, w.storage, w.reactors, w.server} {
		if sys.Has(n) {
			t.Errorf("%s is in the schema-only System", n.Name())
		}
	}
}
