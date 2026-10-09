# goal · go-core-graph

- **State:** idle
- **Task:** none
- **Branch:** none

## Tasks

1. [x] go-core
2. [x] go-storage
3. [x] go-web-sdk
4. [x] go-web-sdk-template
5. [x] go-web-service

## Task brief · go-web-service

```
## Task brief · go-core-graph · go-web-service
Problem       go-web-service requires go-core v0.5.0, go-web-sdk v0.14.0, and go-storage
              v0.4.0. Its composition root registers every service on the stage table
              (lifecycle.Add at stageInfrastructure, stageSchema, stageReactors,
              StageRoot), brackets the stages with telemetry's OnStartup and OnShutdown
              hooks, watches the server's and the sweeper's Err through Monitor, and mounts
              /readyz on the Coordinator. go-core v0.6.0 removes all of that; go-web-sdk
              v0.15.0's RegisterHealth takes a web.Doctor; go-storage v0.5.0 requires
              go-core v0.6.0. The service is the goal's closing test: until it runs on the
              graph, the promotion is unproven on the full stack. The exact /readyz order,
              the ready record's address, a startup failure at the server, and the shutdown
              bound are unpinned, so nothing holds the move to today's behavior.
Behaviors
  1. Both modules build, and the check passes, on go-core v0.6.0, go-web-sdk v0.15.0, and
     go-storage v0.5.0 (the slab module on go-core v0.6.0 and go-web-sdk v0.15.0), with Go
     1.27.2 and grafana/mimir 3.2.2. Currency reports nothing. The go directive stays 1.27,
     and golangci-lint 2.14.0 stays.
  2. Configuration loads at the entrypoint with its contract unchanged:
     - shutdown_timeout is a top-level config.json key, default 10s.
     - APP_SHUTDOWN_TIMEOUT overrides it. An unparsable value is an error naming
       APP_SHUTDOWN_TIMEOUT; a non-positive one gives "shutdown_timeout must be positive".
       An empty prefix reads no override.
     - An overlay's shutdown_timeout replaces the base's; an overlay that leaves it unset
       keeps the base's.
     - A file with a "Config" or "Env" key is rejected as unknown.
     - A config load failure exits nonzero with "config load failed".
  3. Config embeds lifecycle.Config by value, untagged, and drops its own shutdown
     timeout. Merge and Finalize call the embedded ones first; Finalize returns the
     embedded error unwrapped, so error text and order stay. Literals set the timeout as
     a promoted key. The sweeper's grace stays half the shutdown timeout.
  4. The app package describes its graph in one exported Nodes value with these nodes, by
     name: config, logger, telemetry, database, storage, sql (the SQL session with its
     catalog), files (the file store over blobfs and the object store), wake, gate,
     organization, document, schema (with its migrator, migration sets, and seeder
     inside), sweeper, readiness, router, server. Each layer file keeps one define
     function: infrastructure (config through files, database defined before storage),
     telemetry, admin (gate, schema), domain (organization, document), reactors (wake,
     sweeper), and server (readiness, router, server). Nodes.Reactors lists the sweeper.
     The server orders itself After each reactor and After schema. Neither the stage table
     nor the Infrastructure, Domain, and Admin structs remain.
  5. New(cfg, w) only describes the graph and can't fail; the entrypoint's "app init
     failed" path goes. App.Graph() and App.Nodes() are exported, so a caller can Replace
     or Observe a node before Run.
  6. Run(ctx) builds from config, logger, server, schema, and Nodes.Reactors, hands the
     Coordinator the config node's lifecycle block, and runs it:
     - 0 after a clean drain, logging "server stopped".
     - 1 on a Build, startup, runtime, or shutdown failure, logging "service failed" with
       the error. A Build failure is logged through a logger built outside the graph from
       the config's log block.
     - A second Run panics before anything is built.
  7. /healthz answers 200, application/json, Cache-Control no-store, {"status":"ok"}, with
     a request ID header.
  8. /readyz answers 200, application/json, no-store, {"status":"ready","checks":[...]}
     with checks exactly lifecycle, database, storage, schema, sweeper, in that order. The
     health route's web.Doctor is the readiness node's value. With the object store
     unreachable, /readyz answers 503 and lists storage as the only unready check.
  9. The ready record "server ready" names the address the server bound, logged once after
     binding, and that address answers /readyz 200.
 10. A port another listener holds makes the service exit 1 with a "startup:" error
     naming server; "server ready" is never logged.
 11. With APP_SHUTDOWN_TIMEOUT=1s, an upload whose declared body is only partly sent holds
     the drain; an interrupt makes the service exit 1 within the timeout plus a margin.
 12. Startup order is kept, dependencies before dependents:
     - telemetry starts before every other participant and stops after them;
     - database and storage start together; a failure in either exits 1 naming it, and
       schema never starts;
     - schema starts after both;
     - the sweeper starts after schema, so no sweep pass is refused on an empty schema;
     - the server starts last, alone in the built graph's top layer, and drains first.
     The drain runs in reverse.
 13. Telemetry's node value is a service-owned Starter and Stopper over go-observability's
     Telemetry, and its construction performs no I/O. Its Shutdown limits the flush to 1s,
     logs a failure at warn as "telemetry shutdown", and never fails the run. A Shutdown
     after a Start that didn't succeed does nothing.
 14. The wake node's value gives the document domain its Nudge and the sweeper its source,
     and has no Ready method, so /readyz gains no "wake" check. The wake is nudged once at
     construction, so startup runs a sweep.
 15. Each participant is exactly one node's value: the admin storage routes read the
     storage node. The root makes no Monitor call; the server's and sweeper's values are
     found as Monitored, and a sweeper failure ends the run as "run:". sdk's reactor tests
     pin this through a one-node graph with no Monitor call, and a compile-time proof
     states that the reactor is a lifecycle Subsystem, ReadinessChecker, and Monitored.
 16. The integration suite passes: the existing cases plus the pins of behaviors 8 to 11.
     After the pin slice, the only integration change is the harness's Ready comment.
 17. No README, STANDARDS, context note, package doc, or comment names a stage, the stage
     table, StageRoot, lifecycle.Service, Add, OnStartup, OnShutdown, or Monitor on the
     server or the sweeper. The README's startup paragraph describes the graph's order,
     and "how to add a service" defines a node in its layer's define function, with a
     reactor's node joining Nodes.Reactors.
 18. The CHANGELOG's [Unreleased] records:
     - the go-core v0.6.0, go-web-sdk v0.15.0, and go-storage v0.5.0 requirements;
     - the composition root on the graph, with Config embedding lifecycle.Config;
     - telemetry as a node, the wake kept off /readyz, and the probes unchanged;
     - that the libraries' own breaks are in their CHANGELOGs.
Test seams    the integration process runner over Postgres and Azurite (behaviors 7-12,
              16); New and Run hermetic, for startup and Build failure; App.Graph() built
              and read by layer (behaviors 12, 15); Config's Merge, Finalize, and Load over
              files; sdk's Reactor under a Coordinator on a one-node graph
Slices
  1. upgrade: Go 1.27.2 and grafana/mimir 3.2.2. Done when currency reports only the
     go-core, go-web-sdk, and go-storage lines and the check passes.
  2. pins on v0.5.0 (an exception to upgrade-first, round 1 Q1: v0.6.0 removes the API
     the pins would run against): integration cases for the exact /readyz order and both
     probes' headers, the ready record's bound address, the taken port, and the
     partial-upload stall; unit pins for any config contract not yet covered. Done when
     the check and the integration tier pass on v0.5.0 (behaviors 2, 7-11, 16).
  3. upgrade and move: go-core v0.6.0, go-web-sdk v0.15.0, go-storage v0.5.0 in both
     modules; Config embeds lifecycle.Config; the Nodes graph, New and Run; telemetry and
     wake values; the readiness node as Doctor; no Monitor; sdk's reactor tests rewired;
     the stage table and layer structs gone. Done when currency exits 0, the check passes,
     and slice 2's pins and the whole integration suite pass with only the harness comment
     changed (behaviors 1-15).
  4. structural pins, docs, CHANGELOG: the server alone in the top layer, telemetry below
     every other participant, the sweeper above schema, each participant once; stage text
     removed; [Unreleased] entries. Done when the check passes and the docs name no stage
     (behaviors 12, 15, 17, 18).
Out of scope  the role-interface findings (cli · role-interfaces); go-observability's
              Shutdown fix and go-database's doc examples (same task); promoting sdk's
              reactor and gate (v1.messaging); slab beyond its requirements; probe bodies,
              the APP_ namespace, telemetry configuration; a release (the service is
              unreleased); coordinator and architecture edits (goal sync)
Door          two-way: no tag; the sync's pull requests to the coordinator and the
              architecture repository revert like any other
```

