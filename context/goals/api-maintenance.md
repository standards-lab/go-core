# goal · api-maintenance

- **State:** idle
- **Task:** none
- **Branch:** none

## Tasks

1. [x] core
2. [x] libraries
3. [x] storage-cli
4. [ ] messaging
5. [ ] blobfs
6. [ ] template
7. [ ] service

## Task brief · storage-cli

```
Problem       go-storage's storagetest.NewFake takes functional options,
              and go-cli-sdk's StringsVar takes five inputs. Both break
              the construction-and-execution rule. Both repositories
              trail go-core v0.12.0. go-storage's sub-modules re-tag on
              its new release before the next tier migrates.
Behaviors     1. storagetest.NewFake takes a trailing
                 storagetest.Options{Clock, PageSize, Capabilities,
                 NoContainer}. The zero Options keeps today's defaults:
                 time.Now stamped in UTC, a list page size of 3, the
                 default key rules, and an existing container.
              2. A non-nil Options.Clock stamps objects in place of
                 time.Now. A PageSize above zero sets the list page
                 size; zero or less means 3. An Options.Capabilities
                 with MaxKeyLength or ValidateKey set replaces the
                 default key rules whole; a set MaxKeyLength with a nil
                 ValidateKey accepts every key. NoContainer makes the
                 fake behave as a store whose container is missing, as
                 WithoutContainer does today.
              3. storagetest no longer exports Option, WithClock,
                 WithPageSize, WithCapabilities, or WithoutContainer.
              4. cli.StringsValue(p *[]string, value []string)
                 flag.Value is registered with fs.Var. It copies value
                 into *p before returning, so the default isn't
                 aliased. The first use on the command line replaces
                 the default and later uses append in order. Help shows
                 the flag as repeatable with its default, and a tree
                 dispatched again restores the default, as StringsVar
                 does today.
              5. cli no longer exports StringsVar.
              6. go-storage requires go-core v0.12.0. azureblob and s3
                 require go-storage v0.8.0. go-cli-sdk requires go-core
                 v0.12.0. Every module passes its check with the
                 workspace off.
              7. go-storage's and go-cli-sdk's package docs and comments
                 show only the new forms. Each base changelog lists the
                 removals as Breaking, with a "Migrating from" section,
                 and marks its go-core v0.12.0 requirement Breaking.
                 azureblob and s3 each carry one Breaking line for the
                 go-storage v0.8.0 requirement.
              8. Each repository's standards pointer to the
                 construction-and-execution principle names the
                 converted form: storagetest.Options and
                 cli.StringsValue.
              9. Both repositories' currency commands exit 0, and every
                 check passes.
Test seams    The exported constructors, from external test packages:
              storagetest.NewFake with Options values, exercised through
              storage.Store; and cli.StringsValue registered through
              Command.Flags() and fs.Var, driven by cli.Program{Root}.Run
              and its help output.
Slices        1. go-storage upgrade: go-core v0.12.0, nothing to adapt.
                 Done when currency exits 0 and the check passes.
              2. go-cli-sdk upgrade: go-core v0.12.0, nothing to adapt.
                 Same done condition.
              3. storagetest.Options, with the fake's and the store's
                 tests on the new form (1-3).
              4. go-storage's docs, changelog v0.8.0 with its migration
                 section, the standards pointer, and azureblob's and
                 s3's changelogs. Release preparation moves azureblob
                 and s3 onto go-storage v0.8.0 once it is tagged (6-8,
                 go-storage).
              5. cli.StringsValue, with the flag tests on the new form
                 (4-5).
              6. go-cli-sdk's docs, changelog v0.4.0 with its migration
                 section, and the standards pointer (6-8, go-cli-sdk).
Out of scope  go-web-service's NewFake calls and slab's StringsVar call,
              which move in the service task. go-cli-sdk-template's
              requirement, in the template task. blobfs's example
              module's go-storage requirement, in the blobfs task. The
              variadic data lists Command.Add, Use, Require, Exclusive,
              Usagef, scenario.Commands, WriteListing, and Reporter.Note.
              Conforming APIs: Store.Put/Get/List, storage.New, NewEnv,
              azureblob.New, s3.New, storagetest.Run and
              RunMissingContainer.
Door          One-way for the four tags, which reach every consumer's
              module requirements. Two-way for the code and docs, which
              merge by pull request before tagging.
Release       go-storage v0.8.0, azureblob/v0.7.0, s3/v0.3.0;
              go-cli-sdk v0.4.0
```

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

## Pending edits

- architecture · construction-and-execution principle: a context.Context parameter does not count toward the four required inputs.
- standards-lab · context/html-artifacts.md:409: replace go-web-sdk's `WriteProblem` with `Problem.WriteFor` (go-web-sdk v0.17.0); the `problem.go` citation stays, since `WriteFor` is defined there.
