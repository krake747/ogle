# Testing

Conventions established through a design interview on 2026-05-24. All decisions below are resolved; do not re-open
without a specific technical reason.

---

## Service-layer unit tests

These conventions apply to all unit tests in `internal/services/*` (parser, scanner, watcher, docker) and any other
package with pure-function subjects.

### Package style

`package foo_test` (black-box) throughout. Tests interact only with the public API.

If a test needs to assert on unexported state, export the necessary type or function with a `// Exported for testing.`
comment rather than switching to a white-box package.

### Assertions

`testify/require` for preconditions and single-path assertions. `testify/assert` for independent multi-field checks. No
raw `if err != nil { t.Fatal(...) }`.

### Data-driven tables

Use a `testCase` struct scoped to each test function. Single-test functions (flat named subtests) are allowed but
require approval.

Fields are grouped with `// arrange` and `// assert` section comments. Use `expected` terminology, never `want`.

```go
type testCase struct {
 name string
 // arrange
 input string
 setup func(t *testing.T, tc *testCase, dir string)

 // assert
 expected      domain.Project
 expectedError error
}
```

### Struct equality

Assert on the full returned struct with `require.Equal`. Do not assert on named fields selectively — partial assertions
hide regressions in unexamined fields.

Fields whose value cannot be known at table-definition time (e.g. file paths from `t.TempDir()`) are patched inside the
`setup` callback, not hardcoded in the table.

### Error assertions

Store the expected sentinel as `expectedError error` on `testCase`. Assert with `require.ErrorIs`. Never use `wantErr
bool` — it cannot distinguish between wrong sentinels.

```go
if tc.expectedError != nil {
    require.ErrorIs(t, err, tc.expectedError)
    return
}
require.NoError(t, err)
```

### Setup callback

Signature: `func(t *testing.T, tc *testCase, dir string)`. The test loop creates one `t.TempDir()` per subtest and
passes it to `setup`. The callback must not use `t.Fatal` — it receives the subtest's `*testing.T` for proper failure
isolation.

### Fixtures

Inline only. No `testdata/` directory. Small input data (e.g. YAML strings, filenames) is a field on `testCase`. Keeps
the full test case — input and expected output — readable in one place.

File writes are performed in the test loop after `setup` runs, guarded by whether the relevant input field is non-empty:

```go
if tc.yaml != "" {
    require.NoError(t, os.WriteFile(tc.path, []byte(tc.yaml), 0o600))
}
```

### Parallelism

`t.Parallel()` at the top of every test function and every `t.Run` subtest. Each subtest operates on its own
`t.TempDir()` — shared mutable state across cases is not permitted.

### Mocks

Generated via [mockery](https://vektra.github.io/mockery/). Named `MockFoo` in package `mocks`, constructed via
`NewMockFoo(t)`. Live in a `mocks/` subdirectory per package. Generated files must not be edited manually. Mocks are
kept even when currently unused — the seam exists for future tests.

---

## UI model tests

These conventions apply to testing Bubble Tea `tea.Model` state machines (components, flows). All service-layer
conventions apply as the baseline; only the deltas are documented here.

### State machine driving

Call `Init()` or `Update(msg)` and receive `(model, cmd)`. For multi-step flows, feed the returned `tea.Msg` from one
transition into the next `Update()` call.

```go
m, cmd := m.Update(msgs.DaemonConnected{})
```

### Command assertions

Call the returned `tea.Cmd` to obtain the `tea.Msg`, then assert on it directly. Mandatory when the cmd type is
deterministic.

```go
m, cmd := m.Update(msgs.DaemonConnected{})
require.NotNil(t, cmd)
result := cmd()
msg, ok := result.(msgs.SomeMsg)
require.True(t, ok)
require.Equal(t, expectedField, msg.Field)
```

Skip cmd-calling only for inherently non-deterministic cmds (e.g. `tea.Tick`).

### View assertions

`require.Contains(t, m.View(), "expected content")`. No golden files, no full-string equality.

Asserting on colour outputs: compute the expected colour from the theme, not from a hardcoded string.

```go
expected := lipgloss.NewStyle().Foreground(th.StateRunning).Render("")
require.Contains(t, m.View(), expected)
```

### Constructor injection

Every component receives its dependencies as constructor parameters. No model constructs infrastructure internally.
`app.go` is exempted.

---

## Example

```go
package parser_test

import (
 "os"
 "path/filepath"
 "testing"

 "github.com/stretchr/testify/require"

 "github.com/ma-tf/ogle/internal/domain"
 "github.com/ma-tf/ogle/internal/services/parser"
)

func TestParse(t *testing.T) {
 t.Parallel()

 type testCase struct {
  name string
  // arrange
  yaml  string
  setup func(t *testing.T, tc *testCase, dir string)
  path  string

  // assert
  expected      domain.Project
  expectedError error
 }

 cases := []testCase{
  {
   name: "valid file with name field",
   yaml: "name: myproject\nservices:\n  web:\n    image: nginx\n",
   setup: func(t *testing.T, tc *testCase, dir string) {
    tc.path = filepath.Join(dir, "compose.yaml")
    tc.expected.File = tc.path
   },
   expected: domain.Project{
    Name: "myproject",
    Services: []domain.ServiceDef{
     {Name: "web", Image: "nginx"},
    },
   },
  },
 }

 for _, tc := range cases {
  t.Run(tc.name, func(t *testing.T) {
   t.Parallel()

   tc.setup(t, &tc, t.TempDir())

   if tc.yaml != "" {
    require.NoError(t, os.WriteFile(tc.path, []byte(tc.yaml), 0o600))
   }

   svc := parser.New()
   result, err := svc.Parse(tc.path)

   if tc.expectedError != nil {
    require.ErrorIs(t, err, tc.expectedError)
    require.Nil(t, result)
    return
   }

   require.NoError(t, err)
   require.Equal(t, tc.expected, *result)
  })
 }
}
```