## Decisions

- go-core: graph promoted from spike-cli-architecture rather than adopting dig/fx or wire —
  the Core SDK's line is the standard library alone.
- go-core: a node reaches readiness through a lifecycle.Readiness value defined in the graph
  and bound by New(sys, cfg); the composition root doesn't mount /readyz on the Coordinator
  outside the graph.
- go-core: OnStartup and OnShutdown retired; a bracketing value such as telemetry becomes a
  layer-0 Starter/Stopper node.
- go-core: the Coordinator finds each value's runtime error channel by the lifecycle.Monitored
  interface (Err() <-chan error); Monitor(ch) stays for channels outside the graph.
- go-core: Exec carries Run's runtime contract (Ready, OnReady, monitored errors).
- go-core: processtest's one-shot runner is Run(t, Cmd{Args, Stdin, Env}) Result{Stdout,
  Stderr, Code}.
- go-core: keeps go-core's run context (lives until shutdown) and error labels rather than the
  spike's per-layer start context and labels — neither is in the goal's break list.
- go-core: a Run whose context ended before startup starts nothing, stops nothing, and returns
  nil.
- go-core: Exec returns a startup cut short by its context as a "startup:" error; Run treats the
  same cut as a clean stop.
- go-core: Exec returns fn's error unwrapped, joined with any monitored failure wrapped "run:".
- go-core: lifecycle.Config follows go-core's config contract (Merge, Finalize, and Env with
  NewEnv), so config.Load loads it.
