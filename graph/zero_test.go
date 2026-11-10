package graph_test

import (
	"fmt"
	"testing"

	"github.com/standards-lab/go-core/graph"
)

// panicExactly fails the test unless fn panics with exactly want, so a
// message that gains or loses a hint fails it.
func panicExactly(t *testing.T, want string, fn func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		if r == nil {
			t.Fatalf("no panic; want %q", want)
		}
		if msg := fmt.Sprint(r); msg != want {
			t.Fatalf("panic %q; want %q", msg, want)
		}
	}()
	fn()
}

// A zero-value literal is not a handle any Define or Group made: every
// placement and Build that takes one panics as it does on a nil handle,
// with a "graph: " message rather than a nil dereference.
func TestZeroLiteral_PanicsAsANilHandle(t *testing.T) {
	t.Run("after of a zero node", func(t *testing.T) {
		panicExactly(t, `graph: Define "top" with After of a nil node (is its Define later?)`, func() {
			graph.New().Define("top", zero, graph.After(&graph.Node[int]{}))
		})
	})
	t.Run("after of a zero group", func(t *testing.T) {
		panicExactly(t, `graph: Define "top" with After of a nil group (is its Group later?)`, func() {
			graph.New().Define("top", zero, graph.After(&graph.Group{}))
		})
	})
	t.Run("in a zero group", func(t *testing.T) {
		panicExactly(t, `graph: Define "top" with In of a nil group (is its Group later?)`, func() {
			graph.New().Define("top", zero, graph.In(&graph.Group{}))
		})
	})
	t.Run("build of a zero node", func(t *testing.T) {
		panicExactly(t, "graph: Build of a nil node", func() {
			_, _ = graph.New().Build(&graph.Node[int]{})
		})
	})
	t.Run("build of a zero group", func(t *testing.T) {
		panicExactly(t, "graph: Build of a nil group", func() {
			_, _ = graph.New().Build(&graph.Group{})
		})
	})
	t.Run("use of a zero node", func(t *testing.T) {
		g := graph.New()
		top := g.Define("top", func(s *graph.Scope) (int, error) {
			return s.Use(&graph.Node[int]{}), nil
		})
		panicExactly(t, "graph: Use of a nil node", func() { _, _ = g.Build(top) })
	})
	t.Run("replace of a zero node", func(t *testing.T) {
		panicExactly(t, "graph: Replace of a nil node", func() {
			graph.New().Replace(&graph.Node[int]{}, zero)
		})
	})
}

// A nil handle panics with the same message a zero-value literal does.
func TestNilHandle_ReplacePanics(t *testing.T) {
	var n *graph.Node[int]
	panicExactly(t, "graph: Replace of a nil node", func() { graph.New().Replace(n, zero) })
}

// Name of a group handle no Group made is the empty string, not a nil
// dereference.
func TestGroupName_OfANilOrZeroHandleIsEmpty(t *testing.T) {
	var grp *graph.Group
	for _, c := range []struct {
		name string
		ref  graph.Ref
	}{
		{"nil group", grp},
		{"zero group", &graph.Group{}},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := c.ref.Name(); got != "" {
				t.Fatalf("Name() = %q, want empty", got)
			}
		})
	}
}
