# ADR-0005: fileselect is a view, not a top-level package

**Status:** Superseded
**Superseded by:** Code evolution — fileselect moved from `internal/ui/views/` to `internal/ui/components/fileselect/`
during a UI components refactor. Its role as a Bubble Tea sub-model controlled by the startup flow remains unchanged.
**Date:** 2026-05-24

## Context

The Project Selector (file picker shown when File Discovery finds 2+ valid Compose Files) could have been structured as
a standalone top-level package (e.g. `internal/fileselect/`) or as a UI view under `internal/ui/views/`.

## Decision

The file picker lives at `internal/ui/views/fileselect/`. It is a view — a Bubble Tea sub-model rendered by the startup
flow.

## Consequences

- The package hierarchy reflects the component's role: it is a rendered UI component, not a domain or infrastructure
concern.
- It imports `msgs` and `ui/components` like all other views, following the uniform view dependency pattern.
- The startup flow (its only consumer) controls its lifecycle directly, passing file lists in and receiving
`FileSelected` messages out.
- Moving it to top-level would imply it has standalone concerns outside the UI, which is not the case.
