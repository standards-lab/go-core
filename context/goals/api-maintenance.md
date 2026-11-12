# goal · api-maintenance

- **State:** building
- **Task:** blobfs
- **Branch:** blobfs

## Tasks

1. [x] core
2. [x] libraries
3. [x] storage-cli
4. [x] messaging
5. [ ] blobfs
6. [ ] template
7. [ ] service

## Task brief · blobfs

```
Problem       blobfs's data package takes optional inputs through six
              variadic option families. Files.Create, Files.Ensure,
              Files.Move, Directories.Move, and both Continue methods
              take five required inputs, and Store.WriteFile and
              Store.EnsureFile take five and six. All of these break the
              construction-and-execution rule. blobfs trails sqlate
              v0.7.0, so it must pass the new trailing transaction
              options to its Begin and Transact calls. Its postgres
              sub-module and example module can't take sqlate v0.7.0
              until the base tags. After that, postgres re-tags on the
              new base and the example moves onto both releases.
Behaviors     1. data.New(catalog, dialect, Options{Engine}). The zero
                 Options binds the baseline. A set Engine is run as
                 WithEngine's was, and New's errors are unchanged.
              2. Directories.Create and Directories.Ensure take (ctx,
                 sess, parentID, name, CreateOptions{ID}). Files.Create
                 and Files.Ensure take (ctx, sess, keys, FileSpec
                 {DirectoryID, Name, ContentType}, CreateOptions{ID}).
                 An empty ID mints one. A set ID is checked with
                 blobfs.ParseID before any SQL, as WithID's was, and a
                 row an Ensure finds keeps its own id. Outcomes and
                 refusals are unchanged.
              3. Files.Move(ctx, sess, id, FileMove{DirectoryID, Name,
                 Version}) and Directories.Move(ctx, tx, id,
                 DirectoryMove{ParentID, Name, Version}) behave as the
                 positional forms do. Version is required: a zero
                 Version is still a guard that matches no row, so the
                 call returns query.ErrVersionMismatch.
              4. Files.Hold, Files.Delete, Directories.Delete,
                 Directories.MarkDeleting, and Store.RemoveFileID take
                 a trailing VersionOptions{Version}. A zero Version
                 leaves the call unguarded. A non-zero Version guards it
                 as AtVersion did.
              5. Both listings' List takes (ctx, sess, id, req,
                 query.Page, ListOptions{IncludeDeleting}), and their
                 Continue takes (ctx, sess, id, req, CursorPage{After,
                 Size}, ListOptions). Listing[T] has both shapes. An
                 empty After, a Size below 1, and a cursor from another
                 listing are refused as today.
              6. Store.RemoveFile(ctx, db, objects, pick,
                 RemoveOptions{Version, OnDelete}). A zero Version is
                 unguarded and a nil OnDelete runs no hook. A set
                 OnDelete runs in the delete's transaction, as the
                 OnDelete option's function did.
              7. Store.Sweep(ctx, db, objects, SweepOptions{Batch,
                 OnRemoveDirectory, StaleOlderThan}). A zero Batch
                 means 100. A zero StaleOlderThan means no stale
                 reclaim. A nil OnRemoveDirectory runs no hook. A
                 negative Batch or StaleOlderThan is a returned error
                 before any SQL, with today's message.
              8. Store.WriteFile(ctx, WriteFileRequest{DB, Objects,
                 Body, Size, Begin, OnComplete}) and Store.EnsureFile(
                 ctx, EnsureFileRequest{DB, Objects, ID, Body, Size,
                 Begin, OnComplete}) give the outcomes the positional
                 forms give. A nil OnComplete runs no hook. A set one
                 runs as the OnComplete option's function did, only
                 when the call completes the file itself. An invalid
                 EnsureFileRequest.ID is a returned error, as today.
              9. WriteFile and EnsureFile panic before any I/O when DB,
                 Objects, Body, or Begin is nil, with a message such as
                 "data: WriteFileRequest.Body is nil" or
                 "data: EnsureFileRequest.DB is nil". A non-nil
                 interface that wraps a nil pointer counts as set.
              10. data no longer exports Option, WithEngine,
                 CreateOption, WithID, VersionOption, AtVersion,
                 ListOption, IncludeDeleting, WriteOption, the
                 OnComplete function, RemoveOption, the OnDelete
                 function, SweepOption, Batch, the OnRemoveDirectory
                 function, or StaleOlderThan. Every error and wrap
                 message is unchanged.
              11. The base requires sqlate v0.7.0. postgres requires
                 blobfs v0.8.0, sqlate v0.7.0, sqlate/postgres v0.7.0,
                 the sqlint tool v0.4.0, and pgx v5.13.0. example
                 requires blobfs v0.8.0, postgres v0.6.0, go-storage
                 v0.8.0, azureblob v0.7.0, sqlate v0.7.0,
                 sqlate/postgres v0.7.0, and pgx v5.13.0. Every module
                 passes its check with the workspace off, the postgres
                 integration suite passes, and the example runs against
                 the development stack.
              12. The README, the guide, and every package doc show only
                 the new forms. The base changelog's v0.8.0 has an
                 Added entry for each new type. It marks each removal
                 and signature change Breaking under Changed or
                 Removed, and marks the sqlate v0.7.0 requirement
                 Breaking. Its "Migrating from v0.7.0" section maps
                 each option to its field and gives the new forms. It
                 notes the zero calls that are now valid: Batch 0 means
                 100, StaleOlderThan 0 means no reclaim, and an empty ID
                 mints one, where each was refused. It notes that
                 AtVersion(0) becomes an unguarded call, and that a nil
                 Body panics, so an empty file passes http.NoBody. It
                 points to sqlate's changelog for the transaction
                 change. postgres v0.6.0 carries one Breaking line for
                 each breaking direct requirement.
              13. STANDARDS.md points to the construction-and-execution
                 principle. It names WriteFileRequest and
                 EnsureFileRequest, whose methods panic naming a nil DB,
                 Objects, Body, or Begin. It names FileSpec, FileMove,
                 DirectoryMove, and CursorPage, the request values, and
                 Options, CreateOptions, VersionOptions, ListOptions,
                 RemoveOptions, and SweepOptions, the options values.
              14. Currency exits 0 and every check passes.
Test seams    The data package's exported Store and its Files and
              Directories handles, from external test packages over the
              scripted-database unit tier. The conformance suite,
              datatest.Run, run by the postgres integration tier. The
              example module's run.
Slices        1. Upgrade. The base moves onto sqlate v0.7.0 and x/text
                 v0.44.0 and passes a nil *sql.TxOptions at its
                 Transact calls. The conformance suite's and tests'
                 Begin calls take *sql.TxOptions values. postgres and
                 example move onto pgx v5.13.0, and example also moves
                 onto go-storage v0.8.0 and azureblob v0.7.0. Go 1.27.4,
                 golangci-lint 2.15.0, and the two action pins move to
                 their latest. Done when the check passes and currency
                 reports only the sqlate, sqlate/postgres, and sqlint
                 items of postgres and example, which wait for release
                 preparation.
              2. Options, CreateOptions, VersionOptions, and
                 RemoveOptions on New, the create and version calls, and
                 RemoveFile, with their tests (1, 2 for directories, 4,
                 6).
              3. ListOptions and CursorPage on both listings and
                 Listing[T] (5).
              4. SweepOptions (7).
              5. FileSpec, FileMove, and DirectoryMove (2 for files, 3).
              6. WriteFileRequest and EnsureFileRequest, with their
                 panics, and the removals (8-10).
              7. The base's docs and README, changelog v0.8.0 with its
                 migration section, and the standards pointer (12-13,
                 base).
              8. Release preparation for postgres, at SHIP after blobfs
                 v0.8.0 tags, as a follow-up pull request as go-storage's
                 was (f894743). postgres moves onto blobfs v0.8.0 and the
                 sqlate, sqlate/postgres, and sqlint releases that were
                 waiting. Its integration tests move to the new forms,
                 and it gets its v0.6.0 changelog (11-12, postgres).
              9. Release preparation for example, after postgres/v0.6.0
                 tags, as a second follow-up pull request. example moves
                 onto both releases and the new forms. It stays untagged
                 (11, example; 14).
Out of scope  go-web-service's and slab's blobfs calls, which move in
              the service task. Spikes that import blobfs, which are
              outside the goal. A cursor-and-size type in sqlate. The
              data lists datatest.FileRows and DirectoryRows. Conforming
              APIs: Files.Complete, Find, FindByName, FindByPath, Path,
              IsWithin, LockTree, BranchRoots, Purge, Store.PurgeFile,
              Statements, Verify, SweepUntilDone, Patterns, the Variant
              and Engine types, postgres.Engine and Migrations,
              datatest.Run, and the root package's functions. Behavior
              changes beyond those listed.
Door          One-way for the two tags, which reach go-web-service's and
              slab's module requirements. Two-way for the code and docs,
              which merge by pull request before tagging. The example
              stays untagged.
Release       blobfs v0.8.0, postgres/v0.6.0
```

## Progress

slices 0/7 committed · standards — · spec — · editor —

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

## Pending edits

- architecture · construction-and-execution principle: a context.Context parameter does not count toward the four required inputs.
- standards-lab · context/html-artifacts.md:409: replace go-web-sdk's `WriteProblem` with `Problem.WriteFor` (go-web-sdk v0.17.0); the `problem.go` citation stays, since `WriteFor` is defined there.
- standards-lab · context/messaging.md:71,74,110: replace the `messaging.Gated` option with `Setup`'s `Gate` field, `messaging.OnFailure` with `Setup`'s `OnFailure` field, and `outbox.Drain` with `outbox.RelayOptions.Drain` (go-messaging v0.3.0).
