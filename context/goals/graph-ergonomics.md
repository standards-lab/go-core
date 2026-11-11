# goal · graph-ergonomics

- **State:** idle
- **Task:** none
- **Branch:** none

## Tasks

1. [x] design-review
2. [x] release
3. [ ] migrate-libraries
4. [ ] migrate-templates
5. [ ] migrate-service

## Task brief · release

```
Problem       go-core v0.10.0's graph and lifecycle read badly. Order is hidden in s.After calls
              inside constructors, fixed and fresh values need closures, introspection gives only
              names, and a Monitored failure doesn't say which node failed. spike-graph proved the
              settled API on the full go-web-service and slab, and the release's two plan rounds
              settled what was left open. Three migration tasks and wave C's go-ai and client SDK
              wait on one breaking release that ships it.
Behaviors     1. go-core trails nothing: Go is 1.27.4, the currency command exits 0, and the check
                 passes.
              2. Order and groups are declared at Define. graph.After(refs...) orders a node after
                 each node or group it names, each defined earlier, and graph.In(group) makes the
                 node a member of a group. g.Group(name) returns a group handle, which is a Ref.
                 A Build given a group, as a root or as an After target, expands it to its members
                 in definition order, and an empty group builds and orders nothing. An After never
                 builds its target. Scope has only Use, and Scope.After no longer exists. A cycle
                 closed by an After panics, naming its path, including any group it passes
                 through.
              3. Define, Value, Fresh, and Group panic with a "graph: " message naming the call in
                 each of these cases:
                 a. after the first Build;
                 b. an empty name;
                 c. a name containing a comma or a newline;
                 d. a name already taken by a node or a group (they share one name space);
                 e. a nil constructor or a nil option;
                 f. an After or In naming a nil ref, a zero-value ref, or a ref from another
                    Graph. For a nil ref the message asks whether its Define or Group comes later.
                 Each one checks everything before it changes the Graph, so a call that panics
                 leaves the Graph as it was.
              4. System.Dropped lists, as Drop{Node, After} with the handles as declared, each
                 After of a built node that ordered nothing: its target is a node not in the
                 System, or a group with no member in it. Drops are listed in the declaring nodes'
                 definition order, then in each node's declaration order. A group with any member
                 present is not dropped. A Build of the full root set drops nothing.
              5. Replace swaps only a node's constructor; the node keeps its After and In. Replace
                 panics on a nil or zero-value handle, after the first Build, on a node from
                 another Graph, and on a nil constructor.
              6. g.Value(name, v, opts...) defines a node whose value is v in every Build, with T
                 inferred from v's static type. g.Fresh[T](name, opts...) defines a node whose
                 value is a new zero *T in each Build, distinct from one Build to the next. Both
                 take Define's options, so a value can join a group.
              7. Introspection uses the handles that Define, Value, Fresh, and Group return:
                 - Dependency carries Ref, the node's handle.
                 - Observe takes func(Ref) and calls it with the handle as each node begins,
                   before its constructor runs, once per node per Build, depth-first.
                 - An observer persists for every later Build of its Graph, and the docs say so.
                 On a nil or zero-value handle, Name returns "", and Get panics with a "graph: "
                 message.
              8. System.LayerOf(ref) returns a node's layer and panics for a nil ref, a group, or
                 a node not in the System. System.Has(ref) reports whether the node is in the
                 System, and panics for a nil ref or a group. System.Dependencies() returns a
                 fresh copy of every built node in layer order, then definition order, equal to
                 flattening Layers().
              9. System.String prints the layer table, one line per layer in the form
                 "0: config, logger", with no trailing newline. It prints "" for an empty System.
              10. An observer that panics makes Build panic with a "graph: " message that names
                  the node and carries the original value, wrapped when that value is an error.
                  This holds whether the node is a root or was reached through a Use.
              11. A Graph is safe for concurrent use:
                  - Concurrent Builds pass the race detector, as do describing calls and Observe
                    made beside a Build.
                  - Describing still panics after the first Build.
                  - An observer added while a Build runs applies from the next Build on.
                  - Concurrent Builds may call an observer concurrently, and the docs say so.
              12. graph/graphtest.NameErrors(nodes any) []error checks a struct of handles, or a
                  pointer to one. For each exported field whose type implements Ref, it returns an
                  error when the field holds no handle (nil or zero-value), and when the node's or
                  group's name, lowercased with its spaces removed, differs from the field name
                  lowercased. It skips other fields. Anything that is not a struct or a non-nil
                  pointer to one is a single error. It returns nil when every field passes.
                  CheckNames(t, nodes) reports each error as a test error, without stopping the
                  test.
              13. A Monitored value's runtime failure ends the run as "run: <node>: <error>",
                  wrapping the error. A channel registered with Monitor still reports
                  "run: <error>".
              14. Behavior that must not change still holds: edges discovered through Use,
                  build-once, longest-path layers, panics for wiring mistakes, the participation
                  interfaces, and the startup, readiness, and shutdown semantics. go-core's
                  existing graph and lifecycle tests, ported to the new API, prove it.
              15. The package docs for graph, graphtest, and lifecycle and the README describe the
                  shipped API. The CHANGELOG holds a dated v0.11.0 section with its compare link.
                  It marks each break **Breaking:**: Scope.After removed, Observe's signature, the
                  name rule, Dependency's added field, and the run-failure text.
              16. go-core's design note states the shipped API, not a candidate: no open forks,
                  Fresh as a method, the round decisions included, and web.Run and the runners
                  left to go-web-sdk and go-cli-sdk. The context index describes the note as it
                  now stands.
Test seams    go-core's check (`mise run check`). It runs the graph, graphtest, and lifecycle
              tests under the race detector through their exported API. Also go-core's currency
              command. The note and docs are read against the code.
Slices        1. Upgrade go-core to Go 1.27.4. Currency exits 0 and the check passes
                 (behavior 1).
              2. Placement at Define: After, In, and Group; Build expanding groups; Scope.After
                 removed; Dropped; Replace keeping placement; the name rule; graph's tests ported
                 (behaviors 2-5, 14).
              3. Value and Fresh with Define's options (behavior 6).
              4. Introspection: Ref, Observe(func(Ref)), LayerOf, Has, Dependencies, String, and
                 the labelled observer panic (behaviors 7-10).
              5. A Graph safe for concurrent use, with race tests (behavior 11).
              6. graphtest's NameErrors and CheckNames (behavior 12).
              7. lifecycle's node-named run failure, with lifecycle's tests ported
                 (behaviors 13-14).
              8. Release preparation: package docs, README, the CHANGELOG v0.11.0 section, and
                 the design note and its index entry (behaviors 15-16).
Out of scope  The consumers' migrations: the libraries, go-cli-sdk, the templates, and
              go-web-service with slab. web.Run, and cli or scenario reporting the node-named
              failure. Amending composition-terms.md. The spike repositories. Group.Members.
              Replace removing an edge or a membership. A way to stop an observer. Arity adapters.
              Any lifecycle change beyond the run-failure label.
Door          Two-way until the tag; the code and docs land on main by pull request. One-way at
              the tag: once the module proxy fetches v0.11.0, the checksum database pins it and
              the tag is never re-cut.
Release       v0.11.0 (go-core)
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

## Pending edits

- coordinator · roadmap: correct graph-ergonomics' summary: nodes are typed *Node[T] handles; the string is a required name, not the key, and v0.10.0's introspection exposed it as a node's only identity.
- coordinator · standards-lab/context/auth-strategy.md:199-200: the auth plan names an auth.go layer file with a define function; revise it once the spike settles the constructor shape.
- architecture · principles/composition-terms.md: amend it to admit CLI command packages taking the nodes they declare, once the spike settles the API (decision 5).
- architecture · standards/go-elemental/principles/domain-files.md:47-48: reconcile it with the composition-terms.md amendment.
- coordinator · standards-lab/context/graph-ergonomics.md "Open findings": remove the release's bullet, whose outcomes the next edit states; keep the migrate-libraries bullet. "Not shipped" gains Group.Members, a way to stop an observer, and Replace removing an edge or a membership.
- coordinator · standards-lab/context/graph-ergonomics.md "The settled API": decision 9 gains that Replace swaps only the constructor, keeping After and In, and that a Graph is safe for concurrent use (concurrent Builds, describing calls and Observe beside a Build, observers possibly called concurrently). Decision 12 gains the name rule: Define, Value, Fresh, and Group refuse an empty name, a comma or a newline, and a name taken by a node or a group. Decision 13 becomes: `Ref` on a Dependency; `Observe(func(Ref))`, whose observers persist for every later Build and whose panic Build re-panics as a "graph: " message naming the node, wrapping an error value; `LayerOf`, which panics for a node not in the System, a nil ref, or a group; `Has(ref) bool`, false for a node not in the System or from another Graph; `Dependencies()`, every built node in layer, then definition, order; and `String()`, the unquoted layer table, one line per layer as "0: config, logger", with no trailing newline. "The three details": the name check's last clause names the comma and newline rule beside the empty name.
- coordinator · roadmap: migrate-templates' and migrate-service's summaries name Has and Dependencies for the layer tests in place of scanning Layers().