- go-core: Readiness has no constructor: the node's value is new(lifecycle.Readiness). Binding
  one Readiness to a second Coordinator panics, and New computes Checks once.
- go-core: the Coordinator calls each Monitored value's Err once startup completes, and skips a
  nil channel, whether from Err or Monitor.
- go-core: Run reports a monitored failure that wraps context.Canceled as "run:", fixing a
  v0.5.0 flaw the standards review found.
- go-core: a probe verified processtest.Run's stall-past-Failsafe path; no committed test covers
  it, since Failsafe is a 15s constant.
- go-core: processtest.Run's logged shell line names the program, not its build path, which Main
  deletes, and lists only Cmd.Env's overrides.
- go-storage: releases v0.5.0, not v0.4.1 — requiring go-core v0.6.0 breaks importers still
  on lifecycle.Service under a patch update; the repo releases 0.x breaks as minors.
- go-storage: azureblob gets no tag in this task; its azcore v1.23.3 bump waits on main with
  the earlier unreleased bump.
- go-storage: the Service shape test becomes a compile-time proof of Subsystem and
  ReadinessChecker; no graph or Coordinator test in go-storage.
- go-storage: slice 1 added the Subsystem assertion planned for slice 2; its check failed without it.
- go-storage: doc.go's example builds the azureblob client inside the Store's node, for brevity.
- go-storage: doc.go and README say the Coordinator lists Store's readiness in Checks, not runs it.
- go-storage: azureblob's CHANGELOG omits the azcore v1.23.3 bump, as it omitted the earlier one.
- go-web-sdk: RegisterHealth takes `web.Doctor`, an exported interface embedding
  lifecycle.ReadinessChecker plus Checks() []lifecycle.Check, which *lifecycle.Coordinator and
  *lifecycle.Readiness both satisfy; ReadinessReporter and an unnamed interface were rejected.
- go-web-sdk: RegisterHealth panics at wiring on a nil Doctor interface value, naming the fix; no
  typed-nil detection.
