---
name: research
description: Investigate the havn codebase and produce a structured research report. Use for gap analysis (specs vs implementation), bug investigation, architecture tracing, spec digests, or any "what's missing / how does X work / what would it take to do Y" question. Read-only; does not file issues or modify code. Output is consumable directly or by the create-issue-graph skill.
---

# research

Read-only investigator. Report → user or handoff to `create-issue-graph`.

## Rules

1. Read-only. No code edits. No bd writes.
2. Specs = intent. Code = behavior. Read both, cite both.
3. Cite everything: `specs/<f>.md §N` or `path/file.go:LINE`.
4. Check `bd memories <kw>` + `bd search <kw>` first.
5. No TDD prescriptions. *What*, not *how*.
6. No priority/urgency. User's call.
7. Flag surprises (stale state, spec/code contradictions, done-but-open) in Flags section.

## Procedure

1. Restate scope. List specs, code paths, bd issues in play.
2. Read specs fully. Note section numbers.
3. Read code from `cmd/havn/main.go` / `internal/cli/` outward. Tests alongside code.
4. Classify. Gap analysis: `MET` / `PARTIAL` / `MISSING` / `DIVERGENT`. Bugs: `CONFIRMED` / `NOT REPRODUCED` / `ADJACENT`.
5. Write report.

## Report template

Omit sections that don't apply.

```markdown
# Research: <title>

## Scope
<request + what was investigated>

## Summary
<2-4 sentences>

## Findings
<by area; each cited>

## Gap analysis
| Item | Status | Evidence |
|---|---|---|
| … | MET/PARTIAL/MISSING/DIVERGENT | specs/foo.md §2 vs internal/foo/bar.go:45 |

## Critical gaps
<ordered by downstream unblock value, NOT urgency>

## Flags
<surprises, stale state, contradictions, done-but-open>

## Suggested next step
<"create-issue-graph" / "user decides X" / "nothing to do">
```

## Do NOT include

- Implementation plans, pseudocode, code changes.
- TDD breakdowns, task granularity (→ `create-issue-graph`).
- Priorities, time estimates.
- New bd issues (→ `create-issue-graph`).
