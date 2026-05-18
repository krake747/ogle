# feat: settings implementation

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

ogle is a Bubble Tea TUI for monitoring Docker Compose projects. The settings2
component (`internal/ui/components/settings2/settings2.go`) is currently a 61-line
stub showing a 30x5 bordered box with the text "Settings". The compositor layer
architecture, visibility toggle (`,` to open, `esc`/`q`/`,` to close), and
`msgs.SettingsApplied{Theme, LogBufferCap}` message are wired end-to-end:
settings2 emits it, `app.go` handles it by reloading the theme and storing config.

No form controls exist. The stub accepts no current values at construction — it
has no knowledge of the current theme name or log buffer cap. Config file
persistence is not implemented; all changes are session-only.

This plan replaces the stub with a two-field interactive form (Theme, Log Buffer
Cap) with live-apply semantics (each field change immediately applies to the
session and persists to `config.yaml`). Esc simply closes the overlay.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them
without a specific technical reason.

| # | Decision |
|---|---|
| 1 | **Fields:** Theme (cyclical list of built-in names), Log Buffer Cap (numeric with step/fast-step). Poll interval is deferred — no consumer yet. |
| 2 | **Apply model:** Immediate — every field change emits `msgs.SettingsApplied`. Esc just closes the overlay (`SettingsVisibilityChanged{Visible: false}`). No confirm/cancel distinction. |
| 3 | **Persistence:** `config.yaml` is written on every field change via `gopkg.in/yaml.v3`. Warnings logged on failure; the program never crashes on write errors. |
| 4 | **Nav keymap:** `tab`/`shift+tab` cycles field focus. `↑`/`↓` adjusts the focused field value. `shift+↑`/`shift+↓` applies only to the numeric field (fast step). `esc`/`q`/`,` closes. |
| 5 | **Theme value constraints:** cycle through `theme.BuiltinNames()` (wrapping). |
| 6 | **Log buffer cap constraints:** min 100, max 10000, step 100, fast step 1000. |
| 7 | **Dashboard theme sync:** `msgs.SettingsApplied` is handled in `dashboard.go` to reload `m.th` via `theme.Load()` so the dashboard reflects live theme changes without a 1-frame pointer discrepancy. |

---

## Implementation Steps

Each step must leave the build passing before the next begins.

### Step 1 — Add yaml tags to Config struct

**`config/config.go`**

Add `yaml` tags to support `gopkg.in/yaml.v3` marshalling for config file
persistence:

```go
type Config struct {
	Theme        string `mapstructure:"theme" yaml:"theme"`
	LogBufferCap int    `mapstructure:"logBufferCap" yaml:"logBufferCap"`
	Log          struct {
		Level string `mapstructure:"level" yaml:"level"`
	} `mapstructure:"log" yaml:"log"`
}
```

Run `go build ./...`.

### Step 2 — Thread config path into app.Model

**`internal/app/app.go`**

Add `configPath string` field to `Model`:

```go
type Model struct {
	ctx        context.Context
	cfg        config.Config
	configDir  string
	configPath string
	dir        string
	// ... existing fields
}
```

Update `New` signature to accept `configPath string` parameter. Store both
`configDir` and `configPath` in the returned `Model` struct literal (both
construction branches).

**`cmd/root.go`** — `RunE`

Derive `configPath` from Viper's resolved config file:

```go
configPath := filepath.Join(configDir, "config.yaml")
if cf := viper.ConfigFileUsed(); cf != "" {
	configPath = cf
}
```

Pass `configPath` to `app.New`:

```go
model, cleanup, err := app.New(ctx, cfg, configDir, configPath, projectFile, logger, th)
```

Run `go build ./...`.

### Step 3 — Thread current theme name and log buffer cap into dashboard

**`internal/ui/flows/dashboard/dashboard.go`**

Add fields to `Model`:

```go
type Model struct {
	// ... existing fields ...
	themeName    string
	logBufferCap int
}
```

Update `New` signature to accept `themeName string, logBufferCap int`:

```go
func New(
	ctx context.Context,
	project *domain.Project,
	log *slog.Logger,
	th *theme.Theme,
	themeName string,
	logBufferCap int,
	zm *zone.Manager,
	w, h int,
) tea.Model {
```

Store them in the struct literal. Pass them to `settings2.New(th, themeName, logBufferCap, w, h)`.

Also handle `msgs.SettingsApplied` in `Update` to keep `m.th` in sync (so the
dashboard reflects the live theme without waiting for a re-create):

```go
case msgs.SettingsApplied:
	if th, err := theme.Load(msg.Theme, ""); err == nil {
		m.th = th
	}
	m.themeName = msg.Theme
	m.logBufferCap = msg.LogBufferCap
	return m, nil
```

Update the live-reload path (`msgs.FileAvailabilityChanged` handler) to pass
`m.themeName, m.logBufferCap` to `New`.

**`internal/app/app.go`**

Update both `dashboard.New` call sites to pass `m.cfg.Theme, m.cfg.LogBufferCap`:

- Line 88 (dashboard branch)
- Line 170 (`msgs.ProjectLoaded` handler)

Run `go build ./...`.

### Step 4 — Rewrite settings2 with form controls

**`internal/ui/components/settings2/settings2.go`**

