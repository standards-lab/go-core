# Graph ergonomics

This note holds the design direction for the redesign of go-core's `graph` and `lifecycle`
packages, goal `graph-ergonomics` in the workspace roadmap. It is direction in flight, not
package documentation: the API below is a candidate until the spike (`experiment.graph`,
repository `spike-graph`) settles it and go-core's breaking release ships it. A section the code
comes to express is deleted.

Citations name a repository and a file:line in it; `slab` is go-web-service's `tools/slab` CLI,
and the spikes live under `~/experiments/`. Line numbers are as of this note's writing.

## The packages today

The packages are documented in their `doc.go` files (go-core `graph/doc.go`,
`lifecycle/doc.go`), and this note uses their vocabulary: `Define`, `Scope.Use`, `Scope.After`,
`Build`, `System.Get`, `System.Layers`, and `lifecycle.New`.

Both packages behave correctly. They read badly, and nine consumers repeat the same workarounds.
Across the consumers (the three libraries, go-cli-sdk, go-web-sdk, both templates,
go-web-service, and the spikes), `Define` is called 81 times, `Scope.Use` 64, `Scope.After` 10,
`System.Get` 32, `inv.Get` 64, and `Layers()` 8, all eight in tests. Every `s.Use` in
them is unconditional.

## Pain points

1. **Each name is written three times.** A node's struct field, its `Define` string, and a
   `// "name"` comment on the field that records the string for the reader: 43 comment lines
   (go-web-service `internal/app/app.go:48-78`; slab `internal/app/app.go:68-79`;
   go-web-sdk-template `template/internal/app/app.go:33-37`). Deriving the name from the type
   cannot replace them: slab has six nodes of type `*graph.Node[string]` (`app.go:69-75`).
2. **Constructors return constructors.** Twenty `newX(...) func(*graph.Scope) (T, error)`
   wrappers, 17 in go-web-service and 3 in the web template, exist only to close over the node
   handles, and each writes `T` three times (go-web-service
   `internal/app/infrastructure.go:67-133`). Their arguments vary for no reason: the whole
   `*Nodes` (`infrastructure.go:76`), single handles (slab
   `internal/app/infrastructure.go:150,162`), or none (go-web-service
   `internal/app/telemetry.go:99`).
3. **Value-only closures.** A node whose value is fixed, or a fresh zero value, still takes a
   four-line closure: 6 sites (go-web-service `internal/app/infrastructure.go:52`,
   `server.go:12`, `admin.go:35`, `reactors.go:32`; go-web-sdk-template
   `template/internal/app/infrastructure.go:22`, `server.go:12`). The fresh ones must stay fresh
   per Build: a `lifecycle.Readiness` binds to one Coordinator only (go-core
   `lifecycle/doc.go:97-106`), and the gate and nudge are per-run state.
4. **Layer placement is invisible.** Where a node lands is set by `s.After` calls scattered
   through constructor bodies: telemetry three times (go-web-service
   `internal/app/infrastructure.go:78,88,105`), the server after every reactor and the schema
   (`server.go:50-53`), the sweeper and relay after the schema (`reactors.go:110,191`), and the
   reactors added as extra Build roots because nothing Uses them (`app.go:80-83`,
   `reactors.go:36`). Tests check the placement by walking `Layers()` and matching names
   (go-web-service `internal/app/graph_test.go:55-77,166-183`; go-web-sdk-template
   `template/internal/app/graph_test.go:142-157`). Definition order also sets the order of
   readiness checks within a layer, which only a comment states (`infrastructure.go:39-41`).
5. **Use and After are asymmetric.** `s.After` never brings a node into the Build (go-core
   `graph/scope.go:66-77`), so a node that is only ordered against must be reached some other
   way. go-web-service's router middleware calls `s.Use(n.Telemetry)` and discards the value to
   bring telemetry in (`internal/app/middleware.go:39`, explained at `telemetry.go:75-79`).
