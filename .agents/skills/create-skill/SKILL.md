---
name: create-skill
description: Author a new Agent Skill conforming to the agentskills.io specification. Use when the user wants to create, scaffold, or draft a new skill for Claude Code, VS Code Copilot, or any compatible agent. Enforces extreme conciseness (grammar sacrificed), progressive disclosure, and spec-valid frontmatter. Shows a reviewable draft before writing files.
---

# create-skill

Draft → show → write on confirm. Spec: [agentskills.io/specification](https://agentskills.io/specification).

## Two phases

### Phase 1 — DRAFT (always)

Markdown draft in chat. Do NOT write files.

```markdown
## Planned skill

**Path:** .claude/skills/<name>/SKILL.md
**Name:** <name>
**Description:** <1-sentence — what + when>
**Bundled files:** <scripts/ references/ assets/ — or none>

### SKILL.md draft

---
name: <name>
description: <what + when, keyword-rich>
---

# <name>

<body — see rules>

### Rationale
<why this scope, why these sections, what was deliberately left out>

Ready to write? (yes / revise / cancel)
```

`revise` → update + reconfirm. `cancel` → abort. `yes` → Phase 2.

### Phase 2 — WRITE (only after confirm)

1. `.claude/skills/<name>/SKILL.md` via Write tool. Dir auto-created.
2. Bundled files (scripts/, references/, assets/) if any.
3. Report path + how to invoke (`/<name>` in Claude Code, or via description match).

---

## Rules

### 1. Conciseness: sacrifice grammar
Bullet fragments, abbreviations, command-form sentences. No filler words. No "in order to", "you should", "please note". If it can be said in 3 words, don't use 8.
- ✅ *"Cite everything. `specs/<f>.md §N` or `path/file.go:LINE`."*
- ❌ *"You should always make sure to cite every non-obvious claim using either a spec reference or a file-line reference."*

### 2. Frontmatter = spec-valid
Required: `name`, `description`. Optional: `license`, `compatibility`, `metadata`, `allowed-tools`.
- `name`: 1–64 chars. `[a-z0-9-]` only. No leading/trailing/consecutive hyphens. **Must match parent dir name.**
- `description`: 1–1024 chars. Says *what* + *when*. Keyword-rich for activation matching.
- `compatibility`: only if real env requirement (git/docker/Python version).

### 3. Description = activation trigger
Agent matches user prompt against description to decide activation. Include:
- Verbs the user might say ("create", "file", "review", "debug", "plan", "draft").
- Domain nouns ("PDF", "beads issue", "slog logger").
- Use-when clause ("Use when user asks to X, Y, or Z").

- ✅ *"Extract PDF text/tables, fill forms, merge files. Use when user mentions PDFs, forms, or document extraction."*
- ❌ *"Helps with PDFs."*

### 4. Progressive disclosure
- Metadata (~100 tok): always loaded. Keep `description` tight.
- Body (<5000 tok recommended, <500 lines hard cap): loaded on activation.
- Resources: on-demand only. Move detail → `references/<topic>.md`.

If body >500 lines, split. If body >200 lines, consider splitting.

### 5. Scope = one cohesive capability
One skill = one thing. Multi-purpose skills fail to activate reliably.
- ✅ `pdf-processing`, `create-issue-graph`, `roll-dice`
- ❌ `dev-helpers`, `general-utils`, `project-tools`

### 6. Body structure (recommended, not required)
Typical sections in order:
- Opening line: what it does, one sentence.
- Phases / procedure (numbered).
- Rules (numbered, titled).
- Templates (fenced code).
- Anti-patterns (numbered, rejection-framed).
- Hard rules (checklist).

Omit what doesn't apply.

### 7. Confirmation gates for destructive/external work
Skills that file issues, push commits, send messages, or create files outside the skill dir MUST show a plan first and wait for `yes`. Pattern: two-phase (PLAN → ACT).

### 8. No TDD prescriptions, no time estimates, no priority calls
Skills describe *what* and *how*. Urgency, scheduling, and test mechanics belong elsewhere.

### 9. Bundled resources: minimal, focused
- `scripts/` — self-contained, documented deps, good error messages.
- `references/` — one topic per file. Agent loads on demand.
- `assets/` — templates, schemas, images.
- Relative paths from skill root. One level deep, no nested chains.

### 10. Directory structure
```
.claude/skills/<name>/
├── SKILL.md          # required
├── scripts/          # optional
├── references/       # optional
└── assets/           # optional
```

Name of dir == `name` frontmatter field. Enforced.

## SKILL.md template

```markdown
---
name: <name>
description: <what + when, keyword-rich, ≤1024 chars>
---

# <name>

<one-line elevator pitch>

## <Phases | Procedure | Rules>

<numbered sections, concise>

## Template(s)

<fenced code block with fill-in placeholders>

## Anti-patterns

1. <bad pattern> — <why>
2. …

## Hard rules

- <must / must never>
- …
```

## Anti-patterns (reject)

1. **Vague description.** "Helps with X" — agent can't match. Must say what + when.
2. **Multi-purpose skill.** Activation matching fails; split into multiple skills.
3. **Prose walls.** Long paragraphs waste context. Bullets, fragments, examples.
4. **Redundant sections.** "Introduction", "Overview", "Conclusion" — delete.
5. **Example-only skills.** Examples support rules; rules must exist first.
6. **Missing use-when clause.** Without it, activation is a coin flip.
7. **Hyphen errors in name.** Leading/trailing/consecutive hyphens = spec-invalid.
8. **Dir name mismatch.** `name: foo-bar` in `baz/SKILL.md` = spec-invalid.
9. **Bundled file chains.** `SKILL.md → references/a.md → references/b.md → …`. Flatten.
10. **Writing files before confirmation.** Always draft → confirm → write.

## Hard rules

- Never write files before user confirms draft.
- Never skip frontmatter validation (`name`, `description` required, `name` regex, dir match).
- Never exceed 1024 chars in `description` or 64 in `name`.
- Always use `.claude/skills/<name>/` as the install path for this project.
- Always keep body <500 lines; split to `references/` when longer.
- Sacrifice grammar for conciseness. Every line pays rent.

## Validation checklist (run before Phase 2)

- [ ] `name` is 1–64 chars, `[a-z0-9-]`, no edge/consecutive hyphens.
- [ ] `name` matches parent dir.
- [ ] `description` is 1–1024 chars, includes what + when.
- [ ] Body <500 lines.
- [ ] One cohesive capability.
- [ ] Destructive/external actions gated behind confirmation.
- [ ] No filler prose. No grammar tax.

## References

- Spec: https://agentskills.io/specification
- Quickstart: https://agentskills.io/skill-creation/quickstart
- Existing havn skills: `.claude/skills/research/`, `.claude/skills/create-issue-graph/`
