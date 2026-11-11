# goal · graph-ergonomics

- **State:** building
- **Task:** migrate-service
- **Branch:** migrate-service

## Tasks

1. [x] design-review
2. [x] release
3. [x] migrate-libraries
4. [x] migrate-templates
5. [ ] migrate-service

## Task brief · migrate-service

```
Problem       go-core v0.11.0 shipped the new graph and lifecycle API; the
              libraries, go-cli-sdk, and both templates have moved onto it.
              go-web-service and its slab CLI still pin go-core v0.10.0 and
              the old SDKs, and trail their Go, lint, CI action, and compose
              image pins. Their composition roots use removed APIs
              (Scope.After in seven constructors, Observe taking a name,
              cli.Run with WithGraph). The service's Run is a hand-written
              Build and Coordinator with a fallback logger, its reactors are
              a Ref slice the server ranges over, its layer tests scan
              Layers() by name, and both roots name their nodes in comments.
              The service is the workspace's reference consumer, and the last
              of the goal's nine repositories still on the old API.
Behaviors     1. go-web-service current: Go 1.27.4, golangci-lint 2.15.0,
                 jdx/mise-action v5.2.7, taiki-e/create-gh-release-action
                 v1.12.0; compose images grafana 13.3.1, loki 3.7.9, mimir
                 3.2.4, nats 2.15.2-alpine, otel-collector-contrib 0.165.0,
                 tempo 3.1.1. The service module requires go-core v0.11.0,
                 go-web-sdk v0.16.0 + rate-limit v0.4.0, go-database v0.11.0
                 + postgres v0.6.0, go-storage v0.7.0 + azureblob v0.6.0,
                 go-observability + otlp v0.3.0, grpc v1.85.0. slab requires
                 go-core v0.11.0, go-cli-sdk v0.3.0, go-web-sdk v0.16.0,
                 sqlate v0.6.1 + postgres v0.6.0, nats.go v1.55.0, x/sys
                 v0.50.0, x/term v0.48.0. slab stays its own module. The
                 currency command exits 0; the check and the integration
                 tier pass.
              2. The service App runs as one web.Service call; Run(ctx,
                 stderr) returns the exit code:
                 a. on bind it logs "server ready" with the bound address;
                 b. after a clean drain it logs "server stopped", returns 0;
                 c. on startup failure it logs "service failed" with an
                    error naming the node, returns 1, logs no "server ready";
                 d. on Build failure it writes "service failed: <error>" to
                    stderr naming the failing node, logs nothing, returns 1;
                 e. a second Run panics before building anything.
                 The entrypoint passes the process's stderr; the App keeps no
                 fallback logger, configuration, or writer for one.
              3. App.Roots() returns the server and the reactors group. A
                 Build of Roots plus the lifecycle-config node contains
                 exactly the nodes a real Run builds, compared by handle.
              4. Placement, asserted by handle (LayerOf, Has, Dependencies),
                 never by scanning Layers():
                 a. the server is alone in the top layer; a fixture node
                    above it shows the guard fails;
                 b. the reactors group's members are the sweeper and the
                    relay, each joined by its graph.In alone, each below the
                    server;
                 c. telemetry sits beneath every lifecycle participant;
                 d. the schema sits above the database, storage, and broker,
                    and below the sweeper and the relay; the relay sits above
                    the broker;
                 e. each participant is built in exactly one node.
              5. A Build of the root set reports no Drop. A fixture After of
                 an unbuilt node shows the guard fails, and a failing guard
                 prints the System's layer table.
              6. In the service root each node's order and group membership
                 are on its Define line, and no constructor body orders
                 anything: telemetry is defined ahead of the connections that
                 order after it; configuration and the logger's writer are
                 g.Value nodes; the gate, the nudge, and readiness are
                 g.Fresh; a "lifecycle config" node yields the lifecycle
                 block; every constructor that reads a node is a *Nodes
                 method. The reactor Ref slice is gone.
              7. In both modules each Nodes field's node/group name,
                 lowercased without spaces, equals the field name lowercased;
                 graphtest.CheckNames checks it; no Nodes field carries a
                 name comment.
              8. slab dispatches through cli.Program with its graph and
                 lifecycle-config node, exactly as before: existing tests
                 pass unchanged in what they expect (help, usage errors exit
                 2, version, settings layering, scenario listing, need-node
                 startup failures naming the mise task). Commands tested
                 alone run through a cli.Program with only a Root. The
                 build-recording fixture reads names from handles. "grafana
                 up" orders after "service" on its Define line.
              9. slab's layer test ranges over the root's Leaves(): for each
                 leaf, a Build of its node set plus the lifecycle-config node
                 succeeds and drops nothing, and wherever both are built,
                 grafana up sits above service. A fixture leaf, defined on
                 the App's own Graph() and Root(), ordered after a node its
                 path doesn't declare, shows the guard fails, with the layer
                 table.
              10. Package docs, READMEs, and the working context describe the
                  shipped API. Service: web.Service runs it, Build failure
                  goes to stderr, Roots is the root set, a reactor joins the
                  reactors group with graph.In. slab: cli.Program runs the
                  tree, and an App runs one command at a time because the
                  dispatcher restores flags. None names Scope.After, cli.Run,
                  WithGraph, the reactor slice, or name comments.
              11. The CHANGELOG's Unreleased section records, under Changed:
                  the Pins (go-core v0.11.0 and the libraries' new versions
                  in each module, grpc, nats.go, sqlate in slab), deferring
                  to their CHANGELOGs; a Build failure on stderr as "service
                  failed: <error>" in place of a log record; a Monitored
                  node's runtime failure naming its node ("run: <node>:
                  <error>"); slab reporting such a failure as one line; the
                  composition root's order at Define, reactors group, and
                  Roots. No tag. Tool, action, and image bumps take no entry
                  (design-review Decision).
Test seams    The repository's check (`mise run check`, which covers both
              modules), its integration tier (`mise run integration`), and its
              currency command. Service tests drive the App through New, Run,
              Roots, Graph, Nodes and read placement by handle (LayerOf, Has,
              Dependencies, Dropped). slab tests drive App.Run over buffered
              Streams with the apptest fixtures; the layer test iterates
              Leaves(). Docs are read against the code.
Slices        1. go-web-service current (every pin in Behavior 1), absorbing
                 only the break needed to compile and stay green: each After
                 moves to its Define line (telemetry defined first), reactors
                 join a reactors group the server orders after, observers
                 take handles; slab's App.Run and its commands' tests move to
                 cli.Program, and grafana up's After moves to its Define line.
                 Done: currency exits 0, check and integration tier pass. (1)
              2. Service runner: App.Run(ctx, stderr) as one web.Service call,
                 lifecycle-config node, Roots; the fallback logger goes; the
                 entrypoint passes stderr; the Build-failure test reads
                 stderr. Demo: `mise run serve` logs "server ready". (2, 3)
              3. Service Define lines: Value/Fresh nodes, *Nodes methods,
                 every order on its Define line, CheckNames replaces name
                 comments. (6, 7)
              4. Service layer tests on handles: root set vs a real Run,
                 server alone on top, reactor, telemetry, schema, and relay
                 placement, no Drop; each guard shown failing; the
                 Layers()-scanning helpers go. (3-5)
              5. slab names and layers: CheckNames, a recorder on handles,
                 the Leaves layer test with grafana up above service and its
                 fixture guard. (7-9)
              6. Docs: the service's package doc and READMEs, slab's package
                 doc, Run doc's reason, README and usage, the working
                 context. (10)
              7. CHANGELOG: the Unreleased entries. (11)
Out of scope  go-core, the SDKs, the libraries, and the templates; moving
              go-messaging or blobfs onto go-core v0.11.0; folding slab into
              the root module; any tag of go-web-service or slab; new
              reactor, admin, or domain nodes; either runner reading Dropped;
              the construction-and-execution conversions the service calls
              (api-maintenance); the architecture repository and the
              STANDARDS.md pointers pending at sync.
Door          Two-way: pull requests to main, no tag, nothing published; the
              compose image bumps reach only the local stack.
```

