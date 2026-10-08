# goal · go-core-graph

- **State:** building
- **Task:** go-web-sdk
- **Branch:** go-web-sdk

## Tasks

1. [x] go-core
2. [x] go-storage
3. [ ] go-web-sdk
4. [ ] go-web-sdk-template
5. [ ] go-web-service

## Task brief · go-web-sdk

```
## Task brief · go-core-graph · go-web-sdk
Problem       go-web-sdk requires go-core v0.5.0, whose lifecycle.Service, Add, stages, and
              OnStartup v0.6.0 retires. RegisterHealth takes only a *lifecycle.Coordinator,
              which exists only after Build, so a health handler's graph node can't call it.
              Its doc example and tests are written against the stage table. The template and
              go-web-service can't move onto the graph until go-web-sdk takes v0.6.0 and
              RegisterHealth accepts the lifecycle.Readiness their handler node uses.
Behaviors
  1. Both modules build and the check passes on go-core v0.6.0. middleware/rate-limit
     also takes httprate v0.16.1, and currency reports nothing.
  2. web.Doctor is an exported interface that embeds lifecycle.ReadinessChecker and adds
     Checks() []lifecycle.Check. Compile-time proofs show *lifecycle.Coordinator and
     *lifecycle.Readiness both satisfy it.
  3. RegisterHealth(m, d Doctor, notReady) mounts GET /healthz and GET /readyz. A caller
     that passes a *lifecycle.Coordinator compiles unchanged.
  4. RegisterHealth panics at wiring on a nil Doctor, with a message naming the fix.
  5. /healthz answers as in v0.14.0: 200, application/json, Cache-Control no-store,
     {"status":"ok"}. Any other method gets 405.
  6. /readyz answers as in v0.14.0:
     - When every check is ready: 200, application/json, no-store,
       {"status":"ready","checks":[...]}, with "lifecycle" first and then the Doctor's checks.
     - Otherwise: 503, application/problem+json, no-store. The default problem is
       about:blank, "Service Unavailable", detail "one or more readiness checks failed",
       instance = the path. The checks member lists every entry. The caller's Type, Title,
       Detail and Extras show through; its Status and any "checks" it sets are replaced.
     - A nil Checker is not ready. Readiness with zero checks is ready and sends no checks
       member.
  7. RegisterHealth reads the Doctor on every request:
     - Given an unbound lifecycle.Readiness inside a node constructor during Build, /readyz
       reports only "lifecycle": false.
     - Once New binds it, /readyz reports "lifecycle" and then the System's
       ReadinessChecker nodes, named by node, in layer then definition order. Nothing else
       registers anything.
  8. Over a graph Run with a lifecycle.Readiness node:
     - /readyz is 503 while a Starter blocks startup, 200 once OnReady fires, and 503
       after Run returns.
     - /healthz is 200 throughout.
  9. *web.Server as a node's value is a lifecycle.Subsystem and a lifecycle.Monitored
     (compile-time proof). It adds no entry to /readyz.
 10. The package doc's lifecycle section shows the Server as a graph node's value. The
     Coordinator starts it, stops it, and watches its Err without a Monitor call.
     RegisterHealth is shown over a lifecycle.Readiness node's value. No example, godoc or
     README text names lifecycle.Service, Add, a stage, the root stage, or a service added
     after the call. The doc's list of exported names includes Doctor.
 11. The base module's CHANGELOG v0.15.0 section records:
     - the go-core v0.6.0 requirement, as breaking for importers still on
       lifecycle.Service;
     - that RegisterHealth takes a Doctor, so a Readiness node's value works, and panics on
       a nil Doctor;
     - that the checks after "lifecycle" follow the System's layer order.
     rate-limit's [Unreleased] records go-core v0.6.0 and httprate v0.16.1's bucketing
     change: an IPv4-mapped IPv6 address shares the IPv4 address's counter.
Test seams    RegisterHealth over an http.ServeMux, probed by GET; compile-time interface
              assertions on Coordinator, Readiness, and Server
Slices
  1. upgrade: go-core v0.6.0 in both modules, httprate v0.16.1 in rate-limit. The existing
     tests move onto graph, New(sys, cfg), and Run(ctx), keeping their v0.14.0 assertions:
     a blocking Starter node replaces OnStartup, and a Readiness node over a late-defined
     checker replaces the Add regression. Done when currency exits 0 and the check passes
     (behaviors 1, 5, 6, 8).
  2. Doctor: RegisterHealth takes web.Doctor and panics on nil; the interface and Server
     proofs; package doc, Server and webtest godoc, README wording; both CHANGELOG
     sections (behaviors 2-4, 7, 9-11).
Out of scope  tagging middleware/rate-limit; changes to Liveness, Readiness, or the probe
              bodies; typed-nil detection; Doctor in go-core; the template's and
              go-web-service's adoption; architecture page edits (goal sync)
Door          two-way until tagged; one-way at the v0.15.0 tag, since a published version
              can't be withdrawn
Release       v0.15.0 (go-web-sdk, base module); middleware/rate-limit untagged
```

## Progress

slices 2/2 committed · standards ✓ · spec — · editor —

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

## Pending edits

- coordinator · roadmap: go-web-sdk task — RegisterHealth takes an interface with Ready() and
  Checks(), satisfied by both the Coordinator and lifecycle.Readiness.
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
- coordinator · roadmap: go-web-sdk-template and go-web-service tasks — the health handler's node
  uses a lifecycle.Readiness node; the server and the sweeper report runtime failures by
  implementing lifecycle.Monitored; telemetry becomes a layer-0 node whose value implements
  Starter and Stopper.
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