- go-web-sdk: middleware/rate-limit takes go-core v0.6.0 and httprate v0.16.1 on main, untagged,
  with httprate's IPv4-mapped bucketing change under its [Unreleased].
- go-web-sdk: test Coordinators use a fixed 2s ShutdownTimeout, since New panics on a Config that
  hasn't passed Finalize and Finalize reads the environment.
- go-web-sdk: "reads the Doctor on every request" means each request reads the check's current
  state. v0.6.0 fixes Checks at New, so the v0.14.0 case of a check registered after
  RegisterHealth no longer exists.
- go-web-sdk: the compile-time proofs for Doctor, Subsystem, and Monitored live in health.go and
  server.go, not in tests.
- go-web-sdk: the CHANGELOG's v0.15.0 section is dated and linked, as v0.14.0's release prep was,
  with an empty [Unreleased] above it.
- go-web-sdk: the standards review corrected rate-limit's CHANGELOG: under httprate v0.16.0 every
  IPv4-mapped client shared one counter, because its /64 prefix is ::, rather than having its own
  key. ratelimit.New's godoc states the v0.16.1 keying, and
  TestNew_IPv4MappedClientSharesItsIPv4Budget pins it.
- go-web-sdk: middleware/rate-limit still requires go-web-sdk v0.14.0; moving it to v0.15.0 waits
  on the v0.15.0 tag.
- go-web-sdk-template: pins land first on go-core v0.5.0, an exception to upgrade-first, since
  v0.6.0 removes the stage API the pins would be written against; the upgrade is the move.
- go-web-sdk-template: internal/app follows the CLI pattern — an exported Nodes struct, a define
  function per layer, New describes only and can't fail, Run builds and runs with its own
  second-Run guard, App.Graph() and App.Nodes() exported; New building the graph was rejected
  for leaving no Replace seam.
- go-web-sdk-template: Config embeds lifecycle.Config and is served from a Config node; closing
  constructors over cfg was rejected.
- go-web-sdk-template: the layer structs go — a field of a struct takes no lifecycle part; nodes
  are Logger, Config, Readiness, Router, Server; reactors are Build roots with Server After each.
- go-web-sdk-template: pins from outside through New and Run, plus a top-layer structural pin;
  startup order, checker hold, and monitored error stay pinned in go-core and go-web-sdk, and a
  test-only root hook was rejected.
- go-web-sdk-template: lifecycle.Config is embedded by value, untagged — a tag or named field
  nests shutdown_timeout, and a pointer breaks Go 1.27's promoted literal keys and can be nil.
- go-web-sdk-template: Config's Merge and Finalize call the embedded ones; Finalize checks it
  first and returns its error unwrapped, so error text and order stay.
- go-web-sdk-template: code and tests set the timeout as a promoted literal key; the check's
  go fix (embedlit) rejects the nested form.
- go-web-sdk-template: Go moves 1.27.1 → 1.27.2, released after planning; currency must exit 0,
  and the upgrade slice takes a patch that needs no adaptation.
- go-web-sdk-template: Run builds from Config, Logger, Server, and Nodes.Reactors. Every node Run
  reads with Get is a root, so a Replace cannot leave it unbuilt.
- go-web-sdk-template: Run logs a Build failure as "service failed" through a logger it builds
  outside the graph from cfg.Log, since the logger node may be what failed.
- go-web-sdk-template: server.go's defineServer defines Readiness, Router, and Server; app.go
  holds App, Nodes, New, Graph, Nodes, and Run.
- go-web-sdk-template: the reactor roots are an exported Nodes.Reactors []graph.Ref, which
  defineReactors fills.
- go-web-sdk-template: the integration suite is byte-for-byte unchanged except the harness's
  Ready comment; the unit tests pin /readyz's exact checks.
- go-web-sdk-template: a package-main test pins the config-load failure by calling cmd/server's
  run. A missing config.json does not fail the load, so the test writes a rejected key.
- go-web-sdk-template: the standards review made the top-layer pin observe a real Run's Build
  rather than copy Run's roots, and added TestRun_BuildFailureExitsOne.
- go-web-sdk-template: graph_test.go keeps its small helpers in the file rather than in an
  internal apptest package.
