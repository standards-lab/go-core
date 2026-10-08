package lifecycle

import (
	"fmt"
	"sync/atomic"
)

// Readiness reports a [Coordinator]'s readiness to the nodes that depend on
// it, such as a health handler. Define it as a graph node whose value is a
// new *Readiness; [New] binds every *Readiness in its System to the
// Coordinator it returns. The zero value is unbound, and an unbound
// Readiness is not ready and has no checks.
type Readiness struct {
	c atomic.Pointer[Coordinator]
}

// Ready reports the bound Coordinator's [Coordinator.Ready], or false when
// r is unbound.
func (r *Readiness) Ready() bool {
	c := r.c.Load()
	return c != nil && c.Ready()
}

// Checks returns the bound Coordinator's [Coordinator.Checks], or nil when r
// is unbound.
func (r *Readiness) Checks() []Check {
	c := r.c.Load()
	if c == nil {
		return nil
	}
	return c.Checks()
}

// bind binds r, the value of the node name, to c. A Readiness binds once:
// binding it to a second Coordinator panics, as a wiring mistake.
func (r *Readiness) bind(name string, c *Coordinator) {
	if !r.c.CompareAndSwap(nil, c) && r.c.Load() != c {
		panic(fmt.Sprintf("lifecycle: New: Readiness %q is bound to another Coordinator", name))
	}
}
