# ADR-0011: UI model test conventions

**Status:** Proposed

## Context

ADR-0010 establishes unit test conventions for the service layer, where subjects
are pure functions. Bubble Tea models are state machines: `Init()` returns a
`tea.Cmd`; `Update(msg)` returns `(tea.Model, tea.Cmd)`. Three distinct
assertion shapes are required that ADR-0010 does not address:

1. **State transitions** — send a message, assert the returned model's type and fields.
2. **View output** — assert that rendered strings contain expected substrings.
3. **Command emissions** — call the returned `tea.Cmd`, assert the resulting `tea.Msg`.

All conventions in ADR-0010 apply as the baseline. This ADR documents only the
deltas.

## Decision

**Inherited from ADR-0010 (unchanged):** `package foo_test` black-box style,
`testify/require`, full struct equality, `t.Parallel()` everywhere, `testCase`
struct per function, `setup` callback, `t.TempDir()`, mockery-generated mocks.

**Delta — state machine driving:** Call `Init()` or `Update(msg)` and receive
`(model, cmd)`. To assert on a command, call `cmd()` to obtain the `tea.Msg`
and assert on it directly. For multi-step flows, feed the returned message back
into `Update()` to drive the next transition.

**Delta — view assertions:** `require.Contains(t, m.View(), "expected text")`.
No snapshot or golden-file testing. No exact-string equality on full rendered
output. Rationale: full-string equality is brittle under layout changes; golden
files are prohibited by ADR-0010 (`testdata/` banned) and will become
unmaintainable once themes introduce ANSI escape codes.

**Delta — exported-for-testability:** Internal message types and unexported
functions may be exported when they are the natural assertion point for a test
and exporting them is the least-invasive option. Each such export must carry a
`// Exported for testing.` comment on the type or function declaration.

**Delta — constructor injection:** When a model constructs infrastructure
internally (e.g. a filesystem watcher), refactor the constructor to accept the
dependency as a parameter so tests can inject mocks without touching the real
filesystem or network.

## Consequences

- UI packages gain a small number of exported types and functions that would
  otherwise be unexported. In practice these exports remain internal to the
  `ui` subtree and are not part of any public API.
- View tests are resilient to layout changes because they assert on substrings,
  not exact rendered output.
- Constructor injection slightly increases the surface area of `New` functions
  but makes dependency boundaries explicit and removes hidden infrastructure
  construction from model initialisation.
- Mockery-generated mocks must be regenerated when interfaces change. Generated
  files must not be edited manually (inherited from ADR-0010).
