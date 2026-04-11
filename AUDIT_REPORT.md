# Audit Report

Systematic audit of havn's current state. Each section corresponds to a
child task of the parent epic (`havn-qf6`).

---

## Documentation and Onboarding

_Audited: 2026-04-11 | Issue: havn-qf6.6_

### What Exists

| File | Size | Content |
|------|------|---------|
| `README.md` | 6 bytes | Title only (`# havn`) — no description, install instructions, usage, or examples |
| `CLAUDE.md` | 11 bytes | Single `@AGENTS.md` directive — delegates entirely to AGENTS.md |
| `AGENTS.md` | ~5 KB | Comprehensive agent instructions: bd workflow, non-interactive shell conventions, session completion checklist, memory policy |
| `LICENSE` | 1 KB | MIT license present |
| `specs/README.md` | ~400 bytes | Clean index table linking all 9 spec files with one-line descriptions |
| `specs/*.md` (×8) | Varies | Architecture principles, code standards, test standards, quality gates, CLI framework, havn overview, havn doctor, shared Dolt server |
| `Makefile` | 408 bytes | Present (build tooling) |
| `PROMPT.md` | 888 bytes | Headless agent runner prompt |

### What's Missing

| Gap | Impact |
|-----|--------|
| **No user-facing README content** | A new user or contributor visiting the repo sees only `# havn`. No project description, installation instructions, usage examples, prerequisites, or quickstart. This is the single biggest onboarding blocker. |
| **No `docs/` directory** | No user documentation beyond specs. No install guide, no usage guide, no architecture overview for humans. Specs are excellent engineering references but assume deep context. |
| **No `CONTRIBUTING.md`** | No contributor onboarding: no dev setup instructions, no PR workflow, no coding guidelines summary. Contributors must reverse-engineer the workflow from AGENTS.md and specs. |
| **No `CHANGELOG.md`** | No release history. Minor for pre-1.0, but becomes important at first public release. |
| **No Go doc comments on exported types** | `go doc` produces minimal output. The code is internal-only, so impact is limited to developer navigation. |
| **CLAUDE.md is a single redirect** | Works fine for agents that resolve `@AGENTS.md`, but any tool that reads CLAUDE.md literally sees no instructions. Not a blocker in practice (Claude Code resolves it), but fragile if other tools consume the file. |

### Spec Discoverability

**Good:** `specs/README.md` provides a well-organized index table. All 8 non-index specs are listed with descriptions. Navigation between specs uses relative links that work correctly.

**Gap:** The project root has no pointer to `specs/`. A new contributor would need to know to look in that directory. README.md should link to `specs/README.md` as the technical reference.

### Agent Onboarding (AGENTS.md)

**Strengths:**
- Clear bd workflow with examples
- Non-interactive shell conventions (prevents agent hangs)
- Session completion checklist (ensures push)
- Memory policy (bd over file-based for havn-specific context)

**Gaps:**
- No mention of how to run the project (`go build`, `make`, test commands)
- No mention of project structure or where to find code (`cmd/`, `internal/`)
- References `docs/QUICKSTART.md` which does not exist
- Relies on `bd onboard` for context, but doesn't describe what that provides

### Impact Assessment

| Audience | Onboarding Quality |
|----------|--------------------|
| **AI agents (Claude Code)** | Adequate — AGENTS.md + specs + bd provide enough context for structured work. Gaps: stale `docs/QUICKSTART.md` reference, no project structure overview. |
| **Human contributors** | Poor — README is empty, no CONTRIBUTING.md, no dev setup guide. Must discover specs/ independently and read multiple files to understand the project. |
| **End users** | Non-existent — no installation, usage, or configuration documentation. Users cannot learn what havn does or how to use it from the repo alone. |

### Recommendations (not actioned — audit only)

1. **README.md** — Add: project description, prerequisites, installation, basic usage, link to specs/
2. **CONTRIBUTING.md** — Dev setup, make targets, test commands, PR workflow, link to code-standards.md
3. **AGENTS.md** — Remove stale `docs/QUICKSTART.md` reference; add project structure overview
4. **User docs** — At minimum, a usage guide covering `havn .`, `havn stop`, configuration
5. **CHANGELOG.md** — Start tracking before first public release
