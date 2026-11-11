# goal · api-maintenance

- **State:** building
- **Task:** messaging
- **Branch:** messaging

## Tasks

1. [x] core
2. [x] libraries
3. [x] storage-cli
4. [ ] messaging
5. [ ] blobfs
6. [ ] template
7. [ ] service

## Task brief · messaging

```
Problem       go-messaging's New takes six required inputs and functional
              options, and Outbox.Relay takes functional options. Both
              break the construction-and-execution rule. go-messaging
              trails go-core v0.12.0 and sqlate v0.7.0, whose reactor,
              event, and transaction calls it must adapt. Its postgres and
              nats sub-modules can't take those releases until the base
              tags, and then re-tag on it.
Behaviors     1. messaging.Setup{Config, Broker, Outbox, Inbox, Shutdown,
                 Logger, OnFailure, Gate}.Build() (*Runtime, error), with
                 a value receiver, builds the Runtime New builds today.
                 Before any I/O, it returns an error for an invalid
                 Config, a Shutdown of zero or less, or an outbox or inbox
                 engine missing a statement, with New's messages.
              2. Build panics when Broker or Logger is nil, before
                 anything else, with "messaging: Setup.Broker is nil" or
                 "messaging: Setup.Logger is nil".
              3. A nil OnFailure keeps the default hook: one warn line,
                 "messaging failure", for each Failure. A non-nil
                 OnFailure gets each Failure in its place. A nil Gate
                 means the reactors hold no gate. A set Gate is held as
                 Gated's is today: the relay holds it for each outbox
                 row, and a Consume reactor for each delivery. A non-nil
                 interface that wraps a nil pointer counts as set.
              4. messaging no longer exports New, Option, OnFailure, or
                 Gated.
              5. Outbox.Relay(db, poll, RelayOptions{Report, Timeout,
                 Drain, Gate}). The zero RelayOptions keeps today's
                 defaults: reports are discarded, each row's handler is
                 bounded at 10s, there is no last pass, and no gate is
                 held. A non-nil Report gets every error the relay
                 survives, as the report parameter does today. A
                 positive Timeout bounds each row's handler, and the
                 row's transaction at twice that. A positive Drain makes
                 the last pass after cancellation. A set Gate is held
                 shared for each row, as outbox.Gated's is today.
              6. Relay panics when poll is zero or less, or Timeout or
                 Drain is negative. A zero Timeout means 10s and no
                 longer panics.
              7. outbox no longer exports RelayOption, Timeout, Drain, or
                 Gated.
              8. Runtime.Relay keeps its behavior on the new form: the
                 configuration's RelayTimeout bounds each publish, the
                 last pass gets a quarter of the drain timeout, it holds
                 Setup.Gate, and the reactor cancels its handlers at half
                 the drain timeout. Runtime.Consume also keeps its
                 behavior.
              9. go-messaging requires go-core v0.12.0 and sqlate v0.7.0.
                 postgres requires go-messaging v0.3.0, go-core v0.12.0,
                 sqlate v0.7.0, sqlate/postgres v0.7.0, sqlint v0.4.0, and
                 pgx v5.13.0. nats requires go-messaging v0.3.0, go-core
                 v0.12.0, and nats.go v1.55.0. Every module passes its
                 check with the workspace off, and both integration
                 suites pass.
              10. The README and every package doc show only the new
                 forms. The base changelog's v0.3.0 has Added entries
                 for Setup and RelayOptions and marks each removal
                 Breaking under Changed or Removed. It marks the go-core
                 v0.12.0 and sqlate v0.7.0 requirements Breaking. Its
                 "Migrating from v0.2.0" section covers New to
                 Setup.Build and the options to RelayOptions fields,
                 notes that a zero Timeout now means the default, and
                 points to go-core's and sqlate's changelogs for their
                 call changes. postgres v0.3.0 and nats v0.3.0 each carry
                 one Breaking line for each breaking direct requirement.
                 nats's untagged Unreleased line folds into its v0.3.0.
              11. STANDARDS.md points to the construction-and-execution
                 principle and names messaging.Setup, whose Build panics
                 naming a nil Broker or Logger; outbox.RelayOptions, the
                 one options value Outbox.Relay takes; and every new
                 exported function and constructor.
              12. Currency exits 0 and every check passes.
Test seams    The exported constructors, from external test packages:
              messaging.Setup.Build, exercised through Runtime.Relay and
              Runtime.Consume over the memory broker and a scripted
              database; and Outbox.Relay with RelayOptions values, driven
              through Relay.Receive. The postgres and nats integration
              suites and messagingtest.Run cover the sub-modules.
Slices        1. Upgrade. The base moves onto go-core v0.12.0 and sqlate
                 v0.7.0 and adapts: reactor.Options and event.Options at
                 the runtime's and the conformance suite's calls, and a
                 nil *sql.TxOptions at the relay's Begin and the tests'
                 Begin and Transact. postgres moves onto pgx v5.13.0 and
                 nats onto nats.go v1.55.0. Go 1.27.4, golangci-lint
                 2.15.0, the two action pins, and the nats image move to
                 their latest. Done when the check and both integration
                 suites pass, and currency reports only the sub-modules'
                 go-core, sqlate, sqlate/postgres, and sqlint items held
                 for release preparation.
              2. outbox.RelayOptions, with the relay's tests and
                 Runtime.Relay on the new form (5-8).
              3. messaging.Setup, with the runtime's and gate's tests on
                 the new form (1-4).
              4. The base's docs and README, changelog v0.3.0 with its
                 migration section, and the standards pointer (10-11,
                 base).
              5. Release preparation, at SHIP after go-messaging v0.3.0
                 tags, as a follow-up pull request as go-storage's was
                 (f894743). It moves postgres and nats onto the base's
                 new requirement and the held releases, puts their
                 integration tests on the new forms, and writes their
                 v0.3.0 changelogs (9-10, sub-modules; 12).
Out of scope  go-web-service's messaging.New call, which moves in the
              service task. The coordinator's messaging design page,
              which is a pending edit. The variadic data lists:
              the outbox sink's Write and the internal engine check.
              Conforming APIs: Runtime.Relay and Consume, Config's
              methods, the Subscription helpers, inbox.New and Claim,
              outbox.New and Sink, memory.New, nats.New and its Broker,
              messagingtest.Run, pgtest, natstest, and the postgres
              engine, migration, and retention functions.
Door          One-way for the three tags, which reach go-web-service's
              module requirements. Two-way for the code and docs, which
              merge by pull request before tagging.
Release       go-messaging v0.3.0, postgres/v0.3.0, nats/v0.3.0
```

## Progress

slices 4/4 committed (slice 5 runs at SHIP) · standards — · spec — · editor —

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

## Pending edits

- architecture · construction-and-execution principle: a context.Context parameter does not count toward the four required inputs.
- standards-lab · context/html-artifacts.md:409: replace go-web-sdk's `WriteProblem` with `Problem.WriteFor` (go-web-sdk v0.17.0); the `problem.go` citation stays, since `WriteFor` is defined there.
- standards-lab · context/messaging.md:71,74,110: replace the `messaging.Gated` option with `Setup`'s `Gate` field, `messaging.OnFailure` with `Setup`'s `OnFailure` field, and `outbox.Drain` with `outbox.RelayOptions.Drain` (go-messaging v0.3.0).
