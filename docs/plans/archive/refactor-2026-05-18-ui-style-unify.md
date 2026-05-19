# refactor: ui style unify

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

ogle has three UI phases — startup (file select), dashboard (service list + panel), watching (disconnected/scanning). Currently each phase owns its own chrome:

- **Startup**: no top bar, no help bar. Just a bare list.
- **Dashboard**: has a daemon status top bar (🐳 indicator) and a keybinding help bar at the bottom.
- **Watching**: hardcodes an `"ogle\n"` title inline and a manual `"ctrl+c quit"` footer.

This means the daemon status disappears outside dashboard, the help bar only appears in dashboard, and the watching view reinvents chrome by hand. The UI looks and feels different depending on which phase you're in.

Goal: lift chrome (top bar + help bar) to the app level so every phase has a visually consistent frame.

---

## Decision Log

| # | Decision |
|---|---|
| 1 | `app.go` owns the chrome: `topbar.Model`, `helpbar.Model`, and `connection.Machine` (moved up from dashboard). |
| 2 | Top bar is a single line: phase context on the left, Docker daemon status on the right, separated by a pipe character (`|`). |
| 3 | Left context is always `ogle | <phase-specific-text>`. Static text — no animated ellipsis or loading dots on the left side. |
| 4 | Daemon status remains on the right using the existing spinner/countdown states from `connection.Machine` (rendered directly in topbar, not via `daemonstatus.Model`). |
| 5 | Help bar appears at the bottom in all three phases. Each phase sends its keymap via the existing `BindingsMsg` mechanism. |
| 6 | Each phase `View()` returns only its body content. `app.View()` composes `topbar + body + helpbar`. |
| 7 | Phases receive adjusted `tea.WindowSizeMsg` with height = `h - 1 - 2` so the composed frame fills the terminal without overflow. |

---

## Target Structure

```
app.View():
┌────────────────────────────────────────┐
│ ogle | docker-compose.yaml  🐳 ● LIVE  │  ← topbar (1 line)
├────────────────────────────────────────┤
│ servicelist | servicepanel             │  ← phase body (h-3)
│ ...                                    │
├────────────────────────────────────────┤
│ ↑↓ nav · s start · r restart · q quit  │  ← helpbar (2 lines)
└────────────────────────────────────────┘
```

---

## Implementation Steps

Each step leaves the build passing before the next begins.

### Step 1 — Create `msgs.TopbarContext`

**File:** `internal/msgs/msgs.go`

Add a new message type:

```go
type TopbarContext struct {
    Text string
}
```

No build impact yet — messages are just types.

### Step 2 — Create `internal/ui/components/topbar/topbar.go`

New package with a `Model` that:
- Holds a `*connection.Machine` (pointer, shared), a `spinner.Model` (for connecting state), a context text string, and `*theme.Theme`.
- `New(ctxText string, conn *connection.Machine, th *theme.Theme) Model`
- `Init()` fires the spinner tick.
- `Update()` handles `msgs.TopbarContext` (updates left text), `msgs.DaemonConnected/Unavailable/GraceExpired/Tick`, `spinner.TickMsg`.
- `View()` renders one line using lipgloss:
  - Left: `ctxText` styled with theme colours
  - Separator: faint ` | `
  - Right: compact daemon status (`● LIVE`, spinner when connecting, `○ (Ns)` countdown)
- Uses `connection.Machine` state values, not `daemonstatus.Model`.

**Verify:** `go build ./...` passes.

### Step 3 — Move `connection.Machine` ownership to app

**File:** `internal/app/app.go`

- Add fields: `conn *connection.Machine`, `topbar topbar.Model`, `helpbar helpbar.Model`.
- In `New()`: create `connection.Machine`, pass to `topbar.New("ogle | scanning for compose files", conn, th)`, create `helpbar.New()`.
- In `Init()`: return `tea.Batch(topbar.Init(), helpbar.Init(), watcher.Snapshot())`.

**File:** `internal/ui/flows/dashboard/dashboard.go`

- Remove `conn` field. `New()` no longer creates a `connection.Machine`.

