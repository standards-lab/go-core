# goal · graph-ergonomics

- **State:** brief ready
- **Task:** design-review
- **Branch:** design-review

## Tasks

1. [ ] design-review

## Task brief · design-review

```

Problem       go-core's graph and lifecycle behave correctly but read badly: names written three
              times, constructors that return constructors, layer placement hidden in scattered
              s.After calls, value-only closures, name-only introspection, and a web composition
              root that repeats ~30 lines the CLI side gets from cli.Run. go-ai and the client SDK
              build on this API in wave C, so its design must be settled — and proven in a spike —
              first.
Behaviors     1. go-core trails nothing: its currency command exits 0 and its check passes.
              2. go-core's working context holds a design note for the redesign, stating:
                 a. the pain points, each with consumer evidence (repo and site);
                 b. the behavior that must not change: edges discovered through s.Use (so a CLI
                    run builds only what a command declares), build-once, longest-path layers,
                    panics for wiring mistakes, the participation interfaces, and lifecycle's
                    startup, readiness, and shutdown semantics;
                 c. the one candidate API from decisions 9–14, with code sketches of a web
                    composition root, a CLI command, and a layer test written against it;
                 d. the open forks the spike tests (arity adapters beside methods on *Nodes);
                 e. the rejected alternatives (uber-go/fx, dig, samber/do, google/wire) and why;
                 f. the spike's question and the decision its answer changes;
                 g. the migration: one breaking go-core release (no deprecation period), then one
                    migration task per tier (libraries with go-cli-sdk, templates,
                    go-web-service), go-web-sdk gaining web.Run, and composition-terms.md amended
                    for CLI command packages once the spike settles the API.
              3. go-core's context index points to the note.
              4. The goal record holds decisions 1–14 and a pending edit correcting the goal
                 summary's "nodes are string-keyed" claim.
Test seams    go-core's check (`mise run check`) and currency command; the note is read for
              coherence against the code it cites.
Slices        1. Upgrade go-core: go 1.27.3, golangci-lint 2.15.0, mise-action v5.2.7,
                 create-gh-release-action v1.12.0; currency exits 0, check passes.
              2. The design note and its index entry (behaviors 2–3).
Out of scope  Any change to graph or lifecycle's exported API; the spike repository; changes to
              the nine consumers, standards-lab, or the architecture repository (pending edits
              only).
Ship order    Before this task ships, the architect runs `marathon experiment graph` (creating
              experiment.graph and spike-graph) and `marathon plan` (staging spike-graph into
              active, adding graph-ergonomics' release and per-tier migration tasks waiting on
              experiment.graph's intake, and naming the spike a prerequisite of
              capstone.data-intelligence), so design-review is not the goal's last task and
              does not sync it.
Door          two-way (documentation and tooling bumps; nothing released)
```

## Progress

slices 2/2 committed · standards done · spec done (1 round, gaps closed) · editor done

## Decisions

- design-review: redesign go-core's own graph and lifecycle; rejected uber-go/fx and dig (OnStart/OnStop hooks the lifecycle principle retired, run-time reflection), samber/do v2 (no layered concurrent startup), google/wire (archived 2025-08); go-core stays standard-library only.
- design-review: the design lives in a go-core working-context note.
- design-review: the redesign is proven in a spike repository (experiment.graph, spike-graph), whose findings shape go-core and its consumers; no test-only sketch in go-core.
- design-review: migrate everything at once from the spike's findings: one breaking go-core release, no deprecation period.
- design-review: keep the graph behavior the CLI app type relies on (a command declares its nodes so a run builds only what it uses), optimizing its ergonomics; composition-terms.md is amended once the spike settles the API.
- design-review: design-review stays a go-core task; experiment.graph's spike goes into `active` and is a prerequisite of capstone.data-intelligence.
- design-review: the note proposes one candidate API with its open forks named.
- design-review: later tasks are one breaking go-core release, then one migration task per tier (libraries with go-cli-sdk, templates, go-web-service).
- design-review: order-only edges and groups are declared at Define (graph.After, graph.In, a group handle a Build expands); s.After inside constructors is removed.
- design-review: constructors are methods on the *Nodes struct used as method values; typed arity adapters are an open fork for the spike.
- design-review: add g.Value(name, v) for fixed values and graph.Fresh[T](g, name) for a new *T per Build.
- design-review: keep explicit string node names; a fixture check that each name matches its field replaces the `// "name"` comments.
- design-review: System and Observe expose node handles (Ref on Dependency, Observe(func(Ref)), System.LayerOf), plus System.String() printing the layer table.
- design-review: go-web-sdk gains web.Run matching go-cli-sdk's cli.Run, with lifecycle.Config supplied one way on both sides.
- design-review: the tooling bumps take no CHANGELOG entry; none has been recorded for earlier bumps.
- design-review: golangci-lint 2.15's errcheck findings are fixed in code, not in lint config: processtest.Run and a reactor test now signal completion by closing a chan struct{}.
- design-review: the note leaves three details to the spike: what web.Run returns and reports, and where the ready, failed, and stopped logs go; the rule the name check applies; and how a *Nodes method reaches a plain New argument such as an io.Writer field.

## Pending edits

- coordinator · roadmap: correct graph-ergonomics' summary: nodes are typed *Node[T] handles; the string is a required name that introspection exposes as the only identity, not the key.
- coordinator · standards-lab/context/auth-strategy.md:199-200: the auth plan names an auth.go layer file with a define function; revise it once the spike settles the constructor shape.
- architecture · principles/composition-terms.md: amend it to admit CLI command packages taking the nodes they declare, once the spike settles the API (decision 5).
- architecture · standards/go-elemental/principles/domain-files.md:47-48: reconcile it with the composition-terms.md amendment.