6. **Build roots are repeated in tests.** The test fixture copies `Run`'s root list
   (go-web-service `internal/app/graph_test.go:45`) and keeps the copy honest by observing a
   real `Run` and comparing what each built (`graph_test.go:92-119`).
7. **There is no web runner.** About thirty lines (Build, Get, `lifecycle.New`, `OnReady`,
   `Run`, and the logging around them) are identical in go-web-service
   `internal/app/app.go:118-148` and go-web-sdk-template `template/internal/app/app.go:76-106`.
   The CLI side has `cli.Run` with `cli.WithGraph` (go-cli-sdk `run.go:32,245-249`).
8. **`lifecycle.Config` arrives two ways.** A web root reads it off its config node,
   `sys.Get(n.Config).Config` (go-web-service `internal/app/app.go:137`); every CLI defines a
   lifecycle-config node with the same four-line constructor (go-cli-sdk-template
   `template/internal/app/infrastructure.go:31`; slab `internal/app/infrastructure.go:69`;
   spike-ai-pipeline `internal/docinfer/app/infrastructure.go:22`,
   `internal/docproc/app/infrastructure.go:26`).
9. **Introspection gives only names.** `Observe` passes a string (go-core `graph/graph.go:81`)
   and `Dependency` is `{Name, Value}` (`graph/system.go:18-21`), so a test that holds a handle
   compares names: `top[0].Name == n.Server.Name()` (go-web-service
   `internal/app/graph_test.go:62`).
10. **The architecture contradicts itself on CLI commands.** `composition-terms.md` says every
    package below the composition root "never sees a node" (architecture
    `principles/composition-terms.md:19-21`), while the Go Elemental standard makes each CLI
    command "a function of the graph nodes it declares" (architecture
    `standards/go-elemental/principles/domain-files.md:47-48`). The code follows the standard:
    13 slab files outside its composition root take `*graph.Node` (e.g. slab
    `domain/document/commands.go:40`), as do 2 in spike-ai-pipeline
    (`internal/docinfer/infer/commands.go:17`, `internal/docproc/run/commands.go:18`).

## Behavior that must not change

The redesign changes how a composition root writes its graph, not what the graph and lifecycle
do. These hold across it:

- **Edges are discovered through `s.Use`.** A constructor's Use calls are its dependencies,
  found as it runs, and a node neither a root nor reached by a Use is never constructed (go-core
  `graph/doc.go:31-41`). This is what lets a CLI run build only what its command declares:
  go-cli-sdk builds the leaf's declared nodes plus the lifecycle config (`run.go:245`), so help
  or a usage error builds nothing. `s.Use` stays the only way a constructor adds a dependency.
- **Build-once.** Each node is built at most once per Build, and each Build constructs fresh
  values.
- **Longest-path layers.** A node's layer is one above its highest dependency, over the Use and
  order-only edges (`graph/doc.go:47-52`); the composition root never assigns a layer.
- **Panics for wiring mistakes.** A cycle, a foreign or nil node, a Scope used after its
  constructor returned, a Define after Build, a Get of a node not built, a duplicate or empty
  name: each panics with a `graph: ` message (`graph/doc.go:54-74`), as `lifecycle.New` panics
  on its own wiring mistakes (`lifecycle/doc.go:39-42`). A constructor's error is still an
  error, labelled with the node's name.
- **The participation interfaces.** A node takes part in the lifecycle only through its value's
  methods: `Starter`, `Stopper`, `Subsystem`, `ReadyStarter`, `Monitored`, and the `Check` a
  readiness probe lists (`lifecycle/doc.go:44-51`). No registration, no hooks.
- **Startup, readiness, and shutdown.** Layers start lowest first, each layer concurrently; the
  first failure stops higher layers; readiness never flips during a failed startup and goes
  false when shutdown begins; checks are listed in layer order, then definition order; shutdown
  runs on every path, in reverse (`lifecycle/doc.go:64-148`, and the standard's
  `principles/lifecycle-and-context.md`).

## The candidate API

One candidate, from the goal record's decisions 9 to 14. Everything not named here stays as it
is.

