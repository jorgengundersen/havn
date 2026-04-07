---
name: create-issue-graph
description: Plan and file a graph of beads (bd) issues for havn — epics, tasks, dependencies, parent/child links. Use when decomposing an epic, turning a research report into issues, filing bug reports, or adding tasks under an existing epic. Always shows a reviewable plan first, then files via parallel `bd create` + `bd dep add` only after user confirmation.
---

# create-issue-graph

Plans a beads graph → shows the plan → files on confirmation.

## Two phases

### Phase 1 — PLAN (always)

Produce a markdown plan in chat. Do **not** file yet.

**Plan format:**

```markdown
## Planned graph

| Ref | Title | Type | Prio | Parent | Blocks | Notes |
|---|---|---|---|---|---|---|
| n1 | … | task | P2 | havn-xxx | — | — |
| n2 | … | task | P2 | havn-xxx | n1 | — |

### Draft descriptions
**n1 — <title>**
<full description following the havn template below>

**n2 — <title>**
<…>

Ready to file? (yes / revise / cancel)
```

On `revise`, update and re-confirm. On `cancel`, abort. On `yes`, go to Phase 2.

### Phase 2 — FILE (only after confirmation)

1. **Create in dependency order.** Parents before children. Parallelize creates across *different* parents; serialize creates under the *same* parent (see Rule 9).
2. **Use `bd create` with full flags** (see "Filing mechanics" below).
3. **Wire deps with `bd dep add`** after creates succeed.
4. **Verify with `bd list` and `bd ready -n 100`.** Confirm expected leaves are ready and nothing is unexpectedly blocked.
5. **Report filed IDs + `bd ready` shape** back to the user.

---

## Rules

### 1. Priority = urgency, never ordering
P0 critical · P1 high · **P2 default** · P3 low · P4 backlog. Children default to the parent's priority; bump only if a specific child has genuinely different urgency. Never use priority to serialize "do A then B then C".

### 2. `blocks` = real technical dependencies only
Add `blocks` only when B's tests literally cannot compile/run until A's public API exists, or B's behavior genuinely requires runtime state from A. Never for preference, cleanliness, or phase ordering.

### 3. Cross-epic deps go on the LEAF, not the parent
Beads propagates block state parent → children. An epic-level `blocks` dep blocks *every* child. Put cross-graph deps on the specific leaf task that actually needs the prerequisite; siblings that don't need it stay in `bd ready` and parallelize. Epic-level blocks are correct only if *every* child truly needs the prerequisite (rare).

### 4. `--parent` = containment, not blocking
An open (unblocked) parent does not block children. A blocked parent blocks all children transitively. Sibling ordering, when real, requires explicit `blocks` deps between the specific siblings — subject to Rule 2.

### 5. Task granularity: one cohesive behavior (2–5 TDD cycles)
Vertical slice through the package's public API. Large enough to be meaningful, small enough that a wrong turn is cheap.
- ✅ *"Config.Load reads global TOML (handles missing and malformed)"*
- ❌ too fine: *"Add Config.Load signature"* / *"Add default case"*
- ❌ too coarse: *"Implement config package"* (that's the epic)

### 6. No "write tests" sibling tasks
Under TDD the test drives the implementation in the same cycle. Never split "implement X" and "test X" into separate tasks. Never split an epic into "design / implement / test" phases.

### 7. Acceptance criteria describe behavior, not test mechanics
- ✅ *"`havn list --json` emits the shape in havn-overview.md §3"*
- ✅ *"Stopping a missing container returns `container.NotFoundError`"*
- ❌ *"Table-driven tests cover edge cases"* / *"Black-box tests with fake X"* / *"Integration test hits real Docker"*

### 8. Epics can only be blocked by other epics
Beads enforces this as a hard constraint (*"epics can only block other epics, not tasks"*). If you feel you need a task→epic block, you're modeling it wrong — put the block on the specific *child tasks* of the epic that need the prerequisite (Rule 3).

### 9. Serialize sibling creates under the same parent
Parallel `bd create --parent <same-id>` collides on hierarchical IDs and drops siblings. Creates under the *same* parent must be sequential. Creates across *different* parents (or with no parent) may be parallel.

### 10. Beads auto-closes completed molecules
Closing all children of an epic auto-closes the epic. Do not add an explicit epic-blocked-by-children dep to achieve this — it's native.

### 11. Use non-blocking annotations freely when they add context
- `discovered-from` — **only** when the new issue was literally discovered while working on the linked issue.
- `related` — informational link between issues in the same area.
- `caused-by` — root-cause link for bugs.
- `supersedes` — replaces another issue.
These do not affect `bd ready`.

### 12. Issue types
- `epic` — multi-task initiative container.
- `feature` — user-visible functionality.
- `task` — internal work (refactor, wire-up, plumbing). **Default for decomposition children.**
- `bug` — something broken. Use `caused-by` when root cause is known.
- `chore` — tooling, deps, CI.

## havn description template

```markdown
<one-paragraph summary including *why* the work exists>

## Scope
- <what this issue covers>

## Out of scope
- <only if there is ambiguity to resolve>

## Specs
- specs/<file>.md §<section>

## Wires up
- <what gets connected when this closes; only for domain-logic issues>

## Acceptance
- <user-visible behavior or contract guarantee>
```

## Filing mechanics

```bash
bd create "<title>" \
  --type <epic|task|feature|bug|chore> \
  --priority <0-4> \
  --parent <id>            # if applicable
  --spec-id <spec-name>    # if the issue maps directly to a spec file
  --description "<havn template>" \
  --json

# after all creates:
bd dep add <dependent-id> <prerequisite-id>             # default type: blocks
bd dep add <from> <to> --type discovered-from           # non-blocking
bd dep add <from> <to> --type related                   # non-blocking

# verify:
bd list
bd ready -n 100
```

## Anti-patterns (reject explicitly)

1. Phase ordering within an epic (Design → Implement → Test). Beads docs show this; it's wrong for TDD.
2. Separate "write tests" tasks.
3. `blocks` deps between siblings just to give `bd ready` a deterministic order.
4. Mixing priority tiers within an epic to serialize work.
5. Epic-level `blocks` when only some children need the prerequisite.
6. "Skeleton" or "scaffolding" tasks (create dir, add empty file). Not behaviors.
7. TDD mechanics in issue descriptions (the implementation agent handles TDD separately).
8. Using `-f` markdown or `--graph` JSON batch flags — their schemas are undocumented. Use parallel `bd create` + `bd dep add`.

## Hard rules

- **Never file before the user confirms the plan.**
- **Never claim work** (`bd update --claim`) — that's the implementation agent's job.
- **Never update existing issues via this skill** — use `bd update` directly.
- **Always check `bd search` / `bd list`** before filing to avoid duplicates.
