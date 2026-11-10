package graph

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

// System is what one [Graph.Build] constructed: every node the roots
// reached, with its value, in computed layers.
type System struct {
	entries map[*node]*entry
	layers  [][]*entry
	dropped []Drop
}

// Drop is one [After] that orders nothing in a [System]: Node declared it,
// and After is its target as declared, a node not in the System or a
// [Group] with no member in it. Both are the handles [Graph.Define],
// [Graph.Value], [Graph.Fresh], or [Graph.Group] returned, so
// d.Node == graph.Ref(handle) holds.
type Drop struct {
	Node  Ref
	After Ref
}

// Dependency is one built node: its name, its constructed value, and its
// handle. A lifecycle reads what the node takes part in from the value's
// methods.
type Dependency struct {
	Name  string
	Value any
	// Ref is the handle [Graph.Define], [Graph.Value], or [Graph.Fresh]
	// returned for the node, so dep.Ref == graph.Ref(handle) holds and a
	// test compares handles rather than names.
	Ref Ref
}

// Get returns n's value in s. It panics when n is nil or a zero-value
// literal, or not in s: a node the roots did not reach, or one defined on
// another Graph.
func (s *System) Get[T any](n *Node[T]) T {
	core := n.ref()
	if core == nil {
		panic("graph: Get of a nil node")
	}
	e, ok := s.entries[core]
	if !ok {
		panic(fmt.Sprintf("graph: Get of %q, a node not in the System", core.name))
	}
	v, _ := e.value.(T)
	return v
}

// Layers returns the System's nodes in layers, layer 0 first: a node
// with no dependencies is in layer 0, and any other node is one layer
// above the highest of its Use and [After] dependencies in the System, so
// every node's dependencies sit in layers below it. Within a layer, nodes
// are in definition order. Each call returns a fresh copy.
func (s *System) Layers() [][]Dependency {
	out := make([][]Dependency, len(s.layers))
	for i, layer := range s.layers {
		out[i] = make([]Dependency, len(layer))
		for j, e := range layer {
			out[i][j] = e.dependency()
		}
	}
	return out
}

// Dependencies returns every node in s, in layer order, layer 0 first, and
// then in definition order within a layer: [System.Layers] flattened. Each
// call returns a fresh copy; nil for an empty System.
func (s *System) Dependencies() []Dependency {
	var out []Dependency
	for _, layer := range s.layers {
		for _, e := range layer {
			out = append(out, e.dependency())
		}
	}
	return out
}

// dependency returns e as the Dependency a caller sees.
func (e *entry) dependency() Dependency {
	return Dependency{Name: e.node.name, Value: e.value, Ref: e.node.handle}
}

// LayerOf returns the layer of ref's node in s, numbered as
// [System.Layers] numbers them, layer 0 first. It panics when ref is nil,
// a nil node or group, or a zero-value literal of one; when ref is a
// [Group], which has no layer of its own; or when the node is not in s,
// one the roots did not reach or one defined on another Graph.
// [System.Has] reports whether it is.
func (s *System) LayerOf(ref Ref) int {
	core := s.lookup(ref, "LayerOf")
	e, ok := s.entries[core]
	if !ok {
		panic(fmt.Sprintf("graph: LayerOf of %q, a node not in the System", core.name))
	}
	return e.layer
}

// Has reports whether ref's node is in s: a node the roots reached. A node
// defined on another Graph is not. Has panics when ref is nil, a nil node
// or group, or a zero-value literal of one, or a [Group], which is not a
// node of its own.
func (s *System) Has(ref Ref) bool {
	_, ok := s.entries[s.lookup(ref, "Has")]
	return ok
}

// lookup returns the node ref names, for LayerOf and Has, which op names.
// It panics when ref is nil, a zero-value literal, or a group.
func (s *System) lookup(ref Ref, op string) *node {
	if ref == nil {
		panic(fmt.Sprintf("graph: %s of a nil node", op))
	}
	if ref.owner() == nil {
		panic(fmt.Sprintf("graph: %s of a nil %s", op, ref.kind()))
	}
	// A group's members may be changing on a Graph that has not built, so
	// lookup rejects a group before it reads any nodes.
	if _, ok := ref.(*Group); ok {
		panic(fmt.Sprintf("graph: %s of %q, a group, which is not a node", op, ref.Name()))
	}
	return ref.nodes()[0]
}

