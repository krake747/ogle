# ADR-0001: msgs package is top-level, not under ui/

**Status:** Accepted

## Context

The `msgs` package defines all inter-component `tea.Msg` types used across the Bubble Tea runtime. Multiple packages
need to import it: the `watcher` package emits `FileAvailabilityChanged`, the `startup` flow emits `ProjectLoaded`, and
all views consume one or more of these message types.

An early layout option placed `msgs` under `internal/ui/` (i.e. `internal/ui/msgs/`), grouping it with the other UI
packages.

## Decision

`msgs` lives at `internal/msgs/`, as a peer to `watcher`, `compose`, and `app` — not under `ui/`.

## Consequences

- `watcher`, a non-UI package, can import `msgs` without importing any UI package. Placing `msgs` under `ui/` would
force `watcher` to import a UI package, creating an inward dependency from infrastructure to presentation.
- No circular imports are possible. The dependency graph flows: `watcher → msgs`, `ui/views/* → msgs`, never the
reverse.
- Any future non-UI package that needs to emit or consume messages can do so without touching `ui/`.
