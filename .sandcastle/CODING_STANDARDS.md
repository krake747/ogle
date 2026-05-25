# Coding Standards

<!-- Customize this file with your project's coding standards.
     The reviewer agent loads it during code review via @.sandcastle/CODING_STANDARDS.md
     so these standards are enforced during review without costing tokens during implementation. -->

## Style

<!-- Example:
- Use camelCase for variables and functions
- Use PascalCase for classes and types
- Prefer named exports over default exports
-->

- Formatting enforced by `golangci-lint` with `gofumpt` and `golines` (100-char line limit).
- Import order: standard library → third-party → `github.com/ma-tf/ogle` (enforced by `gci`).
- Only comment *why* — motivation, constraint, non-obvious consequence. Never restate what code visibly does.
- Error sentinels prefixed with `Err`, error types suffixed with `Error`.
- Use `log/slog` instead of `log`; no global loggers.
- No package-level globals except in `cmd/root.go` (explicitly exempted by linter config).

## Testing

<!-- Example:
- Every public function must have at least one test
- Use descriptive test names that explain the expected behavior
-->

- Black-box tests: `package foo_test`.
- Table-driven tests with a scoped `testCase` struct per function.
- `testify/require` for preconditions, `testify/assert` for independent multi-field checks.
- `expectedError error` field; assert with `require.ErrorIs`. Never `wantErr bool`.
- `t.Parallel()` on every test function and subtest.
- Mocks generated via `go tool mockery` — never edited manually.
- Inline fixtures only; no `testdata/` directory.
- UI model tests: `Update(msg)` → `(model, cmd)`. Call returned `tea.Cmd` to get `tea.Msg`, then assert.

## Architecture

<!-- Example:
- Keep modules focused on a single responsibility
- Prefer composition over inheritance
-->

- Constructor injection: all dependencies as constructor parameters. Models never construct infrastructure internally.
- Small, focused interfaces. Prefer composition over embedding.
- Errors returned from external packages must be wrapped with context (`fmt.Errorf("…: %w", err)`).
- Avoid `init()` functions (exempted only in `cmd/root.go` for Cobra registration).
- Keep modules focused on a single responsibility.

## Commits (Conventional Commits)

- Format: `<type>(<scope>): <description>`
- Allowed types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`
- Header: 10–72 characters. Body wrap at 100 characters.
- Prefix with `RALPH:` when produced by Sandcastle automation (e.g. `RALPH: feat(core): add service filter`).
