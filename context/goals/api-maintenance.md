# goal · api-maintenance

- **State:** brief ready
- **Task:** libraries
- **Branch:** libraries

## Tasks

1. [x] core
2. [ ] libraries
3. [ ] storage-cli
4. [ ] messaging
5. [ ] blobfs
6. [ ] template
7. [ ] service

## Task brief · libraries

```
Problem       go-database's sweep.New and go-web-sdk's middleware.RequestID
              take functional options, and web.WriteProblem takes five
              inputs. All three break the construction-and-execution rule.
              go-observability's docs name the WithIDSource option that
              goes away. All three repositories trail go-core v0.12.0, and
              go-database also trails sqlate v0.7.0 and pgx v5.13.0. Their
              sub-modules re-tag on each base's new release before the
              next tier migrates.
Behaviors     1. sweep.New takes a db, its registrations, the shutdown
                 drain, a logger, and a trailing sweep.Options{Unlocked,
                 Gate, OnPass}. The zero Options keeps today's defaults:
                 each pass claims its advisory lock, a dialect without
                 the try-lock capability is refused with ErrNoTryLocker,
                 no gate is held, and failed passes are logged at warn.
                 New still refuses its invalid inputs, with an error and
                 before any I/O, and still panics on a nil db or logger.
              2. Options.Unlocked runs the sweeper without claiming
                 locks, on any dialect. A non-nil Options.Gate is held
                 shared for each pass, as Gated does today. A non-nil
                 Options.OnPass receives each Pass in place of the warn
                 log, as OnPass does today.
              3. sweep no longer exports the Option type or the
                 Unlocked, Gated, and OnPass option functions.
              4. The sweeper's reactor still cancels a pass that is
                 still running at half the shutdown drain. On go-core
                 v0.12.0 it sets this through
                 reactor.Options.CancelAfter.
              5. middleware.RequestID takes a
                 middleware.RequestIDOptions{Source, TrustedHeader}. The
                 id is the first of these that yields one: Source's
                 non-empty return; then TrustedHeader's inbound value,
                 when it is 1 to 128 visible ASCII characters; then a
                 generated 16-byte hex id. The id still reaches the
                 request's context and the X-Request-Id response
                 header. A nil Source or an empty TrustedHeader is
                 unset and doesn't panic, so the zero value always
                 generates.
              6. middleware no longer exports RequestIDOption,
                 WithIDSource, or WithTrustedHeader.
              7. web no longer exports WriteProblem. Recoverer,
                 ContentType, and rate-limit's limited and failed
                 responses send the same problems as before through
                 Problem.WriteFor: the same status, the defaulted
                 title, detail, the request path as instance, and the
                 request_id member.
              8. go-observability's package docs and README name
                 RequestIDOptions.Source where they named WithIDSource.
                 RequestIDSource's behavior is unchanged.
              9. go-database requires sqlate v0.7.0 and go-core v0.12.0.
                 postgres requires pgx v5.13.0 and go-database v0.12.0.
                 go-web-sdk requires go-core v0.12.0. rate-limit
                 requires go-core v0.12.0 and go-web-sdk v0.17.0.
                 go-observability requires go-core v0.12.0. otlp
                 requires go-observability v0.4.0. Every module passes
                 its check with the workspace off.
             10. go-database's and go-web-sdk's READMEs, guides, and
                 package docs show only the new forms. Each base
                 changelog lists the removals as Breaking, with a
                 "Migrating from" section, and marks its go-core
                 requirement (and, for go-database, its sqlate
                 requirement) as Breaking. go-observability's v0.4.0
                 marks its go-core requirement Breaking and records the
                 doc fix. postgres, rate-limit, and otlp each carry one
                 Breaking line for their base's new requirement.
             11. go-observability's standards file points to the
                 construction-and-execution principle. go-database's
                 and go-web-sdk's pointers name the converted forms:
                 sweep.Options, RequestIDOptions, and Problem.WriteFor.
             12. All three repositories' currency commands exit 0, and
                 every check passes.
Test seams    The exported constructors, exercised from external test
              packages: sweep.New with Options values over sqlate's
              sqltest recording driver and its locking and non-locking
              dialects, and middleware.RequestID, Recoverer, ContentType,
              and rate-limit's New, driven through net/http/httptest and
              their recorded responses.
Slices        1. go-database upgrade: sqlate v0.7.0 and go-core v0.12.0,
                 with sweep's reactor call moved to
                 reactor.Options{CancelAfter} (4), and postgres onto
                 pgx v5.13.0. Done when currency exits 0 and the check
                 passes.
              2. go-web-sdk upgrade: the base and rate-limit onto
                 go-core v0.12.0, with nothing to adapt.
              3. go-observability upgrade: the base onto go-core
                 v0.12.0, with nothing to adapt. (Slices 2-3: the proxy
                 doesn't list go-core v0.12.0 yet, so currency exits 0
                 either way; done when go.mod requires v0.12.0 and the
                 check passes.)
              4. sweep.Options, with sweep's tests on the new form
                 (1-3).
              5. go-database's docs, changelog v0.12.0 with its
                 migration section, the updated standards pointer, and
                 postgres's changelog. Release preparation moves
                 postgres onto go-database v0.12.0 once it is tagged
                 (9-11, go-database).
              6. RequestIDOptions, with RequestID's tests on the new
                 form (5-6).
              7. WriteProblem removed. Recoverer, ContentType, and
                 rate-limit move to Problem.WriteFor, with their
                 response tests unchanged (7).
              8. go-web-sdk's docs, changelog v0.17.0 with its migration
                 section, the updated standards pointer, and
                 rate-limit's changelog. Release preparation moves
                 rate-limit onto go-web-sdk v0.17.0 once it is tagged
                 (9-11, go-web-sdk).
              9. go-observability's docs fix, changelog v0.4.0, and
                 standards pointer, and otlp's changelog. Release
                 preparation moves otlp onto go-observability v0.4.0
                 once it is tagged (8-11, go-observability).
Out of scope  go-web-service's RequestID and WriteProblem calls, which
              move in the service task. The templates. webtest's
              variadic Header parameters, left as data lists. The
              variadic data lists Chain, Use, ContentType, Readiness,
              NewErrorWriter's matchers, and ErrorWriter.Detail.
              admin.New, which already conforms. go-storage,
              go-messaging, blobfs, and go-cli-sdk, which are later
              tasks. The architecture notes that cite WriteProblem, and
              the principle page's context-exemption sentence, at sync.
Door          One-way for the six tags, which reach every consumer's
              module requirements. Two-way for the code and docs, which
              merge by pull request before tagging.
Release       go-database v0.12.0, postgres/v0.7.0; go-web-sdk v0.17.0,
              middleware/rate-limit/v0.5.0; go-observability v0.4.0,
              otlp/v0.4.0
```

