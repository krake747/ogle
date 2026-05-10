# feat: dashboard two-pane scaffold

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

The Dashboard is the main screen displayed after a Project is loaded. It currently
renders via the `Idle` state (`internal/ui/flows/dashboard/project/states/idle.go`),
which shows a minimal project summary and a `[dashboard not yet implemented]` notice.

The `Idle` state already receives `SetSize(w, h)` calls from `project.Model` on every
`tea.WindowSizeMsg`, and it owns a `bubbles/help` bar pinned to the last row.

This plan replaces `Idle` with a properly named `Dashboard` state that renders a
two-pane horizontal split: service list on the left, log/detail on the right. Both
panes are empty scaffolds with placeholder labels. No real content is wired in yet.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without
a specific technical reason.

| # | Decision |
|---|---|
| 1 | Pane sizing is proportional: 30% service list, 70% log/detail. Widths recalculated on every `tea.WindowSizeMsg`. |
| 2 | Split ratio is fixed for this iteration — no runtime adjustment via keypresses. |
| 3 | Help bar spans full terminal width below both panes. Pane height = `h - helpBarHeight`. |
| 4 | `Idle` is renamed to `Dashboard`: file `dashboard.go`, type `Dashboard`, constructor `NewDashboard`. The `idle.go` file is deleted. |
| 5 | Borders are focus-sensitive: focused pane gets a highlighted (normal) border; unfocused pane gets a dimmed border. |
| 6 | Initial focus is the service list (left pane). |
| 7 | Each pane renders a centred placeholder label: "services" (left), "logs" (right). |

---

## Implementation Steps

Each step must leave the build passing before the next begins.

### Step 1 — Delete `idle.go`, create `dashboard.go`

- Delete `internal/ui/flows/dashboard/project/states/idle.go`.
- Create `internal/ui/flows/dashboard/project/states/dashboard.go` with:
  - Type `Dashboard` (pointer receiver, implements `State`)
  - Constructor `NewDashboard(project *domain.Project) State`
  - Fields: `project`, `keys` (idleKeyMap — keep quit binding for now), `help`, `w`, `h`, `focus` (int, 0 = left, 1 = right)
  - `Init() tea.Cmd` — returns nil
  - `SetSize(w, h int)` — stores dimensions
  - `Update(msg tea.Msg) (State, tea.Cmd)` — handles quit key only
  - `View() string` — see Step 2

### Step 2 — Implement the two-pane `View()`

In `Dashboard.View()`:

1. Compute sizes:
   ```go
   const helpBarHeight = 1
   leftW  := d.w * 30 / 100
   rightW := d.w - leftW
   paneH  := d.h - helpBarHeight - 1  // -1 for the newline before the help bar
   ```
2. Define two lipgloss styles, chosen by `d.focus`:
   - Focused: `lipgloss.NewStyle().Width(...).Height(paneH).Border(lipgloss.NormalBorder()).BorderForeground(<highlight colour>)`
   - Unfocused: same but `BorderForeground(<dimmed colour>)`
3. Render each pane with its placeholder label centred using `lipgloss.NewStyle().Align(lipgloss.Center, lipgloss.Center)` applied to the inner content before wrapping in the border style.
4. Join panes: `lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)`
5. Append help bar below: `+ "\n" + d.help.View(d.keys)`
6. Return the combined string.

### Step 3 — Update `project.go` to reference `NewDashboard`

In `internal/ui/flows/dashboard/project/project.go`, replace any reference to
`states.NewIdle` with `states.NewDashboard`. Confirm the build passes.

### Step 4 — Verify

- `go build ./...` passes with no errors.
- Run the binary against a valid Compose File and confirm the two-pane layout
  renders correctly, with focus highlight on the left pane.
- Resize the terminal and confirm panes reflow at the new 30/70 ratio.

---

## Out of Scope

- Populating the service list pane with real service data.
- Populating the log/detail pane with a Log Stream or container info.
- Runtime split ratio adjustment.
- Tab/arrow key focus switching between panes (no behaviour change this iteration beyond what exists for quit).
- Mouse support for pane focus.
- Service Filter (`/`).

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
