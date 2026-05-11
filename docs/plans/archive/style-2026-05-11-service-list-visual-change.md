# style: service list visual change

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

The service list (`internal/ui/components/servicelist/servicelist.go`) renders a bubbles `list.Model` where the compose filename (e.g. `docker-compose.yaml`) is set as `l.Title` and service names appear as flat list items. The title and item labels sit at the same horizontal position, giving no visual indication that the services belong to the file named above them.

The change adds two purely visual cues: the title is styled as a dimmed section header, and service items are indented 2 columns beneath it.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without a specific technical reason.

| # | Decision |
|---|---|
| 1 | Show hierarchy via **both** a styled title and indented items — not one alone |
| 2 | Apply the indent via **delegate styles** (`PaddingLeft`) not by prefixing spaces in `serviceItem.Title()` |
| 3 | Indent width is **2 columns** |
| 4 | Title style: **bold + dimmed foreground** (`lipgloss.Color("240")`) — recedes without competing with selected items |
| 5 | Title style is **constant** — does not change on pane focus/blur |
| 6 | **No separator** between title and first item — style contrast is sufficient |

---

## Implementation Steps

### Step 1 — Add `PaddingLeft(2)` to all title styles on the base delegate

In `New()`, after constructing `base` (`list.NewDefaultDelegate()`) and before wrapping it in `hoverDelegate`, apply padding to all three title style variants:

```go
base.Styles.NormalTitle   = base.Styles.NormalTitle.PaddingLeft(2)
base.Styles.SelectedTitle = base.Styles.SelectedTitle.PaddingLeft(2)
base.Styles.DimmedTitle   = base.Styles.DimmedTitle.PaddingLeft(2)
```

`hoverDelegate.Render` copies `d.DefaultDelegate` before mutating, so the hover highlight will inherit the padding automatically — no change to `hoverDelegate.Render` is required.

### Step 2 — Override `l.Styles.Title` with bold + dimmed style

In `New()`, after `l.Styles.TitleBar = l.Styles.TitleBar.PaddingBottom(0)` (line 76), add:

```go
l.Styles.Title = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("240"))
```

This is a one-time override; `SetProject()` only updates `l.Title` (the string) and list items — it does not reset styles, so no mirroring is needed in `SetProject()`.

### Step 3 — Verify build and behaviour

```
go build ./...
go test ./...
```

Visually confirm in a running session that:
- The compose filename appears bold and grey above the items
- Each service name is indented 2 columns
- The hover highlight covers the full item row including the indent area
- Behaviour under filter mode and on live reload (`SetProject`) is unchanged

---

## Out of Scope

- Focus-aware title styling (title stays constant regardless of which pane is focused)
- Separator line between title and first item
- Changes to `domain.ServiceDef`, `dashboard.go`, or any file other than `servicelist.go`
- Multi-file / multi-project grouping (the list is currently always a single compose file)

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
