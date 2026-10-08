# goal · go-core-graph

- **State:** idle
- **Task:** none
- **Branch:** none

## Tasks

1. [x] go-core
2. [x] go-storage
3. [ ] go-web-sdk
4. [ ] go-web-sdk-template
5. [ ] go-web-service

## Task brief · go-storage

```
## Task brief · go-core-graph · go-storage
Problem       go-storage requires go-core v0.5.0, whose lifecycle.Service, Add, and stages
              v0.6.0 retires. go-web-sdk-template and go-web-service can't move onto the
              graph while go-storage pins the old lifecycle and documents registration
              through it.
Behaviors
  1. go-storage builds and its check passes on go-core v0.6.0, required by name
     (the proxy's version list doesn't show v0.6.0 yet).
  2. *Store is proven at compile time to satisfy lifecycle.Subsystem (Starter and
     Stopper) and lifecycle.ReadinessChecker; the lifecycle.Service shape test is gone.
     go-storage's tests import neither graph nor a Coordinator.
  3. Store's existing start, ready, and shutdown tests pass unchanged, including
     Shutdown after a failed Start, which v0.6.0's Coordinator now calls.
  4. The package doc and README show Store as a graph node whose value the
     Coordinator starts, checks for readiness, and stops; no example uses
     lifecycle.Service or lc.Add.
  5. go-storage's exported API is unchanged.
  6. azureblob takes azcore v1.23.3; its go-storage requirement and tags are unchanged.
  7. The CHANGELOG's v0.5.0 section records the go-core v0.6.0 requirement as
     breaking for importers still on lifecycle.Service, and that Shutdown now
     follows a failed Start under the Coordinator.
Test seams    lifecycle interface assertions on *Store; Store's Start/Ready/Shutdown tests
Slices
  1. upgrade: go-core v0.6.0 by name, azcore v1.23.3; done when currency exits 0
     (go-core unreported until the proxy lists it) and the check passes
  2. graph registration: interface proof replaces the Service test; package doc,
     README, CHANGELOG v0.5.0
Out of scope  releasing azureblob; go-web-sdk, template, and go-web-service adoption;
              implementing Monitored on Store; architecture page edits (goal sync)
Door          two-way until tagged; one-way at the v0.5.0 tag
Release       v0.5.0 (go-storage, root)
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
