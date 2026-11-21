# Graph design

This page explains why the `graph` and `lifecycle` API is shaped as it is since v0.11.0: the pain
points it answers, the behavior it kept, the reasons behind its decisions, the alternatives it
rejected, and the invariants its unexported code keeps. The package documentation in `graph/doc.go`, `graph/graphtest/doc.go`, and
`lifecycle/doc.go` states what the API does, and the CHANGELOG's v0.11.0 section lists every
change.

## Pain points

v0.10.0's packages behaved correctly but read badly, and its consumers (the libraries, go-cli-sdk,
go-web-sdk, both templates, go-web-service, and the spikes) repeated the same workarounds. Every
`s.Use` in them was unconditional. Each point names what answers it.

1. **Each name was written three times:** a node's struct field, its `Define` string, and a
   comment on the field recording the string for the reader. Deriving the name from the type
   cannot replace them, since one consumer has six nodes of type `*graph.Node[string]`. Answered
   by `graphtest.CheckNames`, which replaces the comments.
2. **Constructors returned constructors.** Wrappers of the form
   `newX(...) func(*graph.Scope) (T, error)` existed only to close over the node handles, wrote
   `T` three times, and took the whole set of handles, single handles, or none, for no reason.
   Answered by a convention, not by go-core: constructors are methods on the root's `*Nodes`,
   passed to Define as method values.
3. **Fixed and fresh values needed closures.** A node whose value is fixed, or a fresh zero value,
   still took a four-line closure. The fresh ones must stay fresh per Build: a
   `lifecycle.Readiness` binds to one Coordinator only, and a gate or a nudge is per-run state.
   Answered by `Graph.Value` and `Graph.Fresh`.
4. **Layer placement was invisible.** Where a node landed was set by `s.After` calls scattered
   through constructor bodies, nodes nothing Used were added as extra Build roots, tests checked
   placement by walking `Layers()` and matching names, and only a comment stated that definition
   order also orders readiness checks within a layer. Answered by `After`, `In`, and `Group` at
   Define, and the removal of `Scope.After`.
5. **Use and After were asymmetric.** `s.After` never brought a node into the Build, so a
   consumer that only ordered against a node called `s.Use` and discarded the value to bring it
   in. Kept on purpose (see The decisions), and made visible: such a node is named as a root, and
   `System.Dropped` lists an After that ordered nothing.
6. **Build roots were repeated in tests.** A test fixture copied `Run`'s root list and kept the
   copy honest by observing a real `Run`. Answered by a convention: the composition root exports
   its root list, and the test builds that list.
7. **There was no web runner.** About thirty lines of Build, Get, `lifecycle.New`, `OnReady`,
   `Run`, and logging were identical across web roots, where the CLI side had `cli.Run`. Answered
   outside go-core, by go-web-sdk's `web.Service`.
8. **`lifecycle.Config` arrived two ways.** A web root read it off its config node, and every CLI
   defined a lifecycle-config node with the same constructor. Answered outside go-core:
   `web.Service` takes it as a node, as `cli.Program` does.
9. **Introspection gave only names.** v0.10.0's `Observe` passed a string and `Dependency` was
   `{Name, Value}`, so a test holding a handle compared names. Answered by handles throughout:
   `Dependency.Ref`, `Observe(func(Ref))`, `LayerOf`, `Has`, `Dependencies`, and `String`.

## Behavior that did not change

v0.11.0 changed how a composition root writes its graph, not what the graph and lifecycle do: edge
discovery through `s.Use`, build-once, longest-path layers, the participation interfaces, and the
startup, readiness, and shutdown semantics are as the package docs state them, and the ported
tests hold them. The one observable change is a Monitored failure's message, which now names its
node. Every other break fails at compile time or panics at Define.

Use discovery is what lets a CLI run build only what its command declares: go-cli-sdk builds the
leaf's declared nodes plus the lifecycle config, so help or a usage error builds nothing. `s.Use`
stays the only way a constructor adds a dependency, which is why order and groups, which carry no
value, moved to Define rather than gaining a second edge in the Scope.

## The decisions

- **An After never builds its target** (pain point 5). An After that built its target would let
  an order-only declaration pull a node, and its whole subtree, into a CLI run that never Uses it,
  undoing the build-only-what-is-declared property. A node only ordered against is named as a root
  instead, and a full-roots test asserts `System.Dropped` is empty. An After naming an empty group
  is dropped too, even in a full-roots Build, so the same assertion catches a forgotten `In`.
- **Groups share the node name space** (pain point 4). A group is a `Ref`, so a Build and an After
  take it wherever they take a node; one name space keeps the layer table and the error labels
  unambiguous. `Group` exports only `Name`: no consumer needed its members, and a group's members
  are its `In` declarations, visible at their Define lines.
- **Replace swaps only the constructor.** A test's substitute stands in for a node's value, not
  its place; keeping After and In means the substitute lands where the real node would. Replace
  cannot remove an edge or a membership, and no test needed it.
- **Fresh is a method** (pain point 3). The design review proposed a function,
  `graph.Fresh[T](g, name)`, because Go had no method type parameters; Go 1.27 has them, so it
  sits beside `g.Value`. Both take Define's options, so a value can join a group. Because
  `g.Value` takes `T` from the argument's static type, a `*Nodes` method reaches a plain value
  such as the logger's writer as `g.Value("writer", w)` with `w io.Writer`.
- **Names stay explicit, and a fixture checks them** (pain point 1). The string labels errors and
  names readiness checks, and type-derived names fail on several nodes of one type. The check
  compares the name with its spaces removed, so "meterprovider" and "meter provider" both match a
  `MeterProvider` field and a multi-word name may keep its spaces. `graphtest` holds the check so
  nothing in `graph` exists for the tests' sake.
- **A name refuses a comma or a newline, and String stays unquoted** (pain point 9).
  `System.String` is for a failing test's message, where quoting every name costs legibility on
  every line; refusing the table's two separators keeps the unquoted table unambiguous.
- **LayerOf panics, and Has answers membership** (pain point 9). `LayerOf` as `(int, bool)` would
  make every layer assertion handle a miss that is the test's own mistake. A panicking `int` reads
  as one expression in a test, and `Has(ref) bool` answers the question when it is the question.
- **Dependencies lists the built nodes flat** (pain point 4). Consumers' tests flattened
  `Layers()` by hand to check every node; `Dependencies` is that flattening, so it equals the
  layers' concatenation.
- **Observers persist.** Consumers observe once, from a test's fixture, so a removal handle would
  be API no one calls; the docs state the persistence instead.
- **A panicking observer is labelled.** In v0.10.0 an observer's panic escaped Build bare, with no
  node named.
- **A Graph is safe for concurrent use, under one mutex.** The spike that proved the design found
  concurrent Builds of one Graph racing. Build holds the mutex only to set the built flag and copy
  the observers, so Builds run concurrently and constructors never run under the lock.
- **A Monitored failure names its node.** v0.10.0's `run: <error>` did not say which value failed.
  A channel registered with `Monitor` has no node and keeps the old form.
- **Methods on `*Nodes` are the only constructor form** (pain point 2). Arity adapters such as
  `graph.Func2(newDatabase, n.Config, n.Logger)` were tried and did not ship: the Define-line
  changes already made the graph legible and fixed the reliability findings, and a second form
  costs one adapter per arity, a projection node or wrapper for each constructor that takes part
  of a config, and no-error variants.
- **One breaking release, no deprecation period.** The old and new APIs never shipped side by
  side; every consumer moved in the migrations that followed.

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
