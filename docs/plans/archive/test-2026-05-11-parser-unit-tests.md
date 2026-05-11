# test: parser unit tests

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

ogle is a Go + Bubble Tea TUI for observing Docker Compose projects. The parser
service (`internal/services/parser/service.go`) reads and validates Compose Files
into `domain.Project` values. It exposes two methods: `Validate(path string) error`
and `Parse(path string) (*domain.Project, error)`, with two sentinel errors:
`ErrReadComposeFile` and `ErrParseComposeFile`.

The codebase currently has zero test coverage. This plan adds a focused,
data-driven test file for the parser service only.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without
a specific technical reason.

| # | Decision |
|---|---|
| 1 | Separate test structs, tables, and functions for `Validate` and `Parse` |
| 2 | Full struct equality on the returned `*domain.Project` (`require.Equal`) |
| 3 | Fixture YAML is inline as a `yaml string` field on the test struct — no `testdata/` files |
| 4 | Expected sentinel error stored as `expectedError error` on the struct; asserted with `require.ErrorIs` |
| 5 | Name-fallback case uses a named subdirectory (`dir string` field) so `expected.Name` is known at table definition time |
| 6 | Both structs are named `testCase`, scoped to their respective test functions |
| 7 | Fields are grouped and commented as `// arrange` and `// assert` |
| 8 | `expected` terminology used throughout (not `want`) |
| 9 | `parser.Service.Parse` is modified to sort `Services` by `Name` before returning, making output deterministic for all callers |
| 10 | `expected.File` is assigned in the test loop (not in the table) since it is the computed absolute path |

---

## Implementation Steps

Each step must leave the build passing before the next begins.

1. **Sort services in `parser.Service.Parse`** — after building the `services`
   slice, add:
   ```go
   slices.SortFunc(services, func(a, b domain.ServiceDef) int {
       return cmp.Compare(a.Name, b.Name)
   })
   ```
   Import `cmp` and `slices` (stdlib, Go 1.21+). Verify `go build ./...` passes.

2. **Ensure `testify` is available** — check `go.mod` for
   `github.com/stretchr/testify`. If absent, run `go get github.com/stretchr/testify`.

3. **Create `internal/services/parser/service_test.go`** — package
   `parser_test`. Implement `TestValidate` and `TestParse` as described below.
   Run `go test ./internal/services/parser/...` and confirm all cases pass.

---

## Target Structure

```
internal/
  services/
    parser/
      service.go        ← modified: sorts Services slice before return
      service_test.go   ← new
```

---

## Test Design

### `TestValidate`

```go
type testCase struct {
    // arrange
    name string
    yaml string

    // assert
    expectedError error
}
```

| name | yaml | expectedError |
|---|---|---|
| `"valid file"` | minimal valid YAML | `nil` |
| `"file does not exist"` | _(no file written)_ | `ErrReadComposeFile` |
| `"invalid YAML"` | `"{"` | `ErrParseComposeFile` |

Loop: write `tc.yaml` to a file in `t.TempDir()` (skip write for the
missing-file case). Call `svc.Validate(path)`. Assert
`require.ErrorIs(t, err, tc.expectedError)`.

---

### `TestParse`

```go
type testCase struct {
    // arrange
    name string
    yaml string
    dir  string

    // assert
    expected      domain.Project
    expectedError error
}
```

| name | dir | expectedError |
|---|---|---|
| `"valid file with name field"` | `""` | `nil` |
| `"name falls back to directory name"` | `"myproject"` | `nil` |
| `"multiple services"` | `""` | `nil` |
| `"file does not exist"` | `""` | `ErrReadComposeFile` |
| `"invalid YAML"` | `""` | `ErrParseComposeFile` |

Loop:
1. If `tc.dir` non-empty: create `filepath.Join(t.TempDir(), tc.dir)` as the
   write directory. Otherwise use `t.TempDir()` directly.
2. Write `tc.yaml` to `compose.yaml` in that directory (skip for missing-file case).
3. Assign `tc.expected.File = path` (the computed absolute path).
4. Call `svc.Parse(path)`.
5. If `tc.expectedError != nil`: assert `require.ErrorIs(t, err, tc.expectedError)`
   and `require.Nil(t, got)`.
6. Otherwise: assert `require.NoError(t, err)` and
   `require.Equal(t, tc.expected, *got)`.

---

## Out of Scope

- Tests for `scanner`, `watcher`, or any UI layer
- Mock generation for the `Parser` interface
- Integration tests
- `testdata/` fixture files

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
