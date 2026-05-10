# feat: plan-to-file skill

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

This project uses grill-me sessions to design features through a structured interview. The output of those sessions currently lives only in conversation history — there is no way to hand the resulting plan to another agent for implementation.

This plan introduces a `plan-to-file` skill and the surrounding conventions (directory structure, naming, archiving) that turn a grill session into a durable, agent-readable plan file.

The existing `AGENTS-services-refactor.md` in the project root is an orphaned example of exactly this kind of plan — it was written manually. This work formalises that pattern.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without a specific technical reason.

| # | Decision |
|---|---|
| 1 | Skill is **project-specific** — lives in `docs/agents/skills/plan-to-file/SKILL.md` |
| 2 | Pending plans are stored in **`docs/plans/`** |
| 3 | Completed plans are moved to **`docs/plans/archive/`** |
| 4 | Filename format: **`{type}-{YYYY-MM-DD}-{slug}.md`** — e.g. `refactor-2026-05-09-services-layer.md` |
| 5 | Type prefix uses **conventional commit types as-is** (`feat`, `fix`, `refactor`, `perf`, `chore`, `docs`, `test`, `build`, `ci`, `style`, `revert`) |
| 6 | Agent **infers type from conversation context**; user corrects it during the review step |
| 7 | Skill flow: ask user for short title → synthesize full draft → **present for review** → write on approval |
| 8 | Mandatory plan sections: **Status, Context, Decision Log, Implementation Steps, Out of Scope, Post-Implementation** |
| 9 | `Post-Implementation` section instructs the implementing agent to **move the file to `docs/plans/archive/`** — no commit instruction |
| 10 | Archive rule lives **both in every plan file and in `AGENTS.md`** (belt and braces) |
| 11 | **`AGENTS.md`** is created at the project root; contains the archive rule and an `## Agent skills` block registering `plan-to-file` |
| 12 | `grill-me` and `grill-with-docs` global skills are **not modified** — skills remain decoupled |

---

## Target Structure

```
docs/
  agents/
    skills/
      plan-to-file/
        SKILL.md
  plans/
    .gitkeep
    archive/
      .gitkeep
AGENTS.md
```

`docs/plans/` and `docs/plans/archive/` may already exist with `.gitkeep` files — create them if absent, skip if present.

---

## Implementation Steps

Work through these in order.

### Step 1 — Create `docs/agents/skills/plan-to-file/SKILL.md`

The skill frontmatter:

```markdown
---
name: plan-to-file
description: Capture the output of a grill-me session as a durable plan file in docs/plans/. Use when a design interview is complete and the user wants to write the plan to disk for another agent to implement.
---
```

The skill body must describe this exact workflow:

1. Ask the user: *"What's a short title for this plan?"* (used to generate the slug)
2. Infer the conventional commit type from the conversation context (feat, fix, refactor, etc.)
3. Synthesize the full plan using the mandatory template (see below)
4. Present the complete draft to the user for review and correction (type, slug, all sections)
5. On approval, write the file to `docs/plans/{type}-{YYYY-MM-DD}-{slug}.md` using today's date

**Mandatory template** the skill must always produce:

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

Optional sections (add when relevant): `Target Structure`, `Import Path Reference`, `API Contract`, etc.

### Step 2 — Create `docs/plans/.gitkeep` and `docs/plans/archive/.gitkeep`

Create both directories and their `.gitkeep` files if they do not already exist.

### Step 3 — Create `AGENTS.md` at the project root

Content:

```markdown
# Agent Instructions

## Plan archiving

When you finish implementing a plan from `docs/plans/` and the build passes, move the plan file to `docs/plans/archive/`. Do not commit the move — leave that to the user.

## Agent skills

The following project-specific skills are available in `docs/agents/skills/`:

| Skill | Description | Trigger |
|---|---|---|
| `plan-to-file` | Captures a grill session as a durable plan file | After a design interview, when the user wants to write the plan to disk |
```

### Step 4 — Move the orphaned plan

`AGENTS-services-refactor.md` in the project root is an existing plan that predates this convention. Move it to `docs/plans/refactor-2026-05-09-services-layer.md` (use the file's apparent creation date or today's date if unknown).

Do not modify its content.

### Step 5 — Verify

- `docs/agents/skills/plan-to-file/SKILL.md` exists and contains the full workflow and template
- `docs/plans/` and `docs/plans/archive/` exist
- `AGENTS.md` exists at project root with archive rule and skills table
- `AGENTS-services-refactor.md` no longer exists in the project root
- `docs/plans/refactor-2026-05-09-services-layer.md` exists with identical content to the original

---

## Out of Scope

- Making this skill global (`~/.agents/skills/`) — evaluate after first use
- Modifying `grill-me` or `grill-with-docs` — skills stay decoupled
- Writing tests for the skill itself
- Updating `docs/CONTEXT.md` — it is domain documentation, not agent instructions
- Moving anything in `docs/deprecated/` — unrelated to this plan

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
