# Graph ergonomics

This note holds what the code does not express about go-core v0.11.0's `graph` and `lifecycle`
API, goal `graph-ergonomics` in the workspace roadmap: why the API changed, the behavior it kept,
the reasons behind its decisions, the alternatives it rejected, and what the consumers' migrations
need. The API itself is stated in `graph/doc.go`, `graph/graphtest/doc.go`, and
`lifecycle/doc.go`, and the CHANGELOG's v0.11.0 section lists every change.

Citations name a repository and a file:line in it; `slab` is go-web-service's `tools/slab` CLI,
and the spikes live under `~/experiments/`. Citations into the consumers are to their code before
migration, on go-core v0.10.0.

## Pain points

go-core v0.10.0's packages behaved correctly but read badly, and nine consumers repeated the same
workarounds. Across them (the three libraries, go-cli-sdk, go-web-sdk, both templates,
go-web-service, and the spikes), `Define` was called 81 times, `Scope.Use` 64, `Scope.After` 10,
`System.Get` 32, `inv.Get` 64, and `Layers()` 8, all eight in tests. Every `s.Use` in them is
unconditional. Each point names what answers it.

1. **Each name is written three times.** A node's struct field, its `Define` string, and a
   `// "name"` comment on the field that records the string for the reader: 43 comment lines
   (go-web-service `internal/app/app.go:48-78`; slab `internal/app/app.go:68-79`;
   go-web-sdk-template `template/internal/app/app.go:33-37`). Deriving the name from the type
   cannot replace them: slab has six nodes of type `*graph.Node[string]` (`app.go:69-75`).
   Answered by `graphtest.CheckNames`, which replaces the comments.
2. **Constructors return constructors.** Twenty `newX(...) func(*graph.Scope) (T, error)`
   wrappers, 17 in go-web-service and 3 in the web template, exist only to close over the node
   handles, and each writes `T` three times (go-web-service
   `internal/app/infrastructure.go:67-133`). Their arguments vary for no reason: the whole
   `*Nodes` (`infrastructure.go:76`), single handles (slab
   `internal/app/infrastructure.go:150,162`), or none (go-web-service
   `internal/app/telemetry.go:99`). Answered by a convention, not by go-core: constructors are
   methods on `*Nodes`, passed to Define as method values.
3. **Fixed and fresh values need closures.** A node whose value is fixed, or a fresh zero
   value, still takes a four-line closure: 6 sites (go-web-service
   `internal/app/infrastructure.go:52`, `server.go:12`, `admin.go:35`, `reactors.go:32`;
   go-web-sdk-template `template/internal/app/infrastructure.go:22`, `server.go:12`). The fresh
   ones must stay fresh per Build: a `lifecycle.Readiness` binds to one Coordinator only, and
   the gate and nudge are per-run state. Answered by `Graph.Value` and `Graph.Fresh`.
4. **Layer placement is invisible.** Where a node lands is set by `s.After` calls scattered
   through constructor bodies: telemetry three times (go-web-service
   `internal/app/infrastructure.go:78,88,105`), the server after every reactor and the schema
   (`server.go:50-53`), the sweeper and relay after the schema (`reactors.go:110,191`), and the
   reactors added as extra Build roots because nothing Uses them (`app.go:80-83`,
   `reactors.go:36`). Tests check the placement by walking `Layers()` and matching names
   (go-web-service `internal/app/graph_test.go:55-77,166-183`; go-web-sdk-template
   `template/internal/app/graph_test.go:142-157`). Definition order also sets the order of
   readiness checks within a layer, which only a comment states (`infrastructure.go:39-41`).
   Answered by `After`, `In`, and `Group` at Define, and `Scope.After`'s removal.
5. **Use and After are asymmetric.** v0.10.0's `s.After` never brought a node into the Build,
   so a node that is only ordered against must be reached some other way. go-web-service's
   router middleware calls `s.Use(n.Telemetry)` and discards the value to bring telemetry in
   (`internal/app/middleware.go:39`, explained at `telemetry.go:75-79`). Kept on purpose (see
   The decisions), and made visible: such a node is named as a root, and `System.Dropped` lists
   an After that ordered nothing.
6. **Build roots are repeated in tests.** The test fixture copies `Run`'s root list
   (go-web-service `internal/app/graph_test.go:45`) and keeps the copy honest by observing a
   real `Run` and comparing what each built (`graph_test.go:92-119`). Answered by a convention:
   the composition root exports its root list, and the test builds that list.
7. **There is no web runner.** About thirty lines (Build, Get, `lifecycle.New`, `OnReady`,
   `Run`, and the logging around them) are identical in go-web-service
   `internal/app/app.go:118-148` and go-web-sdk-template `template/internal/app/app.go:76-106`.
   The CLI side has `cli.Run` with `cli.WithGraph` (go-cli-sdk `run.go:32,245-249`). Answered
   by go-web-sdk's `web.Service`.
