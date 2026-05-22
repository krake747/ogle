# feat: carousel-service-grid

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

The Dashboard currently uses `bubbles/v2/list.Model` as a vertical sidebar for service selection. This creates a modality clash: `↑`/`↓` scroll the service list, while the same keys scroll the log viewport on the right. Workaround keys (`n`/`p`) were introduced, confirming that `list.Model` is the wrong component type for this layout.

We will replace the vertical service list with a spatial 2×2 card carousel in the left pane. Arrow keys will be permanently reserved for the log viewport. Carousel navigation uses `Tab` (focus loop), `1`–`4` (direct card focus), `PgUp`/`PgDown` (pages), and `Enter` (confirm selection).

---

## Decision Log

| # | Decision |
|---|---|
| 1 | Drop `bubbles/v2/list.Model` entirely for the Dashboard service selector. |
| 2 | Use a custom `carousel.Model` with configurable rows/cols (default 2×2). |
| 3 | `Tab` cycles focus inside the carousel only: left chevron → cards → right chevron → loop. |
| 4 | `1`–`4` move focus to the corresponding card on the current page (do not confirm). |
| 5 | `Enter` on a chevron changes page; `Enter` on a card confirms it as the Selected Service. |
| 6 | `PgUp`/`PgDown` change pages directly. |
| 7 | Arrow keys (`↑`/`↓`/`←`/`→`, `j`/`k`/`h`/`l`) are intercepted in `dashboard.go` and routed **only** to the log panel. |
| 8 | Action keys (`s`, `r`, `b`) act on the **confirmed** service, not the focused card. |
| 9 | Empty grid slots render a dim placeholder border. |
| 10 | Left pane width stays at 30 % (24 cols at 80-wide); cards will be very small — acceptable for first iteration. |
| 11 | Card border colour reflects service state (running = green, etc.). |
| 12 | Mouse interaction on the carousel is explicitly out of scope. |

---

## Implementation Steps

1. **Create `internal/ui/components/carousel/carousel.go`**
   - Constructor: `New(project *domain.Project, th *theme.Theme, zm *zone.Manager, w, h, rows, cols int)` (call with `2, 2` from `dashboard.go`).
   - State: `services`, `runtimes`, `inFlight`, `rows`, `cols`, `page`, `focus`, `confirmed`, `theme`, `w`, `h`.
   - `Update` handles: `Tab`, `1`–`4`, `Enter`, `PgUp`/`PgDown`, `s`, `r`, `b`, `ServicesPolled`, `ServiceActionCompleted`, `ThemeChanged`, `WindowSizeMsg`.
   - `View` renders: left chevron (`◀`), card grid, right chevron (`▶`), page indicator, background filler.
   - Card border colour from `colourForState` (reuse helper from `inspector.go`). Empty slots use faint `BorderBlurred` with `StateMuted` foreground. Focused card uses `BorderFocused`. Confirmed but not focused card gets a state-coloured border with slightly dimmer intensity or an inner dot.
   - Implement `ShortHelp() []key.Binding` for dashboard keymap aggregation.
   - Move `KeyToggleService`, `KeyRestart`, `KeyRebuild` bindings here or keep them in `dashboard/keymap.go` and let `carousel.Update` match against them.

2. **Delete `internal/ui/components/servicelist/servicelist.go` and `serviceitem.go`**
   Remove package and all references.

3. **Modify `internal/ui/flows/dashboard/dashboard.go`**
   - Replace field `serviceList servicelist.Model` with `carousel carousel.Model`.
   - In `New`, construct carousel with `2, 2`.
   - In `Init`, replace `servicelist` references with `carousel` in the `BindingsMsg`.
   - In `Update`:
     a. `tea.KeyPressMsg` branch: intercept carousel-only keys first, then global toggles (`q`, `,`, `w`), then arrow keys routed **only** to `m.panel`.
     b. All other message types still broadcast to both `m.carousel` and `m.panel`.
   - `handleServiceAction` forwards completion messages to `m.carousel.Update`.
   - `View` composes `m.carousel.View().Content` instead of `m.serviceList.View().Content`.

4. **Modify `internal/ui/flows/dashboard/keymap.go`**
   - Remove `servicelist.KeyPrev` and `servicelist.KeyNext` from `actions`.
   - Add new bindings for carousel page navigation (`PgUp`, `PgDown`, `Tab`).
   - Add `1`–`4` help entries.
   - Retain `s`, `r`, `b` help entries (now acting on confirmed service).

5. **Verify build and existing flows**
   - `make build` passes.
   - `fileselect` (startup flow) still works unchanged.

---

## Target Structure

```
internal/ui/components/
  carousel/
    carousel.go          # new
  servicelist/           # deleted
    servicelist.go
    serviceitem.go
```

---

## Import Path Reference

| Old | New |
|---|---|
| `internal/ui/components/servicelist` | `internal/ui/components/carousel` |
| `servicelist.Model` | `carousel.Model` |

---

## Out of Scope

- Mouse interaction on carousel cards or chevrons.
- Dynamic grid resizing (e.g. 2×3, 3×2) — the constructor accepts `rows, cols`, but changing them at runtime is not wired.
- Service name truncation improvements for very narrow terminals.
- Animated page transitions.
- Orphan display in the carousel (orphans are not part of the `domain.Project.Services` slice passed to `carousel.New`; adding them later requires a separate step).

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
