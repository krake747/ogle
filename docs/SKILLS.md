# Agent skills

The following project-specific skills are available in `.agents/skills/`:

| Skill | Description | Trigger |
|---|---|---|
| `plan-to-file` | Captures a grill session as a durable plan file | After a design interview, when the user wants to write the plan to disk |
| `scrivener` | Audit project documentation against the codebase, flagging stale content, gaps, conflicts, and quality issues | After code changes that affect documented behaviour, or run standalone at any time |