- go-web-sdk-template: context/scaffolding-cli.md names a Nodes field and defineInfrastructure in
  place of the removed layer structs.
- go-web-service: lifecycle pins written on v0.5.0 before the library upgrade; Go 1.27.2 and
  Mimir 3.2.2 go first, since neither touches the stage API.
- go-web-service: the serving pins (/readyz order and headers, ready address, taken port,
  shutdown bound) are integration cases, written on v0.5.0.
- go-web-service: the wake node's value has no Ready method, so /readyz stays at five checks;
  the sweeper's check already covers the Waker.
- go-web-service: the After-telemetry edge sits on database and storage only, with a structural
  test that telemetry is below every other participant.
- go-web-service: a node per value two or more nodes use; a value one node uses is built inside
  that node. Nodes: config, logger, telemetry, database, storage, sql, files, wake, gate,
  organization, document, schema, sweeper, readiness, router, server ("files" because "storage"
  keeps naming the object store on /readyz).
- go-web-service: the sweeper's runtime-failure pin moves sdk's two Coordinator tests onto a
  one-node graph with no Monitor call, plus a compile-time proof of Subsystem,
  ReadinessChecker, and Monitored; no app-level failure test.
- go-web-service: the integration harness's "root lifecycle stage" comment changes.
- go-web-service: the shutdown-bound pin stalls a raw-TCP upload that sends part of its
  declared body, with APP_SHUTDOWN_TIMEOUT=1s.
- go-web-service: sync repoints lifecycle-and-context.md and composition-terms.md from
  spike-cli-architecture to go-core; cli's template task repoints the other three.
- go-web-service: messaging.md's drain paragraph states the rule only; the relay edge is
  v1.messaging's.
- go-web-service: the workspace's role interfaces were inventoried (no duplicate method sets;
  go-core infers a check from any Ready() bool). The findings go in cli-applications.md and
  are executed as cli's first task, role-interfaces, which decides candidates a–d and folds in
  go-observability's Shutdown-after-failed-Start crash and go-database's lc.Add doc examples;
  the Gate stays with v1.messaging. Not a go-core-graph task, since go-core v0.6.0 is released.
- go-web-service: the ready-address pin binds APP_SERVER_PORT=0. The shutdown pin waits for the
  upload's 100 Continue and bounds the exit below by the timeout and above by a 3s margin.
- go-web-service: the taken-port pin matches the log record's error field ("startup:" naming
  server), not the "service failed" message.
- go-web-service: the router's middleware Uses the telemetry node, since an After edge never
  builds its target and the tracing middleware records to telemetry's providers.
- go-web-service: the sweeper orders itself After schema, which keeps /readyz's order and the
  sweep after the schema.
- go-web-service: app.Telemetry and app.Wake are exported, as node value types. Node constructors
  don't wrap their errors in their own name, since the graph labels them.
- go-web-service: slice 4 reworded integration-test comments and failure messages beyond the
  harness comment, changing no assertion.
- go-web-service: the README gained "Adding a service"; [Unreleased]'s stage-era entries were
  reworded and the stage-table entry deleted.
- go-web-service: the standards review moved the config-load-failure pin to the integration case
  TestStartup_ConfigLoadFailure, on the black-box tier; the template keeps its package-main test.
- go-web-service: no internal/apptest package, since graph_test.go holds assertion helpers, not
  fixtures.
- go-web-service: architect, at the session brief: only the composition root, its test
  fixtures, and the SDKs that run a graph import `graph` — recorded as an architecture
  principle at sync. graph's ergonomics (string-keyed nodes, opaque construction and layering)
  go to the backlog as graph-ergonomics.

## Pending edits

- coordinator · roadmap: the header comment — drop "go-core-graph takes that slot first: it
  promotes spike-cli-architecture's graph and lifecycle into go-core and moves the web stack
  onto them. Once go-core-graph syncs," so the paragraph opens "storage-s3 takes the slot for two
  sessions: it brings spike-s3-storage's s3 provider into go-storage v0.5.0."
