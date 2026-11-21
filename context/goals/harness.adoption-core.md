# goal · harness.adoption-core

- **State:** idle
- **Task:** none
- **Branch:** none

## Tasks

1. [x] adopt

## Decisions

## Pending edits

- coordinator · context/graph-ergonomics.md:11-13: replace the pointer to
  `go-core/context/graph-ergonomics.md` with go-core's `docs/graph-design.md`, which holds the pain
  points, the behavior that did not change, the reasons behind the decisions, the rejected
  alternatives, and an "Implementation invariants" section on what the unexported code keeps; the
  sketches and migration detail are gone.
- coordinator · context/harness-adoption.md, "Settled for every adoption goal": add that SQL file
  headers are package documentation (in sqlate, template metadata) and stay, and that CHANGELOG
  entries record only `go.mod` requirement changes.
- architecture · standards/go-elemental/principles/release-and-ci.md, "One check per repository":
  name lint-context and lint-references among the lints that run after the module loop.
