# ADR-0005: fileselect is a view, not a top-level package

**Status:** Accepted

## Context

The Project Selector (file picker shown when File Discovery finds 2+ valid Compose Files) could have been structured as a standalone top-level package (e.g. `internal/fileselect/`) or as a UI view under `internal/ui/views/`.

## Decision

The file picker lives at `internal/ui/views/fileselect/`. It is a view — a Bubble Tea sub-model rendered by the startup flow.

## Consequences

- The package hierarchy reflects the component's role: it is a rendered UI component, not a domain or infrastructure concern.
- It imports `msgs` and `ui/components` like all other views, following the uniform view dependency pattern.
- The startup flow (its only consumer) controls its lifecycle directly, passing file lists in and receiving `FileSelected` messages out.
- Moving it to top-level would imply it has standalone concerns outside the UI, which is not the case.