- architecture · go-elemental/principles/lifecycle-and-context.md: drop the lifecycle.Service,
  numbered Stage, and hooks-beside-subsystems text, the "shutdown hook" in "Cold start, hot
  start, drain", and "does not follow it yet". go-core v0.6.0's Coordinator infers each value's
  part from Starter, Stopper, and Subsystem, its checks from ReadinessChecker, its readiness
  reporter from a lifecycle.Readiness node, and its runtime failures from Monitored. Three
  sentences change with it:
    - "A web service's composition root declares each subsystem ... hooks and monitors." becomes
      "A web service's composition root describes its subsystems as nodes of one go-core
      dependency graph, and the coordinator runs the built graph; nothing registers with it."
    - "Not every file of the composition root declares a subsystem. ... declares none and
      imports no part of this package" becomes "Not every node of the composition root is a
      subsystem. A layer of the architecture with no resource and nothing that runs, such as the
      Elemental Architecture's Domain Service composition, defines nodes whose values have no
      Start or Shutdown, so they take no part, and imports no part of this package".
    - "A subsystem declares runtime failure through a monitored channel" becomes "A value that
      can fail while running implements `Monitored` (`Err() <-chan error`), and the coordinator
      watches its channel".
- coordinator · cli-applications.md: "Promotions first" — go-core v0.6.0 holds `graph`, the
  graph-backed `lifecycle` with `Coordinator.Exec` as the one-shot form, and
  `processtest.Run(t, Cmd{Args, Stdin, Env}) Result{Stdout, Stderr, Code}` as the one-shot
  runner go-cli-sdk-template's integration suite uses.
- coordinator · roadmap: middleware/rate-limit — its main carries untagged go-core v0.6.0, httprate
  v0.16.1, and the go-web-sdk v0.14.0 requirement. Its next release dates them and can move to
  go-web-sdk v0.15.0.
- coordinator · roadmap: v1.messaging core-reactor — go-core v0.6.0 already ships
  lifecycle.Monitored, found by type assertion and read once startup completes, and infers
  Checks from ReadinessChecker. Drop Monitored from the task's additions; a value implementing
  Subsystem, ReadinessChecker, and Monitored supersedes Component and
  lc.Register(name, stage, c). The task's plan settles the reactor's shape.
- coordinator · roadmap: storage-s3 — builds on go-storage v0.5.0, the go-core-graph release.
- architecture · go-elemental/principles/topology-and-naming.md: the composition-root paragraph
  ("The composition root, `internal/app`, is laid out ...") becomes: "The composition root,
  `internal/app`, describes the application as one dependency graph on go-core's `graph`, one
  file per layer of the architecture. Each layer file defines its layer's nodes in one define
  function: one file the infrastructure services, one the administrative services, one the
  domain services, and one the reactors, and the administrative and domain files also build
  their mounts. `server.go` defines the request edge: the readiness the probes report, the
  router, and the server. The list of mounts and the middleware stack are each a file of their
  own, and a layer the application adds, such as go-web-service's telemetry, is one more file.
  Beside them, `app.go` holds the `Nodes` value, one handle per node; `New`, which describes the
  graph and cannot fail; and `Run`, which builds the graph and runs it under the lifecycle
  coordinator. The layer files are the architecture's layer list, and extending the application
  means defining a node in its layer's define function, a reactor's node also joining
  `Nodes.Reactors`, while the signatures, the entrypoint, and `Run` stay untouched. A package
  that the layers share and that must not import the composition root, the application's
  database infrastructure for one, lives at the root level."
- architecture · go-elemental/principles/lifecycle-and-context.md: "The wiring rule" — under
  the graph, a composition root's wiring defects panic during Run's Build ("graph: " panics),
  not at New; a constructor's error returns from Build and the service exits 1. "a registration
  after the coordinator has run" becomes "a lifecycle.Readiness bound to a second coordinator".
- coordinator · roadmap: cli goal — gains repos = ["go-core", "go-web-sdk",
  "go-observability", "go-database"]; cli's first plan adds go-cli-sdk and
  go-cli-sdk-template, and sets the root, as before.
- coordinator · roadmap: cli goal summary — "Builds on go-core-graph's release." becomes
  "Builds on the go-core release its first task, role-interfaces, cuts."
