# TASK

Audit the project documentation against the codebase on branch `{{BRANCH}}`.
If discrepancies are found, create or update a GitHub issue to track them.

# METHODOLOGY

Use the scrivener skill's audit methodology defined in `.agents/skills/scrivener/SKILL.md`. Follow its workflow exactly:

1. **Establish codebase truth**: Read the code on this branch to understand what the code actually does. Focus on changed areas.
2. **Discover docs**: Enumerate all files under `docs/` and root `*.md` files.
3. **Audit each doc**: Evaluate against codebase truth across all 7 categories — STALE, GAP, CONFLICT, AMBIGUOUS, STRUCTURAL, POLISH, UNWIELDY.

# CONTEXT

## Branch diff

!`git diff {{SOURCE_BRANCH}}...{{BRANCH}}`

## Commits on this branch

!`git log {{SOURCE_BRANCH}}..{{BRANCH}} --oneline`

# REPORT

Produce a structured report in this format:

```markdown
## Scrivener Report

### STALE
- docs/example.md: describes flag --old-flag that no longer exists

### GAP
- README.md: new --filter flag not documented
```

Order findings by severity within each category. Omit categories with zero findings.

If the report is empty (no discrepancies found), output `<promise>NO_ISSUES</promise>` and stop.

# ISSUE MANAGEMENT

If the report has findings:

1. Query for an existing open issue covering documentation quality:

   `gh issue list --label documentation --label Sandcastle --state open --json number,title`

2. **If an open issue exists**: append the full report as a comment:

   `gh issue comment <NUMBER> --body "$REPORT"`

3. **If no open issue exists**: create a new issue with the report as the body:

   `gh issue create --title "Documentation audit: discrepancies found" --label needs-triage,documentation,Sandcastle,
   scrivener-discovered --body "$REPORT"`

Once complete, output `<promise>COMPLETE</promise>`.
