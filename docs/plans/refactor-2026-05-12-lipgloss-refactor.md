# refactor: lipgloss refactor

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

`ogle` is a Bubble Tea TUI application. The `internal/ui/theme` package declares itself
the single source of all style definitions in the UI layer — lipgloss must not be imported
outside it. In practice three files breach or under-use that contract:

- `internal/ui/views/watching/watching.go` re-implements word-wrap (`wrapLine`) and
  vertical anchoring by hand instead of using lipgloss `Width`/`Height` primitives.
- `internal/ui/components/inspector/labels.go` emits raw ANSI `\x1b[4m` for URL-hover
  underline, bypassing the theme seam entirely.
- `internal/ui/components/inspector/header.go` implements `truncate()` (rune-level
  truncation) and `padColumns()` (byte-level column arithmetic) instead of using lipgloss
  `MaxWidth`/`Width`/`Inline` — `padColumns()` contains a latent bug where it uses `len()`
  (byte count) instead of display-width for the gap calculation.

All three issues reduce the depth of the theme module and make the UI layer inconsistent:
some modules use lipgloss for layout, others replicate what it already provides.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without a
specific technical reason.

| # | Decision |
|---|---|
| 1 | `URLHover` is a `lipgloss.Style` field on `Theme` (not `color.Color`), because the style is underline + optional foreground — not a bare colour. |
| 2 | Both `Default()` and `CatppuccinoMocha()` set `URLHover = lipgloss.NewStyle().Underline(true)` — plain underline, no foreground colour override. |
| 3 | `urlHoverColor` is added to the user YAML schema; when set it applies `Foreground(color)` to the base underline style, preserving the underline. |
| 4 | `watching.go` uses `lipgloss.NewStyle().Width(w).Render(text)` for word-wrap. `Width(w)` pads short lines to `w`; trailing spaces are acceptable in the alt-screen context. |
| 5 | Vertical anchoring in `watching.go` is kept as integer arithmetic (`h - 1 - bodyLines - 1 - 1`); the simplification comes from deleting `wrapLine` and the `[]string` assembly loop, not from replacing the arithmetic itself. |
| 6 | `truncate()` and `padColumns()` are deleted. Column layout in `header.go` and `labels.go` uses `Width(n).MaxWidth(n).Inline(true)` per column. |
| 7 | The latent ANSI-before-truncation bug in `labels.go` (underline applied to `val` before `truncate(val, …)`) is fixed as a side-effect: lipgloss `MaxWidth` is ANSI-aware and the style is applied inside the render pipeline after plain-text width is measured. |
| 8 | `shortID()` and `formatAge()` in `header.go` are untouched — they are domain formatting helpers, not layout primitives. |

---

## Implementation Steps

Each step must leave the build passing before the next begins.

### Step 1 — Replace manual layout in `watching.go`

File: `internal/ui/views/watching/watching.go`

1. Add `charm.land/lipgloss/v2` import.
2. Replace every `wrapLine(text, w)` call in `View()` with
   `lipgloss.NewStyle().Width(w).Render(text)`. Each call returns a padded, word-wrapped
   string with embedded newlines.
3. Replace `len(bottomLines)` line-counting with
   `strings.Count(body, "\n") + 1` on the assembled body string.
4. Remove the `for _, l := range bottomLines { … }` assembly loop; replace with direct
   string concatenation (`body + "\n\n" + footer`).
5. Delete `wrapLine()` entirely.

### Step 2 — Add `URLHover` to `Theme`

Files: `internal/ui/theme/theme.go`, `internal/ui/theme/builtin.go`

1. Add `URLHover lipgloss.Style` field to the `Theme` struct.
2. In `Default()`: `URLHover: lipgloss.NewStyle().Underline(true)`.
3. In `CatppuccinoMocha()`: `URLHover: lipgloss.NewStyle().Underline(true)`.
4. Add `URLHoverColor string \`yaml:"urlHoverColor"\`` to `userThemeFile`.
5. Add override case in `applyOverrides()`:
   ```go
   if f.URLHoverColor != "" {
       result.URLHover = result.URLHover.Foreground(lipgloss.Color(f.URLHoverColor))
   }
   ```

This step has no callers of the new field yet — build passes unchanged.

### Step 3 — Thread theme through the Service Inspector; fix labels and header layout

Files: `internal/ui/components/inspector/inspector.go`,
       `internal/ui/components/inspector/labels.go`,
       `internal/ui/components/inspector/header.go`,
       `internal/ui/flows/dashboard/project/states/dashboard.go`

#### `inspector.go`
1. Add `theme *theme.Theme` field to `Model`.
2. Update `New` signature: `func New(service domain.ServiceDef, th *theme.Theme) Model`.
3. Pass `m.theme` as third argument to `m.labels.view()`.

#### `labels.go`
1. Add `charm.land/lipgloss/v2` import.
2. Update `view` signature: `func (m labelsModel) view(width, height int, th *theme.Theme) string`.
3. Replace the column layout per row:
   - `keyW := width / 2`, `valW := width - keyW - 2`
   - `keyBlock := lipgloss.NewStyle().Width(keyW).MaxWidth(keyW).Inline(true).Render(pair.key)`
   - When `m.hover == i && m.ctrlHeld && isURL(pair.value)`:
     `valBlock = th.URLHover.MaxWidth(valW).Inline(true).Render(pair.value)`
     else: `valBlock = lipgloss.NewStyle().MaxWidth(valW).Inline(true).Render(pair.value)`
   - Row = `keyBlock + "  " + valBlock`
4. Delete `underline()`.

#### `header.go`
1. Add `charm.land/lipgloss/v2` import.
2. Row 1 — replace `truncate`/`padColumns` with:
   - `leftW := width / 2`, `rightW := width - leftW`
   - `left := lipgloss.NewStyle().Width(leftW).MaxWidth(leftW).Inline(true).Render(svc.Name)`
   - `right := lipgloss.NewStyle().Width(rightW).MaxWidth(rightW).Inline(true).Align(lipgloss.Right).Render(image)`
   - `row1 := left + right`
3. Row 2 — replace `truncate(fmt.Sprintf(…), width)` with
   `lipgloss.NewStyle().MaxWidth(width).Inline(true).Render(fmt.Sprintf(…))`.
4. Delete `truncate()` and `padColumns()`.

#### `dashboard.go`
1. Update call at line 120: `inspector.New(first, th)`.

---

## Out of Scope

- No changes to `hoverlist`, `servicelist`, `fileselect`, `layout.go`, or any flow
  orchestrators other than the single call-site update in `dashboard.go`.
- No new tests — the UI layer has no existing tests (ADR-0011 covers UI model test
  conventions; adding tests is a separate concern).
- No changes to theme YAML documentation or user-facing config docs.
- No changes to `shortID()` or `formatAge()` in `header.go`.
- No changes to the `image/color` interface on `HoverBackground` — only `URLHover` is
  added as a `lipgloss.Style`.

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
