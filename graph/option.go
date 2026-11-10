package graph

// Option places a node at [Graph.Define], [Graph.Value], or [Graph.Fresh]:
// [After] and [In] return one. It is sealed; only this package's functions
// return an Option.
type Option interface {
	apply(*settings)
}

// settings collects a node's options, which Define, Value, and Fresh
// check before they change the Graph.
type settings struct {
	after []Ref
	in    []*Group
}

type afterOption []Ref

func (o afterOption) apply(s *settings) { s.after = append(s.after, o...) }

type inOption struct{ group *Group }

func (o inOption) apply(s *settings) { s.in = append(s.in, o.group) }

// After orders the node after each ref: it starts after the ref and stops
// before it. A [Group] stands for each of its members. After passes no
// value and never builds a ref; the edge holds only when the ref's node is
// in the System, as a root or a node some Use reached, and is dropped
// otherwise, as [System.Dropped] reports, so a node only ordered against
// is named as a root. Each ref must be defined before the node that names
// it: Define, Value, and Fresh panic when a ref is nil, a zero-value
// literal, or defined on another Graph.
func After(refs ...Ref) Option {
	return afterOption(refs)
}

// In makes the node a member of group, which a Build given the group, as a
// root or an [After] target, expands to. Define, Value, and Fresh panic
// when group is nil, a zero-value literal, or defined on another Graph.
func In(group *Group) Option {
	return inOption{group: group}
}