- **Order and groups are declared at Define (decision 9).** `Define` takes options:
  `graph.After(refs...)` orders the node after each ref, and `graph.In(group)` makes it a member
  of a group. `g.Group(name)` returns a group handle that is a `Ref`; a Build given a group, as
  a root or in `After`, expands it to its members. An order-only edge still builds nothing, as
  `s.After` does today, so a node only ordered against is named as a root. `Scope.After` is
  removed, and every edge that orders without a value is in one place, beside the node's name.
- **Constructors are methods on `*Nodes` (decision 10).** A composition root's constructor is a
  method on its Nodes struct, passed to Define as a method value:
  `n.Database = g.Define("database", n.database)`. The method reads the handles it Uses off its
  receiver, so the wrapper that returns a constructor goes away and `T` is written once, in the
  signature.
- **Values and fresh values (decision 11).** `g.Value(name, v)` defines a node whose value is
  `v` in every Build. `graph.Fresh[T](g, name)` defines a node whose value is a new `*T`, the
  zero value, in each Build: the Readiness, the gate, and the nudge.
- **Names stay explicit (decision 12).** The string is still passed to Define: it labels errors
  and names readiness checks, and type-derived names fail on slab's string nodes. A fixture
  check that each node's name matches its struct field replaces the `// "name"` comments.
- **Introspection exposes handles (decision 13).** `Dependency` gains `Ref`, the handle Define
  returned, so a test compares handles rather than names; `Observe` takes `func(graph.Ref)`;
  `System.LayerOf(ref) int` returns a node's layer; and `System.String()` prints the layer
  table, one line per layer, for a failing test's message.
- **One runner on each side (decision 14).** go-web-sdk gains `web.Run(ctx, g, lifecycleConfig,
  roots...)`, matching go-cli-sdk's `cli.Run` with `cli.WithGraph(g, lifecycleConfig)`: Build,
  `lifecycle.New`, and `Run`. Both take the lifecycle configuration as a
  `*graph.Node[lifecycle.Config]`, so it is supplied one way on both sides.

The sketches below are miniatures, cut down from the real roots they cite. Each shows today's
code first, then the same thing written against the candidate.

### Sketch 1: a web composition root

Today, a miniature of go-web-service `internal/app` (`app.go`, `infrastructure.go`, `server.go`,
`reactors.go`):

```go
type Nodes struct {
	Config    *graph.Node[*config.Config]       // "config"
	Logger    *graph.Node[*slog.Logger]         // "logger"
	Telemetry *graph.Node[*Telemetry]           // "telemetry"
	Database  *graph.Node[*database.DB]         // "database"
	Schema    *graph.Node[*admin.Service]       // "schema"
	Sweeper   *graph.Node[*sweep.Sweeper]       // "sweeper"
	Readiness *graph.Node[*lifecycle.Readiness] // "readiness"
	Router    *graph.Node[*web.Router]          // "router"
	Server    *graph.Node[*web.Server]          // "server"

	Reactors []graph.Ref
}

func defineInfrastructure(g *graph.Graph, n *Nodes, cfg *config.Config, w io.Writer) {
	n.Config = g.Define("config", func(*graph.Scope) (*config.Config, error) {
		return cfg, nil
	})
	n.Logger = g.Define("logger", newLogger(n, w))
	n.Database = g.Define("database", newDatabase(n))
}

func newDatabase(n *Nodes) func(*graph.Scope) (*database.DB, error) {
	return func(s *graph.Scope) (*database.DB, error) {
		s.After(n.Telemetry)
		return postgres.New(s.Use(n.Config).Database)
	}
}

func defineReactors(g *graph.Graph, n *Nodes) {
	n.Sweeper = g.Define("sweeper", newSweeper(n)) // newSweeper calls s.After(n.Schema)
	n.Reactors = []graph.Ref{n.Sweeper}
}

func defineServer(g *graph.Graph, n *Nodes) {
	n.Readiness = g.Define("readiness", func(*graph.Scope) (*lifecycle.Readiness, error) {
		return new(lifecycle.Readiness), nil
	})
	n.Router = g.Define("router", newRouter(n))
	n.Server = g.Define("server", newServer(n))
}

func newServer(n *Nodes) func(*graph.Scope) (*web.Server, error) {
	return func(s *graph.Scope) (*web.Server, error) {
		for _, r := range n.Reactors {
			s.After(r)
		}
		s.After(n.Schema)
		return web.NewServer(s.Use(n.Config).Server, s.Use(n.Router), s.Use(n.Logger)), nil
	}
}

func (a *App) Run(ctx context.Context) int {
	n := a.nodes
	roots := append([]graph.Ref{n.Config, n.Logger, n.Server, n.Schema}, n.Reactors...)
	sys, err := a.graph.Build(roots...)
	if err != nil {
		logging.New(a.w, a.cfg.Log).Error("service failed", "error", err)
		return 1
	}
	logger, server := sys.Get(n.Logger), sys.Get(n.Server)
	lc := lifecycle.New(sys, sys.Get(n.Config).Config)
	lc.OnReady(func() { logger.Info("server ready", "addr", server.Addr()) })
	if err := lc.Run(ctx); err != nil {
		logger.Error("service failed", "error", err)
		return 1
	}
	logger.Info("server stopped")
	return 0
}
```

