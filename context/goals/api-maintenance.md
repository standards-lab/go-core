# goal · api-maintenance

- **State:** building
- **Task:** template
- **Branch:** template

## Tasks

1. [x] core
2. [x] libraries
3. [x] storage-cli
4. [x] messaging
5. [x] blobfs
6. [ ] template
7. [ ] service

## Task brief · template

```
Problem       go-web-sdk-template trails go-core v0.12.0 and go-web-sdk
              v0.17.0, and go-cli-sdk-template trails go-cli-sdk v0.4.0
              and go-core v0.12.0. The web template builds RequestID and
              a test reactor in forms those releases removed, and its
              reactor docs teach CancelWithin as an option. A service or
              CLI generated today starts on the old, non-conforming
              library forms. Neither template exports a library API, so
              both migrate and nothing converts. Each releases so that
              gonew @latest yields the converted forms.
Behaviors     1. The web template requires go-core v0.12.0 and go-web-sdk
                 v0.17.0. Its check passes with the workspace off, and
                 its integration suite passes.
              2. The web middleware stack builds RequestID with the zero
                 middleware.RequestIDOptions. Every response still
                 carries the request-id header, and the stack order is
                 unchanged.
              3. The web graph tests' fixture reactor is built with
                 reactor.Options. The server-above-reactor test and its
                 guard-failure case pass as before.
              4. The web template's reactor docs (defineReactors' comment
                 and the starter README) describe reactor.CancelWithin as
                 the duration that sets reactor.Options.CancelAfter. The
                 README's expiry snippet passes
                 reactor.Options{CancelAfter: reactor.CancelWithin(...)}.
                 No doc in either template shows a removed form.
              5. The CLI template requires go-cli-sdk v0.4.0 and go-core
                 v0.12.0. Its check and integration suite pass, and its
                 code is unchanged.
              6. The web CHANGELOG has a v0.15.0 entry and the CLI
                 CHANGELOG a v0.3.0 entry, each in the v0.14.0/v0.2.0
                 shape: a summary and a porting list linking the
                 libraries' "Migrating from" sections (go-core v0.11.0
                 for both, go-web-sdk v0.16.0 for web, go-cli-sdk v0.3.0
                 for the CLI). The web list names RequestID's options,
                 reactor.New and event.NewRecorder's Options,
                 CancelWithin as a duration, and WriteProblem's
                 replacement by Problem.WriteFor. The CLI list names
                 StringsVar's replacement by fs.Var with
                 cli.StringsValue, and go-core's options. Each entry has
                 a Changed line, "**Breaking:** the pins move" with their
                 from-versions, and compare links. The CLI
                 context/README's version line reads template/v0.3.0.
                 Released entries that name removed forms stay as
                 history.
              7. Each template's STANDARDS.md points to the
                 construction-and-execution principle and names what it
                 covers: the integration harness's Options, and the
                 library calls the template demonstrates.
              8. Currency exits 0 in both templates.
Test seams    internal/app's exported App, from its external test
              packages. Each template's integration suite over the built
              binary.
Slices        1. Upgrade go-web-sdk-template: the pins move, and the
                 RequestID and fixture-reactor calls take their new forms
                 so the build stays green. Done when currency exits 0 and
                 the check passes (1-3).
              2. Upgrade go-cli-sdk-template: the pins move. Done when
                 currency exits 0 and the check passes (5).
              3. The web template's reactor docs, changelog v0.15.0, and
                 standards pointer (4, 6 web, 7 web).
              4. The CLI template's changelog v0.3.0, context version
                 line, and standards pointer (6 CLI, 7 CLI, 8).
Out of scope  go-web-service's and slab's migration (the service task).
              Adopting cli.StringsValue or a reactor in the templates,
              which ship no repeatable flag and no reactor. graph.After
              and graph.In, the rule's named exception. The harnesses'
              conforming shapes (integration.Start/Launch with Options,
              integration.Run's argument list), app.New, apptest.
              Rewriting released changelog entries.
Door          One-way for the two tags, which gonew @latest serves to
              every new service and CLI. Two-way for the code and docs,
              which merge by pull request before tagging.
              go-web-service requires neither template, so no other
              module's requirement moves.
Release       go-web-sdk-template template/v0.15.0,
              go-cli-sdk-template template/v0.3.0
```

