// Package graph is a typed dependency graph. A [Graph] describes [Node]
// values inertly, each a name and a constructor; [Graph.Build] constructs
// what its roots reach and returns it as a [System] of [Dependency] values
// in computed layers. The package knows nothing of lifecycles: a consumer
// that runs a System, starting its layers in order and stopping them in
// reverse, reads each node's part from its value, as the lifecycle package's
// Coordinator does.
//
// The package exports:
//
//   - [Graph], the description of the nodes, and [New], which returns an
//     empty one
//   - [Graph.Define], which adds a node with its constructor and its
//     options, and runs nothing
//   - [Graph.Value], which adds a node whose value is a fixed value in
//     every Build, and [Graph.Fresh], which adds one whose value is a new
//     zero pointer in each Build; both take Define's options
//   - [Option], which places a node at Define, Value, or Fresh: [After],
//     which orders it after nodes or groups without their values, and
//     [In], which makes it a member of a group
//   - [Graph.Group], which defines a [Group], a named set of nodes, and
//     [Group.Name], its name
//   - [Graph.Replace], which swaps a node's constructor before Build, for a
//     test's substitute, keeping its options
//   - [Graph.Observe], which adds a function every later Build calls with
//     each node's handle as it begins to construct it, for tracing what a
//     Build reaches
//   - [Graph.Build], which constructs what the roots reach into a System
//   - [Node], a typed handle on one node, and [Node.Name], its name
//   - [Ref], any Node whatever its type or a Group, sealed to the two, and
//     its Name
//   - [Scope], what a constructor receives
//   - [Scope.Use], which returns a dependency's value, building it if
//     needed
//   - [System], what one Build constructed
//   - [System.Get], which returns a node's value
//   - [System.Layers], which returns the built nodes in layers, and
//     [System.Dependencies], which returns them flattened, in layer order
//   - [System.LayerOf], which returns a node's layer, and [System.Has],
//     which reports whether a node is in the System
//   - [System.String], which prints the layer table, one line per layer
//   - [System.Dropped], which returns each [After] that ordered nothing, as
//     a [Drop]
//   - [Dependency], one built node: its name, its value, and its handle
//
// # Discovery
//
// A constructor declares its dependencies by using them: [Scope.Use] builds
// the dependency if this Build has not, and records the edge, so the edges
// are discovered as the constructors run, depth-first from the roots, and
// can depend on what the constructors see. Each node is built at most once
// per Build, so a node shared by several dependents is constructed once and
// they all receive its value. A node that is neither a root nor reached by
// a Use is never constructed, so a Build of a subset of the roots brings up
// only that subset's dependencies. Each Build is independent and constructs
// fresh values.
//
// # Placement
//
// A node's place is declared at Define, not in its constructor: [After]
// adds an order-only edge, which passes no value and builds nothing, and
// holds only when its target is in the System by some other path, as a
// root or a node some Use reached. [System.Dropped] lists each After that
// ordered nothing, so a test of the full root set can assert there is
// none. [In] makes the node a member of a [Group], defined by
// [Graph.Group]; a group stands for its members wherever a [Ref] is taken,
// so a Build given a group as a root builds its members in definition
// order, and an After of a group orders after each member in the System.
// An After or In names a handle defined earlier, so the nodes and groups
// are defined in dependency order.
//
// # Layers
//
// The layers are the longest-path layering of the discovered Use and After
// edges, as [System.Layers] states: every node's dependencies sit in lower
// layers, so the nodes of one layer can start together once the layers
// below them have.
//
// # Introspection
//
// A System and an observer name a node by the handle [Graph.Define],
// [Graph.Value], or [Graph.Fresh] returned: [Dependency] carries it as its
// Ref, an observer receives it, and [System.LayerOf] and [System.Has] take
// it, so a test compares handles rather than names. [System.String] prints
// the layer table for a failing test's message; a name cannot contain a
// comma or a newline, so the table is unambiguous unquoted. An observer is
// never removed: it applies to every later Build of its Graph. The
// graphtest package checks a program's struct of handles against the names
// its nodes and groups were defined with.
//
// # Concurrency
//
// A [Graph] is safe for concurrent use. Builds of one Graph may run
// concurrently, each constructing its own values into its own System. A
// describing call, [Graph.Define], [Graph.Value], [Graph.Fresh],
// [Graph.Group], or [Graph.Replace], may run beside a Build: one that runs
// before the first Build freezes the Graph completes, and one after
// panics, as it does without concurrency. [Graph.Observe] may run beside a
// Build, or from an observer, and its function applies from the next Build
// on, not to one already running. Concurrent Builds call the same
// observers concurrently, so an observer shared by them must be safe for
// concurrent use. A constructor runs in its Build's goroutine, and runs
// concurrently with the constructors of other Builds.
//
// # Errors and panics
//
// A constructor's error fails the Build, labelled with its node's name and
// wrapping the error; a failing dependency aborts every constructor whose
// Use reached it, and Build reports the dependency's error.
//
// [Scope.Use] returns the dependency's value itself, so the abort unwinds
// by panic, which Build recovers. A constructor must not recover it: one
// that returns, with a value or an error, after a Use it made aborted would
// lose the failure, so Build panics naming the constructor and the Use.
//
// An observer that panics makes Build panic with a "graph: " message that
// names the node it was called with and carries the panic's value, wrapped
// when it is an error, whether the node is a root or one a Use reached.
//
// Wiring mistakes panic with a "graph: " message, as each symbol's
// documentation states: a dependency cycle, a node used on a Graph it was
// not defined on, a Scope used after its constructor returned, a recovered
// abort, a Define, Value, Fresh, Group, or Replace after Build, a Get or
// LayerOf of a node not in the System, a Get, LayerOf, or Has of a nil
// handle, a LayerOf or Has of a group, a nil constructor, option, or
// observer, an After or In naming a nil handle, a zero-value literal, or
// one from another Graph, and a name that is empty, contains a comma or a
// newline, or is already taken by a node or a group, which share one name
// space. Define, Value, Fresh, and Group check everything before they
// change the Graph, so one that panics leaves it as it was. Build checks
// its roots, for a nil node or group or one defined on another Graph,
// before it constructs any of them, so a mistake in any root panics before
// a constructor runs.
package graph
