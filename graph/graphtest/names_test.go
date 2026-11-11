package graphtest_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/standards-lab/go-core/graph"
	"github.com/standards-lab/go-core/graph/graphtest"
)

// zero is a constructor that returns 0 and does nothing else.
func zero(*graph.Scope) (int, error) { return 0, nil }

// nodes is a struct of handles in the shape a program's Nodes has, plus
// fields the check skips: an unexported handle, which the passing cases
// leave undefined, and a field of another type.
type nodes struct {
	MeterProvider *graph.Node[int]
	NATSStream    *graph.Node[int]
	Reactors      *graph.Group
	Root          graph.Ref

	hidden *graph.Node[int]
	Label  string
}

// errorStrings returns each error's message.
func errorStrings(errs []error) []string {
	var got []string
	for _, err := range errs {
		got = append(got, err.Error())
	}
	return got
}

func TestNameErrors_AcceptsWebStyleNames(t *testing.T) {
	g := graph.New()
	n := nodes{
		MeterProvider: g.Define("meterprovider", zero),
		NATSStream:    g.Define("natsstream", zero),
		Reactors:      g.Group("reactors"),
		Root:          g.Define("root", zero),
		hidden:        &graph.Node[int]{},
	}
	if errs := graphtest.NameErrors(n); errs != nil {
		t.Fatalf("NameErrors = %v, want none", errs)
	}
	graphtest.CheckNames(t, &n)
}

func TestNameErrors_AcceptsSlabStyleNames(t *testing.T) {
	g := graph.New()
	n := nodes{
		MeterProvider: g.Define("meter provider", zero),
		NATSStream:    g.Define("nats stream", zero),
		Reactors:      g.Group("reactors"),
		Root:          g.Define("root", zero),
	}
	if errs := graphtest.NameErrors(&n); errs != nil {
		t.Fatalf("NameErrors = %v, want none", errs)
	}
	graphtest.CheckNames(t, n)
}

func TestNameErrors_RejectsAMismatchedName(t *testing.T) {
	g := graph.New()
	n := nodes{
		MeterProvider: g.Define("meters", zero),
		NATSStream:    g.Define("nats stream", zero),
		Reactors:      g.Group("reactor"),
		Root:          g.Define("root", zero),
	}
	got := errorStrings(graphtest.NameErrors(n))
	want := []string{
		`nodes.MeterProvider: named "meters"`,
		`nodes.Reactors: named "reactor"`,
	}
	if !slices.Equal(got, want) {
		t.Fatalf("NameErrors = %q, want %q", got, want)
	}
}

func TestNameErrors_RejectsAnUndefinedHandle(t *testing.T) {
	g := graph.New()
	n := nodes{
		MeterProvider: &graph.Node[int]{},
		Reactors:      g.Group("reactors"),
	}
	got := errorStrings(graphtest.NameErrors(n))
	want := []string{
		"nodes.MeterProvider: no handle",
		"nodes.NATSStream: no handle",
		"nodes.Root: no handle",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("NameErrors = %q, want %q", got, want)
	}
}

func TestNameErrors_RejectsANonStruct(t *testing.T) {
	for _, v := range []any{nil, 1, (*nodes)(nil)} {
		errs := graphtest.NameErrors(v)
		if len(errs) != 1 || !strings.Contains(errs[0].Error(), "not a struct") {
			t.Errorf("NameErrors(%#v) = %v, want one not-a-struct error", v, errs)
		}
	}
}

// recorder is a testing.TB that records the errors reported on it instead
// of failing the test.
type recorder struct {
	testing.TB
	errors []string
}

func (r *recorder) Helper() {}

func (r *recorder) Error(args ...any) {
	for _, a := range args {
		r.errors = append(r.errors, a.(error).Error())
	}
}

func TestCheckNames_ReportsEachError(t *testing.T) {
	g := graph.New()
	n := nodes{
		MeterProvider: g.Define("meters", zero),
		NATSStream:    g.Define("nats stream", zero),
		Reactors:      g.Group("reactors"),
	}
	r := &recorder{TB: t}
	graphtest.CheckNames(r, n)
	want := []string{
		`nodes.MeterProvider: named "meters"`,
		"nodes.Root: no handle",
	}
	if !slices.Equal(r.errors, want) {
		t.Fatalf("CheckNames reported %q, want %q", r.errors, want)
	}
}
