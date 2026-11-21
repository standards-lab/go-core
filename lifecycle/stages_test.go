package lifecycle_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/standards-lab/go-core/graph"
)

func webService(r *recorder, serving chan struct{}) (*graph.Graph, []graph.Ref) {
	g := graph.New()
	config := node(g, &fake{name: "config", r: r})
	database := node(g, &fake{name: "database", r: r}, config)
	store := node(g, &fake{name: "store", r: r}, config)
	domain := node(g, &fake{name: "domain", r: r}, database, store)
	schema := node(g, &fake{name: "schema", r: r}, database)
	storage := node(g, &fake{name: "storage", r: r}, store)
	reactors := node(g, &fake{name: "reactors", r: r}, schema, database)
	server := node(g, &fake{name: "server", r: r, start: func(context.Context) error {
		close(serving)
		return nil
	}}, reactors, domain, schema, storage)
	return g, []graph.Ref{server, reactors}
}

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