8. **`lifecycle.Config` arrives two ways.** A web root reads it off its config node,
   `sys.Get(n.Config).Config` (go-web-service `internal/app/app.go:137`); every CLI defines a
   lifecycle-config node with the same four-line constructor (go-cli-sdk-template
   `template/internal/app/infrastructure.go:31`; slab `internal/app/infrastructure.go:69`;
   spike-ai-pipeline `internal/docinfer/app/infrastructure.go:22`,
   `internal/docproc/app/infrastructure.go:26`). Answered by `web.Service`, which takes it as a
   node, as `cli.Program` does.
9. **Introspection gives only names.** v0.10.0's `Observe` passed a string and `Dependency` was
   `{Name, Value}`, so a test that holds a handle compares names:
   `top[0].Name == n.Server.Name()` (go-web-service `internal/app/graph_test.go:62`). Answered
   by handles throughout: `Dependency.Ref`, `Observe(func(Ref))`, `LayerOf`, `Has`,
   `Dependencies`, and `String`.
10. **The architecture contradicts itself on CLI commands.** `composition-terms.md` says every
    package below the composition root "never sees a node" (architecture
    `principles/composition-terms.md:19-21`), while the Go Elemental standard makes each CLI
    command "a function of the graph nodes it declares" (architecture
    `standards/go-elemental/principles/domain-files.md:47-48`). The code follows the standard:
    13 slab files outside its composition root take `*graph.Node` (e.g. slab
    `domain/document/commands.go:40`), as do 2 in spike-ai-pipeline
    (`internal/docinfer/infer/commands.go:17`, `internal/docproc/run/commands.go:18`). Answered
    outside go-core, by amending `composition-terms.md`.

## Behavior that did not change

v0.11.0 changes how a composition root writes its graph, not what the graph and lifecycle do:
edge discovery through `s.Use`, build-once, longest-path layers, the participation interfaces,
and the startup, readiness, and shutdown semantics are as `graph/doc.go` and `lifecycle/doc.go`
state them, and go-core's ported tests hold them. A migrated consumer's behavior tests are therefore
expected to keep passing, apart from a test that matches a Monitored value's `run: <error>`. The CHANGELOG's
other breaks fail at compile time or panic at Define.

Use discovery is what lets a CLI run build only what its command declares: go-cli-sdk builds the
leaf's declared nodes plus the lifecycle config (`run.go:245`), so help or a usage error builds
nothing. `s.Use` stays the only way a constructor adds a dependency, which is why order and
groups, which carry no value, moved to Define rather than gaining a second edge in the Scope.

## The decisions

The design review proposed one candidate API; spike-graph (`experiment.graph`) proved it on the
full go-web-service and slab, and the release's two plan rounds settled what the spike left. The
reasons below are what the package docs do not state.

- **An After never builds its target.** The spike kept v0.10.0's asymmetry (pain point 5): an
  After that built its target would let an order-only declaration pull a node, and its whole
  subtree, into a CLI run that never Uses it, undoing the build-only-what-is-declared property.
  A node only ordered against is named as a root instead, and `System.Dropped` makes a dropped
  After observable, so the full-roots test asserts there is none. An After naming an empty group
  is dropped too, even in a full-roots Build, so the same assertion catches a forgotten `In`.
- **Groups expand in definition order and share the node name space.** A group is a `Ref`, so a
  Build and an After take it wherever they take a node; one name space keeps the layer table and
  the error labels unambiguous. `Group` exports only `Name`: no consumer needed
  `Group.Members`, and a group's members are its `In` declarations, visible at their Define
  lines.
- **Replace swaps only the constructor.** A test's substitute stands in for a node's value, not
  its place; keeping After and In means the substitute lands where the real node would.
  Replace cannot remove an edge or a membership, and no test needed it.
- **Fresh is a method.** The design review proposed `graph.Fresh[T](g, name)`, a function,
  because Go had no method type parameters; Go 1.27 does, so it is `g.Fresh[T](name)`, beside
  `g.Value`. Both take Define's options, so a value can join a group. `g.Value` infers `T` from
  the argument's static type, so `g.Value("writer", w)` with `w io.Writer` is a
  `*Node[io.Writer]`: that is how a `*Nodes` method reaches a plain value such as the logger's
  writer.
- **Names stay explicit, and a fixture checks them.** The string labels errors and names
  readiness checks, and type-derived names fail on slab's string nodes (pain point 1). The rule,
  lowercased with spaces removed against the field name lowercased, admits both "meterprovider"
  and "meter provider" for a `MeterProvider` field, so a multi-word name may keep its spaces.
  `graphtest` holds the check so nothing in `graph` exists for the tests' sake.