## Progress

slices 7/7 committed · standards — · spec — · editor —

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

- release: LayerOf stays a panicking int; System.Has(ref) bool is added for membership.
- release: no Group.Members; Group exports only Name.
- release: observers persist across every later Build of their Graph, documented.
- release: Replace swaps only the constructor and keeps the node's After and In.
- release: adopt Go 1.27.4 in an opening upgrade slice, with no CHANGELOG entry.
- release: fix all four unassigned spike findings here: String's ambiguity, a panicking observer escaping Build, concurrent Builds racing, no list of built nodes.
- release: the release is go-core v0.11.0.
- release: Define, Value, Fresh, and Group refuse a name containing a comma or a newline; String stays unquoted.
- release: an observer's panic re-panics with a "graph: " message naming the node and wrapping the original value.
- release: a sync.Mutex makes the Graph safe for concurrent use; Build holds it only to set built and copy the observers.
- release: System.Dependencies() []Dependency lists the built nodes in layer order, then definition order.
- release: Has returns false for a node from another Graph, as LayerOf treats it as not in the System; LayerOf and Has share one panic message for a group, "a group, which is not a node".
- release: an observer's panic value that is not an error is re-panicked as a %v string; an error is wrapped with %w.
- release: Ref's unexported methods split into owner() and nodes(), so a foreign group is rejected before its members are read; reading them first would race with the other Graph's Define.
- release: an After naming an empty group is reported by Dropped even in a Build of the full root set, so the empty-Dropped assertion catches a forgotten In.
- release: graph/doc.go keeps its overview list of wiring panics, which predates the task.
- release: the CHANGELOG's v0.11.0 section carries a "Migrating from v0.10.0" subsection, as v0.6.0 does.
- release: the CHANGELOG's v0.11.0 date is the day the tag is cut, 2026-11-11.

