# feat: Settings page

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

ogle is a Bubble Tea TUI for monitoring Docker Compose projects. The project flow
state machine has a single concrete state today (`states.Dashboard`), governed by
a `State` interface in `internal/ui/flows/dashboard/project/states/`. The
project-level orchestrator is `project.Model`; the root orchestrator is
`dashboard.Model`.

`CONTEXT.md` defines **Settings** as an in-session overlay that lets the user
adjust configuration values without leaving the TUI or editing the config file.
The concept is defined but not yet implemented. The three fields in scope are
poll interval, log buffer cap, and theme. The first two have no consumers yet
(State Polling and Log Buffer are planned but unimplemented) — the config fields
and Settings form are added now so they are ready when those features land.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them
without a specific technical reason.

| # | Decision |
|---|---|
| 1 | **Fields in scope:** poll interval, log buffer cap, theme. |
| 2 | **Architecture:** `states.Settings` — new concrete project State, replaces Dashboard while open. Consistent with ADR-0007 state machine pattern. |
| 3 | **Trigger key:** `,` from Dashboard. |
| 4 | **Apply behavior:** deferred (confirm=`enter`, cancel=`esc`). Exception: theme change is live within the Settings view — the form re-renders with the selected theme immediately. All changes commit or revert on confirm/cancel. |
| 5 | **Field navigation:** `tab`/`shift+tab` cycles focus; `↑/↓` adjusts the focused field; `shift+↑/↓` fast-steps numeric fields. |
| 6 | **Poll interval:** default 2s, min 1s, max 60s, step 1s, fast-step 5s. |
| 7 | **Log buffer cap:** default 1000, min 100, max 10 000, step 100, fast-step 1 000. |
| 8 | **Uncapped buffer:** deferred. |
| 9 | **Persistence to config file:** deferred. Settings changes are session-only. |
| 10 | **Theme picker:** built-in themes only (`default`, `catppuccino_mocha`). User YAML themes deferred. |
| 11 | **`msgs.SettingsApplied`** carries plain scalar values (`Theme string`, `PollInterval time.Duration`, `LogBufferCap int`). `dashboard.Model` reloads the theme via `theme.Load(msg.Theme, m.configDir)`. To support this, `configDir` is added as a field to `dashboard.Model` and threaded in through `dashboard.New`. |

---

## Implementation Steps

Each step must leave the build passing before the next begins.

> **Prerequisite — check for active test plan.**
> `docs/plans/test-2026-05-11-ui-model-test-conventions.md` also modifies
> `dashboard.New` (Step 4 of that plan adds `w svcwatcher.Watcher, watcherErr error`).
> If that plan has already been implemented, the starting signature for Step 4
> of this plan is:
> `New(cfg config.Config, logger *slog.Logger, sc scanner.Scanner, p parser.Parser, th *theme.Theme, w svcwatcher.Watcher, watcherErr error)`
> and the resulting signature after adding `configDir` becomes:
> `New(cfg config.Config, configDir string, logger *slog.Logger, sc scanner.Scanner, p parser.Parser, th *theme.Theme, w svcwatcher.Watcher, watcherErr error)`
> Adjust Step 4 and `cmd/root.go` accordingly. If the test plan has not yet run,
> proceed as written.

### Step 1 — Extend `config.Config` and defaults

**`config/config.go`**

Add `"time"` to the import block, then add two fields:

```go
type Config struct {
    ProjectFile  string        `mapstructure:"project-file"`
    Theme        string        `mapstructure:"theme"`
    PollInterval time.Duration `mapstructure:"poll-interval"`
    LogBufferCap int           `mapstructure:"log-buffer-cap"`
    Log struct {
        Level string `mapstructure:"level"`
    } `mapstructure:"log"`
}
```

**`config.yaml`**

Add defaults:

```yaml
log:
  level: debug
project-file:
poll-interval: 2s
log-buffer-cap: 1000
```

**`cmd/root.go`** — `initialiseConfig`

Add Viper defaults as fallback (config file may omit these fields):

```go
viper.SetDefault("poll-interval", 2*time.Second)
viper.SetDefault("log-buffer-cap", 1000)
```

Verify that `viper.Unmarshal` decodes the YAML string `"2s"` into `time.Duration`
correctly. Viper v1.21 uses `github.com/go-viper/mapstructure/v2` and includes
`StringToTimeDurationHookFunc` by default, so this should work without changes.
If it does not (zero value for `PollInterval` after unmarshal), add an explicit
decode hook using the v2 import path:

```go
import "github.com/go-viper/mapstructure/v2"

viper.Unmarshal(&cfg, func(dc *mapstructure.DecoderConfig) {
    dc.DecodeHook = mapstructure.ComposeDecodeHookFunc(
        mapstructure.StringToTimeDurationHookFunc(),
        dc.DecodeHook,
    )
})
```

Run `go build ./...` to confirm.

