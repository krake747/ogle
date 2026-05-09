# ADR-0002: compose package has no UI dependencies; ScanAll/Validate split from Parse

**Status:** Accepted

## Context

The `compose` package handles Compose File discovery and parsing. Three operations are required:

- **ScanAll** — scans CWD for candidate filenames
- **Validate** — checks whether a candidate is a valid compose YAML (cheap: file read + basic parse)
- **Parse** — full parse into an in-memory `Project` (expensive: full YAML decode + validation)

The watcher needs to test candidate files on every filesystem event. If Validate and Parse were a single operation, every filesystem event would trigger a full parse for each candidate file.

## Decision

`compose` imports only the standard library and a YAML library. It has no `ui/`, `tea`, or TUI dependencies. `ScanAll` and `Validate` are separate from `Parse`.

## Consequences

- The watcher can call `Validate` on filesystem events cheaply, without a full parse.
- `Parse` is called exactly once per file selection, not on every filesystem event.
- `compose` is independently testable without any TUI or Bubble Tea setup.
- The two-step approach (Validate then Parse) creates a narrow race window: a file can pass Validate and then fail Parse if it changes between the two calls. This is handled explicitly as the `startupError` / `fileselectError` state in the startup flow.
