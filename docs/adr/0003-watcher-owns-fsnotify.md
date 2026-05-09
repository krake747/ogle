# ADR-0003: watcher owns the fsnotify lifecycle

**Status:** Accepted

## Context

The application needs to monitor the working directory for Compose File changes. fsnotify provides the underlying OS-level filesystem events. The question was where to manage the fsnotify watcher: in the `watcher` package, in the startup flow, or in the app root.

## Decision

`internal/watcher` is the sole owner of the fsnotify lifecycle. Views and flows never interact with fsnotify directly. They receive `tea.Msg` values (`FileAvailabilityChanged`) from the watcher via the Bubble Tea runtime.

## Consequences

- fsnotify is isolated behind the `Watcher` interface. UI code is decoupled from filesystem concerns and cannot accidentally misuse fsnotify (e.g. subscribing twice, forgetting to close).
- The watcher can be substituted with a `NullWatcher` in tests or error conditions without any UI code changing (see ADR-0006).
- The watcher can be decorated with middleware (e.g. `LoggingWatcher`) transparently (see ADR-0009).
- The watcher runs for the entire process lifetime, including while the dashboard is active. The app root re-subscribes with `watcher.Next()` after every `FileAvailabilityChanged`.
