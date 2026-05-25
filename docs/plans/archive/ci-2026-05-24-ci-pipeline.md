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
| Triggers | push to `main` + PR targeting `main` | Standard CI gate |
| Runner | `ubuntu-latest` | Single OS; cross-platform handled by GoReleaser at release |
| Go version | `1.26.x` | Matches `go.mod` |
| Stage order | build → lint → test → check-generated | Fail fast — compile errors caught before slower lint/test |
| Caching | GOMODCACHE + GOCACHE via `actions/cache` | Keyed on hash of `go.sum` + `go.mod` with restore-key fallback |
| Concurrency | cancel-in-progress for non-`main` | Saves CI minutes on outdated runs |
| check-generated | PR only (`github.event_name == 'pull_request'`) | Generated-file assertions only meaningful relative to PR diff |
| check-generated steps | `make generate`, `make docs`, `make man`, `make notice`, then `git diff --exit-code` | Identical to `lefthook.yml` pre-push `verify-generated` |

## Implementation

Module: `.github/workflows/ci.yml` — single workflow file with four jobs.

## Post-merge

Move this file to `docs/plans/archive/`.
