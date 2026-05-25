# TASK

Fix issue {{TASK_ID}}: {{ISSUE_TITLE}}

Pull in the issue using `gh issue view <ID>`. If it has a parent PRD, pull that in too.

Only work on the issue specified.

Work on branch {{BRANCH}}. Make commits and run tests.

# CONTEXT

Here are the last 10 commits:

<recent-commits>

!`git log -n 10 --format="%H%n%ad%n%B---" --date=short`

</recent-commits>

# EXPLORATION

Explore the repo and fill your context window with relevant information that will allow you to complete the task.

Pay extra attention to test files that touch the relevant parts of the code. Understand the project's testing conventions by reading `docs/TESTING.md`.

# EXECUTION

If applicable, use RGR to complete the task.

1. RED: write one test
2. GREEN: write the implementation to pass that test
3. REPEAT until done
4. REFACTOR the code

Follow these Go test conventions (see docs/TESTING.md for full details):

- `package foo_test` (black-box) for service-layer tests
- Table-driven tests with a scoped `testCase` struct per function
- `testify/require` for preconditions, `testify/assert` for independent multi-field checks
- `expectedError error` + `require.ErrorIs` (never `wantErr bool`)
- `t.Parallel()` on every test function and subtest
- Use `go tool mockery` for mocks (generated, never edited manually)

# FEEDBACK LOOPS

Before committing, run `make tools` to install linting tools, then run `make lint` and `make test` to ensure the code passes linting and tests.

# COMMIT

Make a git commit. The commit message must:

1. End the subject line with ` [RALPH]` (e.g. `feat(core): add service filter [RALPH]`)
2. Include task completed + PRD reference
3. Key decisions made
4. Files changed
5. Blockers or notes for next iteration

Keep it concise.

# THE ISSUE

If the task is not complete, leave a comment on the issue with what was done.

Do not close the issue - this will be done later.

Once complete, output <promise>COMPLETE</promise>.

# FINAL RULES

ONLY WORK ON A SINGLE TASK.
