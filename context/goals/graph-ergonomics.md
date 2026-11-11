# goal · graph-ergonomics

- **State:** building
- **Task:** migrate-templates
- **Branch:** migrate-templates

## Tasks

1. [x] design-review
2. [x] release
3. [x] migrate-libraries
4. [ ] migrate-templates
5. [ ] migrate-service

## Task brief · migrate-templates

```
Problem       go-core v0.11.0 shipped the new graph and lifecycle API, and
              go-web-sdk v0.16.0 and go-cli-sdk v0.3.0 moved onto it. The two
              templates still pin go-core v0.10.0 / v0.8.0 and the old SDKs,
              and they trail their Go, lint, and CI action pins. Their
              composition roots use removed APIs (Scope.After, Observe taking
              a name, cli.Run with WithGraph). The web root's Run is a
              hand-written Build and Coordinator with a fallback logger, its
              layer tests scan Layers() by name, and both roots name their
              nodes in comments. gonew scaffolds only tagged versions, so
              every new service and CLI starts on the old API until each
              template ships a release.
Behaviors     1. Both templates current: Go 1.27.4, golangci-lint 2.15.0,
                 jdx/mise-action v5.2.7 and taiki-e/create-gh-release-action
                 v1.12.0 in the repository's and the generated module's
                 workflows; web requires go-core v0.11.0 + go-web-sdk
                 v0.16.0, CLI requires go-core v0.11.0 + go-cli-sdk v0.3.0.
                 Each currency command exits 0; each check and integration
                 tier passes.
              2. The web App runs as one web.Service call; Run(ctx, stderr)
                 returns the exit code:
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
              4. The server is alone in the top layer; a fixture node above
                 it shows the guard fails. A reactor joins by its
                 graph.In(Reactors) alone and sits below the server; a
                 fixture reactor ordered after the router, added black-box
                 through the group, proves it.
              5. While the reactors group is empty, a Build of the web root
                 set reports exactly one Drop (the server's After of the
                 reactors group); with any member, none. Any other Drop fails
                 the guard (shown by a fixture After of an unbuilt node). A
                 failing guard prints the System's layer table.
              6. In the web root each node's order and group membership are
                 on its Define line, and no constructor body orders anything:
                 configuration and the logger's writer are g.Value nodes,
                 readiness is g.Fresh; a "lifecycle config" node yields the
                 lifecycle block; every constructor that reads a node is a
                 *Nodes method.
              7. In both templates each Nodes field's node/group name,
                 lowercased without spaces, equals the field name lowercased;
                 graphtest.CheckNames checks it; no Nodes field carries a
                 name comment.
              8. The CLI App dispatches through cli.Program with its graph
                 and lifecycle-config node, exactly as before: existing tests
                 pass unchanged in what they expect (root help, unknown
                 command, version, New is cold, Halt, invalid shutdown
                 timeout). The build-recording fixture reads names from
                 handles.
              9. The CLI layer test ranges over the root's Leaves(); for each
                 leaf, a Build of its node set plus the lifecycle-config node
                 succeeds and drops nothing. A fixture leaf ordered after a
                 node its path doesn't declare shows the guard fails, with
                 the layer table.
              10. The web template exports no test-only reactor hook; the
                  lint testpackage note allows a clock or probe hook only;
                  the testpackage skip stays for generated services.
              11. Package docs and READMEs describe the shipped API — web:
                  web.Service runs the service, Build failure to stderr,
                  Roots is the root set, a reactor is added with graph.In on
                  the reactors group (README snippet); CLI: cli.Program runs
                  the tree, an App runs one command at a time because the
                  dispatcher restores flags. Neither names Scope.After,
                  cli.Run, WithGraph, appending to Nodes.Reactors, or name
                  comments.
              12. Each CHANGELOG has a dated section with compare link — web
                  v0.14.0, CLI v0.2.0 — naming the new go-core and SDK
                  requirements as **Breaking:** and listing what a generated
                  app meets when porting (web: Run signature and web.Service,
                  Build failure on stderr, Roots, reactors group and
                  graph.In, lifecycle-config node, name check, removed hook
                  and narrowed lint note; CLI: cli.Program for cli.Run, name
                  check, Leaves layer test, one-line monitored failure). Each
                  links go-core's "Migrating from v0.10.0". Tool and action
                  bumps take no entry (design-review Decision).
Test seams    Each template's check (`mise -C template run check`), its
              integration tier, and its currency command. Web tests drive the
              App through New, Run, Roots, Graph, Nodes and read placement by
              handle (LayerOf, Has, Dependencies, Dropped). CLI tests drive
              App.Run over buffered Streams with the apptest fixtures; the
              layer test iterates Leaves(). Docs read against the code.
Slices        1. Web template current (pins, go-core v0.11.0, go-web-sdk
                 v0.16.0), absorbing only the break needed to compile and
                 stay green: server's order after reactors via a reactors
                 group on its Define line; fixture reactor joins with
                 graph.In; test hook and lint note go; observers take
                 handles. Done: currency exits 0, check passes. (1, 10)
              2. CLI template current (pins, go-core v0.11.0, go-cli-sdk
                 v0.3.0); App.Run moves to cli.Program; build recorder reads
                 names from handles. Done: currency exits 0, check passes.
                 (1, 8)
              3. Web runner: App.Run(ctx, stderr) as one web.Service call,
                 lifecycle-config node, Roots; fallback logger goes;
                 entrypoint passes stderr; Build-failure test reads stderr.
                 Demo: `mise run serve` logs "server ready". (2, 3)
              4. Web Define lines: Value/Fresh nodes, *Nodes methods, every
                 order on its Define line, CheckNames replaces name comments.
                 (6, 7)
              5. Web layer tests on handles: root set vs a real Run, server
                 alone on top, server above a fixture reactor, the one
                 allowed Drop; each guard shown failing; Layers()-scanning
                 helpers go. (3-5)
              6. CLI names and layers: CheckNames, Leaves layer test with its
                 fixture guard. (7, 9)
              7. Web docs: package doc, starter README, repository README,
                 capability map. (11)
              8. CLI docs: package doc, Run doc's reason, starter README,
                 capability map. (11)
              9. go-web-sdk-template release prep: v0.14.0 CHANGELOG section
                 and link lines. (12)
              10. go-cli-sdk-template release prep: v0.2.0 CHANGELOG section
                  and link lines. (12)
Out of scope  go-core, go-web-sdk, go-cli-sdk changes; go-web-service and
              slab (migrate-service); defining any reactor, admin, or domain
              node in the templates; either runner reading Dropped; the
              construction-and-execution conversion (api-maintenance); the
              architecture repository and the STANDARDS.md pointers already
              pending at sync.
Door          Two-way until each tag (pull requests to main). One-way at each
              tag: the module proxy and checksum database pin it, gonew's
              @latest resolves to it, it's never re-cut. Each tag is cut only
              from a main whose CI, integration tier included, has passed.
Release       go-web-sdk-template: template/v0.14.0
              go-cli-sdk-template: template/v0.2.0
```

