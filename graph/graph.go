package graph

import (
	"fmt"
	"slices"
	"strings"
)

// Graph describes a set of nodes and their constructors. Describing is
// inert: [Graph.Define], [Graph.Value], [Graph.Fresh], [Graph.Group], and
// [Graph.Replace] record nodes, groups, and constructors and run none, and
// only [Graph.Build] constructs anything. The first Build freezes the
// description: Define, Value, Fresh, Group, and Replace panic after it. A Graph is not safe for concurrent use.
type Graph struct {
	nodes []*node
	// groups are the groups [Graph.Group] defined, in definition order. They
	// share the name space with nodes.
	groups []*Group
	// built is set by the first Build, which freezes the description:
	// Define, Value, Fresh, Group, and Replace panic after it.
	built bool
	// observers are called, in the order added, as a Build begins each
	// node.
	observers []func(name string)
}

// New returns an empty Graph.
func New() *Graph {
	return &Graph{}
}

// erase wraps a typed constructor to return its value as any.
func erase[T any](ctor func(*Scope) (T, error)) func(*Scope) (any, error) {
	return func(s *Scope) (any, error) {
		v, err := ctor(s)
		return v, err
	}
}

// Define adds a node named name, constructed by ctor, and returns its
// handle. It runs nothing: ctor runs only when a [Graph.Build] reaches the
// node. The options place the node: [After] orders it after other nodes or
// groups, and [In] makes it a member of a group. Define panics once g has
// run a Build; when name is empty, contains a comma or a newline, or is
// already defined on g as a node or a group; when ctor or an option is
// nil; or when an option names a nil ref, a zero-value literal, or one
// defined on another Graph. It checks everything before it changes g, so
// a Define that panics leaves g as it was.
func (g *Graph) Define[T any](name string, ctor func(*Scope) (T, error), opts ...Option) *Node[T] {
	return define(g, "Define", name, ctor, opts)
}

// Value defines a node named name whose value is v in every Build, and
// returns its handle, a [Node] like any Define returns: a Use target, a
// root, an [After] target, and replaceable. T is v's static type. It takes
// Define's options: [After] orders the node after other nodes or groups,
// and [In] makes it a member of a group. Every Build's value is v itself,
// so a pointer or map v is shared across Builds; a value a run mutates,
// such as a lifecycle participant or a readiness flag, belongs in
// [Graph.Fresh] or [Graph.Define], or every System would start, stop, or
// set the one shared value. Value panics as [Graph.Define] does, the
// message naming Value, and checks everything before it changes g.
func (g *Graph) Value[T any](name string, v T, opts ...Option) *Node[T] {
	return define(g, "Value", name, func(*Scope) (T, error) { return v, nil }, opts)
}

// Fresh defines a node named name whose value is a new zero *T in each
// Build, and returns its handle, a [Node] like any Define returns. It takes
// Define's options, [After] and [In]. Two Builds give two distinct
// pointers, so a value a run mutates, a readiness flag or a gate, starts
// zero in every System. Fresh panics as [Graph.Define] does, the message
// naming Fresh, and checks everything before it changes g.
func (g *Graph) Fresh[T any](name string, opts ...Option) *Node[*T] {
	return define(g, "Fresh", name, func(*Scope) (*T, error) { return new(T), nil }, opts)
}

// define is Define, Value, and Fresh: it checks the name, the constructor,
// and every option before it changes g, then adds the node. call names the
// exported function for a panic's message.
func define[T any](g *Graph, call, name string, ctor func(*Scope) (T, error), opts []Option) *Node[T] {
	g.checkName(call, name)
	if ctor == nil {
		panic(fmt.Sprintf("graph: %s %q with a nil constructor", call, name))
	}
	var set settings
	for _, opt := range opts {
		if opt == nil {
			panic(fmt.Sprintf("graph: %s %q with a nil option", call, name))
		}
		opt.apply(&set)
	}
	// After and In take a handle already defined, so a nil one most often
	// names a node or group whose Define or Group comes later; the message
	// asks.
	op := fmt.Sprintf("%s %q with ", call, name)
	for _, r := range set.after {
		g.resolve(r, op+"After", true)
	}
	for _, grp := range set.in {
		g.resolve(grp, op+"In", true)
	}

	core := &node{graph: g, name: name, index: len(g.nodes), ctor: erase(ctor), after: set.after}
	g.nodes = append(g.nodes, core)
	for _, grp := range set.in {
		if !slices.Contains(grp.members, core) {
			grp.members = append(grp.members, core)
		}
	}
	n := &Node[T]{core: core}
	core.handle = n
	return n
}