Proposed:

```go
type Nodes struct {
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

	w io.Writer // the logger's writer, from New
}

func (n *Nodes) define(g *graph.Graph, cfg *config.Config) {
	n.Config = g.Value("config", cfg)
	n.Lifecycle = g.Define("lifecycle", n.lifecycle)
	n.Logger = g.Define("logger", n.logger)
	n.Telemetry = g.Define("telemetry", n.telemetry)
	n.Database = g.Define("database", n.database, graph.After(n.Telemetry))
	n.Schema = g.Define("schema", n.schema)

	n.Reactors = g.Group("reactors")
	n.Sweeper = g.Define("sweeper", n.sweeper, graph.In(n.Reactors), graph.After(n.Schema))

	n.Readiness = graph.Fresh[lifecycle.Readiness](g, "readiness")
	n.Router = g.Define("router", n.router)
	n.Server = g.Define("server", n.server, graph.After(n.Reactors, n.Schema))
}

func (n *Nodes) lifecycle(s *graph.Scope) (lifecycle.Config, error) {
	return s.Use(n.Config).Config, nil
}

func (n *Nodes) database(s *graph.Scope) (*database.DB, error) {
	return postgres.New(s.Use(n.Config).Database)
}

func (n *Nodes) server(s *graph.Scope) (*web.Server, error) {
	return web.NewServer(s.Use(n.Config).Server, s.Use(n.Router), s.Use(n.Logger)), nil
}

// logger, telemetry, schema, sweeper, and router are methods of the same shape.

// Roots is what Run builds; a test builds the same list rather than a copy.
func (a *App) Roots() []graph.Ref {
	n := a.nodes
	return []graph.Ref{n.Server, n.Schema, n.Telemetry, n.Reactors}
}

func (a *App) Run(ctx context.Context) error {
	return web.Run(ctx, a.graph, a.nodes.Lifecycle, a.Roots()...)
}
```

What changed: the placement of every node is in `define`, beside its name, and no constructor
body orders anything. The reactors are a group the server orders after, not a slice of roots
maintained by hand. Telemetry, which only orders, is a root rather than a discarded `s.Use` in
the router's middleware. The thirty-line `Run` is one call, and the lifecycle config is a node,
as it is on the CLI side. `Roots` is the composition root's own method, not go-core API.

### Sketch 2: a CLI command declaring its nodes

Today, a miniature of slab (`internal/app/app.go`, `internal/app/infrastructure.go`,
`domain/organization/commands.go`):

