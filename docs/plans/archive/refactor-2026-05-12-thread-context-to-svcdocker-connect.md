# refactor: Thread root context through to svcdocker.Connect

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

`cmd/root.go` creates a cancellable `ctx` (cobra command context) and passes it
to `tea.WithContext(ctx)`, which cancels the BubbleTea runtime on
SIGINT/SIGTERM. However, this context is never propagated into any model.

Both `svcdocker.Connect` call sites in `states/dashboard.go` use
`context.Background()`, meaning in-flight Docker `/_ping` requests cannot be
cancelled on shutdown. The fix threads the root `ctx` down through the
constructor chain so `svcdocker.Connect` holds a cancellable context.

`openURLCmd` in `inspector/labels.go` is fire-and-forget (`Start()`, no
`Wait()`); its `context.Background()` is left unchanged.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them
without a specific technical reason.

| # | Decision |
|---|---|
| 1 | Thread the root cobra `ctx` — not a derived context — all the way down the constructor chain. |
| 2 | On `msgs.ProjectLoaded` (project reload), the same root `ctx` is reused for the new `project.Model` and `Dashboard`. No separate cancel per reload. |
| 3 | `openURLCmd` in `labels.go` keeps `context.Background()` — fire-and-forget subprocess, cancellation is meaningless. |
| 4 | BubbleTea `Init`/`Update`/`View` signatures are unchanged. Context is carried on model structs, set at construction time only. |

---

## Implementation Steps

Each step must leave the build passing before the next begins.

### Step 1 — `states.NewDashboard`: accept and store `ctx`

File: `internal/ui/flows/dashboard/project/states/dashboard.go`

- Add `ctx context.Context` field to the `Dashboard` struct.
- Add `ctx context.Context` parameter to `NewDashboard(project, th, ctx)`.
- Store it on construction: `ctx: ctx`.
- Replace `context.Background()` on line 139 (`Init`) with `d.ctx`.
- Replace `context.Background()` on line 273 (`handleRetryTick`) with `d.ctx`.

### Step 2 — `project.New`: accept and store `ctx`

File: `internal/ui/flows/dashboard/project/project.go`

- Add `ctx context.Context` field to the `Model` struct.
- Add `ctx context.Context` parameter to `New(project, th, ctx, w, h)`.
- Store it: `ctx: ctx`.
- Pass `ctx` to `states.NewDashboard(project, th, ctx)`.

### Step 3 — root `dashboard.Model`: accept and store `ctx`

File: `internal/ui/flows/dashboard/dashboard.go`

- Add `ctx context.Context` field to the `Model` struct.
- Add `ctx context.Context` parameter to `New(cfg, logger, sc, p, th, ctx)`.
- Store it: `ctx: ctx`.
- In the `msgs.ProjectLoaded` handler, pass `m.ctx` to `project.New(...)`.

### Step 4 — `cmd/root.go`: pass `ctx` to `dashboard.New`

File: `cmd/root.go`

- Update the call from `dashboard.New(cfg, logger, sc, p, th)` to
  `dashboard.New(cfg, logger, sc, p, th, ctx)`.
- `ctx` is already in scope at this call site.

### Step 5 — Build and vet

Run `go build ./...` and `go vet ./...` to confirm no compilation errors or
type mismatches.

---

## Out of Scope

- `openURLCmd` in `inspector/labels.go` — intentionally unchanged.
- Adding timeouts or deadline propagation beyond what the root ctx provides.
- Any changes to `svcdocker.Connect` itself.
- BubbleTea `Init`/`Update`/`View` interface signatures.

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