- migrate-libraries: web.Run ships as web.Service{Graph, LifecycleConfig, Logger, Server, Roots} with Run(ctx, stderr) int; Server and LifecycleConfig join every Build beside Roots, and a Logger the Build doesn't reach panics with a "web: " message before anything starts.
- migrate-libraries: web.Service's Server is the concrete *graph.Node[*web.Server], with no type parameter; go-web-sdk's web.Server is already the struct.
- migrate-libraries: a nil Service field or empty Roots panics with a "web: " message naming the field.
- migrate-libraries: web.Service's tests port spike-graph's runner tests, with a real *web.Server on port 0 and fake participant nodes for the failures.
- migrate-libraries: neither web.Service.Run nor cli.Program.Run reads System.Dropped; the composition roots' layer tests assert it is empty.
- migrate-libraries: go-cli-sdk reports a Monitored failure during a leaf or scenario step as one line; the run failure (from context.Cause) replaces an error that is only the context's end, and an unrelated leaf error is still reported beside it.
- migrate-libraries: cli.Program{Root, Graph, LifecycleConfig}.Run(ctx, args, streams) int replaces cli.Run, Option, and WithGraph; nil Root panics; Graph and LifecycleConfig are both set or both nil.
- migrate-libraries: (*cli.Command).Leaves() iter.Seq2[string, []graph.Ref] yields each leaf's command path and the union of nodes its path declares with Use.
- migrate-libraries: construction-and-execution rule: at most 4 required positional inputs plus at most one trailing options struct whose zero value is the default; no functional options; more inputs, or an API that runs a configured thing, becomes an exported struct with a method; a nil required field panics naming the field with the package prefix.
- migrate-libraries: graph.After and graph.In are a named exception to the construction-and-execution rule.
- migrate-libraries: the construction-and-execution rule covers test-support packages for new API; existing helpers convert at their next break.
- migrate-libraries: the other construction-and-execution candidates (sweep.New, middleware.RequestID, storagetest.NewFake, go-messaging New/Relay, sqlate TxOption, blobfs WriteFile/EnsureFile and their option families, go-core reactor/event options) go to a new maintenance goal, api-maintenance, handled after graph-ergonomics completes.
- migrate-libraries: the providers ripple and are tagged in this task: postgres, otlp, s3, azureblob, middleware/rate-limit.
- migrate-libraries: every touched repository is brought fully current (dependencies, container images, mise tools, CI actions), and each currency command exits 0.
- migrate-libraries: every module takes a minor release with its go-core (or base) requirement marked **Breaking:** (versions as in the Release line).
- migrate-libraries: an upgrade slice moves each provider to its base's latest existing tag; release preparation moves it to the base's new tag once that tag exists, since each check builds with GOWORK=off.
- migrate-libraries: postgres's untagged go-database v0.9.0 entry folds into its v0.6.0 section.
- migrate-libraries: the goal record changes on a migrate-libraries branch in go-core, the root, which the task otherwise leaves untouched, so it merges by pull request with the task.
- migrate-libraries: web.Service.Run's panics read "web: Service.Run requires <field>"; the panic for a Logger the Build didn't reach fires after a successful Build, so a Build failure goes to stderr first.
- migrate-libraries: empty Roots panics as the brief says, even when Server alone is the service; that Run doesn't write into the caller's Roots slice stays undocumented.
- migrate-libraries: cli.Program's panics read "cli: Program.Root is nil", and, when only one of Graph and LifecycleConfig is set, name the missing field then the set one; Use without a Graph panics "cli: <path>: Use declared but Program.Graph is nil".
- migrate-libraries: cli runs each leaf under a context detached from the run context; context.AfterFunc ends it with an unexported runFailure cause ("run: <node>: <error>"), which unwraps to the failure and matches context.Canceled. Err waits for that end and returns the caller's DeadlineExceeded. cli and scenario replace an error that is only the context's end with its cause only when errors.Is(cause, err), so a caller's own cause still reports "context canceled" or "context deadline exceeded".
- migrate-libraries: a leaf error that wraps ctx.Err() is reported beside the run failure; a caller cancel racing a Monitored failure may still print two lines; a caller cause that wraps context.Canceled is reported in place of the context's end.
- migrate-libraries: Leaves yields paths and node sets from the tree's root even when called on a subtree, a nil set for a leaf whose path declares no nodes, and nothing for a command with neither Run nor subcommands.
- migrate-libraries: each CHANGELOG lists every break go-core marks **Breaking:** between the module's previous requirement and v0.11.0; postgres's range starts at v0.7.0, which postgres/v0.5.0 pulled.
- migrate-libraries: go-storage's new CHANGELOG link lines are releases/tag links, following that repository's convention, not compare links.
- migrate-libraries: go-storage's store test asserts lifecycle.ReadyStarter and that the Coordinator lists the Store's check under its node name; s3's docs drop AWS_NEW_RETRIES_2026, which aws-sdk-go-v2 v1.48.0 no longer reads.
- migrate-libraries: tests whose constructors only returned a fixed or new zero value moved to g.Value and graph.Fresh.
- migrate-libraries: the principle is named construction-and-execution (page construction-and-execution.md), replacing the working name api-shape; the maintenance goal stays api-maintenance.
- migrate-libraries: the coordinator pending edits were applied ahead of sync at the architect's request (standards-lab#139); the architecture and STANDARDS.md edits stay for sync.
- migrate-templates: while the web template's reactors group is empty, its layer test allows exactly one Drop, the server's After of the reactors group; the server keeps graph.After(Reactors) as shipped.
- migrate-templates: the web template's test-only AppendReactor hook and its .golangci.yml note go; a fixture reactor joins with graph.In(Reactors).
- migrate-templates: both templates release, marked **Breaking:**: go-web-sdk-template template/v0.14.0, go-cli-sdk-template template/v0.2.0.
- migrate-templates: the upgrade slices carry only the break needed to compile and stay green (web: the reactors group; CLI: cli.Program); later slices finish the migration.
- migrate-templates: the goal record changes on a migrate-templates branch in go-core, the root, as migrate-libraries did.
- migrate-templates: the web template creates the reactors group in defineReactors, which New calls before defineServer, so the server's Define line can order after it.
- migrate-templates: the web App keeps its own run-once guard, because web.Service keeps no state between runs.
- migrate-templates: the web template's Roots lists only the server and the reactors group; the server reaches the config and logger nodes through Use.
- migrate-templates: the web template adds a "writer" g.Value node for the logger's writer, and groups the Nodes fields by layer.
- migrate-templates: the web router helpers (middleware, routes, mountAPI, mountAdmin) keep their (s, n) signatures.
- migrate-templates: the web template's rootDrops test helper checks whether the reactors group is empty by building the group alone, since go-core exports no list of a group's members.
- migrate-templates: the web starter README's database snippet names its node "db" and makes its constructor a *Nodes method, so it passes CheckNames.
- migrate-templates: the CLI Leaves layer test's fixture leaf defines its nodes on the App's own Graph() and Root(), with no test-only export.
- migrate-templates: the CLI layer test builds the lifecycle-config node even for a leaf that declares no nodes.
- migrate-templates: both CHANGELOGs link go-core's migration guide at https://github.com/standards-lab/go-core/blob/main/CHANGELOG.md#migrating-from-v0100.
- migrate-templates: the CLI template's context/README.md version line moves to template/v0.2.0; the web template records its version only in its CHANGELOG.
- migrate-templates: standards review changed the web Build-failure test's fixture error to "unavailable", so the stderr match finds "logger" only in the node's name.
- migrate-service: go-web-service's Roots lists only the server and the reactors group; the router reaches telemetry and the schema through Use, and the root guard expects no Drop.
- migrate-service: go-web-service and slab stay untagged; the migration is recorded under the CHANGELOG's Unreleased section, and the door is two-way.

