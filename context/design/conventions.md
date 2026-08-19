# Package conventions

The patterns settled for this SDK's packages.

## A standard-library base

The module may depend only on packages as idiomatic and stable as the standard library itself —
`golang.org/x/…`, `google/uuid`, and the like; today it depends on the standard library alone. Heavy or
vendor-specific dependencies — cloud SDKs, database drivers — never enter this module. This keeps the
base SDK effectively free to depend on: importing one of its packages compiles no vendor SDK and pulls
nothing beyond the standard library.

## Process lifecycle and context ownership

The `lifecycle` package hosts startup, readiness, and graceful shutdown, and it fixes the ecosystem's
context-ownership convention. The composition root owns the signal context — it traps signals
(`signal.NotifyContext`) and passes the context to `Coordinator.Run`, the one blocking call that owns
the sequence. Hooks and monitors are declared while the coordinator waits, and nothing executes before
`Run`; the coordinator installs no signal handlers of its own.

Hooks describe only what is bespoke to the program. Startup hooks run concurrently and return errors —
a failure drains what did start and readiness never flips, so a probe cannot report a partially started
process. The run context reaches every startup hook, and work that outlives its hook keeps watching it.
Runtime failure is declared rather than hand-rolled: a monitored channel's first non-nil error ends the
run.

The drain is two-phase and coordinator-driven: the run context is cancelled, then each shutdown hook
runs against a fresh timeout-bounded context derived from `context.Background()`, so cleanup is not
pre-cancelled. A shutdown hook needs no cancellation guard of its own — it runs its graceful drain
(`http.Server.Shutdown`, an in-flight wait) against the drain context and returns its error into `Run`'s
joined result.

Readiness is non-monotonic: the coordinator is ready once every startup hook succeeds and not-ready
again the moment draining begins, so a `/readyz` probe reports a draining process as unavailable. A
subsystem exposes its own readiness through `lifecycle.ReadinessChecker`; its leaf components take a
plain `context.Context` rather than the coordinator, keeping them usable without it.

## Tests: co-located and black-box

Tests are `{file}_test.go` files co-located with the source they cover, in an external test package
(`package <pkg>_test`). They exercise only the public API; private infrastructure is covered transitively
through the public entry points that use it.

## doc.go and godoc

Production source is written without doc comments; the agent writes godoc. Each package has exactly one
`doc.go` holding only the package comment.
