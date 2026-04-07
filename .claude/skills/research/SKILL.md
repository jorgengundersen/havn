---
name: research
description: Investigate the havn codebase and produce a structured research report. Use for gap analysis (specs vs implementation), bug investigation, architecture tracing, spec digests, or any "what's missing / how does X work / what would it take to do Y" question. Read-only; does not file issues or modify code. Output is consumable directly or by the create-issue-graph skill.
---

# research

Read-only investigator. Produces a report for the user and/or for handoff to `create-issue-graph`.

## Rules

1. **Read-only.** Do not modify code. Do not file beads issues. That's for other skills.
2. **Specs are intent; code is current behavior.** Read both; cite both.
3. **Cite everything.** Every non-obvious claim gets `specs/<file>.md §N` or `path/file.go:LINE`.
4. **Check beads memory first.** `bd memories <keyword>` may already have the answer. Also `bd search <keyword>` for related existing issues.
5. **No TDD prescriptions.** Describe *what* needs doing, not *how* to TDD it.
6. **No priority or urgency calls.** Report what exists; urgency is the user's call.
7. **Flag surprises explicitly.** Stale state, spec/code contradictions, already-done-but-not-closed work go in a dedicated "Flags" section.

## Procedure

1. **Restate scope** in your own words. Identify specs, code paths, and existing beads issues in play.
2. **Read relevant specs** fully (not skim). Note section numbers.
3. **Read relevant code**, starting from `cmd/havn/main.go` / `internal/cli/` and following imports. Read tests alongside code.
4. **Compare and classify** (for gap analysis): `MET` / `PARTIAL` / `MISSING` / `DIVERGENT`. For bugs: `CONFIRMED` / `NOT REPRODUCED` / `ADJACENT`.
5. **Write the report** using the template below.

## Report template

Omit sections that don't apply.

```markdown
# Research: <short title>

## Scope
<restated request + what was investigated>

## Summary
<2-4 sentence executive summary>

## Findings
<organized by area; each with spec:§ or file:line citation>

## Gap analysis
| Item | Status | Evidence |
|---|---|---|
| … | MET/PARTIAL/MISSING/DIVERGENT | specs/foo.md §2 vs internal/foo/bar.go:45 |

## Critical gaps
<ordered by what unblocks the most downstream work, NOT by urgency>

## Flags
<surprises, stale state, contradictions, already-done-but-not-closed work>

## Suggested next step
<"file issue graph with create-issue-graph" / "user decides X first" / "nothing to do">
```

## What NOT to include

- Implementation plans, pseudocode, or code changes.
- TDD cycle breakdowns or task granularity (that's `create-issue-graph`).
- Priority assignments or time estimates.
- New beads issues (use `create-issue-graph` as a follow-up).
