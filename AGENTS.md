# Agent Instructions

ogle — A terminal UI for observing and operating Docker Compose projects, no setup required.

## Commands

| Command | What it does |
|---|---|
| `make generate` | Regenerate mockery mocks |
| `make test` | Run tests with race detector |
| `make lint` | `go vet ./...` + `golangci-lint run ./...` |
| `make build` | Build to `./bin/ogle` |

## Agent workflow

- After writing files, run `make lint` before finishing. Run `make test` when feasible.
- If files were written, provide a commit message inline when finishing the task (but do not commit unless asked).

## Progressive disclosure

For deeper context the agent can pull on demand:

| File | Contents |
|---|---|
| [docs/TOOLCHAIN.md](./docs/TOOLCHAIN.md) | Go version, mockery, golangci-lint |
| [docs/CONVENTIONS.md](./docs/CONVENTIONS.md) | Coding conventions |
| [docs/plans/WORKFLOW.md](./docs/plans/WORKFLOW.md) | Plan workflow |
| [docs/SKILLS.md](./docs/SKILLS.md) | Available agent skills |
| [docs/CONTEXT.md](./docs/CONTEXT.md) | Domain terminology — Service, Project, Dashboard, etc. |
| [docs/arch.md](./docs/arch.md) | Package structure, dependency graph |
| [docs/flows.md](./docs/flows.md) | State machines, screen transitions |
| [docs/charm-ecosystem.md](./docs/charm-ecosystem.md) | Charm library compatibility notes |
| [docs/TESTING.md](./docs/TESTING.md) | Unit test and UI model test conventions |
