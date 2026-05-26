# CI Pipeline

## Problem

The project has local githooks via `lefthook.yml` enforcing lint, test,
generated-mock freshness, CLI doc freshness, and NOTICE file freshness on
commit and push. However, there is no CI pipeline. Contributors can bypass
local hooks, and there is no automated gate on PRs to `main` or pushes to
`main`.

## Solution

Add a GitHub Actions CI pipeline consisting of four sequential stages:
**build → lint → test → check-generated** (the last stage runs only on PRs).
This replicates the quality gates from `lefthook.yml` in CI.

## Design decisions

| Decision | Choice | Rationale |
|---|---|---|
| Platform | GitHub Actions (`ubuntu-latest`) | Project hosted at github.com/ma-tf/ogle |
| Triggers | push to any branch + PR targeting `main` | Standard CI gate; removed `branches: [main]` from push trigger in commit 72863c8 |
| Runner | `ubuntu-latest` | Single OS; cross-platform handled by GoReleaser at release |
| Go version | `1.26.x` | Matches `go.mod` |
| Stage order | build → lint → test → check-generated | Fail fast — compile errors caught before slower lint/test |
| Caching | Automatic via `setup-go@v5` and `golangci-lint-action` | Explicit `actions/cache` steps removed in commit 72863c8; `setup-go@v5` and `golangci-lint-action` handle caching internally |
| Concurrency | cancel-in-progress for non-`main` | Saves CI minutes on outdated runs |
| check-generated | PR only (`github.event_name == 'pull_request'`) | Generated-file assertions only meaningful relative to PR diff |
| check-generated steps | `make generate`, `make docs`, `make man`, `make notice`, then `git diff --exit-code` | Identical to `lefthook.yml` pre-push `verify-generated` |

## Implementation

Module: `.github/workflows/ci.yml` — single workflow file with four jobs.

## Post-merge

Move this file to `docs/plans/archive/`.