// String returns the layer table, one line per layer, layer 0 first: the
// layer's number, a colon, and its nodes' names in definition order,
// separated by commas, as in
//
//	0: config, logger
//	1: database
//	2: server
//
// with no newline after the last line. A name cannot contain a comma or a
// newline, so the table is unambiguous unquoted. It is for a failing
// test's message. An empty System returns the empty string.
func (s *System) String() string {
	var b strings.Builder
	for i, layer := range s.layers {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "%d:", i)
		for j, e := range layer {
			if j > 0 {
				b.WriteByte(',')
			}
			b.WriteByte(' ')
			b.WriteString(e.node.name)
		}
	}
	return b.String()
}

// Dropped returns each [After] that orders nothing in s, which Build drops
// silently so that a Build of a subset of the roots works: a node target
// not in s, or a group target with no member in s. A group with at least
// one member in s orders after that member and is not dropped, even when
// other members are absent. Only the Afters of nodes in s are considered.
// The drops are in the declaring nodes' definition order, then in the
// order of each node's After refs. A Build of the full root set drops
// nothing, so a root's layer test asserts len(sys.Dropped()) == 0 to make
// a forgotten root, or a forgotten [In], fail instead of silently losing
// an order. Each call returns a fresh copy; nil when nothing is dropped.
func (s *System) Dropped() []Drop {
	return slices.Clone(s.dropped)
}

// system computes the layers of b's entries by longest path and returns
// them as a System. The Use edges are acyclic, since construct panics on a
// cycle, but an After edge can close one; that panics with its path here,
// which names a group the cycle passes through between its member and the
// node ordered after it.
func (b *build) system() *System {
	level := make(map[*node]int, len(b.entries))
	// path is the chain of names being visited, outermost first, groups
	// included; onPath holds each visiting node's place in it.
	var path []string
	onPath := make(map[*node]int)
	var layer func(n *node) int
	layer = func(n *node) int {
		if l, ok := level[n]; ok {
			return l
		}
		if i, ok := onPath[n]; ok {
			panic("graph: dependency cycle: " + strings.Join(append(slices.Clone(path[i:]), n.name), " -> "))
		}
		onPath[n] = len(path)
		path = append(path, n.name)
		e := b.entries[n]
		l := 0
		for _, d := range e.uses {
			l = max(l, layer(d)+1)
		}
		for _, r := range n.after {
			grp, isGroup := r.(*Group)
			if isGroup {
				path = append(path, grp.name)
			}
			for _, d := range r.nodes() {
				if _, ok := b.entries[d]; ok {
					l = max(l, layer(d)+1)
				}
			}
			if isGroup {
				path = path[:len(path)-1]
			}
		}
		path = path[:len(path)-1]
		delete(onPath, n)
		level[n] = l
		return l
	}

	ordered := make([]*entry, 0, len(b.entries))
	for _, e := range b.entries {
		ordered = append(ordered, e)
	}
	slices.SortFunc(ordered, func(a, b *entry) int { return cmp.Compare(a.node.index, b.node.index) })

	var layers [][]*entry
	for _, e := range ordered {
		l := layer(e.node)
		e.layer = l
		for len(layers) <= l {
			layers = append(layers, nil)
		}
		layers[l] = append(layers[l], e)
	}
	return &System{entries: b.entries, layers: layers, dropped: b.dropped(ordered)}
}

// dropped returns the Afters of the ordered entries whose targets have no
// node among b's entries, in the entries' order and then each node's After
// order.
func (b *build) dropped(ordered []*entry) []Drop {
	var drops []Drop
	for _, e := range ordered {
		for _, r := range e.node.after {
			if !slices.ContainsFunc(r.nodes(), func(d *node) bool { _, ok := b.entries[d]; return ok }) {
				drops = append(drops, Drop{Node: e.node.handle, After: r})
			}
		}
	}
	return drops
}
