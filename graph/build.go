package graph

import (
	"fmt"
	"slices"
	"strings"
)

// entry is one node in one Build: its state, its value once built, and the
// edges its constructor recorded.
type entry struct {
	node     *node
	building bool
	value    any
	// layer is the node's layer in the System, set once the Build has
	// computed the layers.
	layer int
	// uses are the nodes the constructor reached through Use, deduplicated,
	// in the order it first reached them.
	uses []*node
}

// build is the state of one [Graph.Build].
type build struct {
	graph *Graph
	// observers are the Graph's observers as the Build started, so one
	// Observe adds while the Build runs applies from the next Build on.
	observers []func(Ref)
	entries   map[*node]*entry
	// stack is the chain of nodes under construction, outermost first,
	// which names the path of a cycle.
	stack []*node
}

// abort carries a constructor's labelled error out of the dependents whose
// Use reached it, up to Build, which recovers it and returns the error. A
// constructor that recovers it instead breaks that unwinding, which
// construct detects and panics on.
type abort struct {
	err error
}

// Build constructs every node the roots reach through [Scope.Use], each at
// most once, depth-first, and returns them as a [System] in computed
// layers. A [Group] root stands for its members, built in definition order;
// an empty one builds nothing. An [After] never builds its target. Each
// Build is independent and constructs fresh values. A constructor's error
// fails the Build, labelled with its node's name and wrapping the error,
// whether the node is a root or a dependency some Use reached. Each
// observer [Graph.Observe] added is called as Build begins a node; one that
// panics makes Build panic with a "graph: " message naming the node.
// Build panics on a nil root, a root defined on another Graph, or a
// dependency cycle, among the nodes Use reaches or the order-only edges
// [After] adds among them, a group's included; the message names the
// cycle's path. It checks every root before it constructs any, so a wiring
// mistake in one root panics before another root's constructor runs.
// It also panics when a constructor returns after a Use it made aborted,
// which only a recover in the constructor allows: a constructor must not
// recover. The first Build freezes g: [Graph.Define], [Graph.Value],
// [Graph.Fresh], [Graph.Group], and [Graph.Replace] panic after it.
// Builds may run concurrently, and call the observers concurrently.
func (g *Graph) Build(roots ...Ref) (sys *System, err error) {
	// The lock covers only the freeze and the copy. Once built is set,
	// every describing call panics before it changes g, so the nodes, their
	// constructors and Afters, and the groups' members are immutable, and
	// the Build reads them without the lock, beside other Builds and beside
	// an observer that calls Observe.
	g.mu.Lock()
	g.built = true
	observers := slices.Clone(g.observers)
	g.mu.Unlock()
	b := &build{graph: g, observers: observers, entries: make(map[*node]*entry)}
	defer func() {
		if r := recover(); r != nil {
			a, ok := r.(abort)
			if !ok {
				panic(r)
			}
			sys, err = nil, a.err
		}
	}()
	// Resolve every root first, so a nil or foreign root anywhere in the
	// list panics before any constructor runs. A group expands here to its
	// members, complete since Define panics after the first Build.
	var nodes []*node
	for _, r := range roots {
		nodes = append(nodes, g.resolve(r, "Build", false)...)
	}
	for _, n := range nodes {
		if err := b.construct(n); err != nil {
			return nil, err
		}
	}
	return b.system(), nil
}

// construct builds n unless it is built already, and returns its
// constructor's labelled error. Reaching a node under construction is a
// cycle, and panics with its path. A constructor that returns, with a value
// or an error, after a Use it made aborted has recovered the abort, and
// construct panics naming it, since the failure that aborted the Use would
// otherwise be lost.
func (b *build) construct(n *node) error {
	if e, ok := b.entries[n]; ok {
		if e.building {
			panic("graph: dependency cycle: " + cyclePath(b.stack, n))
		}
		return nil
	}
	e := &entry{node: n, building: true}
	b.entries[n] = e
	b.stack = append(b.stack, n)
	for _, fn := range b.observers {
		observe(fn, n)
	}
	s := &Scope{build: b, entry: e}
	defer func() {
		s.done = true
		b.stack = b.stack[:len(b.stack)-1]
	}()
	v, err := n.ctor(s)
	if s.aborted != "" {
		panic(s.recovered())
	}
	if err != nil {
		return fmt.Errorf("%s: %w", n.name, err)
	}
	e.value = v
	e.building = false
	return nil
}

// observe calls fn with n's handle. When fn panics, observe recovers the
// panic here, at the call, and panics again with a "graph: " message naming
// n and carrying the value, wrapped when it is an error; the new panic
// passes through any Use and constructor between n and Build unchanged, so
// it is the same for a root and for a node some Use reached. A
// runtime.Goexit, the t.FailNow of a test's observer, is not a panic:
// recover returns nil for it, and it unwinds the Build untouched.
func observe(fn func(Ref), n *node) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		if err, ok := r.(error); ok {
			panic(fmt.Errorf("graph: observer of %q panicked: %w", n.name, err))
		}
		panic(fmt.Sprintf("graph: observer of %q panicked: %v", n.name, r))
	}()
	fn(n.handle)
}

// cyclePath names the cycle that reaching n again closes, from n's place in
// path, the chain of nodes being visited outermost first, back to n.
func cyclePath(path []*node, n *node) string {
	names := make([]string, 0, len(path)+1)
	for _, m := range path[max(slices.Index(path, n), 0):] {
		names = append(names, m.name)
	}
	return strings.Join(append(names, n.name), " -> ")
}
