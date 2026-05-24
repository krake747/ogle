# ADR-0008: Parse cancellation via context.CancelFunc on the Parsing state

**Status:** Superseded
**Superseded by:** Decision not to implement context cancellation for parsing. The startup flow is synchronous (parse on `FileSelected`) and the dashboard re-parses inline on `FileAvailabilityChanged`. No concurrent parses exist, so cancellation is unnecessary.
**Date:** 2026-05-24

## Context

The `Parsing` state launches a `parseCmd` goroutine. A `FileAvailabilityChanged` event can arrive while the parse is in flight, superseding it. Without cancellation, the stale `parseDoneMsg` would arrive after the state has already transitioned and would be acted on incorrectly.

Options considered:

1. **Ignore stale results by sequence number** — attach a monotonic counter; discard results whose counter doesn't match current state.
2. **Context cancellation** — store a `context.CancelFunc` on the `Parsing` struct; call it when superseded; detect cancellation via `errors.Is(err, context.Canceled)`.
3. **Channel-based cancellation** — close a `done` channel to signal the goroutine.

## Decision

`Parsing` holds a `context.Context` and `context.CancelFunc`. `parseCmd` accepts the context and checks `ctx.Err()` at the start. When a `FileAvailabilityChanged` supersedes the in-flight parse, `cancel()` is called. The goroutine returns `parseDoneMsg{err: context.Canceled}`. The `Parsing.Update` handler detects this with `errors.Is(msg.err, context.Canceled)` and discards the stale result.

`cancel()` is also called on every non-stale `parseDoneMsg` path (success and failure) to prevent context leaks. `context.WithCancel` is idempotent on multiple calls.

## Consequences

- Standard library only — no custom sentinel errors or sequence counters.
- `context.Canceled` is a well-known, unambiguous sentinel; future developers will recognise it immediately.
- The goroutine is cooperative: it checks `ctx.Err()` at entry only. If the YAML parse library does not accept a context, cancellation cannot interrupt a parse in progress — the goroutine blocks to completion and the stale `parseDoneMsg` is discarded on arrival. For typical Compose Files this is acceptable; for unusually large files on slow disks, the goroutine may outlive its usefulness briefly.
- `cancel()` must be called on every exit path from `Parsing.Update` to prevent context leaks. The implementation calls it unconditionally on every `parseDoneMsg` and on every `FileAvailabilityChanged`.