// checkName panics, with call naming the exported function, when g has run
// a Build or name cannot name a new node or group: it is empty, contains a
// comma or a newline, which would make a list of names ambiguous, or is
// already taken on g by a node or a group.
func (g *Graph) checkName(call, name string) {
	if g.built {
		panic(fmt.Sprintf("graph: %s of %q after Build", call, name))
	}
	if name == "" {
		panic(fmt.Sprintf("graph: %s with an empty name", call))
	}
	if strings.ContainsAny(name, ",\n") {
		panic(fmt.Sprintf("graph: %s of %q, a name with a comma or a newline", call, name))
	}
	if slices.ContainsFunc(g.nodes, func(n *node) bool { return n.name == name }) ||
		slices.ContainsFunc(g.groups, func(grp *Group) bool { return grp.name == name }) {
		panic(fmt.Sprintf("graph: %s of duplicate name %q", call, name))
	}
}

// resolve returns the nodes r stands for on g: a node itself, or a group's
// members in definition order. It panics when r is nil, a zero-value
// literal, or defined on another Graph; op names the call for the message.
// When later is set, a nil ref's message asks whether the ref's Define, or
// a group's Group, comes later, as it does for an option at Define.
func (g *Graph) resolve(r Ref, op string, later bool) []*node {
	kind := "node"
	var owner *Graph
	var nodes []*node
	if r != nil {
		kind = r.kind()
		owner, nodes = r.expand()
	}
	if owner == nil {
		msg := fmt.Sprintf("graph: %s of a nil %s", op, kind)
		if later {
			maker := "Define"
			if kind == "group" {
				maker = "Group"
			}
			msg += fmt.Sprintf(" (is its %s later?)", maker)
		}
		panic(msg)
	}
	if owner != g {
		panic(fmt.Sprintf("graph: %s of %q, a %s defined on another Graph", op, r.Name(), kind))
	}
	return nodes
}

// Replace swaps n's constructor for ctor, which every later Build runs in
// its place exactly as it would a defined one. It swaps only the
// constructor: n keeps the [After] and [In] it was defined with. Replace
// panics when n is nil or a zero-value literal, once g has run a Build,
// when n was defined on another Graph, or when ctor is nil.
func (g *Graph) Replace[T any](n *Node[T], ctor func(*Scope) (T, error)) {
	core := n.ref()
	if core == nil {
		panic("graph: Replace of a nil node")
	}
	if g.built {
		panic(fmt.Sprintf("graph: Replace of %q after Build", core.name))
	}
	if core.graph != g {
		panic(fmt.Sprintf("graph: Replace of %q, a node defined on another Graph", core.name))
	}
	if ctor == nil {
		panic(fmt.Sprintf("graph: Replace of %q with a nil constructor", core.name))
	}
	core.ctor = erase(ctor)
}

// Observe adds fn, which every later Build calls with a node's name as it
// begins to construct the node, before the constructor runs: each node a
// Build reaches, once, in the order it reaches them depth-first, the one
// whose constructor fails included. It is for tracing what a Build
// constructs, such as a test asserting which nodes a run brought up; fn
// sees only the name and cannot change the Build. Observe panics on a nil
// fn.
func (g *Graph) Observe(fn func(name string)) {
	if fn == nil {
		panic("graph: Observe with a nil function")
	}
	g.observers = append(g.observers, fn)
}
