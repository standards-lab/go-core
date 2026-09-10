# go-core

go-core is the Core SDK of Go Elemental, the Standards Lab organization's Go implementation of
the Elemental Architecture. It provides the common primitives every Go Elemental application
type uses: layered configuration, the process lifecycle, the logger, and the pre-infrastructure
main sequence with its integration toolkit. The repository is managed with the marathon
workflow; start from `context/README.md`.

## Documentation lives in the repository

This repository documents its own implementation: the README states its place in the standard
and the principles it enhances, and each package's `doc.go` is the authority for its API. The
organization's [architecture repository](https://github.com/standards-lab/architecture) states the principles this repository follows
and holds nothing a reader can infer from this source. `context/` records only working
knowledge the code and the README do not express; do not restate documented design here. A
change that alters documented behavior updates the README and the package documentation in
the same effort, and a design note that generalizes past this repository is promoted to the
architecture repository through its `context/`.

## Repository specifics

- **Module layout** — one Go module rooted at `github.com/standards-lab/go-core`; each primitive
  is a package, and the README lists them. No sub-modules.
- **Dependencies** — the standard library alone, the enhancement the README's Standard section
  states.
- **Releases, CI, tests, tasks** — per the Go Elemental standard principles in the architecture repository
  (root `v<semver>` tags from `CHANGELOG.md`, co-located black-box tests, mise tasks).
- **Public repo.** The module resolves through the public Go proxy; CI carries no private-module
  config.
