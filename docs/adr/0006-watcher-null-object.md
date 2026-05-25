# ADR-0006: NullWatcher — Null Object pattern for the Watcher interface

**Status:** Accepted

## Context

The watcher (`internal/services/watcher`) monitors a directory for changes to known compose filenames. Two situations
require a watcher that never emits events:

1. **Testing** — unit tests for the app or downstream components should not require fsnotify or a real filesystem.
2. **startup Watching state** — when the watcher fails to initialise (permissions, missing CWD), the Watching state uses
a NullWatcher so the UI can render without crashing.

## Decision

A `NullWatcher` adapter satisfies the `Watcher` interface. It never delivers events. It is implemented as
`internal/services/watcher/null.go`.

## Consequences

- Tests can instantiate a `NullWatcher` without filesystem setup.
- The app returns a `NullWatcher` when the real watcher fails to initialise, keeping the UI alive.
- The `Watcher` interface is the enabling prerequisite — without an interface, the Null Object pattern is not possible.
