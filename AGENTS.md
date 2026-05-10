# Agent Instructions

## Plan archiving

When you finish implementing a plan from `docs/plans/` and the build passes, move the plan file to `docs/plans/archive/`. Do not commit the move — leave that to the user.

## Agent skills

The following project-specific skills are available in `docs/agents/skills/`:

| Skill | Description | Trigger |
|---|---|---|
| `plan-to-file` | Captures a grill session as a durable plan file | After a design interview, when the user wants to write the plan to disk |