```go
type Nodes struct {
	LifecycleConfig *graph.Node[lifecycle.Config] // "lifecycle config"
	Base            *graph.Node[string]           // "base", the service's base URL
	Client          *graph.Node[*httpx.Client]    // "client"
}

func defineInfrastructure(g *graph.Graph, n *Nodes, flags *rootFlags) {
	n.LifecycleConfig = g.Define("lifecycle config", newLifecycleConfig)
	n.Base = g.Define("base", newSetting(flags, "base", defaultBase, parseString))
	n.Client = g.Define("client", newClient(n.Base))
}

func newClient(base *graph.Node[string]) func(*graph.Scope) (*httpx.Client, error) {
	return func(s *graph.Scope) (*httpx.Client, error) {
		return httpx.NewClient(s.Use(base)), nil
	}
}

func (a *App) Run(ctx context.Context, args []string) int {
	return cli.Run(ctx, a.root, args, a.streams, cli.WithGraph(a.graph, a.nodes.LifecycleConfig))
}

// In package organization, below the composition root.
func Commands(client *graph.Node[*httpx.Client]) []*cli.Command {
	org := &cli.Command{Name: "org", Summary: "Read and change organizations"}
	org.Add(getCommand(client)).Use(client)
	return []*cli.Command{org}
}

func getCommand(client *graph.Node[*httpx.Client]) *cli.Command {
	return &cli.Command{
		Name: "get",
		Args: cli.ExactArgs(1),
		Run: func(ctx context.Context, inv *cli.Invocation) error {
			res, err := NewClient(inv.Get(client)).Get(ctx, inv.Args[0])
			return render(inv, res, err)
		},
	}
}
```

Proposed:

```go
type Nodes struct {
	LifecycleConfig *graph.Node[lifecycle.Config]
	Base            *graph.Node[string] // the service's base URL
	Client          *graph.Node[*httpx.Client]
}

func (n *Nodes) define(g *graph.Graph, flags *rootFlags) {
	n.LifecycleConfig = g.Define("lifecycle config", n.lifecycleConfig)
	n.Base = g.Define("base", flags.setting("base", defaultBase, parseString))
	n.Client = g.Define("client", n.client)
}

func (n *Nodes) client(s *graph.Scope) (*httpx.Client, error) {
	return httpx.NewClient(s.Use(n.Base)), nil
}

func (a *App) Run(ctx context.Context, args []string) int {
	return cli.Run(ctx, a.root, args, a.streams, cli.WithGraph(a.graph, a.nodes.LifecycleConfig))
}

// Package organization is unchanged: a command still takes the node it declares,
// declares it with Use, and reads it with inv.Get.
```

The command package keeps its shape on purpose: declaring nodes on the command is what makes a
run build only what the command reaches, so the redesign keeps it and amends
`composition-terms.md` to admit it (pain point 10). The gain is in the composition root. A
setting family parameterized by flag name, like `newSetting`, still returns a constructor; a
method cannot carry its parameters, and that is the case the pattern is for.

### Sketch 3: a layer test

Today, go-web-service `internal/app/graph_test.go:42-77`, trimmed:

```go
func buildAsRun(t *testing.T, a *app.App) *graph.System {
	n := a.Nodes()
	roots := append([]graph.Ref{n.Config, n.Logger, n.Server, n.Schema}, n.Reactors...)
	sys, err := a.Graph().Build(roots...)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return sys
}

func layerOf(sys *graph.System, name string) int {
	for i, layer := range sys.Layers() {
		for _, d := range layer {
			if d.Name == name {
				return i
			}
		}
	}
	return -1
}

func TestGraph_ServerAloneInTopLayer(t *testing.T) {
	a := app.New(configtest.Config(t), io.Discard)
	// ...observe a real Run and compare its nodes with buildAsRun's...
	sys := buildAsRun(t, a)
	layers := sys.Layers()
	top := layers[len(layers)-1]
	if len(top) != 1 || top[0].Name != a.Nodes().Server.Name() {
		t.Errorf("top layer = %+v, want the server node alone", names(top))
	}
}
```

Proposed:

