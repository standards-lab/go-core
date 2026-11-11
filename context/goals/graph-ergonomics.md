# goal · graph-ergonomics

- **State:** idle
- **Task:** none
- **Branch:** none

## Tasks

1. [x] design-review
2. [x] release
3. [x] migrate-libraries
4. [ ] migrate-templates
5. [ ] migrate-service

## Task brief · migrate-libraries

```
Problem       go-core v0.11.0 shipped the new graph and lifecycle, and the libraries above it still
              require older go-core versions (v0.7.0 to v0.10.0). They also trail their own
              dependencies, tools, CI actions, and test images, and four docs still name the
              removed Scope.After. A service still writes its own build-run-log sequence because
              go-web-sdk has no runner, and go-cli-sdk's runner takes inputs in a shape the
              construction-and-execution rule retires. When a Monitored value fails, cli and scenario print a
              useless "context canceled" line above the node-named failure. The templates,
              go-web-service, and wave C wait on these libraries' tagged releases.
Behaviors     1. Every repository is current. go-database, go-observability, go-storage,
                 go-web-sdk, and go-cli-sdk each run Go 1.27.4 and golangci-lint 2.15.0, the
                 latest CI action pins, the latest container images, and every module's direct
                 requirements at their latest. Each repository's currency command exits 0 and its
                 check passes, race detector included.
              2. Every base module requires go-core v0.11.0 and each provider sub-module requires
                 its base's new release. Each module's tests use the v0.11.0 graph and lifecycle
                 API, and go-storage's acceptance suites pass against SeaweedFS and Azurite.
              3. No package doc, README, or doc comment in the five repositories names
                 Scope.After. Each one that described an order-only edge now describes
                 graph.After on the Define line.
              4. web.Service{Graph, LifecycleConfig, Logger, Server, Roots}.Run(ctx, stderr) int
                 builds Roots, Server, and LifecycleConfig, runs the System under a lifecycle
                 Coordinator configured with LifecycleConfig's value until ctx ends, and returns
                 go-core's process exit codes:
                 a. once startup completes, it logs "server ready" on the logger node's value,
                    with the server's bound address as addr;
                 b. when ctx ends and the System drains cleanly, it logs "server stopped" and
                    returns ExitOK;
                 c. when startup, a Monitored value at runtime, or shutdown fails, it logs
                    "service failed" with the Coordinator's error, which names the node, and
                    returns ExitFailure; a failed startup logs no "server ready";
                 d. when the Build fails, it writes "service failed: <error>" to stderr, logs
                    nothing, starts nothing, and returns ExitFailure.
              5. Service.Run panics with a "web: " message naming the field when Graph,
                 LifecycleConfig, Logger, or Server is nil or Roots is empty. It also panics with
                 a "web: " message, before anything starts, when the Build did not reach the
                 Logger.
              6. cli.Program{Root, Graph, LifecycleConfig}.Run(ctx, args, streams) int dispatches
                 exactly as cli.Run did: the same parse order, Use builds, lifecycle, flag
                 restoration across dispatches, and exit codes. cli.Run, cli.Option, and
                 cli.WithGraph no longer exist. Program.Run panics with a "cli: " message naming
                 the field when Root is nil, and when only one of Graph and LifecycleConfig is
                 set. A tree that declares Use with neither set panics, as before.
              7. When a Monitored value fails while a leaf runs, and the leaf's error is only
                 the end of its context, stderr holds one line: "<path>: run: <node>: <error>",
                 e.g. "prog watch: run: watched: connection lost". When a scenario step fails
                 because that failure ended its context, the line names the step:
                 "prog scenario tour: step 1 (Wait): run: watched: connection lost". When the
                 leaf returns an unrelated error, that error is still reported beside the run
                 failure. A context the caller cancelled still reports "context canceled" as
                 before. The exit code stays ExitFailure.
              8. (*cli.Command).Leaves() iter.Seq2[string, []graph.Ref] yields, depth-first in
                 the order commands were added, each leaf under the command. With each leaf it
                 yields the leaf's command path and the union of the nodes declared with Use
                 along that path, each node once in first-declared order: exactly the set a
                 dispatch to that leaf builds, apart from the lifecycle configuration. Each
                 yielded slice is the caller's own. Iteration stops when the consumer stops.
              9. Neither web.Service.Run nor cli.Program.Run reads System.Dropped.
              10. Each module's package docs and README describe its shipped API. go-web-sdk's
                  doc example ends in a web.Service, go-cli-sdk's docs and README use
                  cli.Program and describe the one-line run failure, and each repository's
                  STANDARDS.md that names cli.Run names cli.Program.
              11. Each released module's CHANGELOG has a dated section with its compare link.
                  The section names each requirement the release changes, and marks the go-core
                  requirement **Breaking:** (it pulls v0.11.0's graph break into importers). In a
                  provider, the base requirement is marked **Breaking:**, and the section notes
                  pgx v5.12.0's two security fixes (postgres) and the AWS SDK's Retry Behavior
                  2.1 default (s3). go-cli-sdk marks cli.Run, Option, and WithGraph removed and
                  the changed stderr text **Breaking:**; go-web-sdk adds web.Service, and
                  go-cli-sdk adds Program and Leaves. go-database's postgres folds its untagged
                  go-database v0.9.0 entry into its new section.
              12. Behavior that must not change still holds: each library's existing tests pass
                  with only their graph and lifecycle wiring migrated, and the expected text
                  changes only where behavior 7 says.
Test seams    Each repository's check (`mise run check`: build, vet, gofmt, go fix, tidy,
              `go test -race`, golangci-lint) and its currency command. go-storage's acceptance
              suites (`mise run acceptance`, SeaweedFS and Azurite under compose). go-database
              has no container suite: its postgres tests run in the check. web.Service.Run is
              exercised with a real *web.Server on port 0 and fake participant nodes that fail
              at startup, at runtime, and at shutdown, ported from spike-graph's runner tests.
              cli.Program.Run uses Streams buffers, and Leaves is exercised by iterating it. Docs
              are read against the code.
Slices        1. go-database current on go-core v0.11.0: Go, lint, actions, sqlate v0.6.1, pgx
                 v5.12.0, postgres on go-database's latest tag; tests on the new graph API;
                 admin's doc says graph.After. Currency exits 0, check passes (behaviors 1-3, 12).
              2. go-web-sdk current on go-core v0.11.0, rate-limit on go-core v0.11.0 and
                 go-web-sdk's latest tag. Currency exits 0, check passes (1-2, 12).
              3. go-observability current on go-core v0.11.0, otlp on grpc v1.85.0; doc, README,
                 and Telemetry's comment say graph.After. Currency exits 0, check passes
                 (1-3, 12).
              4. go-storage current on go-core v0.11.0, azureblob and s3 at their latest SDKs,
                 SeaweedFS 4.53. Currency exits 0, check and acceptance pass (1-2, 12).
              5. go-cli-sdk current on go-core v0.11.0, the monitored-failure test expecting the
                 node-named failure. Currency exits 0, check passes (1-2, 12).
              6. web.Service and its Run, with its panics and ported tests; go-web-sdk's doc
                 example and README (4-5, 9-10).
              7. cli.Program replacing cli.Run, Option, and WithGraph, with scenario and every
                 test moved onto it (6, 9).
              8. The one-line run-failure report in cli and scenario, with a scenario
                 monitored-failure test (7).
              9. (*cli.Command).Leaves (8).
              10. Release preparation for go-database and postgres: docs, README, CHANGELOG
                  sections. postgres requires go-database v0.11.0 once that tag exists
                  (2, 10-11).
              11. Release preparation for go-web-sdk and rate-limit, likewise (2, 10-11).
              12. Release preparation for go-observability and otlp, likewise (2, 10-11).
              13. Release preparation for go-storage, s3, and azureblob, likewise (2, 10-11).
              14. Release preparation for go-cli-sdk: docs, README, STANDARDS.md's runner
                  reference, CHANGELOG (10-11).
Out of scope  go-core changes. The templates, go-web-service, and slab (migrate-templates,
              migrate-service). Converting any other API to the construction-and-execution rule: sweep.New,
              middleware.RequestID, storagetest.NewFake, go-messaging, sqlate, blobfs, and
              go-core's reactor and event options go to the api-maintenance goal. The construction-and-execution
              principle page and the STANDARDS.md pointers to it (pending edits at sync).
              Amending composition-terms.md. Either runner reading Dropped. A general
              Command.Children accessor. Arity adapters.
Door          Two-way until each tag: the code and docs land on main by pull request in each
              repository. One-way at each of the ten tags: once the module proxy fetches a
              version, the checksum database pins it and the tag is never re-cut. A provider is
              tagged only after its base's tag exists and its requirement moves to it.
Release       go-database v0.11.0, postgres/v0.6.0; go-web-sdk v0.16.0,
              middleware/rate-limit/v0.4.0; go-observability v0.3.0, otlp/v0.3.0;
              go-storage v0.7.0, s3/v0.2.0, azureblob/v0.6.0; go-cli-sdk v0.3.0
```

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

## Pending edits

- architecture · principles/composition-terms.md: amend it to admit CLI command packages taking the nodes they declare, once the spike settles the API (decision 5).
- architecture · standards/go-elemental/principles/domain-files.md:47-48: reconcile it with the composition-terms.md amendment.
- architecture · standards/go-elemental/principles/construction-and-execution.md: new principle page (rule, graph.After/In exception, test-support scope); index it in the principles README and the go-elemental README.
- go-web-sdk, go-cli-sdk, go-database, go-storage · STANDARDS.md: point to construction-and-execution.md.
