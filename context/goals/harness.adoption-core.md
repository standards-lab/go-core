# goal · harness.adoption-core

- **State:** building
- **Task:** adopt
- **Branch:** adopt

## Tasks

1. [ ] adopt

## Task brief · adopt

**Problem**

go-core and sqlate run marathon's harness without its checks. Neither check runs lint-context or
lint-references, and both fail lint-context. go-core's graph note is over budget, partly stale,
and names a retired roadmap goal. Neither capability map marks built or planned. Neither
STANDARDS.md has a comment rule. Both carry Go, script, and configuration comments outside the
five kinds, and neither requires a reason on a lint suppression.

**Behaviors**
1. Both repositories are current: mise-action v5.2.10 in both, and sqlate's development and
   integration PostgreSQL on 19.0-alpine, with its quick start naming 19 and acceptance passing.
2. Each `mise.toml` pins marathon's tools at 0.22.0 (`marathon/v` prefix). Currency reports a
   newer marathon tag.
3. Each check runs lint-context and lint-references after its existing steps and fails when either
   fails. A lint missing from PATH fails the check with a message naming `mise install`. The
   check's description names every step, sqlint included.
4. lint-context passes in both. Each capability map lists only `(built)` entries linking their
   package, or `(planned)` entries. go-core: nine built. sqlate: seven built packages,
   dollar-quoting planned and linked to its note, and a second engine planned with its open points
   in a short note.
5. go-core's graph note leaves `context/`. Its decisions' reasoning and rejected alternatives move
   to one go-core docs page, with the pain points each answers and any kept behavior the package
   docs don't state. The migration section, sketches, roadmap pointer, and evidence citations into
   pre-migration code drop. The README and the context README point at the page.
6. Every comment in Go source, tests, scripts, and configuration is one of the five kinds.
   Unexported-identifier docs, test-function docs, in-body comments, and test-file package
   comments go; their reasoning moves into names, exported docs, task descriptions, or a one-line
   reference comment. Script sections become named functions. SQL file headers stay: a SQL file's
   header is its package documentation and, in sqlate's format, its template metadata.
7. golangci-lint enables nolintlint with require-explanation and require-specific in every module.
8. Each STANDARDS.md cites comments.md for Go source, tests, scripts, and every configuration file
   (TOML, YAML, `.gitignore`, Dockerfile, compose), and says a data file's header is its package
   documentation (in sqlate, also template metadata).
9. Changes stay under Unreleased, with no CHANGELOG entries (no `go.mod` requirement changes). CI
   passes on both pull requests.

**Test seams**
- `mise run check` per repository, `mise run currency`, and sqlate's `mise run acceptance`.

**Slices**
1. go-core upgrade: mise-action v5.2.10.
2. sqlate upgrade: mise-action v5.2.10; PostgreSQL 19.0-alpine; quick start names 19; acceptance.
3. go-core context: capability map; graph note to a docs page.
4. go-core comments: Go, tests, scripts, configuration; nolintlint; STANDARDS comment rule.
5. go-core pin and check: marathon 0.22.0; the check runs both lints.
6. sqlate context: capability map; second-engine note; dollar-quoting linked.
7. sqlate comments, base module: root, header, query, migrate, sqltest.
8. sqlate comments, sub-modules and configuration; nolintlint; STANDARDS comment rule.
9. sqlate pin and check: marathon 0.22.0; the check runs both lints.

**Out of scope**

go-core's docs/ guide beyond the one reasoning page (v1.repository-docs), any API or behavior
change, other repositories' copies of currency.sh and .golangci.yml, releases.

**Door**

Two-way. No release; every change is internal to the two repositories.

## Progress

slices 0/9 committed · standards — · spec — · editor —

## Decisions

- adopt: SQL file headers are package documentation and stay; in sqlate they are also template
  metadata, like front matter. Comments inside a statement body follow the five kinds. (wave 1, W1)
- adopt: CHANGELOG entries record only changes to a module's `go.mod` requirements. (wave 1, W2)

## Pending edits

- coordinator · context/graph-ergonomics.md:11-13: name go-core's new docs page and what it holds
  (reasoning and rejected alternatives; no sketches or migration).
- coordinator · context/harness-adoption.md, "Settled for every adoption goal": add that SQL file
  headers are package documentation (in sqlate, template metadata) and stay, and that CHANGELOG
  entries record only `go.mod` requirement changes.
- architecture · standards/go-elemental/principles/release-and-ci.md, "One check per repository":
  name lint-context and lint-references among the lints that run after the module loop.