## Pending edits

- architecture · principles/composition-terms.md: amend it to admit CLI command packages taking the nodes they declare, once the spike settles the API (decision 5).
- architecture · standards/go-elemental/principles/domain-files.md:47-48: reconcile it with the composition-terms.md amendment.
- architecture · standards/go-elemental/principles/construction-and-execution.md: new principle page (rule, graph.After/In exception, test-support scope); index it in the principles README and the go-elemental README.
- go-web-sdk, go-cli-sdk, go-database, go-storage · STANDARDS.md: point to construction-and-execution.md.
- architecture · standards/go-elemental/principles/cli-applications.md:39, 87: names cli.Run with WithGraph; name cli.Program{Root, Graph, LifecycleConfig}.Run instead.
- architecture · standards/go-elemental/principles/tests-and-docs.md ("The unit tier"): cites go-web-sdk-template's AppendReactor in internal/app/export_test.go as the fixture-node hook example; migrate-templates deletes it — cite a fixture reactor joining with graph.In(Reactors) instead, or drop the clause.
- coordinator · context/graph-ergonomics.md (opening paragraph): name the templates' releases, go-web-sdk-template template/v0.14.0 and go-cli-sdk-template template/v0.2.0, beside the libraries' releases that moved onto go-core v0.11.0.
- architecture · standards/go-elemental/principles/topology-and-naming.md:88-92: names `Run` as building the graph and running it under the lifecycle coordinator, and a reactor's node as "also joining `Nodes.Reactors`"; say instead that `Run` is one runner call (`web.Service`, `cli.Program`), and that a reactor joins the reactors group with `graph.In` on its Define line.
