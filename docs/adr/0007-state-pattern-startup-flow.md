# ADR-0007: State pattern for startup flow; states in a sub-package

**Status:** Superseded
**Superseded by:** Decision to keep the startup flow as a simple model. After implementation, the startup flow proved simple enough (82 lines, 4 states with straightforward transitions) that the State pattern overhead was unnecessary. The current code uses a `Model` with direct message dispatch and delegates to a single `fileSelect` sub-model.
**Date:** 2026-05-24

## Context

The startup flow (`internal/ui/flows/startup/startup.go`) originally used a `state int` enum with four parallel `switch` statements across `Init`, `Update`, `View`, and `forwardToActiveView`. A `prevState` field existed solely because the `Parsing` state is invisible — it must remember the last visible state as an integer to know which sub-model to render. This required secondary switches to resolve the correct view.

Two alternatives were considered:

1. Keep the enum-based switch approach, refactor to reduce duplication.
2. Replace with the State pattern: each state is a concrete struct implementing a `State` interface.

## Decision

The State pattern is adopted. Each startup state (`Scanning`, `Watching`, `Selecting`, `Parsing`) is a concrete struct in `internal/ui/flows/startup/states/`. They implement a `State` interface with `Init()`, `Update()`, and `View()` methods. `startup.Model` shrinks to two fields and delegates all three methods to `m.current`.

`Error` was considered as a fifth state but rejected. Parse failures transition directly back to `Watching` or `Selecting` with the error set on the sub-model, which already carries error/notice sub-states (`SetError`, `SetNotice`). A wrapping `Error` struct adds no unique behaviour.

States live in a `states/` sub-package (not in `startup/` directly) to keep the startup package thin and to make states independently testable.

## Consequences

- `prevState int` is eliminated. `Parsing` holds the actual display `State` object (`Parsing.display State`), so `View()` delegates to it directly — no secondary switch.
- Adding a new startup state requires adding one file in `states/` and one transition; no switch statements need updating.
- The `startup.go` file is reduced to a factory (`New`) and three one-line delegation methods.
- States are value types; transitions return new state instances. This is safe with Bubble Tea's immutable model convention.
- `startup.Model` shrinks to **two** fields (`dir string`, `current State`) — not three as originally stated. `cfg` is removed entirely; `dir` is retained for cross-cutting `msgs.WatcherError` transitions.
- The `states/` sub-package introduces an additional import path but avoids a circular dependency: `startup` imports `states`; `states` imports `ui/views/*` directly. This import direction is intentional — state objects own transitions that instantiate view sub-models (`Watching`, `Selecting`). If view construction were delegated to the startup flow instead, the flow would need to understand each state's view requirements, which defeats the purpose of the pattern.
