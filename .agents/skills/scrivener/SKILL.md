---
name: scrivener
description: Audit project documentation against the codebase, flagging stale content, gaps, conflicts, and quality issues. Codebase-agnostic — works for any language or project structure. Use after code changes that affect documented behaviour, or run standalone at any time.
---

# Scrivener

Audit project documentation against the codebase. Reads no source files
directly — the agent supplies the truth from recent code work or an
exploration phase. Produces a structured report of findings. Does not edit
docs.

Loaded automatically when code changes affect public API, architecture,
user-facing behaviour, or domain terminology. Also loadable explicitly
for standalone audits.

## Workflow

### Phase 0 — Explore (if needed)

If loaded as a standalone audit (no recent code changes in agent context),
first establish current codebase truth:

1. Launch an explore subagent to walk the codebase. Instruct it to return:
   - High-level packages/modules and their responsibilities
   - All exported public API (commands, flags, types, functions, interfaces)
   - Current state of any documented features
   - Any recent changes that may have outrun documentation
2. Keep the result in context as the codebase truth for the checklist phase.

If loaded post-implementation, skip exploration. The agent's existing
knowledge of what it just changed is the truth.

### Phase 1 — Discover docs

Enumerate all documentation targets:

List `docs/` at the repo root with the Read tool. Classify by path:

- Each subdirectory under `docs/` is its own document category
- Each file directly under `docs/` is a user-facing doc
- Root `*.md` files (`README.md`, `CONTRIBUTING.md`) are user-facing docs

### Phase 2 — Audit

Read each discovered doc file. For each, evaluate against the codebase truth
across all 7 categories. Collect findings.

### Phase 3 — Report

Write an inline structured report.

```markdown
## Scrivener Report

### STALE
- README.md: lists a CLI flag that was removed
- docs/config.md: default value differs from codebase

### GAP
- README.md: new command not listed in usage section
- docs/architecture.md: new module not documented

### CONFLICT
- docs/setup.md and docs/install.md: describe different versions
- docs/adr/0001-decision.md: implementation contradicts the decision

### AMBIGUOUS
- docs/guide.md: "token" used for both auth and API keys
- docs/reference.md: "client" means HTTP client in one place, SDK in another

### STRUCTURAL
- docs/advanced.md: install instructions duplicated from docs/quickstart.md
- README.md: contains design rationale better suited to an ADR

### POLISH
- docs/guide.md: line 42 — "recieve" → "receive"
- docs/reference.md: line 89 — broken link to #configuration

### UNWIELDY
- docs/reference.md: 500+ lines covering API, config, and troubleshooting
```

---

## Category definitions

### STALE

A doc describes behaviour, API surface, architecture, or convention that no
longer matches the codebase.

**Canonical form:** "Doc says X, code does Y."

**Examples:** CLI flag was removed but doc references it; package was renamed
but doc uses old name; default value changed in code but doc lists old
default; program output shown in doc differs from actual output; state in a
flow diagram no longer exists.

**How to detect:** For each doc, read every meaningful claim (flags,
commands, defaults, package names, file names, state names, keybindings,
protocols, schemas) and compare against codebase truth. Different truth =
STALE.

**Not STALE:** Content that is merely unclear or unhelpful — that is POLISH
or AMBIGUOUS.

---

### GAP

A feature, concept, API element, or behaviour exists in the codebase but has
no documentation coverage.

**Canonical form:** "Code has X but no doc mentions it."

**Examples:** new CLI flag added but not listed in README usage; new package
introduced but not described in architecture docs; new state added to a
flow but flow docs omit it; new config option exists but isn't documented.

**How to detect:** Walk codebase truth elements (flags, packages, states,
types, config keys) and check whether each appears in at least one relevant
doc. Orphan element = GAP.

**Not GAP:** Missing docs for internal implementation details with no
user-facing or architectural significance — those are intentionally
undocumented surfaces.

---

### CONFLICT

Two or more documentation sources contradict each other, or a doc contradicts
an architectural decision.

**Canonical form:** "Doc A says X, doc B says not-X."

**Examples:** README says `--port` defaults to 8080, docs/config.md says
9090; an ADR says parser must have zero UI dependencies, architecture doc
describes it importing a UI package; glossary defines a term one way,
another doc uses it differently.

