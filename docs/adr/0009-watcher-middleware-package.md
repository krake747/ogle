# ADR-0009: Watcher middleware in internal/watcher/middleware/

**Status:** Accepted

## Context

Observability of watcher events (logging each `FileAvailabilityChanged` snapshot) was needed for debugging. Options considered:

1. **Add logging directly to `fsWatcher`** — couples logging to the concrete implementation.
2. **Add logging in `app.go`** — couples infrastructure concerns to the root model.
3. **Decorator pattern via a middleware sub-package** — `LoggingWatcher` wraps any `Watcher` and logs each event before forwarding it.

## Decision

A `middleware` sub-package is introduced at `internal/watcher/middleware/`. `LoggingWatcher` wraps a `Watcher`, logs each `FileAvailabilityChanged` snapshot at `Debug` level, and forwards the message unchanged.

`app.go` wraps the real watcher after a successful `watcher.New()`. When `watcher.New()` fails, a `NullWatcher` is used instead (see ADR-0006) and is not wrapped — `NullWatcher` never emits events, so logging middleware would have nothing to observe.

```go
w = middleware.NewLogging(w, slog.Default())
```

`slog` discards `Debug` messages at higher log levels, so the wrap is zero-cost in production.

## Consequences

- `watcher` (parent) does not import `middleware` (child). The import flows `middleware → watcher`, not the reverse. No cycle is introduced.
- The pattern is consistent with Chi HTTP middleware conventions already used in the project.
- Additional watcher behaviours (e.g. metrics, rate limiting) can be added as new decorators without modifying `fsWatcher` or `NullWatcher`.
- The `Watcher` interface (ADR-0003, ADR-0006) is the enabling prerequisite — without an interface, the decorator pattern is not possible.