Replace the entire file. The new `Model` struct:

```go
type Model struct {
	th           *theme.Theme
	themeNames   []string
	themeIdx     int
	logBufferCap int
	focusField   int // 0 = theme, 1 = logBufferCap
	w, h         int
}
```

Constants:

```go
const (
	fieldTheme = iota
	fieldLogBufferCap
	fieldCount
)

const (
	logBufCapMin    = 100
	logBufCapMax    = 10000
	logBufCapStep   = 100
	logBufCapFStep  = 1000
)
```

**Constructor:**

```go
func New(th *theme.Theme, themeName string, logBufferCap int, w, h int) Model {
	names := theme.BuiltinNames()
	idx := 0
	for i, n := range names {
		if n == themeName {
			idx = i
			break
		}
	}
	return Model{
		th:           th,
		themeNames:   names,
		themeIdx:     idx,
		logBufferCap: logBufferCap,
		focusField:   0,
		w:            w,
		h:            h,
	}
}
```

**Update behavior:**

- `tab`: `focusField = (focusField + 1) % fieldCount`
- `shift+tab`: `focusField = (focusField - 1 + fieldCount) % fieldCount`
- `↑` on theme: `themeIdx = (themeIdx + 1) % len(themeNames)` → emit `SettingsApplied`
- `↓` on theme: `themeIdx = (themeIdx - 1 + len(themeNames)) % len(themeNames)` → emit `SettingsApplied`
- `↑` on logBufferCap: `logBufferCap = min(logBufferCap + logBufCapStep, logBufCapMax)` → emit `SettingsApplied`
- `↓` on logBufferCap: `logBufferCap = max(logBufferCap - logBufCapStep, logBufCapMin)` → emit `SettingsApplied`
- `shift+↑` on logBufferCap: `logBufferCap = min(logBufferCap + logBufCapFStep, logBufCapMax)` → emit `SettingsApplied`
- `shift+↓` on logBufferCap: `logBufferCap = max(logBufferCap - logBufCapFStep, logBufCapMin)` → emit `SettingsApplied`
- `shift+↑`/`shift+↓` on theme: no-op (themes don't have fast-step)
- `esc`/`q`/`,`: emit `SettingsVisibilityChanged{Visible: false}`
- `tea.WindowSizeMsg`: store `w, h`

The `settingsAppliedCmd` helper:

```go
func (m Model) settingsAppliedCmd() tea.Cmd {
	return func() tea.Msg {
		return msgs.SettingsApplied{
			Theme:        m.themeNames[m.themeIdx],
			LogBufferCap: m.logBufferCap,
		}
	}
}
```

**View layout:**

```
┌─────────────────────────────────────┐
│  Settings                            │
│                                     │
│  Theme          [default        ]   │
│  Log Buffer Cap [  1000         ]   │
│                                     │
│  ↑↓ adjust · tab next · esc close   │
└─────────────────────────────────────┘
```

Render a bordered box using `m.th.StateMuted` for the outer border. The focused
field's label and value bracket use `m.th.BorderFocused` color; the blurred field
uses `m.th.BorderBlurred`. The whole form is sized to fit content (no hard-coded
dimensions).

Remove the old `boxWidth`/`boxHeight` constants — sizing is computed from content.

Run `go build ./...`.

### Step 5 — Persist config on field change

**`internal/app/app.go`** — `msgs.SettingsApplied` handler

After the existing theme loading and config storage, add config file writing:

```go
case msgs.SettingsApplied:
	th, err := theme.Load(msg.Theme, m.configDir)
	if err != nil {
		m.log.WarnContext(
			m.ctx,
			"settings: theme load failed, keeping previous",
			slog.Any("err", err),
		)
	} else {
		m.theme = th
	}

	m.cfg.Theme = msg.Theme
	m.cfg.LogBufferCap = msg.LogBufferCap

	data, marshalErr := yaml.Marshal(&m.cfg)
	if marshalErr != nil {
		m.log.WarnContext(
			m.ctx,
			"settings: failed to marshal config",
			slog.Any("err", marshalErr),
		)
	} else if writeErr := os.WriteFile(m.configPath, data, 0o600); writeErr != nil {
		m.log.WarnContext(
			m.ctx,
			"settings: failed to write config file",
			slog.Any("err", writeErr),
		)
	}

	return m, nil
```

Add imports: `"go.yaml.in/yaml/v3"` or `"gopkg.in/yaml.v3"`.

Run `go build ./...`.

### Step 6 — Build verification

```bash
go build ./...
```

All packages must compile cleanly. Manual smoke test: open settings with `,`,
cycle theme with ↑/↓ (dashboard should re-render with new theme), adjust log
buffer cap with ↑/↓ and shift+↑/↓, close with `esc`, re-open to confirm values
persist. Verify `~/.ogle/config.yaml` or custom config file contains the updated
values.

---

## Out of Scope

- Poll interval field (no consumer yet; deferred)
- User YAML themes in the picker (deferred; built-in themes only)
- Help bar keymap switching when settings overlay is open
- Debouncing config file writes
- Orphan Toggle / Label Toggle (not yet implemented in the codebase)
- Log wrap toggle as a settings field (remains an instant `w` keybinding)
- Mouse interaction with settings form fields

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
