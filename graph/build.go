package graph

import (
	"fmt"
	"slices"
	"strings"
)

type entry struct {
	node     *node
	building bool
	value    any
	layer    int
	uses     []*node
}

// See docs/graph-design.md, "Implementation invariants".
type build struct {
	graph     *Graph
	observers []func(Ref)
	entries   map[*node]*entry
	stack     []*node
}

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

func cyclePath(path []*node, n *node) string {
	names := make([]string, 0, len(path)+1)
	for _, m := range path[max(slices.Index(path, n), 0):] {
		names = append(names, m.name)
	}
	return strings.Join(append(names, n.name), " -> ")
}
