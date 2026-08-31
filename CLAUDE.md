# go-core

The Core SDK of Go Minimal, the Standards Lab organization's minimal-dependency Go standard:
the common primitives useful across all Go Minimal application types — layered configuration,
the process lifecycle, and the logger. Managed with the marathon workflow; start from
`context/README.md`.

## Design is documented in the landing zone

The design and conventions of this repository are documented in the organization's
[documentation landing zone](https://github.com/standards-lab/docs) — that is the authority.
`context/` records only working knowledge the landing zone and the code do not express; do not
restate documented design here. A change that alters documented behavior updates the landing
zone page in the same effort.

## Repository specifics

- **Module layout** — one Go module rooted at `github.com/standards-lab/go-core`; each primitive
  is a package (`config`, `lifecycle`, `logging`). No sub-modules.
- **Dependencies** — the standard library alone; this repository enhances Go Minimal's
  dependency line.
- **Releases, CI, tests, tasks** — per the Go Minimal standard principles in the landing zone
  (root `v<semver>` tags from `CHANGELOG.md`, co-located black-box tests, mise tasks).
- **Public repo.** The module resolves through the public Go proxy; CI carries no private-module
  config.
