# fix: fix layout

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

`ogle` has two startup views — `fileselect` (the Project Selector) and `watching`
(the Watching state). Both currently use the same layout: content block vertically
centred, footer pinned to the last row, content horizontally centred and capped at
`maxContentWidth = 120` columns.

On a very wide terminal the content block ends up adrift in the middle of the
screen in both views. The fix grounds both views with `ogle` pinned top-left and
all functional content anchored bottom-left, just above the footer.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without
a specific technical reason.

| # | Decision |
|---|---|
| 1 | Footer stays pinned to the last row in both views. |
| 2 | Both views: the `ogle` title is pinned to row 1, column 1, no indent. |
| 3 | Both views: the functional content block (prompt/body, files/notice/error, "Parsing...") is bottom-left — no horizontal centering, anchored so its last line is 2 rows above the last row (1 blank row + footer). |
| 4 | Project Selector: structural lines ("Multiple compose files found...", notice, "Parsing...") start at column 1. The existing `"  > file.yaml"` / `"    file.yaml"` cursor indent within list entries is preserved — those spaces are part of the cursor indicator, not layout padding. |
| 5 | Both views: `contentWidth`, `leftPad`, and `pad` are removed entirely; the `maxContentWidth` constant is removed from both files as it is no longer referenced. |
| 6 | Watching view: `wrapLine` calls use `w` (full terminal width) as the wrap boundary instead of `contentWidth`. `wrapLine` itself is retained. |

---

## Implementation Steps

Each step must leave the build passing before the next begins.

1. **`internal/ui/views/fileselect/fileselect.go` — rewrite `View()`**

   Split the single content block into two independent regions:

   **Title region** (top):
   - Write `"ogle\n"` at the very start of the output — row 1, column 1, no pad.

   **Bottom block** (assembled separately, no pad):
   ```
   Multiple compose files found. Select one:

     > file-a.yaml
       file-b.yaml

   notice: ...        ← only in stateError
   Parsing...         ← only when m.parsing && !stateError
   ```
   The bottom block is the current `lines` slice minus the leading `"ogle"` and
   its trailing blank row.

   **Layout arithmetic**:
   - `availableRows := h - 1` (last row reserved for footer, unchanged).
   - `bottomStart := availableRows - len(bottomLines) - 1` (1 = blank row before footer).
   - Gap between title and bottom block: `max(bottomStart-1, 0)` blank lines
     (`-1` because the title already consumed row 1).
   - If the terminal is too small to fit everything, clamp gap to 0 — render title
     then bottom block immediately, accepting overflow rather than crashing.

   **Rendering order**:
   1. `"ogle\n"`
   2. `max(bottomStart-1, 0)` blank lines.
   3. Each line in `bottomLines` (no pad prefix; empty lines as bare `\n`).
   4. One blank line.
   5. Footer (no pad prefix, no trailing newline).

   Remove `contentWidth`, `leftPad`, `pad`, `topPad` and their computation.
   Remove the `maxContentWidth` constant — it is no longer referenced.

2. **`internal/ui/views/watching/watching.go` — rewrite `View()`**

   Apply the identical split:

   **Title region** (top):
   - Write `"ogle\n"` at row 1, column 1, no pad.

   **Bottom block**:
   ```
   Watching /some/dir for a compose file...   ← ModeCold
   — or —
   Disconnected — waiting for file.yaml...    ← ModeDisconnected

   notice: ...    ← stateNotice
   — or —
   Error: ...     ← stateError

   Parsing...     ← only when m.parsing
   ```

   **Layout arithmetic**: identical to step 1.

   **`wrapLine` calls**: replace `contentWidth` argument with `w` (full terminal
   width) in all three call sites.

   Remove `contentWidth`, `leftPad`, `pad`, `topPad` and their computation.
   Remove the `maxContentWidth` constant.

3. **Verify the build passes** — `go build ./...`.

---

## Out of Scope

- Line-wrapping of long file paths in the Project Selector list.
- Any changes to `minWidth` or `minHeight` fallback constants.
- Any other views or flows beyond `fileselect` and `watching`.

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
