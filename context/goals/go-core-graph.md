# goal · go-core-graph

- **State:** building
- **Task:** go-web-sdk-template
- **Branch:** go-web-sdk-template

## Tasks

1. [x] go-core
2. [x] go-storage
3. [x] go-web-sdk
4. [ ] go-web-sdk-template
5. [ ] go-web-service

## Task brief · go-web-sdk-template

```
## Task brief · go-core-graph · go-web-sdk-template
Problem       The template requires go-core v0.5.0 and go-web-sdk v0.14.0. Its composition
              root registers the server on the stage table (lifecycle.Add at StageRoot),
              watches server.Err() through Monitor, and mounts /readyz on the Coordinator.
              go-core v0.6.0 retires Add, Service, the stages, StageRoot, and Run's timeout;
              go-web-sdk v0.15.0's RegisterHealth takes a web.Doctor. Every generated
              service seeds from an API that no longer exists. The exact probe answers, a
              startup failure, the shutdown bound, and OnReady after bind aren't pinned, so
              nothing holds the move to today's behavior.
Behaviors
  1. The template builds and the check passes on go-core v0.6.0 and go-web-sdk v0.15.0;
     currency reports nothing. Go 1.27.1 and golangci-lint 2.14.0 stay.
  2. Configuration loads at the entrypoint with its contract unchanged:
     - shutdown_timeout is a top-level config.json key, default 10s.
     - APP_SHUTDOWN_TIMEOUT overrides it; an unparsable value is an error naming
       APP_SHUTDOWN_TIMEOUT, a non-positive one "shutdown_timeout must be positive";
       an empty prefix reads no override.
     - An overlay's shutdown_timeout replaces the base's; one that leaves it unset keeps it.
     - A file with a "Config" or "Env" key is rejected as unknown.
     - A config load failure exits nonzero with "config load failed".
  3. The service's Config embeds lifecycle.Config by value, untagged, and drops its own
     shutdown timeout. A literal sets the timeout as a promoted key.
  4. The app package describes its graph in one exported Nodes value: Logger, Config,
     Readiness, Router, Server. Each layer keeps a define function. The reactor layer lists
     the Build roots, and Server takes an After edge to each, so it stays topmost. Neither
     the stage table nor the layer structs remain.
  5. New(cfg, w) describes the graph only and can't fail; the entrypoint's "app init
     failed" path goes. App.Graph() and App.Nodes() are exported, so a caller can Replace
     or Observe a node before Run.
  6. Run(ctx) builds the graph, hands the Coordinator the Config node's lifecycle block,
     and runs it:
     - 0 after a clean drain, logging "server stopped".
     - 1 on a startup, build, or shutdown failure, logging "service failed" with the error.
     - A second Run panics before anything is built.
  7. /healthz answers 200, application/json, Cache-Control no-store, {"status":"ok"}, with
     a request ID header.
  8. /readyz answers 200, application/json, no-store, {"status":"ready","checks":[...]},
     checks exactly ["lifecycle"]. The health route's web.Doctor is the Readiness node's
     value, not the Coordinator.
  9. OnReady logs "server ready addr=<addr>" once the server has bound, and that address
     answers /readyz 200.
 10. A taken port makes Run return 1 with a "startup:" error naming server; "server ready"
     is never logged.
 11. A connection stalled past shutdown_timeout makes Run return 1 within shutdown_timeout
     plus a margin.
 12. Server is alone in the built graph's top layer. The root makes no Monitor call:
     *web.Server is inferred Subsystem and Monitored, so its runtime error ends the run.
     Startup order, readiness held until every check passes, the monitored error, and
     reverse drain are pinned in go-core's lifecycle and go-web-sdk's health/server tests.
 13. The integration suite passes unchanged: boot, probe, drain; two instances; the local
     overlay.
 14. No README, STANDARDS, context, package doc, or harness comment names a stage, the
     stage table, StageRoot, lifecycle.Service, Add, or Monitor on the server. "How to add
     a service" defines a node in its layer's define function, part inferred, and a
     reactor as a Build root.
 15. The template module's CHANGELOG template/v0.12.0 section, dated and linked, carries
     the [Unreleased] entries and records:
     - the go-core v0.6.0 and go-web-sdk v0.15.0 requirement;
     - for a generated service porting: stage table and layer structs give way to Nodes
       and define functions; New can't fail and Run builds; Config embeds
       lifecycle.Config (cfg.Config is the lifecycle block); RegisterHealth takes the
       Readiness node; lc.Monitor(server.Err()) dropped, since it would watch twice;
       /readyz lists checks after "lifecycle" in layer, then definition, order;
     - that the libraries' own breaks are in their CHANGELOGs.
Test seams    New and Run over HTTP, the log as the ready signal; Config's Merge, Finalize,
              and Load over files; App.Graph() built and read by layer (behavior 12); the
              integration process runner, unchanged
Slices
  1. pins on v0.5.0 — an exception to upgrade-first (round 1, Q1): v0.6.0 removes the API
     the pins would be written against. Exact probe answers, readiness at the logged
     address, startup failure, shutdown bound, and the config contract, pinned on the
     stage table. Done when the check and integration tier pass on v0.5.0
     (behaviors 2, 7-11, 13).
  2. upgrade and move: go-core v0.6.0, go-web-sdk pinned @v0.15.0; Config embeds
     lifecycle.Config; Nodes graph, New and Run, Readiness node as Doctor, no Monitor;
     stages and layer structs gone. Done when currency exits 0, the check passes, and
     slice 1's pins and integration pass with only New's call site changed
     (behaviors 1-11, 13).
  3. structural pin, docs, CHANGELOG: Server alone in the top layer; stage text removed;
     the v0.12.0 section. Done when the check passes and the docs name no stage
     (behaviors 12, 14, 15).
Out of scope  go-web-service's move, telemetry, sweeper; a test-only root hook or fake
              participant; probe body or APP_ namespace changes; middleware/rate-limit;
              architecture page edits (goal sync)
Door          two-way until tagged; one-way at the template/v0.12.0 tag
Release       template/v0.12.0 (go-web-sdk-template, template module)
```

