# goal · go-core-graph

- **State:** building
- **Task:** go-core
- **Branch:** go-core

## Tasks

1. [ ] go-core
2. [ ] go-storage
3. [ ] go-web-sdk
4. [ ] go-web-sdk-template
5. [ ] go-web-service

## Task brief · go-core

```
## Task brief · go-core-graph · go-core
Problem       go-core's lifecycle orders subsystems by hand-numbered stages and function
              fields; the architecture calls for a computed graph whose values take part
              through interfaces. The cli goal and the web stack's move need graph, the
              graph-backed Coordinator, and a one-shot process runner in a go-core release.
Behaviors
  graph
   1. Define/Replace/Observe/Build/Use/After/Get/Layers behave as the spike's validate API:
      depth-first discovery, each node once per Build, longest-path layers in definition
      order, labelled constructor errors, documented wiring panics; standard library only.
  lifecycle
   2. New(sys, cfg) panics on a nil System or a non-positive ShutdownTimeout;
      Config.Finalize defaults 10s and reads <PREFIX>_SHUTDOWN_TIMEOUT.
   3. Layers start lowest first, each concurrently, under the run context that lives until
      shutdown; the first failure cancels the rest of the layer, its consequent
      cancellations are dropped, and no higher layer starts.
   4. Shutdown runs on every path in reverse layer order, each layer concurrently, under one
      fresh ShutdownTimeout budget, including a participant whose Start failed; a timeout
      adds one error wrapping DeadlineExceeded.
   5. A value takes part only as Starter, Stopper, or Subsystem; others are skipped.
   6. Ready is false until startup completes, true while running, false once draining
      begins; OnReady hooks run in order after the flip.
   7. Checks lists every System value implementing ReadinessChecker, named by node, in
      layer then definition order, excluding Readiness values.
   8. A Readiness node value reports the Coordinator's Ready and Checks once New binds it;
      unbound, it is not ready.
   9. A value implementing Monitored (Err() <-chan error), and any Monitor(ch), is watched
      from readiness: the first non-nil error ends the run as "run:"; nil and close are
      ignored.
  10. Run: a context end (including before or mid-startup) is a clean stop. Exec: fn runs
      only after full startup, under Run's runtime contract; its ctx is cancelled by a
      monitored failure, and its error joins shutdown's; a second Run or Exec panics.
  11. Errors keep go-core's labels ("startup:", "run:", "shutdown:", names, "drain timeout
      after"); registration after Run or Exec panics.
  processtest
  12. Run(t, Cmd{Args, Stdin, Env}) Result{Stdout, Stderr, Code} executes Main's build once;
      returns stdout, stderr, and exit code separately; a stall past Failsafe is
      interrupted, killed, and fails the test; logs the shell line and output.
  13. go-core's existing black-box tests pass, changed only for the break (Add, Service,
      stages, StageRoot, Run's timeout, OnStartup/OnShutdown, the zero-value Coordinator,
      Shutdown after a failed Start).
Test seams    graph's exported API; lifecycle.Coordinator over a built graph.System of
              recorder fakes; processtest.Run over the internal test program
Slices
  1. graph promoted with its tests
  2. lifecycle Coordinator core: New, Config, Exec, Run, Starter/Stopper/Subsystem,
     shutdown semantics; existing tests adapted
  3. readiness: Ready, OnReady, inferred Checks, Readiness binding
  4. monitoring: Monitored inference and Monitor, in Run and Exec
  5. processtest.Run one-shot runner
  6. docs and release prep: doc.go inventories, README, CHANGELOG v0.6.0 with the
     breaking list
  (no upgrade slice: go-core is current)
Out of scope  downstream adoption (go-web-sdk, go-storage, template, go-web-service);
              architecture page edits (goal sync); go-cli-sdk; changing graph's validated API
Door          two-way for the code until tagged; one-way at the v0.6.0 tag
Release       v0.6.0 (go-core, root)
```

## Progress

slices 6/6 committed · standards ✓ (f94363f) · spec ✓ no gaps · editor ✓

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
