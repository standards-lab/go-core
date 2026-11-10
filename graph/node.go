package graph

// node is the untyped core of a [Node]: what a Build needs to construct it
// without knowing its type.
type node struct {
	graph *Graph
	name  string
	// index is the node's position in definition order, the order of the
	// nodes within one layer.
	index int
	// ctor is the typed constructor, wrapped to return its value as any.
	ctor func(*Scope) (any, error)
	// after are the order-only targets [After] named at Define, nodes or
	// groups, as declared. A Build expands a group to its members then,
	// and drops the targets not in the System when it computes the layers.
	after []Ref
	// handle is the *Node[T] that defined the node, the one handle a
	// [Drop] carries, so a caller compares it with what Define returned.
	handle Ref
}

// Node is one node of a [Graph], defined by [Graph.Define]: a name and a
// constructor of a T. It is a handle, not a value; a constructor reaches
// the value through [Scope.Use], and a program through [System.Get].
type Node[T any] struct {
	core *node
}

// Ref is any [Node], whatever its type, or a [Group]: what [Graph.Build]
// takes as roots and [After] as an ordering target, and what names a node
// or group without its type. A group stands for its members wherever a Ref
// is taken. It is sealed; only *Node[T] and *Group implement it.
type Ref interface {
	// Name returns the name the node or group was defined with.
	Name() string
	// expand returns the Graph the ref was defined on, nil for a nil
	// handle or a zero-value literal, and the nodes it stands for.
	expand() (*Graph, []*node)
	// kind names what the ref is, "node" or "group", for a panic message.
	kind() string
}

// ref returns n's core, nil for a nil handle or a zero-value literal.
func (n *Node[T]) ref() *node {
	if n == nil {
		return nil
	}
	return n.core
}

// expand treats a zero-value literal, &Node[T]{}, as a nil handle: it has
// no core, since only Define makes one.
func (n *Node[T]) expand() (*Graph, []*node) {
	if core := n.ref(); core != nil {
		return core.graph, []*node{core}
	}
	return nil, nil
}

func (*Node[T]) kind() string { return "node" }

// Name returns the name the node was defined with.
func (n *Node[T]) Name() string { return n.core.name }