## Progress

slices 0/3 committed · standards — · spec — · editor —

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

## Pending edits

- coordinator · roadmap: go-web-sdk task — RegisterHealth takes `web.Doctor`, an interface with
  Ready() and Checks() that both the Coordinator and lifecycle.Readiness satisfy.
- coordinator · roadmap: go-web-service task — sdk.Waker implements Ready() bool but isn't a
  Check today; as its own node, inferred Checks would add a "wake" entry to /readyz, against
  "/readyz answering as today". Its plan decides.
- architecture · go-elemental/principles/lifecycle-and-context.md: drop the lifecycle.Service,
  numbered Stage, and hooks-beside-subsystems text, the "shutdown hook" in "Cold start, hot
  start, drain", and "does not follow it yet". go-core v0.6.0's Coordinator infers each value's
  part from Starter, Stopper, and Subsystem, its checks from ReadinessChecker, its readiness
  reporter from a lifecycle.Readiness node, and its runtime failures from Monitored.
- coordinator · cli-applications.md: "Promotions first" — go-core v0.6.0 holds `graph`, the
  graph-backed `lifecycle` with `Coordinator.Exec` as the one-shot form, and
  `processtest.Run(t, Cmd{Args, Stdin, Env}) Result{Stdout, Stderr, Code}` as the one-shot
  runner go-cli-sdk-template's integration suite uses.
- coordinator · roadmap: go-web-sdk-template and go-web-service tasks — both take go-web-sdk
  v0.15.0. The health handler's node passes a lifecycle.Readiness node's value to
  web.RegisterHealth as a web.Doctor, in place of lc. The sweeper reports runtime failures by
  implementing lifecycle.Monitored. Telemetry becomes a layer-0 node whose value implements
  Starter and Stopper.
- coordinator · roadmap: go-web-sdk-template and go-web-service tasks — a *web.Server node's
  value is inferred as a Subsystem and Monitored, so the composition root drops
  lc.Monitor(server.Err()), which would watch the channel twice.
- coordinator · roadmap: go-web-sdk-template and go-web-service tasks — /readyz lists "lifecycle"
  and then the ReadinessChecker nodes in layer then definition order, not stage order. "/readyz
  answering as today" holds only if the graph's layers keep the old stage order, so each task's
  tests pin the order.
- coordinator · roadmap: middleware/rate-limit — its main carries untagged go-core v0.6.0, httprate
  v0.16.1, and the go-web-sdk v0.14.0 requirement. Its next release dates them and can move to
  go-web-sdk v0.15.0.
- coordinator · roadmap: v1.messaging core-reactor — go-core v0.6.0 already ships
  lifecycle.Monitored, found by type assertion and read once startup completes, and infers
  Checks from ReadinessChecker. Drop Monitored from the task's additions; a value implementing
  Subsystem, ReadinessChecker, and Monitored supersedes Component and
  lc.Register(name, stage, c). The task's plan settles the reactor's shape.
- coordinator · roadmap: go-core-graph go-storage task summary — "Releases go-storage
  v0.5.0" (was v0.4.1); go-web-service task takes go-storage v0.5.0.
- coordinator · roadmap: storage-s3 — builds on go-storage v0.5.0, the go-core-graph release.
- coordinator · messaging.md: "Lifecycle registration" — go-storage v0.5.0's Store joins the
  Coordinator as a graph node's value, inferred as Starter, Stopper, and ReadinessChecker; no
  composition root copies its methods into a `lifecycle.Service`.
- coordinator · roadmap: go-web-service task — its Config meets the same lifecycle.Config
  embedding points as the template (by value, untagged, promoted literal key), and its
  stageSchema and stageReactors become Build roots with the server After each.