- **A name refuses a comma or a newline; String stays unquoted.** `System.String` is for a
  failing test's message, where quoting every name costs legibility on every line. Refusing the
  two characters the table uses as separators makes the unquoted table unambiguous.
- **LayerOf panics; Has answers membership.** `LayerOf` as `(int, bool)` would make every layer
  assertion handle a miss that is a test's own mistake. A panicking `int` reads as one
  expression in a test, and `Has(ref) bool` answers the question when it is the question.
- **Dependencies lists the built nodes flat.** The consumers' tests flattened `Layers()` by
  hand to check every node; `Dependencies` is that flattening, in layer then definition order,
  so it equals the layers' concatenation.
- **Observers persist.** An observer applies to every later Build of its Graph, with no way to
  remove it: the consumers observe once, from a test's fixture, and a removal handle would be API
  no one calls. The docs say so instead.
- **A panicking observer is labelled.** In v0.10.0 an observer's panic escaped Build bare, with
  no node named. Build now re-panics naming the node, wrapping an error value so `errors.Is`
  reaches it.
- **A Graph is safe for concurrent use, under one mutex.** The spike found concurrent Builds of
  one Graph racing. A `sync.Mutex` guards the description; Build holds it only to set the built
  flag and copy the observers, so Builds run concurrently and constructors never run under the
  lock.
- **A Monitored failure names its node.** v0.10.0's `run: <error>` did not say which value
  failed; `run: <node>: <error>` does, wrapping the error. A channel registered with `Monitor`
  has no node and keeps `run: <error>`.
- **Arity adapters did not ship.** The spike tried `graph.Func2(newDatabase, n.Config,
  n.Logger)` beside methods on `*Nodes`. The architect decided against them: the Define-line
  changes already made the graph legible and fixed the reliability findings, and a second
  constructor form costs one adapter per arity, a projection node or wrapper for each
  constructor that takes part of a config (`postgres.New(cfg.Database)`), and no-error
  variants. Methods on `*Nodes` are the only constructor form.
- **One breaking release, no deprecation period.** The old and new APIs never ship side by
  side; every consumer moves in the migrations that follow.

## Rejected alternatives

- **uber-go/fx (v1.24.0) and dig (v1.19.0), its container.** fx runs components through
  `OnStart`/`OnStop` hooks registered on a lifecycle, the model the lifecycle principle retired
  in favor of participation by the value's own methods; dig resolves the graph by run-time
  reflection over constructor signatures, so a wiring mistake surfaces at run time rather than
  in a typed handle. Both are third-party dependencies in go-core, which is standard-library
  only.
- **samber/do (v2.1.0).** A typed container with lazy services, but it has no layered,
  concurrent startup: it cannot start a layer's subsystems together and shut down in reverse
  layers, which is the lifecycle's core semantics.
- **google/wire.** Compile-time code generation that fits the build-once model, but the
  repository was archived in August 2025.

None of them meets the sourcing rule either: the standard writes a capability in-house when it
is generic to its layer and has no specification or security surface (architecture
`standards/go-elemental/principles/dependencies.md:65-68`), and go-cli-sdk's lint holds that
module to the standard library and go-core.

## Migration

v0.11.0 ships go-core's part. go-web-service, with slab, is planned to move onto it in task
migrate-service. The libraries and both templates run on it, and their code is the target at full
size: go-web-sdk's `web.Service` and go-cli-sdk's `cli.Program` run a composition root, and
go-web-sdk-template's `internal/app` is a web root with its graph tests. The spike repositories that
import `graph` or `lifecycle` (spike-ai-pipeline and others) are evidence, read and never migrated.

`composition-terms.md` is amended to admit CLI command packages taking the nodes they declare
(pain point 10).

The sketches below are the migration's target in miniature, cut down from the real roots they
cite; spike-graph holds them at full size.

### Sketch 1: a web composition root

