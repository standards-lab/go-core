# go-core standards

The judgement calls the standards-reviewer applies to go-core, beyond what `mise run check` enforces.

- A change that alters documented behavior updates the README and the affected `doc.go` in the same change.
- `architecture/standards/go-elemental/principles/dependencies.md`: the bottom-up line and no provider in a base, held by review; go-core is the bottom, so any `require` in `go.mod` is a finding.
- `architecture/standards/go-elemental/principles/tests-and-docs.md`: the doc.go inventory rule, held by review, and the harness rules `process/processtest` realizes.
- `architecture/standards/go-elemental/principles/topology-and-naming.md`: the core SDK's layout; a new primitive is a short-named package of the root module.
- `architecture/standards/go-elemental/principles/release-and-ci.md`: one root `v<semver>` artifact with one `CHANGELOG.md`.
- `architecture/standards/go-elemental/principles/lifecycle-and-context.md`: `lifecycle` and `process` realize it; a new primitive of the main sequence goes to `process` when it runs before the infrastructure exists and to `lifecycle` when it runs from `Run` on.
- `architecture/principles/composition-root.md`: go-core has no composition root of its own; its documentation names the entrypoint and the composition root in this page's terms.
- `architecture/principles/validation-first.md`: a new configuration field validates in its type's Finalize, and a new `config.Options` field validates before `Load` reads the first file.
- `architecture/principles/context-architecture.md`: the README and each `doc.go` are the homes; `context/` records only what they do not express.
