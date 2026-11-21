# Graph design

This page explains why the `graph` and `lifecycle` API is shaped as it is: what each part of it is
for, the reasons behind its decisions, the alternatives it rejected, and the invariants its
unexported code keeps. The package documentation in `graph/doc.go`, `graph/graphtest/doc.go`, and
`lifecycle/doc.go` states what the API does, and the CHANGELOG lists every change.

## What each part of the API is for

Each part of the API spares go-core's consumers a workaround they would otherwise repeat in every
composition root.

1. **`graphtest.CheckNames` keeps each name written once.** A node has a struct field and a
   `Define` string, and the check ties the two, so no comment on the field records the string for
   the reader. Deriving the name from the type cannot replace the string, since several nodes can
   share one type, such as `*graph.Node[string]`.
2. **Constructors are methods on the root's `*Nodes`,** passed to Define as method values. They
   reach every handle through the receiver, so no wrapper of the form
   `newX(...) func(*graph.Scope) (T, error)` exists only to close over handles and write `T` three
   times. This is a convention, not go-core code.
3. **`Graph.Value` and `Graph.Fresh` define fixed and fresh values** without a closure. A fresh
   value stays fresh per Build: a `lifecycle.Readiness` binds to one Coordinator only, and a gate or
   a nudge is per-run state.
4. **`After`, `In`, and `Group` make layer placement visible** at Define, where no `Scope` call
   inside a constructor body can set it. A node nothing Uses joins the Build through a group or as
   a named root, a test checks placement with `LayerOf` and `Dependencies` rather than by walking
   `Layers()` and matching names, and the package docs state that definition order also orders
   readiness checks within a layer.
5. **An After orders without building.** Unlike `s.Use`, an After never brings its target into the
   Build (see The decisions). A node only ordered against is named as a root, and
   `System.Dropped` lists an After that ordered nothing.
6. **The composition root exports its root list,** and its test builds that list, so no fixture
   copies `Run`'s roots and observes a real `Run` to keep the copy honest. This is a convention,
   not go-core code.
7. **The runners live outside go-core.** The Build, Get, `lifecycle.New`, `OnReady`, `Run`, and
   logging every root would repeat are answered outside go-core, by the application SDKs' runners.
8. **`lifecycle.Config` arrives one way:** as a node the application SDKs' runners take, so no
   root reads it two ways.
9. **Introspection works with handles throughout:** `Dependency.Ref`, `Observe(func(Ref))`,
   `LayerOf`, `Has`, `Dependencies`, and `String`, so a test holding a handle compares handles,
   not names.

Use discovery is what lets a CLI run build only what its command declares: a CLI runner builds the
leaf's declared nodes plus the lifecycle config, so help or a usage error builds nothing. `s.Use`
stays the only way a constructor adds a dependency, which is why order and groups, which carry no
value, are declared at Define rather than as a second edge in the Scope.

## The decisions

- **An After never builds its target** (point 5). An After that built its target would let
  an order-only declaration pull a node, and its whole subtree, into a CLI run that never Uses it,
  undoing the build-only-what-is-declared property. A node only ordered against is named as a root
  instead, and a full-roots test asserts `System.Dropped` is empty. An After naming an empty group
  is dropped too, even in a full-roots Build, so the same assertion catches a forgotten `In`.
- **Groups share the node name space** (point 4). A group is a `Ref`, so a Build and an After
  take it wherever they take a node; one name space keeps the layer table and the error labels
  unambiguous. `Group` exports only `Name`: no consumer needed its members, and a group's members
  are its `In` declarations, visible at their Define lines.
- **Replace swaps only the constructor.** A test's substitute stands in for a node's value, not
  its place; keeping After and In means the substitute lands where the real node would. Replace
  cannot remove an edge or a membership, and no test needed it.
- **Fresh is a method** (point 3). The design review proposed a function,
  `graph.Fresh[T](g, name)`, because Go had no method type parameters; Go 1.27 has them, so it
  sits beside `g.Value`. Both take Define's options, so a value can join a group. Because
  `g.Value` takes `T` from the argument's static type, a `*Nodes` method reaches a plain value
  such as the logger's writer as `g.Value("writer", w)` with `w io.Writer`.
- **Names stay explicit, and a fixture checks them** (point 1). The string labels errors and
  names readiness checks, and type-derived names fail on several nodes of one type. The check
  compares the name with its spaces removed, so "meterprovider" and "meter provider" both match a
  `MeterProvider` field and a multi-word name may keep its spaces. `graphtest` holds the check so
  nothing in `graph` exists for the tests' sake.
- **A name refuses a comma or a newline, and String stays unquoted** (point 9).
  `System.String` is for a failing test's message, where quoting every name costs legibility on
  every line; refusing the table's two separators keeps the unquoted table unambiguous.
