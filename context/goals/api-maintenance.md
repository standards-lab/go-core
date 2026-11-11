# goal · api-maintenance

- **State:** building
- **Task:** core
- **Branch:** core

## Tasks

1. [ ] core
2. [ ] libraries
3. [ ] storage-cli
4. [ ] messaging
5. [ ] blobfs
6. [ ] template
7. [ ] service

## Task brief · core

```
Problem       go-core's reactor.New and event.NewRecorder take functional
              options, and sqlate's Begin, Transact, and Beginner take
              TxOption functions. Both break the construction-and-execution
              rule. They sit at the bottom of the dependency order, so every
              later task in the goal migrates onto their converted forms.
              sqlate also trails its currency.
Behaviors     1. reactor.New takes a source, a handler, and a trailing
                 reactor.Options. The zero Options keeps today's default:
                 handlers are cancelled only when Shutdown's context ends.
              2. Options.CancelAfter d makes Shutdown cancel the handlers'
                 contexts after d and then wait for them to unwind, bounded
                 by Shutdown's context, as the CancelAfter option does today.
              3. reactor.CancelWithin(drain) returns half of drain as a
                 time.Duration, for Options.CancelAfter.
              4. reactor no longer exports the Option type or the
                 CancelAfter option function.
              5. event.NewRecorder takes a sink, a source, and a trailing
                 event.Options{Clock, IDs}. The zero Options stamps
                 time.Now in UTC and version 7 UUIDs. A set Clock or IDs
                 replaces its default. A nil sink or an empty source still
                 panics with an "event: " message.
              6. event no longer exports the Option type or the Clock and
                 IDs option functions.
              7. sqlate's DB.Begin takes a context and a *sql.TxOptions.
                 nil opens a default transaction. A non-nil value's
                 isolation level and read-only flag reach the driver as
                 given.
              8. DB.Transact and Transact take a trailing *sql.TxOptions
                 and pass it to Begin. Commit on success, rollback on error
                 with the rollback failure joined, and rollback then
                 re-panic on panic are unchanged.
              9. Beginner's Begin has the same signature as DB.Begin.
                 *DB, and any type that embeds it, still satisfies it.
             10. sqlate no longer exports TxOption, Isolation, or ReadOnly.
             11. The postgres and sqlint sub-modules each require sqlate
                 v0.7.0 and pass their checks with the workspace off.
             12. The READMEs, guides, and package docs of both repositories
                 show only the new forms. Each changelog's release lists
                 the removals as Breaking, with a "Migrating from" section.
                 Each repository's standards file points to the
                 construction-and-execution principle.
             13. go-core's and sqlate's currency commands exit 0, and both
                 checks pass.
Test seams    The exported constructors and runners, exercised from
              external test packages: reactor.New and event.NewRecorder
              with Options values, and sqlate's DB.Begin, DB.Transact, and
              Transact over sqltest's recording driver, which records the
              transaction options each begin receives.
Slices        1. sqlate upgrade: go 1.27.4, golangci-lint 2.15.0,
                 mise-action v5.2.7, create-gh-release-action v1.12.0,
                 postgres onto pgx v5.12.0 and sqlate v0.6.1, sqlint onto
                 sqlate v0.6.1. Done when currency exits 0 and the check
                 passes. (go-core is current: no upgrade slice.)
              2. reactor.Options and CancelWithin as a duration, with
                 reactor's tests on the new form (1-4).
              3. event.Options, with the recorder's tests on the new form
                 (5-6).
              4. go-core docs, changelog v0.12.0 with its migration
                 section, and the standards pointer (12, go-core).
              5. sqlate's Begin, Transact, and Beginner on *sql.TxOptions,
                 with the library's own callers and tests moved (7-10).
              6. The postgres sub-module's test implementer of Beginner and
                 both sub-modules' changelogs. Release preparation moves
                 postgres and sqlint onto sqlate v0.7.0 once it is tagged
                 (11).
              7. sqlate's guide, package docs, changelog v0.7.0 with its
                 migration section, and the standards pointer (12, sqlate).
Out of scope  Migrating any consumer: go-database's sweeper, go-messaging,
              the templates, and go-web-service are later tasks.
              reactor.Every, Wake, and Gate, which take no options.
              query.Projection.Continue, whose context is exempt from the
              count. graph.After and graph.In, the rule's named exception.
              The principle page's context-exemption sentence, at sync.
Door          One-way for the four tags, which reach every consumer's
              module requirements. Two-way for the code and docs, which
              merge by pull request before tagging.
Release       go-core v0.12.0; sqlate v0.7.0, postgres/v0.7.0,
              sqlint/v0.4.0
```

## Progress

slices 7/7 committed (1 sqlate b6178ba, 2 go-core ec89674, 3 go-core 8ebf7b9, 4 go-core f2a9b86, 5 sqlate 838e3e9, 6 sqlate 980c58a, 7 sqlate cb9e580) · standards done (go-core 01a47bd, sqlate 38d203c) · spec done (no gaps; 11 deferred to release) · editor —

## Decisions

- plan: root is go-core, the lowest tier and graph-ergonomics' root.
- plan: one task per dependency tier, lowest first, each tagging before the next migrates; the library tier splits into libraries, storage-cli, messaging, and blobfs.
- plan: scope is every exported library API in the goal's repositories that breaks construction-and-execution, beyond the summary's list (blobfs's Files and Directories positional methods, web.WriteProblem, sqlate's Beginner, go-cli-sdk's StringsVar); application code in go-web-service and slab migrates but is not converted.
- plan: a context.Context parameter is exempt from the four-input count; query.Projection.Continue is therefore not a violation.
- plan: old forms are removed outright in each breaking release, with no deprecated aliases.
- plan: every sub-module re-tags on its base's new requirement, code change or not, so the workspace aligns.
- plan: go-observability fixes its docs that name WithIDSource and releases them.
- plan: each converting task adds its repositories' missing STANDARDS.md pointer to the construction-and-execution principle.
- plan: go-cli-sdk and go-cli-sdk-template join the goal; StringsVar becomes a flag.Value constructor used through fs.Var, the standard library's shape for custom flags.
- core: reactor.CancelWithin stays, as a plain function returning half the drain as a duration.
- core: sqlate takes the standard library's *sql.TxOptions in place of TxOption (existing solution adopted).
- core: took pgx v5.13.0 (released 2026-11-11, after the brief named v5.12.0) in the upgrade slice; a minor bump with nothing to adapt, and the slice is done only when currency exits 0.
- core: sqlate's go.mod files stay at `go 1.27` with no toolchain line; the repository tracks only the Go minor version.
- core: event.Options applies its defaults per field: a set Clock keeps UUID v7 IDs, a set IDs keeps time.Now in UTC, as the old options did.
- core: Transact and DB.Transact take *sql.TxOptions as a required trailing parameter, not variadic; callers pass nil for a default transaction.
- core: postgres v0.7.0 and sqlint v0.4.0 mark their sqlate v0.7.0 requirement Breaking, as their earlier entries did for breaking base releases.
- core: behavior 12's "Migrating from" section applies to the changelogs that record removals (go-core, sqlate); postgres and sqlint remove nothing, so each carries one Breaking line for the sqlate v0.7.0 requirement.

## Pending edits

- architecture · construction-and-execution principle: a context.Context parameter does not count toward the four required inputs.
- libraries · plan round: go-observability's version (v0.3.1 patch, or a minor once it moves onto go-core v0.12.0, per graph-ergonomics' precedent) and whether otlp re-tags on it.
