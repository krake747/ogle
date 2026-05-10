# fix: fix layout

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

`ogle` has two startup views — `fileselect` (the Project Selector) and `watching`
(the Watching state). Both currently use the same layout: content block vertically
centred, footer pinned to the last row, content horizontally centred and capped at
`maxContentWidth = 120` columns.

On a very wide terminal the Project Selector content block ends up adrift in the
middle of the screen. The design decision is to ground it bottom-left, with the
`ogle` title remaining anchored at the top. The `watching` view is unaffected.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without
a specific technical reason.

| # | Decision |
|---|---|
| 1 | Footer stays pinned to the last row in both views. |
| 2 | Project Selector (`fileselect`): the `ogle` title is pinned to row 1, column 1, no indent. |
| 3 | Project Selector: the file list block (prompt, files, notice, "Parsing...") is bottom-left — no horizontal centering, anchored so its last line is 2 rows above the last row (1 blank row + footer). |
| 4 | Project Selector: structural lines ("Multiple compose files found...", notice, "Parsing...") start at column 1. The existing `"  > file.yaml"` / `"    file.yaml"` cursor indent within list entries is preserved — those spaces are part of the cursor indicator, not layout padding. |
| 5 | Project Selector: `contentWidth`, `leftPad`, and `pad` are removed entirely; the `maxContentWidth` constant can be kept or removed — it is unused after this change. |
| 6 | `watching` view: no change — keeps existing vertical + horizontal centering. |

---

## Implementation Steps

Each step must leave the build passing before the next begins.

1. **`internal/ui/views/fileselect/fileselect.go` — rewrite `View()`**

   Split the single content block into two independent regions:

   **Title region** (top):
   - Write `"ogle\n"` at the very start of the output — row 1, column 1, no pad.

   **Bottom block** (assembled separately, no `pad`):
   ```
   Multiple compose files found. Select one:

     > file-a.yaml
       file-b.yaml

   notice: ...        ← only in stateError
   Parsing...         ← only when m.parsing && !stateError
   ```
   The bottom block is identical to the current `lines` slice minus the leading
   `"ogle"` and its trailing blank row.

   **Layout arithmetic**:
   - `availableRows := h - 1` (last row reserved for footer, unchanged).
   - `titleRows := 1` (the `"ogle"` line).
   - `blankRow := 1` (separator before footer).
   - `bottomStart := availableRows - len(bottomLines) - blankRow`
   - `topPad` for the title is 0 (already at row 1); the gap between title and
     bottom block is `bottomStart - titleRows` blank lines.
   - If `bottomStart - titleRows < 0` (terminal too small to fit everything),
     clamp: render title then bottom block immediately, accepting overflow rather
     than crashing.

   **Rendering order**:
   1. `"ogle\n"`
   2. `(bottomStart - 1)` blank lines (gap between title and bottom block; min 0).
   3. Each line in `bottomLines` (no pad prefix).
   4. One blank line.
   5. Footer (no pad prefix).

   Remove `contentWidth`, `leftPad`, `pad` and their computation. The
   `maxContentWidth` constant can be removed too — it is no longer referenced.

2. **Verify the build passes** — `go build ./...`.

---

## Out of Scope

- Any changes to `internal/ui/views/watching/watching.go`.
- Line-wrapping of long file paths in the list (not currently implemented; out of scope).
- The `maxContentWidth` wrapping behaviour that was applied to structural lines — the bottom block is now left-aligned so wrapping is irrelevant.
- Any changes to the `watching` view's centering constants (`minWidth`, `minHeight`, `maxContentWidth`).

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
