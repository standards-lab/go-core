package lifecycle_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/standards-lab/go-core/graph"
)

// webService is a web service's composition, the graph graph's
// stages_test.go builds, over this package's recorder fakes: the same nodes
// and edges, so the stage table falls out of the layers, each value a fake
// that records its start and stop. The shape is restated here because
// graph's tests cannot run it on a Coordinator, and this package cannot
// import theirs.
func webService(r *recorder, serving chan struct{}) (*graph.Graph, []graph.Ref) {
	g := graph.New()
	config := node(g, &fake{name: "config", r: r})
	// Infrastructure: the pool and the object store, over the config.
	database := node(g, &fake{name: "database", r: r}, config)
	store := node(g, &fake{name: "store", r: r}, config)
	// The domain over the database and the object store; the schema, the
	// admin service over the pool; the storage admin domain over the store.
	domain := node(g, &fake{name: "domain", r: r}, database, store)
	schema := node(g, &fake{name: "schema", r: r}, database)
	storage := node(g, &fake{name: "storage", r: r}, store)
	// The reactors: the sweeper over the pool, once the schema is verified.
	reactors := node(g, &fake{name: "reactors", r: r}, schema, database)
	// The request edge: the server over the domain and admin pieces, started
	// after the reactors and drained before them. Its start marks it serving.
	server := node(g, &fake{name: "server", r: r, start: func(context.Context) error {
		close(serving)
		return nil
	}}, reactors, domain, schema, storage)
	// The reactors are a root: nothing uses their value.
	return g, []graph.Ref{server, reactors}
}

// TestRun_ServesAGraphShapedLikeTheWebService runs the web service's graph
// on Run: its layers start in stage order, it serves until its context
// ends, and it shuts down in reverse.
func TestRun_ServesAGraphShapedLikeTheWebService(t *testing.T) {
	var r recorder
	serving := make(chan struct{})
	g, roots := webService(&r, serving)
	lc := coordinator(t, failsafe, g, roots...)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := run(ctx, lc)

	recvOrFail(t, serving, "the server's Start")
	select {
	case err := <-done:
		t.Fatalf("Run returned %v before its context ended", err)
	case <-time.After(20 * time.Millisecond):
	}
	starts := [][]string{
		{"start config"},
		{"start database", "start store"},
		{"start domain", "start schema", "start storage"},
		{"start reactors"},
		{"start server"},
	}
	if got := r.list(); !phased(got, starts...) {
		t.Fatalf("events while serving = %q, want the starts %q and nothing more", got, starts)
	}

	cancel()
	if err := recvOrFail(t, done, "Run to return"); err != nil {
		t.Fatalf("Run = %v, want nil on a clean stop", err)
	}
	stops := [][]string{
		{"stop server"},
		{"stop reactors"},
		{"stop domain", "stop schema", "stop storage"},
		{"stop database", "stop store"},
		{"stop config"},
	}
	if got := r.list(); !phased(got, slices.Concat(starts, stops)...) {
		t.Errorf("events = %q, want the starts %q, then the stops %q", got, starts, stops)
	}
}
