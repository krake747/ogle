# docs: Documentation overhaul

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

The project's documentation has drifted substantially from the actual codebase. The architecture doc (`docs/arch.md`) and flow doc (`docs/flows.md`) describe package layouts and state machines that no longer exist. Several ADRs are Accepted but unimplemented (ADR-0007), Proposed but de facto standard (ADR-0010), or missing entirely (ADR-0006). The `docs/deprecated/` directory contains 2,300+ lines of historical plans for refactors that were never executed. `README.md` has no usable content. `test-coverage.md` has stale numbers. The core domain glossary (`CONTEXT.md`) and ecosystem doc (`charm-ecosystem.md`) are accurate.

This plan fixes all of this to bring documentation into alignment with the real codebase.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without a specific technical reason.

| # | Decision |
|---|---|
| 1 | ADR-0007 (State pattern for startup flow) — mark Superseded, not implement. The startup flow is simple enough that a flat model is appropriate; implementing the State pattern retroactively would be code change for documentation's sake. |
| 2 | ADR-0002, ADR-0005 — mark Superseded, noting what changed. Code evolution split packages; the ADRs' core principles still hold but their file paths/packages are wrong. |
| 3 | ADR-0006 — create retroactively from context (NullWatcher / Null Object pattern). The file was clearly lost; the decision was real and implemented. |
| 4 | ADR-0008, ADR-0009 — mark Superseded. These were Proposed and never implemented; no current need. If a future need arises, a new ADR should be authored. |
| 5 | ADR-0010 — mark Accepted. It's the de facto test standard with 4 test files following its conventions. |
| 6 | ADR-0011 — mark Superseded. Not implemented beyond servicehost; the UI test conventions weren't broadly adopted. |
| 7 | `docs/deprecated/` — archive into a timestamped subdirectory with a README explaining these are historical. Not delete — they may contain useful context for future design decisions. |
| 8 | No architectural code changes — all modifications are to `.md` files and inline doc comments only. |

---

## Implementation Steps

### Phase 1 — Fix actively misleading documentation

1. **`docs/arch.md`** — rewrite entire file with current package tree, dependency graph, and import rules. Include `internal/app/`, `internal/services/docker/` (with `connection/`, `logs/`), all UI components under `internal/ui/components/`, and correct `flows/dashboard/` + `flows/startup/` layout.

2. **`docs/flows.md`** — rewrite to match actual flows:
   - `app.go` as root orchestrator (not `dashboard.go`)
   - Startup flow: simple Model with `subState`, not State pattern
   - Dashboard: no `dashboardReloading`/`dashboardParseError` sub-states
   - Full message type table with all current types
   - Three app-level phases: `appStartup`, `appDashboard`, `appWatching`

3. **`docs/adr/0007-state-pattern-startup-flow.md`** — prepend Superseded status with date, add note about why State pattern was not implemented (startup flow proved simple enough for flat model).

4. **`docs/adr/0006-watcher-null-object.md`** — create retroactively. Status Accepted. Document the null watcher decision (NullWatcher adapter satisfying Watcher interface).

5. **`docs/adr/0002-compose-no-ui-dependencies.md`** — prepend Superseded status, noting that `compose` package was split into `scanner`, `parser`, `docker`.

6. **`docs/adr/0005-fileselect-is-a-view.md`** — prepend Superseded status, noting fileselect moved from `internal/ui/views/` to `internal/ui/components/fileselect/`.

### Phase 2 — Fix outdated documentation

7. **`docs/test-coverage.md`** — re-audit:
   - Component count (was 15, verify current)
   - Add parser, scanner, watcher to tested list
   - Update line counts
   - Check `svcdocker.Docker` interface status and update testability flags
   - Check `connection.Machine` concreteness and update topbar testability

8. **ADR status updates:**
   - `0008-parse-cancellation-context.md` — Proposed → Superseded
   - `0009-watcher-middleware-package.md` — Proposed → Superseded
   - `0010-unit-test-conventions.md` — Proposed → Accepted
   - `0011-ui-model-test-conventions.md` — Proposed → Superseded

### Phase 3 — Add missing documentation

9. **`README.md`** — add installation (`go install`), quick start, keybindings summary, requirements, link to `docs/`.

10. **`docs/deprecated/README.md`** — create with explanation of historical content. Archive all files into `docs/deprecated/archive/`.

11. **Inline doc comments:**
    - `internal/msgs/msgs.go` — add `// Package msgs defines all inter-component tea.Msg types...`
    - `internal/ui/flows/dashboard/dashboard.go` — expand package comment to describe message dispatch
    - `internal/services/docker/service.go` — expand package comment to describe Docker interface

---

## Out of Scope

- Any code changes (refactors, feature additions, bug fixes)
- Writing new user-facing documentation (keybinding reference, screenshots, tutorial) — only fixing existing docs to match reality
- Deleting any historical documents from `docs/deprecated/` — only archiving
- Adding new ADRs for undocumented decisions not covered above
- Updating auto-generated CLI docs (`docs/cli/`) — they regenerate from cobra

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