```go
type Nodes struct {
	Writer    *graph.Node[io.Writer]
	Config    *graph.Node[*config.Config]
	Lifecycle *graph.Node[lifecycle.Config]
	Logger    *graph.Node[*slog.Logger]
	Telemetry *graph.Node[*Telemetry]
	Database  *graph.Node[*database.DB]
	Schema    *graph.Node[*admin.Service]
	Reactors  *graph.Group
	Sweeper   *graph.Node[*sweep.Sweeper]
	Readiness *graph.Node[*lifecycle.Readiness]
	Router    *graph.Node[*web.Router]
	Server    *graph.Node[*web.Server]
}

func (n *Nodes) define(g *graph.Graph, cfg *config.Config, w io.Writer) {
	n.Writer = g.Value("writer", w)
	n.Config = g.Value("config", cfg)
	n.Lifecycle = g.Define("lifecycle", n.lifecycle)
	n.Logger = g.Define("logger", n.logger) // Uses n.Writer
	n.Telemetry = g.Define("telemetry", n.telemetry)
	n.Database = g.Define("database", n.database, graph.After(n.Telemetry))
	n.Schema = g.Define("schema", n.schema)

	n.Reactors = g.Group("reactors")
	n.Sweeper = g.Define("sweeper", n.sweeper, graph.In(n.Reactors), graph.After(n.Schema))

	n.Readiness = g.Fresh[lifecycle.Readiness]("readiness")
	n.Router = g.Define("router", n.router)
	n.Server = g.Define("server", n.server, graph.After(n.Reactors, n.Schema))
}

func (n *Nodes) database(s *graph.Scope) (*database.DB, error) {
	return postgres.New(s.Use(n.Config).Database)
}

// Roots is what Run builds; a test builds the same list rather than a copy.
func (a *App) Roots() []graph.Ref {
	n := a.nodes
	return []graph.Ref{n.Server, n.Schema, n.Telemetry, n.Reactors}
}

func (a *App) Run(ctx context.Context, stderr io.Writer) int {
	n := a.nodes
	return web.Service{Graph: a.graph, LifecycleConfig: n.Lifecycle, Logger: n.Logger,
		Server: n.Server, Roots: a.Roots()}.Run(ctx, stderr)
}
```

Every node's placement is in `define`, beside its name, and no constructor body orders
anything. The reactors are a group the server orders after, not a slice of roots maintained by
hand. Telemetry, which only orders, is a root rather than a discarded `s.Use` in the router's
middleware. The thirty-line `Run` is one call, and App drops its fallback logger.

### Sketch 2: a CLI command declaring its nodes

```go
func (n *Nodes) define(g *graph.Graph, flags *rootFlags) {
	n.LifecycleConfig = g.Define("lifecycle config", n.lifecycleConfig)
	n.Base = g.Define("base", flags.setting("base", defaultBase, parseString))
	n.Client = g.Define("client", n.client)
}

func (n *Nodes) client(s *graph.Scope) (*httpx.Client, error) {
	return httpx.NewClient(s.Use(n.Base)), nil
}

// Package organization is unchanged: a command still takes the node it declares,
// declares it with Use, and reads it with inv.Get.
```

The command package keeps its shape on purpose: declaring nodes on the command is what makes a
run build only what the command reaches. A family of settings parameterized by flag name, such
as `flags.setting` (slab's `newSetting`, moved onto the flags it reads), still returns a
constructor: a method cannot carry those parameters, and returning a constructor is the pattern
for that case.

### Sketch 3: layer tests

```go
func TestGraph_ServerAloneInTopLayer(t *testing.T) {
	a := app.New(configtest.Config(t), io.Discard)
	sys, err := a.Graph().Build(a.Roots()...)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	top := len(sys.Layers()) - 1
	if sys.LayerOf(a.Nodes().Server) != top || len(sys.Layers()[top]) != 1 {
		t.Errorf("want the server alone in the top layer; layers:\n%s", sys)
	}
	if d := sys.Dropped(); len(d) != 0 {
		t.Errorf("dropped Afters: %v", d)
	}
}

func TestGraph_TelemetryBeneathParticipants(t *testing.T) {
	a := app.New(configtest.Config(t), io.Discard)
	n := a.Nodes()
	sys, err := a.Graph().Build(a.Roots()...)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	tel := sys.LayerOf(n.Telemetry)
	// participant reports whether a value implements a lifecycle interface.
	for _, d := range sys.Dependencies() {
		if d.Ref != graph.Ref(n.Telemetry) && participant(d.Value) && sys.LayerOf(d.Ref) <= tel {
			t.Errorf("%s is not above telemetry; layers:\n%s", d.Name, sys)
		}
	}
}

func TestNodes_Names(t *testing.T) {
	graphtest.CheckNames(t, app.New(configtest.Config(t), io.Discard).Nodes())
}
```

The helpers that walked `Layers()` and matched names go away, the roots come from the one list
`Run` uses, and a failure prints the whole layer table instead of a slice of names.

## Evidence

spike-graph answered yes: both real roots are shorter and their layer placement legible, with
every node's layer stated at its Define line. `mise run measure` against its baseline:

| Root | Code lines | Closures | Wrapper constructors | Name comments | `s.After` sites |
|---|---|---|---|---|---|
| go-web-service | 460 → 411 | 21 → 0 | 14 → 0 | 21 → 0 | 7 → 0 |
| slab | 207 → 202 | 11 → 7 | 3 → 1 | 12 → 0 | 1 → 0 |

[spike-graph](https://github.com/JaimeStill/spike-graph) (archived, checked out at
`~/experiments/spike-graph`).