- coordinator · roadmap: new first cli task, ahead of sdk:
    [goals.cli.tasks.role-interfaces]
    name = "Role interfaces across the stack"
    repos = ["go-core", "go-web-sdk", "go-observability", "go-database"]
    summary = '''
    Decides the role-interface findings in `cli-applications.md` ("Role interfaces")
    before go-cli-sdk adds the CLI's own: (a) go-core infers a /readyz check from any
    `Ready() bool`; (b) `web.Doctor`'s home; (c) `lifecycle.Subsystem`, never asserted at
    runtime; (d) the consumer-side `Ready()` in `Source[T]` and admin/storage's `Store`. A
    change (d) rules for go-web-service lands as a pending edit on v1.messaging's
    service-reactors task. The Gate stays with v1.messaging's core-reactor. Also fixes
    go-observability's `Telemetry.Shutdown` after a failed Start (nil providers under go-core
    v0.6.0) and go-database's `lc.Add(lifecycle.Service{...})` doc examples. Its plan
    settles the tags it releases; go-cli-sdk builds on its go-core release.
    '''
    context = ["standards-lab/context/cli-applications.md"]
- coordinator · roadmap: cli · sdk task summary — "at go-core v0.6.0's graph and
  lifecycle" becomes "at the graph and lifecycle of the go-core release role-interfaces
  cuts".
- coordinator · roadmap: cli · template task summary — add "Repoints the architecture's
  three remaining spike-cli-architecture links (tests-and-docs.md's apptest fixtures,
  domain-files.md's files domain, baseline-standards.md's schema commands) to
  go-cli-sdk-template once it holds those examples."
- coordinator · cli-applications.md: add before "## The plan":
    ## Role interfaces

    go-core v0.6.0's Coordinator finds each value's part from its methods, so a role
    interface is a contract every value in a graph can meet by accident. The
    go-core-graph goal took inventory before cli adds the CLI's own:

    | Interface | Home | Methods | Implemented by | Read by |
    |---|---|---|---|---|
    | Starter | go-core lifecycle | `Start(ctx) error` | database.DB, storage.Store, admin.Service, observability.Telemetry, web.Server, sdk.Reactor | the Coordinator, by assertion |
    | Stopper | go-core lifecycle | `Shutdown(ctx) error` | the same except admin.Service | the Coordinator, by assertion |
    | Subsystem | go-core lifecycle | Starter + Stopper | DB, Store, Telemetry, Server, Reactor | docs and compile-time proofs only |
    | ReadinessChecker | go-core lifecycle | `Ready() bool` | Coordinator, Readiness, DB, Store, admin.Service, Reactor, Waker | `Coordinator.Checks`, by assertion; `web.Doctor` |
    | Monitored | go-core lifecycle | `Err() <-chan error` | web.Server, Reactor | the Coordinator, by assertion |
    | Doctor | go-web-sdk | ReadinessChecker + `Checks()` | Coordinator, Readiness | `RegisterHealth` |
    | Mounter | go-web-sdk | `Handle(pattern, handler)` | http.ServeMux, web.Router | `RegisterHealth` |
    | Source[T] | go-web-service sdk | `Receive`, `Ready() bool` | Waker, Every | Reactor |

    No two packages declare the same method set. The candidates, which cli's
    role-interfaces task decides:

    a. Any `Ready() bool` becomes a /readyz check. A value with Ready in another sense
       joins the probe: go-web-service hides its Waker behind a value with no Ready.
    b. `web.Doctor` lives in go-web-sdk, though only go-core's two types implement it.
    c. `Subsystem` names Starter plus Stopper, but the Coordinator never asserts it.
    d. `Source[T]` and admin/storage's `Store` declare `Ready()` on the consumer side,
       which (a) makes a readiness check wherever such a value is a node.

    The Gate's consumer views (`Shared`, `Exclusive`) stay with v1.messaging's
    core-reactor. Two fixes ride along: go-observability v0.1.0's `Telemetry.Shutdown`
    dereferences providers a failed Start never set, which v0.6.0 now reaches, and
    go-database v0.7.0's doc examples still call `lc.Add(lifecycle.Service{...})`.