## Progress

slices 4/4 committed · standards done · spec done · editor done

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
- storage-cli: storagetest.Options.Capabilities is a storage.Capabilities value; it replaces the default key rules whole when MaxKeyLength or ValidateKey is set, and a set MaxKeyLength with a nil ValidateKey accepts every key (per-field defaulting would turn a nil ValidateKey back into the default rules; a pointer was rejected).
- storage-cli: go-cli-sdk's repeatable-flag constructor is cli.StringsValue(p *[]string, value []string) flag.Value, used through fs.Var, the flag package's shape for custom flags; its concrete type stays unexported (cli.Strings rejected).
- storage-cli: azureblob's and s3's go.mod files stay on go-storage v0.7.0 (go-core v0.11.0 indirect) until go-storage v0.8.0 is tagged; release preparation moves them in a follow-up pull request before azureblob/v0.7.0 and s3/v0.3.0 tag, as go-storage's last release did (f894743).
- storage-cli: a zero storagetest.Options.Capabilities keeps the default key rules, where WithCapabilities(storage.Capabilities{}) cleared them; go-storage's v0.8.0 "Migrating from v0.7.0" section says so.
- storage-cli: test names drop the removed option names (TestFake_PageSizeIgnoresNonPositive, TestRun_FakeNoContainer, TestStringsValue_*).
- storage-cli: the changelogs mark each break inline under Added, Changed, or Removed (Keep a Changelog), with an Added entry for the new form, as go-web-sdk v0.17.0 did.
- storage-cli: standards review stated go-storage's Options.Capabilities replace-whole rule once, on the field (tests-and-docs: a contract is stated once, on its symbol), and made go-cli-sdk's aliasing test call StringsValue directly.
- messaging: New becomes messaging.Setup{Config, Broker, Outbox, Inbox, Shutdown, Logger}.Build() (*Runtime, error), with a value receiver.
- messaging: OnFailure Hook and Gate Gate are nil-default fields on Setup; Option, OnFailure, and Gated are removed with no aliases.
- messaging: Build panics naming a nil Setup.Broker or Setup.Logger; Config, Shutdown, and engine problems stay returned errors.
- messaging: the relay's options struct is outbox.RelayOptions (outbox.Options rejected: it would read as the Outbox's).
- messaging: Outbox.Relay takes (db, poll, RelayOptions{Report, Timeout, Drain, Gate}); report moves into the options, being optional; RelayOption, Timeout, Drain, and outbox.Gated are removed.
- messaging: a zero RelayOptions.Timeout means 10s and a zero Drain no last pass; a negative Timeout or Drain panics, as a poll of zero or less does.
- messaging: tags go-messaging v0.3.0, postgres/v0.3.0, nats/v0.3.0; sub-modules move onto go-core v0.12.0, sqlate v0.7.0, sqlate/postgres v0.7.0, sqlint v0.4.0, and go-messaging v0.3.0 in a release-prep follow-up pull request after the base tags (f894743); one Breaking line per breaking direct requirement; nats's untagged Unreleased line folds into v0.3.0.
- messaging: pgx v5.13.0, nats.go v1.55.0, and the mise, CI, and image pins move in the upgrade slice; only the go-core and sqlate items wait for the base tag.
- messaging: marathon-architecture applies; its declared marathon version is a minimum.
- messaging: held the nats image at 2.15.2-alpine: 2.15.3-alpine (latest) publishes an image index with no platforms and can't be pulled; go-web-service pins 2.15.2.
- messaging: golangci-lint 2.15.0's errcheck flags a bare receive on a chan error; two relay tests now assert Receive returns nil.
- messaging: the relay tests keep TestTimeoutEndsAHungHandler, TestDrain…, and TestNoDrain…, since Timeout and Drain stay field names; only the test named after RelayOption is renamed.
- messaging: RelayOptions.Report's doc lists the gate's failures, other than its wait ending, among the errors it gets.
- messaging: the typed-nil gate tests use a test-only gate type, since a nil *reactor.Gate panics in Shared.
- messaging: Build's error tests match message prefixes.
- messaging: changelog v0.3.0 omits the tool and action pins, as go-storage v0.8.0 and go-web-sdk v0.17.0 did.
- messaging: changelog v0.3.0's go-core entry also summarizes v0.11.0's graph breaks, since the requirement moves from v0.10.0.
- messaging: changelog v0.3.0 gives the Relay signature change its own Breaking entry under Changed.
- messaging: standards review kept Relay's one panic message for poll, Timeout, and Drain; the changelog quotes it.
- messaging: editor dropped go-messaging's context/README.md line naming the deleted v1.messaging record (stale since ca8621e).
- blobfs: Store.WriteFile and Store.EnsureFile take WriteFileRequest and EnsureFileRequest (named for the method, as outbox.RelayOptions was); OnComplete is a nil-default field on each, and WriteOption and the OnComplete function are removed.
- blobfs: Files.Create, Files.Ensure, Files.Move, and Directories.Move each take one request value as their last required input; a shared Place value and struct-with-Run forms were rejected.
- blobfs: Files.Continue and Directories.Continue replace (after, size) with one cursor-and-size value beside query.Page, so Listing[T] keeps two methods of the same shape.
- blobfs: VersionOptions.Version and RemoveOptions.Version are int64, and zero leaves the call unguarded, since row versions start at 1 (*int64 and sql.Null[int64] rejected).
- blobfs: tags blobfs v0.8.0 and postgres/v0.6.0; example stays untagged.
- blobfs: Files.Create and Files.Ensure share data.FileSpec{DirectoryID, Name, ContentType}, since their inputs are identical and an Ensure request type would clash with EnsureFileRequest; FileSpec kept over FileInfo, which reads as io/fs.FileInfo's existing-file metadata.
- blobfs: a supplied file id stays in the trailing CreateOptions{ID}, as on Directories.Create and Ensure; FileSpec has no ID field.
- blobfs: the move requests are data.FileMove{DirectoryID, Name, Version} and data.DirectoryMove{ParentID, Name, Version} (MoveFileRequest and MoveDirectoryRequest rejected; the "Request" suffix is kept for the Store protocols' structs).
- blobfs: the id of the row being moved stays positional, as on Find, Delete, Hold, MarkDeleting, and Complete.
- blobfs: Continue's value is data.CursorPage{After query.Cursor, Size int}, beside query.Page and named after sqlate's Continue parameter.
- blobfs: a nil WriteFileRequest.Body or EnsureFileRequest.Body panics by name, as DB, Objects, and Begin do; an empty file passes http.NoBody.
- blobfs: New takes data.Options{Engine}, as reactor.New and event.NewRecorder take theirs; the families become CreateOptions, VersionOptions, ListOptions, RemoveOptions, and SweepOptions, with field names kept from the option functions.
- blobfs: options structs are required trailing parameters, not variadic, as sqlate's Transact took *sql.TxOptions.
- blobfs: a zero option field means the default: Batch 0 is 100, StaleOlderThan 0 is no stale reclaim, and an empty ID is minted, where each was refused; a negative Batch or StaleOlderThan stays a returned error with today's message.
- blobfs: a zero FileMove or DirectoryMove Version stays a guard (query.ErrVersionMismatch); zero means unguarded only on the optional VersionOptions.
- blobfs: keys stays a positional input before FileSpec; Directories.Create and Ensure stay positional (three inputs) with a trailing CreateOptions.
- blobfs: the only new panics are a nil DB, Objects, Body, or Begin on the Store requests, in the "data: <Type>.<Field> is nil" form; every other refusal and every error and wrap message is unchanged.
- blobfs: postgres's code needs no change; it re-tags on the new base (goal decision).
- blobfs: postgres's sqlate, sqlate/postgres, and sqlint moves and example's sqlate moves wait for the blobfs v0.8.0 tag, since the tagged v0.7.0 doesn't build against sqlate v0.7.0; pgx, go-storage, azureblob, x/text, Go, golangci-lint, and the action pins move in the upgrade slice.
- blobfs: release preparation is two follow-up pull requests, postgres after blobfs v0.8.0 tags and example after postgres/v0.6.0 tags, as blobfs v0.7.0's #14 and #15 and go-storage's f894743 were.
- blobfs: postgres v0.6.0 carries one Breaking line for each breaking direct requirement; the base's v0.8.0 "Migrating from v0.7.0" section covers the zero calls that are now valid, AtVersion(0), and http.NoBody, and points to sqlate's changelog.
- blobfs: no item is held; currency exits 0 at the end of the task.
- blobfs: standards review kept Sweep's refusal messages ("the batch %d is below 1", "the stale age %s is not positive") unchanged, as the brief requires today's message for a negative value.
- blobfs: SweepOptions.Batch's zero-means-100 default stays, carried over from the old option; STANDARDS.md's no-default rule names only the stale age.
- blobfs: spec review found postgres's package docs still naming data.WithEngine (behavior 12); fixed on the task branch before the tag, since doc comments aren't the code the "no change" decision covers.
- blobfs: the conformance subtests Deletes/AtVersion and Branches/MarkAtVersion are renamed Guarded and MarkGuarded with AtVersion's removal; the base changelog records the rename for -run filters.
- blobfs: spec review ran postgres's integration suite (1409 pass) and the example against the dev stack on the new forms in a scratch copy, before any tag.
- blobfs: WriteFile and EnsureFile check DB, Objects, Body, and Begin in that order and panic naming the first nil one; EnsureFile panics before it checks its ID, so an invalid ID stays a returned IDError only on a fully wired request.
- template: both templates' STANDARDS.md point to construction-and-execution, though neither converts an API; every service and CLI starts from them.
- template: both release minor and Breaking (go-web-sdk-template template/v0.15.0, go-cli-sdk-template template/v0.3.0); a generated CLI calling the removed forms stops compiling on the new pins, as web template v0.13.0 went minor with its composition root unchanged.
- template: standards review aligned both changelogs' Breaking pin lines to the prior-entry form ("the <lib> pin moves to vX (from vY)"), not an arrow.
- template: spec review found no gaps; the CLI STANDARDS.md pointer names the harness's argument list (Run(t, args...) over processtest.Cmd) where behavior 7 says "Options", since the CLI harness has no Options type (the brief's Out of scope names that argument list).
- template: editor found no citation outside the templates that this task makes stale; standards-lab's graph-ergonomics.md (template/v0.14.0, template/v0.2.0) and html-artifacts.md (the emulator's template/v0.13.0) name the releases those goals used, and stay.

## Pending edits

- architecture · construction-and-execution principle: a context.Context parameter does not count toward the four required inputs.
- standards-lab · context/html-artifacts.md:409: replace go-web-sdk's `WriteProblem` with `Problem.WriteFor` (go-web-sdk v0.17.0); the `problem.go` citation stays, since `WriteFor` is defined there.
- standards-lab · context/messaging.md:71,74,110: replace the `messaging.Gated` option with `Setup`'s `Gate` field, `messaging.OnFailure` with `Setup`'s `OnFailure` field, and `outbox.Drain` with `outbox.RelayOptions.Drain` (go-messaging v0.3.0).
- standards-lab · context/blobfs-composition.md:23-24: replace "passing `OnComplete` and `OnDelete`" with "setting `WriteFileRequest.OnComplete`, `EnsureFileRequest.OnComplete`, and `RemoveOptions.OnDelete`" (blobfs v0.8.0).
