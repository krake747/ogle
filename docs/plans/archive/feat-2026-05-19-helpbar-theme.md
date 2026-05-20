# feat: helpbar theme

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

`helpbar` is a thin wrapper around `charm.land/bubbles/v2/help.Model`. It renders the active keymap delivered via `msgs.BindingsMsg`. Currently it is constructed with `helpbar.New()` — no arguments, no theme pointer — and renders using `help.DefaultDarkStyles()`: hardcoded grey values unrelated to the active theme.

Every other chrome component (`topbar`) already holds a `*theme.Theme`, applies it in `View()`, and refreshes via `msgs.ThemeChanged`. The helpbar is the only chrome component that does not participate in the theme system.

`help.Model` exposes a public `Styles help.Styles` field with seven `lipgloss.Style` slots: `ShortKey`, `ShortDesc`, `ShortSeparator`, `FullKey`, `FullDesc`, `FullSeparator`, `Ellipsis`. These can be set directly.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without a specific technical reason.

| # | Decision |
|---|---|
| 1 | New theme fields are typed `lipgloss.Style` (not `color.Color`), matching the existing `BorderFocused`, `BorderBlurred`, `ServiceListTitle` pattern. |
| 2 | Three fields added to `Theme`: `HelpKey`, `HelpDesc`, `HelpSep`. `HelpSep` covers both separator and ellipsis. |
| 3 | Color semantics: `HelpKey` ← `Text` color (primary foreground); `HelpDesc` ← `Subtext` (secondary); `HelpSep` ← `StateMuted` (muted/structural). |
| 4 | Short and Full help variants share the same style values — no separate `FullKey`/`FullDesc`/`FullSep` fields needed. |
| 5 | User YAML override keys added for all three fields so user-defined themes can customise them. |
| 6 | `helpbar` stores `*theme.Theme`; `help.Styles` is re-applied on `msgs.ThemeChanged` to stay in sync at runtime. |

---

## Implementation Steps

Each step must leave the build passing before the next begins.

**Step 1 — Extend `Theme` struct and YAML schema (`internal/ui/theme/theme.go`)**

Add to the `Theme` struct:
```go
HelpKey  lipgloss.Style
HelpDesc lipgloss.Style
HelpSep  lipgloss.Style
```

Add to `userThemeFile`:
```go
HelpKeyColor  string `yaml:"helpKeyColor"`
HelpDescColor string `yaml:"helpDescColor"`
HelpSepColor  string `yaml:"helpSepColor"`
```

Add override cases in `applyOverrides`:
```go
if f.HelpKeyColor != "" {
    result.HelpKey = result.HelpKey.Foreground(lipgloss.Color(f.HelpKeyColor))
}
// same for HelpDesc, HelpSep
```

**Step 2 — Populate in `Default()` (`internal/ui/theme/builtin.go`)**
```go
HelpKey:  lipgloss.NewStyle().Foreground(defaultWhite),
HelpDesc: lipgloss.NewStyle().Foreground(defaultBrightBlack),
HelpSep:  lipgloss.NewStyle().Foreground(defaultBrightBlack),
```

**Step 3 — Populate in `CatppuccinoMocha()` (`internal/ui/theme/catppuccino_mocha.go`)**
```go
HelpKey:  lipgloss.NewStyle().Foreground(mochaText),
HelpDesc: lipgloss.NewStyle().Foreground(mochaSubtext0),
HelpSep:  lipgloss.NewStyle().Foreground(mochaOverlay1),
```

**Step 4 — Theme-aware helpbar (`internal/ui/components/helpbar/helpbar.go`)**

- Add `th *theme.Theme` field to `Model`
- Change `New() Model` → `New(th *theme.Theme) Model`
- On construction, call a local helper `applyStyles` that sets all seven `help.Styles` slots from the three theme fields:
  - `ShortKey` / `FullKey` ← `th.HelpKey`
  - `ShortDesc` / `FullDesc` ← `th.HelpDesc`
  - `ShortSeparator` / `FullSeparator` / `Ellipsis` ← `th.HelpSep`
- In `Update`, handle `msgs.ThemeChanged`: store new theme, re-call `applyStyles`

**Step 5 — Wire in app root (`internal/app/app.go`)**

Change `helpbar.New()` → `helpbar.New(th)` (line 147).

---

## Out of Scope

- Changes to how `BindingsMsg` delivers keymaps to the helpbar
- Changes to the `msgs.ThemeChanged` broadcast path — it already reaches the helpbar
- Adding a full-help toggle to the helpbar
- Any changes to other components

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