**How to detect:** Cross-reference statements across docs that describe the
same thing. If two sources disagree on a factual statement, it is CONFLICT.

**Not CONFLICT:** A doc describing something differently from how the agent
would prefer it — the standard is internal consistency, not subjective
quality.

---

### AMBIGUOUS

A term is used inconsistently across documents, or language in a document
could reasonably be interpreted in multiple ways.

**Canonical form:** "Term X means Y in one place and Z in another." / "It is
unclear whether X includes Y."

**Examples:** "service" refers to a microservice in one doc and an OS
daemon in another; "filter" means search filter in one place and access
control filter in another; "client" could mean HTTP client or library user.

**How to detect:** Track every domain term used across docs. If the same term
appears in contexts implying different meanings, or a sentence parses in more
than one way, it is AMBIGUOUS. If a glossary exists, use it as the canonical
vocabulary source — flag terms used in docs but not defined there.

**Not AMBIGUOUS:** Incorrect content (that is STALE). Vague but harmless
language ("various options") — only flag when ambiguity could lead to
misunderstanding.

---

### STRUCTURAL

Content lives in the wrong document, or the same content is maintained in
multiple places creating drift risk.

**Canonical form:** "X belongs in Y, not Z." / "X is duplicated in Y and Z
but should be authored once."

**Examples:** Implementation rationale embedded in a user-facing doc; config
reference duplicated in README and a dedicated config doc with different
values; installation steps in both README and a contributing guide.

**How to detect:** Scan for cross-doc duplication — the same information
appearing in two or more files. Scan for content mismatched to doc purpose —
user-facing doc containing design rationale, decision doc containing
implementation details, glossary containing procedural instructions.

**Not STRUCTURAL:** Closely related content in neighbouring files that
naturally overlap — only flag when boundary is clearly wrong or duplication
provably creates drift.

---

### POLISH

Surface-level quality issues: spelling mistakes, grammar errors, broken
formatting, markdown lint failures, broken links, inconsistent heading
styles.

**Canonical form:** "File:line — specific issue."

**Examples:** misspelled word; broken internal anchor link; mixed heading
levels; fenced code block without language specifier; trailing whitespace;
inconsistent list style.

**How to detect:** Probe for `markdownlint` and `aspell`/`hunspell` on the
system. If available, run them against doc files. If not, review manually:
check links resolve within the repo, check spelling, check heading structure
is consistent.

**Not POLISH:** Content errors — those are STALE, GAP, or CONFLICT. POLISH is
about presentation quality, not correctness.

---

### UNWIELDY

A document has grown too large, too dense, or too mixed in purpose to be
useful.

**Canonical form:** "X.md covers Y distinct concerns over N lines."

**Examples:** A glossary that exceeds 500 lines and mixes term definitions
with implementation notes; README that contains all documentation (install,
usage, API, architecture, contributing) instead of linking to sub-docs; a
single decision doc covering multiple unrelated topics; a prose-heavy doc
that would benefit from a diagram.

**How to detect:** Any doc over ~500 lines of prose should be examined for
splitting. Any doc that covers more than one distinct audience (user +
contributor + architect) should be examined for per-audience decomposition.
Any doc where a reader would struggle to find a specific piece of information
is UNWIELDY.

**Not UNWIELDY:** A naturally long but well-structured doc where each section
serves one audience. Only flag when length actively harms findability or
maintainability.

---

## General rules

1. **One finding per line.** Each line must contain a file path and a
   specific discrepancy. Do not combine findings into a single line.
2. **Order findings by severity** within each category. STALE about a public
   API or user-facing feature is higher priority than STALE about an internal
   implementation detail.
3. **Do not edit any file.** The scrivener's output is the report — nothing
   more. The user decides what to do with the findings.
4. **If a category has zero findings, omit it from the report.** Do not
   include empty headers.
5. **If every category has zero findings, report:** "No discrepancies found.
   All documentation is accurate and current."
6. **Verify first, report second.** Do not begin writing the report until all
   docs have been read and evaluated against codebase truth. A partial report
   that later gets corrected is confusing.