**Verify:** `go build ./...` passes (app won't compile yet since imports need wiring).

### Step 4 — Wire daemon messages to topbar

**File:** `internal/app/app.go`

In `Update()`, before the phase switch, forward daemon-relevant messages to `m.topbar`:

```go
m.topbar, topbarCmd = m.topbar.Update(msg)
```

Apply the same forwarding pattern used for `daemonstatus.Model` in the current dashboard.

**Verify:** `go build ./...` passes.

### Step 5 — Wire phase transitions to `TopbarContext`

**File:** `internal/app/app.go`

On each phase transition, send a `msgs.TopbarContext` so the topbar updates its left text:

| Transition | Context text |
|---|---|
| Initial startup | `"ogle | scanning for compose files"` |
| Startup → Dashboard (`ProjectLoaded`) | `"ogle | <project.File>"` |
| Dashboard → Watching (`FileRemoved`) | `"ogle | disconnected"` |

Use `func() tea.Msg { return msgs.TopbarContext{...} }` batched with other init commands.

**Verify:** `go build ./...` passes.

### Step 6 — Compose chrome in app View

**File:** `internal/app/app.go`

Rewrite `View()`:

```go
func (m Model) View() tea.View {
    const topbarH = 1
    const helpbarH = 2

    var body tea.View
    switch m.phase {
    case phaseStartup:
        body = m.startup.View()
    case phaseDashboard:
        body = m.dashboard.View()
    case phaseWatching:
        body = m.watching.View()
    }

    content := lipgloss.JoinVertical(lipgloss.Top,
        m.topbar.View().Content,
        body.Content,
        m.helpbar.View().Content,
    )

    v := tea.NewView(content)
    v.Content = m.zm.Scan(v.Content)
    v.AltScreen = true
    v.MouseMode = tea.MouseModeCellMotion
    return v
}
```

Forward adjusted `tea.WindowSizeMsg` to phases:

```go
case tea.WindowSizeMsg:
    m.width = msg.Width
    m.height = msg.Height
    bodyMsg := tea.WindowSizeMsg{Width: msg.Width, Height: msg.Height - topbarH - helpbarH}
    // forward bodyMsg to active phase
```

**Verify:** `go build ./...` passes.

### Step 7 — Strip chrome from dashboard

**File:** `internal/ui/flows/dashboard/dashboard.go`

- Remove `daemonstatus.Model` field, `helpbar.Model` field.
- Remove `daemonstatus` and `helpbar` imports.
- Remove daemon/helpbar from `Init()` (keep sending `BindingsMsg` — helpbar at app level catches it).
- Remove daemon/helpbar from `Update()`.
- `View()` returns only body: `lipgloss.JoinHorizontal(lipgloss.Top, listContent, panContent)`.
- Remove `conn` field (already done in step 3).

**Verify:** `go build ./...` passes. Dashboard renders without its own chrome.

### Step 8 — Strip chrome from watching

**File:** `internal/ui/components/watching/watching.go`

- Remove the `"ogle\n"` title prefix in `View()`.
- Remove the hardcoded footer (`"ctrl+c quit"` / `"r retry   ctrl+c quit"`).
- `View()` returns only the body text (watching message + error/notice/parsing lines).

**File:** `internal/app/app.go`

- On transition to `phaseWatching`, send a `BindingsMsg` with watching keymap (quit, retry).

**Verify:** `go build ./...` passes.

### Step 9 — Add help bar to startup

**File:** `internal/ui/flows/startup/startup.go`

- `Init()` sends a `BindingsMsg` with a keymap for: ↑↓ navigation, enter to select, ctrl+c to quit.

**File:** `internal/app/app.go`

- Forward adjusted `tea.WindowSizeMsg` to startup phase (already covered by step 6).

**Verify:** `go build ./...` passes. Startup should now have a top bar and help bar.

### Step 10 — Clean up

- Remove `internal/ui/components/daemonstatus/` directory if nothing imports it anymore (verify with `rg "daemonstatus"`).
- Remove unused `helpbar` import from dashboard.
- Run `go vet ./...` and `golangci-lint run ./...`.

**Verify:** `go build ./...`, `go vet ./...`, lint passes.

---

## Import Path Reference

| Current | New |
|---|---|
| `connection.Machine` created in `dashboard` | Created in `app`, shared via pointer |
| `daemonstatus.Model` in `dashboard` | Replaced by `topbar.Model` in `app` |
| `helpbar.Model` in `dashboard` | Moved to `app` |
| Watching inline `"ogle\n"` title | Removed — topbar provides it |
| Watching inline footer | Removed — helpbar provides it |

---

## Out of Scope

- No changes to the settings overlay.
- No changes to theme or colour palette.
- No changes to key binding sets (only where they're sent from).
- No new features — pure restructuring.
- The `views/` packages (as opposed to `components/`) are left untouched.
- No changes to `servicepanel`, `servicelist`, `inspector`, `logpane`, or any service-level component.

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