### Step 2 — Add `theme.BuiltinNames()`

**`internal/ui/theme/builtin.go`**

```go
// BuiltinNames returns the names of all built-in themes in display order.
func BuiltinNames() []string {
    return []string{"default", "catppuccino_mocha"}
}
```

### Step 3 — Add `msgs.SettingsApplied`

**`internal/msgs/msgs.go`**

Add `"time"` to the import block, then add:

```go
// SettingsApplied is emitted by states.Settings when the user confirms changes.
// dashboard.Model handles it to update the active configuration for the session.
type SettingsApplied struct {
    Theme        string
    PollInterval time.Duration
    LogBufferCap int
}
```

### Step 4 — Thread `configDir` into `dashboard.Model`

**`internal/ui/flows/dashboard/dashboard.go`**

Add `configDir string` to `Model`:

```go
type Model struct {
    cfg       config.Config
    configDir string
    dir       string
    logger    *slog.Logger
    scanner   scanner.Scanner
    parser    parser.Parser
    theme     *theme.Theme
    w         svcwatcher.Watcher
    current   tea.Model
    width     int
    height    int
}
```

Update `New` signature:

```go
func New(cfg config.Config, configDir string, logger *slog.Logger, sc scanner.Scanner, p parser.Parser, th *theme.Theme) Model
```

Handle `msgs.SettingsApplied` in `Update`:

```go
case msgs.SettingsApplied:
    th, err := theme.Load(msg.Theme, m.configDir)
    if err != nil {
        m.logger.Warn("settings: theme load failed, keeping previous", slog.Any("err", err))
    } else {
        m.theme = th
    }
    m.cfg.Theme = msg.Theme
    m.cfg.PollInterval = msg.PollInterval
    m.cfg.LogBufferCap = msg.LogBufferCap
    return m, nil
```

**`cmd/root.go`** — `RunE`

Pass `configDir` to `dashboard.New`:

```go
model := dashboard.New(cfg, configDir, logger, sc, p, th)
```

Run `go build ./...`.

### Step 5 — Expand `states.NewDashboard` and `project.New` signatures

`states.Dashboard` must store the current config values to pass them to
`states.NewSettings` when `,` is pressed.

**`internal/ui/flows/dashboard/project/states/dashboard.go`**

Add fields to `Dashboard`:

```go
type Dashboard struct {
    project      *domain.Project
    theme        *theme.Theme
    themeName    string
    pollInterval time.Duration
    logBufferCap int
    keys         dashboardKeyMap
    help         help.Model
    serviceList  servicelist.Model
    layout       paneLayout
    focus        int
}
```

Update `NewDashboard`:

```go
func NewDashboard(project *domain.Project, th *theme.Theme, themeName string, poll time.Duration, logBufCap int) State
```

**`internal/ui/flows/dashboard/project/project.go`**

Update `New`:

```go
func New(project *domain.Project, th *theme.Theme, themeName string, poll time.Duration, logBufCap int, w, h int) Model
```

**`internal/ui/flows/dashboard/dashboard.go`** — `msgs.ProjectLoaded` handler

```go
case msgs.ProjectLoaded:
    m.current = project.New(msg.Project, m.theme, m.cfg.Theme, m.cfg.PollInterval, m.cfg.LogBufferCap, m.width, m.height)
    return m, m.current.Init()
```

Run `go build ./...`.

### Step 6 — Implement `states.Settings`

Create **`internal/ui/flows/dashboard/project/states/settings.go`**.

#### Field constants

```go
const (
    fieldTheme = iota
    fieldPollInterval
    fieldLogBufferCap
    fieldCount
)
```

#### Struct

```go
type Settings struct {
    project *domain.Project

    // original values — restored on cancel
    origThemeName string
    origTheme     *theme.Theme
    origPoll      time.Duration
    origCap       int

    // staged values
    themeIdx     int
    themeNames   []string
    themes       []*theme.Theme // parallel to themeNames; preloaded at construction
    pollInterval time.Duration
    logBufferCap int

    // live theme — index into themes; updated immediately on theme field change
    liveTheme *theme.Theme

    // UI
    focusField int
    keys       settingsKeyMap
    help       help.Model
    w, h       int
}
```

#### Constructor

Themes are preloaded at construction time so cycling never calls `theme.Load`
during `Update` — avoiding a fragile `os.ReadFile` call relative to the working
directory.

```go
func NewSettings(project *domain.Project, themeName string, poll time.Duration, logBufCap int, th *theme.Theme) State {
    names := theme.BuiltinNames()
    themes := make([]*theme.Theme, len(names))
    idx := 0
    for i, n := range names {
        themes[i], _ = theme.Load(n, "") // built-ins resolve without configDir
        if n == themeName {
            idx = i
        }
    }
    return &Settings{
        project:       project,
        origThemeName: themeName,
        origTheme:     th,
        origPoll:      poll,
        origCap:       logBufCap,
        themeIdx:      idx,
        themeNames:    names,
        themes:        themes,
        pollInterval:  poll,
        logBufferCap:  logBufCap,
        liveTheme:     themes[idx],
        keys:          defaultSettingsKeys,
        help:          help.New(),
    }
}
```

