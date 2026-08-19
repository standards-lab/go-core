# go-core

The base SDK of `go-minimal`, the Standards Lab organization's minimal-dependency Go standard: the
process-level packages every program in the standard builds on — layered configuration, the process
lifecycle, and the logger. Managed with the marathon workflow; start from `context/README.md`.

## Conventions are settled in the repository

The design and conventions for this SDK are recorded in `context/design/` — that is the authority.
Keep them there; do not restate them here.

## Role boundary

go-core is a marathon **code** project (`.claude/marathon.toml` declares `kind = "code"`). The
developer owns the production Go source — they apply it and answer for it. The agent writes
everything else: tests, godoc and `doc.go`, prose documentation, the files in `context/`, the
implementation guide, and the reset file.

## Repository specifics

- **Module layout** — one Go module rooted at `github.com/standards-lab/go-core`; each capability is
  a package (`config`, `lifecycle`, `logging`). No sub-modules.
- **Dependencies** — the standard library alone; at most, packages as idiomatic and stable as the
  standard library. Vendor SDKs never enter this module.
- **Releases** — the module is tagged `v<semver>` at the root from `CHANGELOG.md`, cut by
  `.github/workflows/release.yml`.
- **Tests** are co-located `{file}_test.go` files in an external black-box package
  (`package <pkg>_test`) that exercise the public API.
- **Tasks** run through `mise` (`build`, `test`, `vet`, `fmt`, `tidy`, `lint`).
- **Public repo.** The module resolves through the public Go proxy; CI carries no private-module
  config.
