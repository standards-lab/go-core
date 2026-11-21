package graph

type node struct {
	graph  *Graph
	name   string
	index  int
	ctor   func(*Scope) (any, error)
	after  []Ref
	handle Ref
}

// Node is one node of a [Graph], defined by [Graph.Define], [Graph.Value],
// or [Graph.Fresh]: a name and a constructor of a T. It is a handle, not a
// value; a constructor reaches the value through [Scope.Use], and a program
// through [System.Get].
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
	owner() *Graph
	nodes() []*node
	kind() string
}

func (n *Node[T]) ref() *node {
	if n == nil {
		return nil
	}
	return n.core
}

func (n *Node[T]) owner() *Graph {
	if core := n.ref(); core != nil {
		return core.graph
	}
	return nil
}

func (n *Node[T]) nodes() []*node {
	if core := n.ref(); core != nil {
		return []*node{core}
	}
	return nil
}

func (*Node[T]) kind() string { return "node" }

// Name returns the name the node was defined with, or the empty string for
// a nil handle or a zero-value literal, &Node[T]{}.
func (n *Node[T]) Name() string {
	if core := n.ref(); core != nil {
		return core.name
	}
	return ""
}