## Progress

slices 10/10 committed · standards ✓ · spec ✓ · editor ✓

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

## Pending edits

- architecture · principles/composition-terms.md: amend it to admit CLI command packages taking the nodes they declare, once the spike settles the API (decision 5).
- architecture · standards/go-elemental/principles/domain-files.md:47-48: reconcile it with the composition-terms.md amendment.
- architecture · standards/go-elemental/principles/construction-and-execution.md: new principle page (rule, graph.After/In exception, test-support scope); index it in the principles README and the go-elemental README.
- go-web-sdk, go-cli-sdk, go-database, go-storage · STANDARDS.md: point to construction-and-execution.md.
- architecture · standards/go-elemental/principles/cli-applications.md:39, 87: names cli.Run with WithGraph; name cli.Program{Root, Graph, LifecycleConfig}.Run instead.
- architecture · standards/go-elemental/principles/tests-and-docs.md ("The unit tier"): cites go-web-sdk-template's AppendReactor in internal/app/export_test.go as the fixture-node hook example; migrate-templates deletes it — cite a fixture reactor joining with graph.In(Reactors) instead, or drop the clause.
- coordinator · context/graph-ergonomics.md (opening paragraph): name the templates' releases, go-web-sdk-template template/v0.14.0 and go-cli-sdk-template template/v0.2.0, beside the libraries' releases that moved onto go-core v0.11.0.
