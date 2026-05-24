---
name: plan-to-file
description: Capture the output of a grill-me session as a durable plan file in docs/plans/. Use when a design interview
is complete and the user wants to write the plan to disk for another agent to implement.
---

# plan-to-file

Captures the output of a design interview (grill-me session) as a structured, agent-readable plan file written to
`docs/plans/`.

## Workflow

1. Ask the user: *"What's a short title for this plan?"* — this becomes the slug (lowercase, hyphen-separated, no
punctuation).
2. Infer the conventional commit type from the conversation context. Valid types: `feat`, `fix`, `refactor`, `perf`,
`chore`, `docs`, `test`, `build`, `ci`, `style`, `revert`.
3. Synthesize the full plan draft using the mandatory template below.
4. Present the complete draft to the user for review. Explicitly state the inferred type and proposed filename so the
user can correct them before writing.
5. On approval, write the file to `docs/plans/{type}-{YYYY-MM-DD}-{slug}.md` using today's date.

## Mandatory Template

Every plan file must contain all of the following sections. Do not omit any.

```markdown
# {type}: {title}

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

[Brief description of the codebase/domain relevant to this plan. Current state. Why this change is needed.]

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without a specific technical reason.

| # | Decision |
|---|---|
| 1 | ... |

---

## Implementation Steps

[Ordered, concrete steps. Each step must leave the build passing before the next begins.]

---

## Out of Scope

[Explicit list of things this plan does NOT cover.]

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
```

## Optional Sections

Add these when relevant to the plan:

- `## Target Structure` — directory/file layout diagrams
- `## Import Path Reference` — old → new import path mappings
- `## API Contract` — request/response shapes for API changes

## Notes

- The slug must be lowercase, hyphen-separated, and derived from the user's short title.
- The date in the filename is always today's date at time of writing, not the date of the interview.
- The `Post-Implementation` section is mandatory in every plan — it instructs the implementing agent to archive the file
on completion.
- Do not commit the written file — leave that to the user.