- coordinator · roadmap: v1.messaging template task summary — becomes "The template's
  `reactors.go` defines each reactor as a graph node on go-core's `reactor`, appended to
  `Nodes.Reactors`. The template stays engine-free and imports no go-messaging."
- coordinator · roadmap: v1.messaging service-reactors task summary — "every component
  registers through `Register`" becomes "each component is a graph node whose part go-core
  infers".
- coordinator · messaging.md: the "A broker constructs without I/O ..." bullet — "starts as
  a stage-0 lifecycle component" becomes "starts as a lowest-layer graph node".
- coordinator · messaging.md: "Lifecycle registration" — replace the section's body:
    Infrastructure joins the lifecycle through `Start`, `Shutdown`, and `Ready`
    (go-database's pool, go-storage's store, the broker). Under go-core v0.6.0 each is a
    graph node's value, and the Coordinator infers its part: Starter, Stopper, and
    ReadinessChecker from those methods, and Monitored from `Err() <-chan error`. No
    composition root copies methods into a `lifecycle.Service`, and nothing registers.
    The coordinator's two error-drop windows (after the signal, and at the drain
    deadline) stay as they are, since the reactor covers both itself.

    The graph's layers order the drain: a node drains before every node it uses or
    orders After, so the server drains first, and the database and the broker drain after
    every node that uses them.
- architecture · go-elemental/principles/lifecycle-and-context.md: the spike-cli-architecture
  sentence becomes "[go-core](https://github.com/standards-lab/go-core)'s `lifecycle`
  package realizes the rule." (with the existing edit dropping "does not follow it yet").
- architecture · principles/composition-terms.md: "[spike-cli-architecture](...) expresses
  go-web-service's startup order as a graph and runs a CLI and a graph shaped like the web
  service on the same coordinator" becomes "[go-core](https://github.com/standards-lab/go-core)'s
  `graph` and `lifecycle` run both:
  [go-web-service](https://github.com/standards-lab/go-web-service) describes its
  composition root as one graph on that coordinator, and the CLI application type builds
  on the same one."
- coordinator · messaging.md: the paragraph after the definitions — "`go-web-service/internal/app/reactors.go`
  is the service's reactors layer, registered on the lifecycle coordinator," becomes
  "`go-web-service/internal/app/reactors.go` is the service's reactors layer, which defines each
  reactor as a graph node and appends it to `Nodes.Reactors`,".
- coordinator · auth-strategy.md: "`internal/app` gains `auth.go` as a layer file, ... rather than
  passing `infra` itself." becomes "`internal/app` gains `auth.go` as a layer file, parallel to
  `admin.go` and `domain.go`, whose define function defines the verifier, the resolver, and the
  evaluator as graph nodes built from the infrastructure nodes they use; `mountAPI` reads them
  from `Nodes` with `Use`, as it reads every other node." go-web-service has no layer structs
  and no `infra` value to thread.
- coordinator · migration-sets.md: "add the shipper's statements to its own verify stage" becomes
  "add the shipper's statements to the ones its startup verifies"; go-web-service has no verify
  stage, and its schema node's Start verifies every registered store's statements.
- coordinator · roadmap: v1.data evaluation task summary — add "It also decides whether
  go-web-sdk-template's package-main test of the config-load failure moves to its integration
  tier, as go-web-service's TestStartup_ConfigLoadFailure did."
- architecture · principles/composition-terms.md: after the Graph definition, add "Only the
  composition root, its test fixtures, and the SDKs that run a graph import `graph`. Every
  package below the root takes its dependencies as plain values through its constructor and
  never sees a node, a scope, or the system, so the graph's API can change without reaching
  them."
- coordinator · roadmap: backlog += "graph-ergonomics" (after "v1.harness"), with a goal table:
  name "graph ergonomics"; summary: go-core's `graph` works but its API is early: nodes are
  string-keyed, and how a node is constructed and which layer it lands in is hard to see from
  the define functions. Iron these out as more applications use the graph; the
  composition-root principle (composition-terms.md) keeps the breaks inside composition roots,
  their fixtures, and the SDKs that run a graph. cli's plan may pull it forward if go-cli-sdk's
  WithGraph meets the same friction.
