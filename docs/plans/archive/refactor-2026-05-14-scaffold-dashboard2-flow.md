# refactor: scaffold dashboard2 flow

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

The ogle application uses a state pattern across its flow architecture. Currently, the root `dashboard.Model` (the orchestrator) transitions to `project.Model` (the project flow) when `msgs.ProjectLoaded` is received from the startup flow. This couples the project-specific UI entirely to the post-load flow.

To cleanly separate concerns, a new `dashboard2` flow is being scaffolded as a replacement for the current project flow. The existing `internal/ui/flows/dashboard/project/` code remains untouched for reference and will be phased out after the new flow is complete.

This refactoring establishes a clear separation:
- **Startup flow** owns File Discovery, file selection, and parsing.
- **Root orchestrator** owns watcher lifecycle and message routing.
- **dashboard2 flow** (new) will own all post-load UI and Docker connectivity.

The current project dashboard is kept in place to avoid breaking the build and to provide reference material during the redesign.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without a specific technical reason.

| # | Decision |
|---|---|
| 1 | Parsing is not a dashboard concern. The startup flow owns parsing and hands off a `Project` via `msgs.ProjectLoaded`. |
| 2 | The root `dashboard.Model` remains the orchestrator and owns watcher lifecycle, startup flow, and message routing. |
| 3 | On `msgs.ProjectLoaded`, the root orchestrator transitions to the new `dashboard2` flow instead of the existing `project.Model`. |
| 4 | The existing `internal/ui/flows/dashboard/project/` package is left untouched and remains on disk for reference. It is not deleted or imported. |
| 5 | The `dashboard2` stub is minimal — it implements `tea.Model` and displays "dashboard2". No fields, no complex initialization. |

---

## Implementation Steps

1. **Create `internal/ui/flows/dashboard2/dashboard2.go`** — a minimal `tea.Model` that displays "dashboard2".
   - Export `New()` function that returns a `tea.Model`.
   - Implement `Init() tea.Cmd` (return `nil`).
   - Implement `Update(tea.Msg) (tea.Model, tea.Cmd)` (return `(m, nil)` for all messages).
   - Implement `View() tea.View` (return `tea.NewView("dashboard2")`).

2. **Modify `internal/ui/flows/dashboard/dashboard.go`**
   - Add import: `"github.com/ma-tf/ogle/internal/ui/flows/dashboard2"`
   - In `Update()`, find the `msgs.ProjectLoaded` case.
   - Replace the current `m.current = project.New(...)` line with `m.current = dashboard2.New()`.
   - Remove the now-unused `project` import.

3. **Verify the build passes** — run `go build ./...` to ensure no compilation errors.

4. **Test the startup → ProjectLoaded → dashboard2 flow manually** — start ogle, complete the startup flow (scan/select/parse), and verify the screen displays "dashboard2".

---

## Out of Scope

- Implementing the full dashboard2 UI (logging, service list, inspector, etc.).
- Deleting or modifying `internal/ui/flows/dashboard/project/`.
- Changing the startup flow or root orchestrator logic.
- Modifying `cmd/root.go` or any entry point wiring.
- Implementing Docker connectivity logic in dashboard2 (that comes in a later phase).
- Live Reload re-parsing at runtime (deferred to a separate plan).

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
