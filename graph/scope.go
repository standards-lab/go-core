package graph

import (
	"fmt"
	"slices"
)

// Scope is what a constructor receives: the means to reach its
// dependencies. It has [Scope.Use] only: a node's order-only edges are
// declared at Define, with [After]. It is valid only while the constructor
// runs; Use panics once the constructor has returned.
type Scope struct {
	build   *build
	entry   *entry
	done    bool
	aborted string
}

func (s *Scope) check(op string) {
	if s.done {
		panic(fmt.Sprintf("graph: %s called after the constructor of %q returned", op, s.entry.node.name))
	}
}

// Use returns n's value, building n first when this Build has not, and
// records that the node under construction depends on n. When n's
// constructor fails, or a dependency's does as n is built, Use does not
// return: it aborts the calling constructor by panicking, and Build
// recovers the panic and reports the failing node's error. A constructor
// must not recover that panic: when one returns after a Use it made
// aborted, Build panics naming it. Use panics when n was defined on another
// Graph, when n is under construction (a cycle, named by its path), after a
// Use in the same constructor aborted, or after the constructor has
// returned.
func (s *Scope) Use[T any](n *Node[T]) T {
	s.check("Use")
	if s.aborted != "" {
		panic(s.recovered())
	}
	core := s.build.graph.resolve(n, "Use", false)[0]
	returned := false
	defer func() {
		if !returned {
			s.aborted = core.name
		}
	}()
	if err := s.build.construct(core); err != nil {
		panic(abort{err: err})
	}
	returned = true
	if !slices.Contains(s.entry.uses, core) {
		s.entry.uses = append(s.entry.uses, core)
	}
	v, _ := s.build.entries[core].value.(T)
	return v
}

func (s *Scope) recovered() string {
	return fmt.Sprintf(
		"graph: the constructor of %q carried on after its Use of %q aborted; a constructor must not recover",
		s.entry.node.name, s.aborted,
	)
}