- **LayerOf panics, and Has answers membership** (point 9). `LayerOf` as `(int, bool)` would
  make every layer assertion handle a miss that is the test's own mistake. A panicking `int` reads
  as one expression in a test, and `Has(ref) bool` answers the question when it is the question.
- **Dependencies lists the built nodes flat** (point 4). Consumers' tests flattened
  `Layers()` by hand to check every node; `Dependencies` is that flattening, so it equals the
  layers' concatenation.
- **Observers persist.** Consumers observe once, from a test's fixture, so a removal handle would
  be API no one calls; the docs state the persistence instead.
- **A panicking observer is labelled.** An observer's panic that escaped Build bare would name no
  node.
- **A Graph is safe for concurrent use, under one mutex.** The spike that proved the design found
  concurrent Builds of one Graph racing. Build holds the mutex only to set the built flag and copy
  the observers, so Builds run concurrently and constructors never run under the lock.
- **A Monitored failure names its node.** A bare `run: <error>` would not say which value failed.
  A channel registered with `Monitor` has no node, so its failure keeps that form.
- **Methods on `*Nodes` are the only constructor form** (point 2). Arity adapters such as
  `graph.Func2(newDatabase, n.Config, n.Logger)` were tried and did not ship: the Define-line
  changes already made the graph legible and fixed the reliability findings, and a second form
  costs one adapter per arity, a projection node or wrapper for each constructor that takes part
  of a config, and no-error variants.

## Rejected alternatives

- **uber-go/fx and dig, its container.** fx runs components through `OnStart`/`OnStop` hooks
  registered on a lifecycle, the model the lifecycle principle retired in favor of participation
  by the value's own methods; dig resolves the graph by run-time reflection over constructor
  signatures, so a wiring mistake surfaces at run time rather than in a typed handle.
- **samber/do.** A typed container with lazy services, but with no layered, concurrent startup: it
  cannot start a layer's subsystems together and shut down in reverse layers, which is the
  lifecycle's core semantics.
- **google/wire.** Compile-time code generation that fits the build-once model, but the repository
  was archived in August 2025.

None of them meets the sourcing rule either: the Go Elemental standard writes a capability
in-house when it is generic to its layer and has no specification or security surface, and
go-core depends on the standard library alone.

## Implementation invariants

The unexported code of `graph` and `lifecycle` keeps these invariants, which no exported doc
states.

- **The freeze makes the description immutable.** Once a Build sets the built flag, every
  describing call panics before it changes the Graph, so the nodes, their constructors and Afters,
  and the groups' members are read without the lock, beside other Builds and beside an observer
  that calls Observe. An Observe that runs during a Build applies from the next Build on. Every
  describing call defers its unlock, so a check that panics leaves the Graph usable.
- **A Build resolves every root before any constructor runs,** so a nil or foreign root anywhere
  in the list panics first. A group expands to its members then, complete since the freeze.
- **A constructor's error unwinds as a panic.** The labelled error travels as an internal abort
  value out of every dependent whose Use reached it, up to Build, which recovers it and returns
  the error. A Scope records the node whose Use did not return; a constructor that recovers the
  abort and returns anyway would lose the failure, so the Build panics naming it.
- **An observer's panic is relabelled at the call.** It is recovered where the observer runs and
  raised again with a `graph: ` message naming the node, so it reads the same for a root and for
  a node some Use reached. A `runtime.Goexit`, a test observer's `t.FailNow`, is not a panic and
  unwinds the Build untouched.
- **Layers are longest paths.** A node's layer is one above the highest of its Uses and in-System
  Afters, not its last Use. Use edges are acyclic, since reaching a node under construction panics
  with the cycle's path; an After edge can still close a cycle, which panics when the layers are
  computed, naming any group the cycle passes through. An After whose target has no node in the
  System is dropped, in definition order and then each node's After order.
- **A lifecycle layer is one engine phase.** Its participants start concurrently, and the phase
  is recorded for unwinding whether or not their starts succeed, so a participant whose start
  failed is still stopped. The first failure cancels the run context so the rest of the layer can
  stop early, and no higher layer starts; a cancellation consequent on a recorded failure is not
  recorded as a second one, and the check and the record hold one lock.
- **Shutdown has its whole budget.** The unwind runs every recorded phase, the last first, under
  one context derived from `context.Background` and bounded by the shutdown timeout, whatever
  became of the run context. A phase cut short contributes what it recorded by the deadline.
- **The first runtime failure is the run's cause.** A Monitored value's channel is read once
  startup completes; recording a failure and cancelling the run hold one lock, so the failure
  reported is the one that ended the run.
