# ADR-0010: Unit test conventions

**Status:** Proposed

## Context

The codebase had zero test coverage. A prior attempt to add tests
(`test-2026-05-10-add-unit-tests.md`) lacked sufficient design detail and
could not be implemented cleanly. A design interview was conducted to resolve
the outstanding decisions before implementation began.

The following conventions apply to all unit tests in this codebase. The parser
service (`internal/services/parser`) is the first application of these
conventions.

Options considered per decision are noted inline.

## Decision

**Test package style:** `package foo_test` (black-box) throughout. Internal
whitebox tests are not used. Rationale: forces tests to interact only through
the public API, preventing tests from encoding implementation details.

**Assertions:** `testify/assert` and `testify/require` throughout. `require`
for preconditions and single-path assertions; `assert` for independent
multi-field checks. No raw `if err != nil { t.Fatal(...) }` patterns.

**Sentinel error assertions:** store the expected sentinel as `expectedError
error` on the test struct; assert with `require.ErrorIs`. Never use
`wantErr bool` — it cannot distinguish between wrong sentinels.

**Struct equality:** assert on the full returned struct with `require.Equal`.
Do not assert on named fields selectively — partial assertions hide regressions
in unexamined fields.

**Data-driven tests:** each test function owns a local `testCase` struct
(named `testCase`, scoped to the function — two functions in one file may both
define `testCase` without conflict). Fields are grouped and commented as
`// arrange` and `// assert`. Terminology uses `expected` throughout, not
`want`.

**Fixture YAML:** inlined as a `yaml string` field on the test struct. No
`testdata/` directory. Rationale: keeps the full test case (input and expected
output) readable in one place; fixtures for this domain are small and not
shared across packages.

**Filesystem:** `t.TempDir()` with real files written in the test loop. No
filesystem abstraction layer. Where a known directory name is required (e.g.
to assert on a name derived from the parent directory), create a named
subdirectory via a `dir string` arrange field rather than attempting to predict
the generated temp path.

**Computed fields:** fields whose value cannot be known at table-definition
time (e.g. an absolute file path) are assigned in the test loop immediately
before the assertion, not hardcoded in the table.

**Mocks:** hand-written `testify/mock` structs; no codegen tooling. Live in a
`mocks/` subdirectory per package (e.g. `internal/services/parser/mocks/`).

## Consequences

- All test cases for a given function are visible in one place with no
  indirection to external fixture files.
- Full struct equality will fail loudly if any field regresses, including
  fields not under active test focus.
- The `testCase` naming convention requires developers to read the enclosing
  function name for context — the struct name alone is not self-describing.
- `t.TempDir()` ties tests to the real filesystem; tests that write files are
  slightly slower than pure in-memory tests, which is acceptable for this
  domain.
- Hand-written mocks require manual updates when interfaces change, but avoid
  a codegen dependency in the build.