#### Key map

```go
type settingsKeyMap struct {
    Next      key.Binding // tab
    Prev      key.Binding // shift+tab
    Inc       key.Binding // up
    Dec       key.Binding // down
    FastInc   key.Binding // shift+up
    FastDec   key.Binding // shift+down
    Confirm   key.Binding // enter
    Cancel    key.Binding // esc
}
```

```go
//nolint:gochecknoglobals
var defaultSettingsKeys = settingsKeyMap{
    Next:    key.NewBinding(key.WithKeys("tab"),        key.WithHelp("tab", "next")),
    Prev:    key.NewBinding(key.WithKeys("shift+tab"),  key.WithHelp("shift+tab", "prev")),
    Inc:     key.NewBinding(key.WithKeys("up"),         key.WithHelp("↑", "inc")),
    Dec:     key.NewBinding(key.WithKeys("down"),       key.WithHelp("↓", "dec")),
    FastInc: key.NewBinding(key.WithKeys("shift+up"),   key.WithHelp("shift+↑", "fast inc")),
    FastDec: key.NewBinding(key.WithKeys("shift+down"), key.WithHelp("shift+↓", "fast dec")),
    Confirm: key.NewBinding(key.WithKeys("enter"),      key.WithHelp("enter", "confirm")),
    Cancel:  key.NewBinding(key.WithKeys("esc"),        key.WithHelp("esc", "cancel")),
}
```

#### `Update` behaviour

- `tab` / `shift+tab`: advance/retreat `focusField` modulo `fieldCount`.
- On `fieldTheme`, `up`/`down`: cycle `themeIdx` (wrapping). After each change,
  set `liveTheme = s.themes[s.themeIdx]` — no I/O; the slice was preloaded at
  construction.
- On `fieldPollInterval`, `up`/`shift+up`: increment by 1s / 5s, clamp to `[1s, 60s]`.
  `down`/`shift+down`: decrement, clamp.
- On `fieldLogBufferCap`, `up`/`shift+up`: increment by 100 / 1 000, clamp to `[100, 10 000]`.
  `down`/`shift+down`: decrement, clamp.
- `enter`: return `(NewDashboard(project, liveTheme, selectedThemeName, poll, cap), settingsAppliedCmd)`.  
  `settingsAppliedCmd` is a `tea.Cmd` closure defined as:
  ```go
  settingsAppliedCmd := func() tea.Msg {
      return msgs.SettingsApplied{
          Theme:        s.themeNames[s.themeIdx],
          PollInterval: s.pollInterval,
          LogBufferCap: s.logBufferCap,
      }
  }
  ```
- `esc`: return `(NewDashboard(project, origTheme, origThemeName, origPoll, origCap), nil)`.

#### `View`

Render a centred form. The focused field is highlighted using a colour from
`liveTheme` (e.g. `BorderFocused` foreground). The entire form is styled using
`liveTheme` so selecting a different theme reflects immediately.

Approximate layout:

```
Settings

  Theme          [default        ]
  Poll Interval  [  2s           ]
  Log Buffer Cap [  1000         ]

  enter confirm · esc cancel · tab next
```

#### `SetSize`

Store `w`, `h`; call `s.help.SetWidth(w)`.

### Step 7 — Add `,` binding to Dashboard and open Settings

**`internal/ui/flows/dashboard/project/states/dashboard.go`**

Add `Settings key.Binding` to `dashboardKeyMap` and `defaultDashboardKeys`:

```go
Settings: key.NewBinding(
    key.WithKeys(","),
    key.WithHelp(",", "settings"),
),
```

In `Dashboard.Update`, handle the key:

```go
if key.Matches(msg, d.keys.Settings) && !d.serviceList.IsFiltering() {
    return NewSettings(d.project, d.themeName, d.pollInterval, d.logBufferCap, d.theme), nil
}
```

Add `d.keys.Settings` to `combinedKeyMap.ShortHelp()`.

Run `go build ./...`.

### Step 8 — Verify

```
go build ./...
```

All packages must compile cleanly. No runtime behaviour changes for poll interval
or log buffer cap (no consumers yet). Theme switching and Settings navigation
should be manually smoke-tested.

---

## Out of Scope

- **State Polling implementation** — `PollInterval` is wired into config but has no consumer yet.
- **Log Buffer implementation** — `LogBufferCap` is wired into config but has no consumer yet.
- **Uncapped log buffer** — deferred.
- **Persistence to config file** — deferred; Settings changes are session-only.
- **User YAML themes in the theme picker** — deferred; built-in themes only.
- **Tests for `states.Settings`** — deferred per ADR-0011 (project flow states deferred until the project flow stabilises).

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
