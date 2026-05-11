# feat: themes

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

Ogle is a Bubbletea v2 TUI application using Lipgloss v2 for styling. Currently there is no theme system — only two hardcoded `lipgloss.Color` values in `internal/ui/flows/dashboard/project/states/dashboard.go`. All other views are unstyled.

This plan introduces a named-theme system: built-in themes defined in Go, user-defined themes loaded from `~/.config/ogle/themes/`, and a `*Theme` pointer threaded through the UI layer via constructor injection.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without a specific technical reason.

| # | Decision |
|---|---|
| 1 | Theming scope is named user-selectable themes (not light/dark only, not per-color config) |
| 2 | Active theme is a shared `*Theme` pointer injected via constructors; views dereference at render time for instant runtime switching |
| 3 | Theme package lives at `internal/ui/theme/` — lipgloss must not be imported outside the UI layer |
| 4 | `Theme` struct contains a color palette (internal) and pre-composed semantic `lipgloss.Style` fields (exported); flat struct design so adding a new color/style is a one-liner |
| 5 | Built-in themes are Go vars in the `theme` package; two ship initially: `default` and `catppuccin-mocha` |
| 6 | User themes live in `~/.config/ogle/themes/` as YAML files |
| 7 | User theme YAML is flat; supports an optional `base` field naming a built-in to inherit from; only overridden fields need to be present |
| 8 | Active theme is selected via `theme: <name>` at the top level of `config.yaml` |
| 9 | Theme selector and duplicate-to-disk action live in a settings flow (not yet implemented), triggered by a keybinding from the dashboard; neither is in scope for this plan |

---

## Implementation Steps

Each step must leave the build passing before the next begins.

1. **Create `internal/ui/theme/theme.go`** — define the `Theme` struct with a color palette and semantic `lipgloss.Style` fields. Export all style fields. Keep palette fields unexported or as a nested `Palette` struct.

2. **Define built-in themes** — implement `Default()` and `CatppuccinoMocha()` constructor functions returning `*Theme`. Replace the two hardcoded `lipgloss.Color` values in `dashboard.go` with references to the appropriate theme style fields.

3. **Add theme loading logic** — implement `Load(name string, configDir string) (*Theme, error)` in the `theme` package:
   - Check `~/.config/ogle/themes/<name>.yaml` first
   - Fall back to built-in by name
   - If `base` field is present in YAML, start from that built-in and apply overrides
   - Return an error if the name resolves to nothing

4. **Wire theme into config** — add a `Theme string` field to the config struct in `config/config.go`. Default to `"default"`. Read `theme:` from `config.yaml` via Viper.

5. **Instantiate and inject at startup** — in the root model or `cmd/`, call `theme.Load(...)` once at startup to produce a `*Theme`. Pass the pointer into the root `dashboard.Model` constructor, which threads it down to child flows and states via their constructors.

6. **Replace all remaining hardcoded styles** — audit all views and states; replace any inline `lipgloss.Color` or `lipgloss.Style` literals with references to the injected `*Theme`.

---

## Target Structure

```
internal/
└── ui/
    └── theme/
        ├── theme.go        # Theme struct, Load(), palette types
        └── builtin.go      # Default(), CatppuccinoMocha()
```

---

## Out of Scope

- Settings flow and theme selector UI
- Duplicate-theme-to-disk action
- More than two built-in themes
- Live file-watching of `~/.config/ogle/themes/` for hot reload
- Per-component style overrides beyond what the `Theme` struct exposes

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
