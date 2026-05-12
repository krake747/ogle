# feat: themes

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

Ogle is a Bubbletea v2 TUI application using Lipgloss v2 for styling. Currently there is no theme
system — four hardcoded `lipgloss.Color` values are scattered across three files in the UI layer.
All other views are unstyled.

This plan introduces a named-theme system: built-in themes defined in Go, user-defined themes loaded
from `~/.ogle/themes/`, and a `*Theme` pointer threaded through the UI layer via constructor
injection.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without a specific
technical reason.

| # | Decision |
|---|---|
| 1 | Theming scope is named user-selectable themes (not light/dark only, not per-color config) |
| 2 | Active theme is a shared `*Theme` pointer injected via constructors; views dereference at render time for instant runtime switching |
| 3 | Theme package lives at `internal/ui/theme/` — lipgloss must not be imported outside the UI layer |
| 4 | `Theme` struct has four exported fields only: `BorderFocused lipgloss.Style`, `BorderBlurred lipgloss.Style`, `ServiceListTitle lipgloss.Style`, `HoverBackground lipgloss.Color`. No palette struct — palette colours are locals inside each built-in constructor function. Flat struct; adding a new themeable element is a one-liner. |
| 5 | `BorderFocused` and `BorderBlurred` pre-compose `lipgloss.NormalBorder()` and the foreground colour. Call sites extend with `Width`/`Height` only. Border shape is structural — not overridable via user YAML. |
| 6 | Built-in themes are functions returning `*Theme`; two ship initially: `Default()` and `CatppuccinoMocha()` |
| 7 | User themes live in `~/.ogle/themes/<name>.yaml` — sibling to `config.yaml`, not in `~/.config/ogle/`. Active theme is selected via `theme: <name>` at the top level of `config.yaml`. |
| 8 | User theme YAML is flat; supports an optional `base` field naming a built-in to inherit from; only overridden fields need to be present. Overridable fields: `border_focused_color`, `border_blurred_color`, `service_list_title_color`, `hover_background_color`. Values are any string lipgloss accepts (hex `"#cba6f7"` or ANSI-256 index `"62"`). |
| 9 | `Load(name, configDir string) (*Theme, error)` derives `configDir` from `filepath.Dir(viper.ConfigFileUsed())`, falling back to `home + "/.ogle"`. On unresolved name it returns `Default()` + a logged `Warn`-level error — consistent with the null-object pattern (ADR-0006). |
| 10 | `hoverlist.NewDelegate` gains a `theme *Theme` parameter. `Render()` reads `theme.HoverBackground` at call time — reactive to runtime theme switches. |
| 11 | `servicelist.View()` applies `theme.ServiceListTitle` to `m.list.Styles.Title` before calling `m.list.View()`. The assignment is removed from `New()`. Single source of truth at render time. |
| 12 | `paneLayout` gains a `theme *Theme` field, set via `newPaneLayout(theme *Theme)`. `View()` reads `p.theme` for border colours. |
| 13 | `dashboard.Model` stores `*Theme` as a field. `project.New` is called inside `Update` (not the constructor), so the stored pointer is passed there. |
| 14 | Theme selector and duplicate-to-disk action live in a settings flow (not yet implemented); neither is in scope for this plan. Viper XDG multi-directory config search is a separate PR. |

---

## Hardcoded values being replaced

| File | Line | Value | Replaced by |
|---|---|---|---|
| `internal/ui/flows/dashboard/project/states/layout.go` | 95 | `lipgloss.Color("62")` | `theme.BorderFocused` foreground |
| `internal/ui/flows/dashboard/project/states/layout.go` | 96 | `lipgloss.Color("240")` | `theme.BorderBlurred` foreground |
| `internal/ui/components/servicelist/servicelist.go` | 58 | `lipgloss.Color("240")` | `theme.ServiceListTitle` |
| `internal/ui/hoverlist/hoverlist.go` | 43 | `lipgloss.Color("237")` | `theme.HoverBackground` |

---

## Constructor chain

`*Theme` must be threaded through the following sites (9 touch points across 7 files):

| Site | Change |
|---|---|
| `cmd/root.go` | Call `theme.Load(cfg.Theme, configDir)` after config init; pass `*Theme` to `dashboard.New` |
| `config/config.go` | Add `Theme string \`mapstructure:"theme"\`` with Viper default `"default"` |
| `dashboard.New` / `dashboard.Model` | Accept `*Theme`; store as field; pass to `startup.New`; pass to `project.New` inside `Update` |
| `startup.New` + `fileHandler` struct | Accept `*Theme`; store in `fileHandler`; pass to `fileselect.New` |
| `fileselect.New` | Accept `*Theme`; pass to `hoverlist.NewDelegate` |
| `project.New` | Accept `*Theme`; pass to `states.NewDashboard` |
| `states.NewDashboard` | Accept `*Theme`; pass to `servicelist.New` and `newPaneLayout` |
| `servicelist.New` | Accept `*Theme`; pass to `hoverlist.NewDelegate`; remove `l.Styles.Title` assignment |
| `hoverlist.NewDelegate` | Accept `theme *Theme`; store as field; read `theme.HoverBackground` in `Render()` |
| `newPaneLayout` | Accept `theme *Theme`; store as field; read in `View()` |

---

## Implementation Steps

Each step must leave the build passing before the next begins.

1. **Create `internal/ui/theme/theme.go`** — define the `Theme` struct with the four exported
   fields (Decision 4). Implement `Load(name, configDir string) (*Theme, error)` (Decision 9):
   check `configDir/themes/<name>.yaml` first, fall back to built-in by name, apply YAML colour
   overrides onto the base built-in's styles (Decision 8).

2. **Create `internal/ui/theme/builtin.go`** — implement `Default()` and `CatppuccinoMocha()`
   returning `*Theme`. Each function defines its palette as local variables and composes the four
   exported style fields from them (Decisions 5–6). `BorderFocused`/`BorderBlurred` pre-compose
   `lipgloss.NormalBorder()` (Decision 5).

3. **Wire theme into config** — add `Theme string \`mapstructure:"theme"\`` to `config/config.go`.
   Add `viper.SetDefault("theme", "default")` in `cmd/root.go`. Call `theme.Load(...)` in `RunE`
   after config is initialised; derive `configDir` from `filepath.Dir(viper.ConfigFileUsed())`
   with `home + "/.ogle"` fallback. Log a warning and continue on error (Decision 9).

4. **Thread `*Theme` through the constructor chain** — update all 9 touch points listed in the
   Constructor Chain section above. No behaviour changes; purely additive signatures.

5. **Replace hardcoded colour literals** — update the four sites in the Hardcoded Values table.
   Apply `ServiceListTitle` in `servicelist.View()` (Decision 11); read `BorderFocused`/`BorderBlurred`
   from `paneLayout.theme` in `layout.go` (Decision 12); read `HoverBackground` from `theme` in
   `hoverlist.Render()` (Decision 10).

---

## Target Structure

```
internal/
└── ui/
    └── theme/
        ├── theme.go      # Theme struct, Load()
        └── builtin.go    # Default(), CatppuccinoMocha()
```

---

## Out of Scope

- Settings flow and theme selector UI
- Duplicate-theme-to-disk action
- More than two built-in themes
- Live file-watching of `~/.ogle/themes/` for hot reload
- Per-component style overrides beyond what the `Theme` struct exposes
- Viper XDG multi-directory config search (separate PR)

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
