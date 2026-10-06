# go-core standards

The judgement calls the standards-reviewer applies to go-core, beyond what `mise run check` enforces.

- A change that alters documented behavior updates the README and the affected `doc.go` in the same change.
- A new package joins go-core only when its pattern is process-level and universal, never for one application type or one external technology.
- `architecture/standards/go-elemental/principles/dependencies.md`: the bottom-up line and no provider in a base, at `go.mod`.
- `architecture/standards/go-elemental/principles/tests-and-docs.md`: the doc.go inventory of every package, and the harness rules `process/processtest` realizes.
- `architecture/standards/go-elemental/principles/topology-and-naming.md`: the root module and its packages.
- `architecture/standards/go-elemental/principles/release-and-ci.md`: one root `v<semver>` artifact with one `CHANGELOG.md`.
- `architecture/standards/go-elemental/principles/lifecycle-and-context.md`: `lifecycle` and `process` realize it, and their documentation names the entrypoint and the composition root in its terms; a new primitive of the main sequence goes to `process` when it runs before the infrastructure exists and to `lifecycle` when it runs from `Run` on.
- `architecture/principles/validation-first.md`: a new configuration field validates in its type's Finalize, and a new `config.Options` field validates before `Load` reads the first file.
- `architecture/principles/context-architecture.md`: the README and each `doc.go` are the homes.