## Progress

slices 9/9 committed · standards ✓ · spec ✓ (no gaps) · editor ✓

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
- libraries: web.WriteProblem is removed outright; callers use Problem{...}.WriteFor(w, r), the rule's existing struct-with-method form (no standard library form for RFC 9457).
- libraries: middleware.RequestID takes RequestIDOptions{Source, TrustedHeader}; a nil Source or empty TrustedHeader is unset, and the old options' panics go.
- libraries: webtest Client.Do, Get, Post, Put, and Delete keep their variadic Header parameters, which are data lists, not functional options.
- libraries: go-observability releases v0.4.0 on go-core v0.12.0, marking the requirement Breaking per graph-ergonomics' precedent; otlp re-tags as otlp/v0.4.0.
- libraries: a nil sweep.Options.Gate or OnPass means the default, as passing nil to Gated or OnPass did; a non-nil interface that wraps a nil pointer counts as set.
- libraries: RequestID takes RequestIDOptions by value, and the middleware reads it once, at construction.
- libraries: standards review cut TestRequestID_UnsetOptionsDoNotPanic, which could fail only where go test's own panic handling already would; its one new case joined the precedence table.
- libraries: go-observability's docs pass RequestIDSource as a value (`Source: observability.RequestIDSource`); it is the source function, not a constructor.
- libraries: released changelog entries that name removed forms (go-observability v0.1.0's WithIDSource, go-database v0.10.0's sweep options) stay as history.
- libraries: rate-limit v0.5.0 carries two Breaking lines, go-core v0.12.0 and go-web-sdk v0.17.0, because it requires both directly (one line per requirement, as in v0.4.0).
- libraries: each repository's "Migrating from" section covers its own removals and points to go-core's and sqlate's changelogs for their call changes.
- libraries: the go-database v0.12.0, go-web-sdk v0.17.0, and go-observability v0.4.0 changelog entries (and their sub-modules') are dated 2026-11-11; release preparation re-dates them if the tags land later.

## Pending edits

- architecture · construction-and-execution principle: a context.Context parameter does not count toward the four required inputs.
- standards-lab · context/html-artifacts.md:409: replace go-web-sdk's `WriteProblem` with `Problem.WriteFor` (go-web-sdk v0.17.0); the `problem.go` citation stays, since `WriteFor` is defined there.