```go
func TestGraph_ServerAloneInTopLayer(t *testing.T) {
	a := app.New(configtest.Config(t), io.Discard)
	n := a.Nodes()
	sys, err := a.Graph().Build(a.Roots()...)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	top := len(sys.Layers()) - 1
	if sys.LayerOf(n.Server) != top || len(sys.Layers()[top]) != 1 {
		t.Errorf("want the server alone in the top layer; layers:\n%s", sys)
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
	for _, layer := range sys.Layers() {
		for _, d := range layer {
			if d.Ref != graph.Ref(n.Telemetry) && participant(d.Value) && sys.LayerOf(d.Ref) <= tel {
				t.Errorf("%s is not above telemetry; layers:\n%s", d.Ref.Name(), sys)
			}
		}
	}
}
```

The test helpers `buildAsRun`, `layerOf`, and `names` go away, the roots come from the one list
`Run` uses, and a failure prints the whole layer table instead of a slice of names.

## The open fork: arity adapters

Decision 10 makes constructors methods on `*Nodes`. The spike also tries typed arity adapters
beside them: `graph.Func2(newDatabase, n.Config, n.Logger)` returns a constructor that Uses each
handle and passes the values to an ordinary function, so a constructor needs no Scope and no
receiver at all.

```go
n.Client = g.Define("client", graph.Func1(httpx.NewClientE, n.Base))
```

For: the dependencies are visible at the Define line, and a constructor is a plain function a
package below the root can export. Against: one adapter per arity; a function that takes part of
a value, such as `postgres.New(cfg.Database)`, needs a projection node or a wrapper; a
constructor that returns no error needs an adapter variant or a wrapper (`NewClientE` above is
invented for the example); and two ways to write a constructor where today there is one. The
spike writes one tier both ways and keeps adapters only if they read better there; otherwise
methods on `*Nodes` are the only form.

## Rejected alternatives

- **uber-go/fx (v1.24.0) and dig (v1.19.0), its container.** fx runs components through
  `OnStart`/`OnStop` hooks registered on a lifecycle, the model the lifecycle principle retired
  in favor of participation by the value's own methods; dig resolves the graph by run-time
  reflection over constructor signatures, so a wiring mistake surfaces at run time rather than
  in a typed handle. Both are third-party dependencies in go-core, which is standard-library
  only; fx's last release was May 2025.
- **samber/do (v2.1.0).** A typed container with lazy services, but it has no layered,
  concurrent startup: it cannot start a layer's subsystems together and shut down in reverse
  layers, which is the lifecycle's core semantics.
- **google/wire.** Compile-time code generation that fits the build-once model, but the
  repository was archived in August 2025.

None of them meets the sourcing rule either: the standard writes a capability in-house when it
is generic to its layer and has no specification or security surface (architecture
`standards/go-elemental/principles/dependencies.md:65-68`), and go-cli-sdk's lint holds that
module to the standard library and go-core.

## The spike

`experiment.graph`, in repository `spike-graph` (`github.com/JaimeStill/spike-graph`, checked
out at `~/experiments/spike-graph`), asks one question: does the candidate API, built on today's
graph semantics, make a web composition root (a miniature of go-web-service) and a CLI (a
miniature of slab, with commands declaring their nodes) shorter, and their layer placement
legible, with arity adapters and methods on `*Nodes` compared on one tier?

Its answer decides the exact API go-core's breaking release ships: which of decisions 9 to 14
survive as written, whether arity adapters ship, and the shape of the details this note leaves
loose (the rule the name check applies, how `web.Run` reports readiness and failure, and how a
Nodes method reaches a plain value such as the logger's writer). The spike is a prerequisite of
`capstone.data-intelligence`.

## Migration

- **One breaking go-core release, no deprecation period.** The old and new APIs never ship side
  by side; every consumer moves in the migration that follows.
- **Then one migration task per tier:**
  1. the libraries with go-cli-sdk: go-database, go-observability, go-storage, go-web-sdk
     (adding `web.Run`), and go-cli-sdk;
  2. the templates: go-web-sdk-template and go-cli-sdk-template;
  3. go-web-service, with slab.
- **`composition-terms.md` is amended** to admit CLI command packages taking the nodes they
  declare, once the spike settles the API.
