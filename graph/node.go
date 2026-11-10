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
	// [Drop], a [Dependency], and an observer carry, so a caller compares
	// it with what Define returned.
	handle Ref
}

// Node is one node of a [Graph], defined by [Graph.Define], [Graph.Value],
// or [Graph.Fresh]: a name and a constructor of a T. It is a handle, not a value; a constructor reaches
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
	// owner returns the Graph the ref was defined on, nil for a nil
	// handle or a zero-value literal.
	owner() *Graph
	// nodes returns the nodes the ref stands for. A group's members change
	// under its Graph's lock until the first Build, so a caller calls
	// nodes only on a ref whose owner it has checked is the Graph it holds,
	// one that has run a Build or whose lock it holds.
	nodes() []*node
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

// owner treats a zero-value literal, &Node[T]{}, as a nil handle: it has
// no core, since only Define, Value, and Fresh make one.
func (n *Node[T]) owner() *Graph {
	if core := n.ref(); core != nil {
		return core.graph
	}
	return nil
}

// nodes returns n's core alone, nil for a nil handle or a zero-value
// literal.
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
