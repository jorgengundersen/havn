# Architecture Principles

Guiding star for havn's design and implementation. Other specs define _what_
havn does; this defines _how_ it should be built.

havn is developed by AI agents, humans on-the-loop. Agents can't infer
intent -- implicit = gap where assumptions compound. Some principles here
are conventional engineering applied with more discipline because agents
amplify the cost of violations. Others -- like treating the codebase itself
as the primary channel for guiding agent behavior -- are specific to this
way of working.

## 1. Primitives and Composition

Build the system from small, composable units -- like lego bricks. Each unit
has a clear contract (inputs→outputs), a single responsibility, and is
independently understandable and testable.

**Primitives** are the smallest units:

- **Pure functions**: no side effects, deterministic. Config merging, name
  derivation, mount resolution, validation -- all pure functions.
- **Dependency wrappers**: thin adapters around external systems. Translate
  external interface→internal contract. See
  [Section 4](#4-dependency-isolation).

**Layered composition.** Primitives compose into higher-level units, which
compose further, as many layers as needed. Each composed unit is itself a
building block for the layer above. A unit that creates a database container
is built from primitives -- and is itself a building block for "start
development environment."

**The rule for every unit at every layer: one nameable job.** If you need
"and" to describe what a unit does, it's two units. "Create database
container" is one job. "Resolve config and start container" is two jobs
that belong in separate units, composed by something above them.

**The CLI layer** is the top entry point -- it wires user input to the
composed logic below. It is not special; it follows the same composition
rules. CLI commands should be thin: parse input, call composed units,
surface results and errors.

**CLI surface is a contract.** Follow established CLI conventions -- flags,
subcommands, and patterns should be intuitive to anyone familiar with
modern command-line tools. Don't invent novel syntax when a convention
exists. Once a flag or subcommand is published, treat it as stable.
Changing it breaks scripts, muscle memory, and trust. Changes to the CLI
surface must be intentional and deliberate, not incidental side effects of
internal refactoring.

_Command structure, flag naming, and UX conventions live in a dedicated
CLI spec._

**When decomposition isn't obvious.** The "and" test catches clear cases,
but some units have natural sub-steps where the boundary is ambiguous. When
unsure, ask:

1. **Can each piece be independently tested with a meaningful contract?**
   If splitting produces a unit with no useful contract of its own, it's
   not a real primitive -- keep it together.
2. **Does separating them give the caller useful control over ordering or
   error handling?** If the caller would always call them in the same
   sequence with no decisions in between, separation adds indirection
   without value.
3. **Would combining them force a single unit to know about concerns at
   different abstraction levels?** If one part is pure data transformation
   and the other talks to Docker, they belong in separate units.

Yes to any → separate. No to all → keeping them together is pragmatism,
not a violation.

_Example:_ config has a natural chain -- load raw file, resolve defaults,
validate. Each is independently testable with a clear contract (partial
config→complete config, complete config→valid config or error). Separating
them means downstream code always receives complete, validated config
instead of wondering whether missing fields are intentional. That's three
primitives composed by a higher-level unit, not one unit doing three jobs.

**Naming:** domain-first. `DeriveContainerName`, not
`SanitizeAndJoinStrings`. Organize by domain concern (e.g., `config/`,
`container/`), not by role (`utils/`, `helpers/`). Poor naming→duplication.
Agent that can't find `ResolveProjectPath` writes a new one.

_Package structure and specific organization live in implementation specs._

## 2. Testing

Agent's primary self-verification. Weak tests = agents can't self-verify =
humans verify everything = no leverage.

**Black-box by default.** Test contracts, not internals. Litmus: can you
refactor internals without breaking tests? If no, tests are wrong.

**Boundaries:**

- **Unit** (fast, cheap, the majority): primitives and composed units tested
  with test doubles. Known inputs, expected outputs. Every unit is tested at
  its own level -- test the unit's contract, not the primitives inside it
  (those have their own tests).
- **Integration**: units compose correctly, wrappers interact with real
  systems. Verifies that boundaries work in practice -- the places where
  test doubles can diverge from reality.
- **System** (few, focused): e2e for critical paths only. Expensive -- reserve
  for the workflows that matter most.

**Exception:** complex algorithms with many internal edge cases -- internal
tests acceptable. Document why.

Coverage is signal, not target. Chase contracts not numbers.

_Testing standards/patterns/tooling in a dedicated spec._

## 3. Build vs Depend

AI-led development lowers the cost of building. Don't depend by default --
evaluate whether ownership cost is less than dependency cost.

- **Depend** when correctness is hard-won: container runtimes, crypto,
  database drivers, SSH. These have years of edge cases and security
  implications you don't want to re-discover.
- **Own** when the wrapper surface is small and there's no meaningful upgrade
  cycle. If the wrapper _is_ the implementation, own it.

**Decision procedure -- work through in order:**

1. **Does Go's standard library cover it?** Use it. The stdlib is
   extensive and well-maintained. This is not an external dependency.
2. **Is correctness hard-won or security-critical?** Crypto, protocol
   implementations, container runtimes -- depend. The cost of subtle bugs
   far exceeds the cost of the dependency.
3. **How much of the library would you actually use?** If you need 10% of
   a package's surface, you're paying for 100% of its supply chain risk
   and upgrade burden. A tailored implementation of the 10% you need is
   often simpler and more maintainable.
4. **What's the transitive dependency cost?** A library that pulls in a
   tree of sub-dependencies multiplies supply chain risk. Fewer external
   dependencies = smaller attack surface.
5. **Can an agent implement it in a sitting?** Agentic coding makes
   implementation cheap. If the functionality is straightforward enough
   to build, test, and verify in one pass, the ownership cost is low --
   and you get code tailored to havn's exact needs.

**When in doubt, lean toward owning.** The JS/npm ecosystem normalized
depending on packages for trivial functionality. havn does the opposite:
external dependencies are a conscious, justified choice -- not the default.
Every dependency is code you don't control that can break, change, or be
compromised.

This decision happens before dependency isolation ([Section 4](#4-dependency-isolation)).
First decide whether to depend at all. If yes, Section 4 governs how that
dependency enters the codebase.

## 4. Dependency Isolation

External dependencies enter through wrappers that translate the external
world into havn's own language. Domain code never sees dependency-specific
types, naming, or behavior -- it works exclusively against havn's contracts.

**The hard rule:** dependency types do not cross into domain code. Docker's
`ContainerJSON` → havn's `ContainerInfo`. Docker's error types → havn's
error types. If you can grep domain code and find imports from a dependency's
package, isolation is broken.

**Why this is non-negotiable for havn:** Docker is the runtime today but may
not be tomorrow. If Docker types are woven through domain code, switching to
Podman means rewriting the domain -- not just a wrapper. Isolation means:

- Domain depends on havn's contracts, not Docker's API
- Replacing a runtime = rewriting one wrapper, not the codebase
- Testing domain code doesn't require the real dependency
- Dependency API changes have blast radius of one wrapper

**Wrapper guidelines:**

- **Thin -- translate, don't decide.** Logic stays in domain code.
  Wrappers only convert between external and internal representations.
- **Organize by domain concern, not by dependency.** An external dependency
  may serve multiple domain concerns (Docker handles containers, networks,
  and volumes). Domain code is structured around those concerns
  independently. If a domain concern later moves to a different dependency,
  domain code doesn't change -- only the wrapper behind it.
- **Don't leak.** No dependency types, naming conventions, or error formats
  in domain code. Translate everything at the wrapper boundary.

**Interface extraction:** you don't need every interface designed upfront.
Extract the minimal interface when you first write code that touches a
dependency. The constraint is not "define all interfaces early" -- it's
"dependency types never appear in domain code." If domain code is about to
import a dependency package, that's the signal to introduce a wrapper and
interface.

**Interface granularity:** start with the interface your first consumer
needs -- no broader, no narrower. When a second consumer needs a different
subset of methods, that's the signal to split. This follows naturally from
the existing principles:

- **One nameable job** (Section 1) applies to interfaces too. If an
  interface serves unrelated callers with different needs, it's doing
  two jobs.
- **Organize by domain concern** produces naturally narrow interfaces. A
  unit that only creates containers shouldn't depend on an interface that
  also knows about networks.
- **Incremental complexity** ([Section 12](#12-incremental-complexity))
  means don't pre-split interfaces before you have a reason. One consumer
  = one interface. Split when reality demands it, not in anticipation.

_Specific wrapper designs and interface definitions live in implementation
specs, not here._

## 5. Error Handling

**Core philosophy: fail fast, fail loudly.** If something is wrong, surface
it immediately and clearly. Silent failures are the worst kind -- they
propagate through the system and surface far from the cause. Invalid config
fails at load, not ten steps later at container start. Missing Docker daemon
reported before building a mount list.

Inconsistent error handling is invisible for months, untraceable in
production. Establish conventions before agents write code.

**Errors are part of the contract.** Not just "given X return Y" but "given
X, fail _this specific way_."

**Principles:**

1. **Created at failure point.** The closest function to the failure creates
   the error with sufficient context.
2. **Wrapped as they propagate.** Each layer adds context as the error moves
   up the call stack. Error chains should read like call paths.
3. **Handled at entry points.** CLI commands and top-level composed units are
   the handling boundary. Internal code propagates transparently. Entry
   points retry, fall back, or surface errors to the user.
4. **Propagate or handle, never both.** Don't log an error and also return
   it. That produces duplicate noise and obscures where the error was
   actually handled. Each error has exactly one handler -- everything else
   wraps and propagates.
5. **Translate at wrapper boundaries.** External dependency errors don't
   leak into domain code. Wrappers catch dependency-specific errors and
   return havn's own error types. This keeps dependency isolation
   ([Section 4](#4-dependency-isolation)) intact and gives callers errors
   they can reason about.
6. **No exceptions as flow control.** Error paths use explicit returns, not
   panics or exceptions. A crash from a programming mistake is acceptable --
   using crash mechanisms for expected conditions is not.

**User-facing errors must be actionable.** "Docker daemon not running" >
"connection refused." Tell the user what's wrong and what they can do about
it.

_Error types, wrapping conventions, and Go-specific patterns in coding
standards spec._

## 6. Multi-Unit Operations

Section 5 governs how individual units fail. This section governs what
happens when an operation spans multiple independent units -- starting
several containers, stopping a set of services, applying config to
multiple resources.

**Best-effort, not all-or-nothing.** When an operation acts on multiple
independent units (e.g., starting five containers), a failure in one does
not abort the rest. Each unit either succeeds or fails on its own terms.
Fail-fast ([Section 5](#5-error-handling)) applies _within_ each unit --
a single container that can't start fails immediately and loudly. But that
failure does not cascade to unrelated units. A healthy container is not
rolled back because a sibling failed.

**Report everything.** The user must know the full outcome -- what
succeeded, what failed, and why. Partial success is not silent success.
The entry point collects results from all units and presents a complete
picture. This is non-negotiable: a user who runs a multi-resource command
and sees no errors must be able to trust that everything is running.

**Respect declared dependencies.** When units have ordering requirements
(database must be running before the app connects), those dependencies
must be explicit in configuration -- not inferred. If unit B depends on
unit A, A must succeed before B starts. If A fails, B is skipped with a
clear explanation (not attempted and left to fail with a confusing error).

**Parallelize independent work.** Units with no declared dependency
between them can run concurrently. This is a performance concern but also
a correctness one: concurrent operations must not produce race conditions
or interfere with each other's state. If two units share a resource
(network, volume), the design must account for that explicitly.

**The composition pattern:** the orchestrating unit is responsible for
dependency ordering, concurrency decisions, result collection, and final
reporting. Individual units know nothing about siblings -- they do their
one job and return success or failure. The orchestrator composes the
overall outcome.

## 7. Blast Radius

Every decision contains or expands blast radius.

**Containment:**

- Small primitives: bug in `DeriveContainerName` affects naming, not mounts
- Module boundaries: `internal/config/` changes don't ripple into
  `internal/container/`
- Dependency wrappers: Docker API change affects one wrapper
- Black-box tests: contract unchanged = zero refactor blast radius

**Structural property, not carefulness.** Agent on config must not silently
break container lifecycle. Can't count on carefulness you can't observe.

**Import direction.** Dependencies flow downward through composition layers.
Higher-level packages import lower-level ones, never the reverse.

- If two packages at the same level need to interact, define an interface
  in the consumer -- don't create a direct import dependency. Consumers
  define the interfaces they need.
- Shared types needed by multiple packages get their own package, not
  stuffed into one of the consumers.
- No circular dependencies, no shared mutable state.
- When in doubt: if adding an import creates a dependency that feels wrong,
  it probably is. Extract an interface or restructure.

Packages have well-defined public APIs. Need shared logic? Extract a
primitive.

## 8. State and Configuration

**havn is stateless by design.** No persistent state beyond config files.
Container state = Docker. Database state = Dolt. Volume state = disk. havn
queries at runtime, never caches or duplicates.

This means: no local DB to corrupt, no sync problems, no stale cache, any
process killable without cleanup.

**Sane defaults, full override.** havn should work out of the box with zero
config for the common case. But every default must be overridable. CLIs are
power tools for power users -- making configuration awkward or inaccessible
is a design failure.

**Only expose what the user has a reason to change.** Not every internal
value is configuration. Connection timeouts, buffer sizes, internal retry
counts -- these are implementation details, not user-facing knobs. The
test: can you explain to a user _why_ they'd want to change this value?
If not, it's code, not config. The composable primitive design makes it
straightforward to expose more configuration later when a real need
emerges -- so err on the side of less upfront.

**Expectations:**

- **Config files** for persistent preferences and project-specific settings.
- **CLI flags** for ephemeral overrides and scriptability. A power user
  should be able to integrate havn into shell scripts without touching
  config files.
- **Deterministic precedence.** When multiple sources set the same value,
  the resolution order must be documented and predictable. No surprises.

**Design test:** if a user asks "how do I change X?", the answer should
always be one of: a flag, an env var, or a line in config. Never "edit this
source file" or "it's not possible."

_Specific config layers, file paths, and precedence rules live in a config
spec._

**Config is a pure data problem.** Merging/validation/defaults = pure
functions, no dependency on Docker or filesystem beyond reading files.

**State at the edges.** Core logic stateless. CLI commands read config,
query Docker, pass results into pure functions.

## 9. Observability

Testing, types, and linting catch problems during development. Observability
catches problems when the tool is actually running. The user must be able to
understand what havn is doing and why -- without reading source code.

**Surface problems to the user.** If something affects the user's system or
workflow, they must know immediately. Not silently fail, not bury it in a
log. This complements fail-fast error handling
([Section 5](#5-error-handling)) -- errors are surfaced clearly, but observability
also covers degraded states, warnings, and unexpected conditions that aren't
errors yet.

**Logs for diagnosis.** Default output is minimal and human-friendly.
Detailed logs are available when the user needs to understand behavior or
diagnose problems -- but not shown unless requested. The user controls how
much detail they see.

**`--json` for machine-readable output.** Any command that displays
information should support a `--json` flag. This is an established CLI
convention that serves two audiences: AI agents that need to parse output
using tools, and power users building shell script wrappers. Without
`--json`, output is human-friendly and visually organized for
comprehension. With `--json`, output is structured and stable for
programmatic consumption.

_Logging levels, output formats, and instrumentation patterns in
implementation specs._

## 10. Quality Gates and Feedback Loops

Catch drift before humans see it. Tighter loop = less review burden.

**Automate everything that can be a pass/fail check:**

- **Tests**: primary feedback loop. Pass/fail → agent adjusts. Only works
  if tests encode actual contracts.
- **Static analysis and formatting**: zero ambiguity, free guardrails.
  Automate style so it's never a review discussion.
- **Build**: the codebase must compile cleanly. Trivial but load-bearing.
- **Type system**: use distinct types to prevent confusion at compile time.
  If two values are semantically different (e.g., a container name vs an
  image name), they should be different types -- not bare strings that can
  be silently swapped.

**CI as backpressure.** Recurring quality issue → encode as CI check, not
review comment. Review comments are lost -- next agent starts fresh.

**Review → codebase signal.** Human catches mistake → ask what context was
missing. Update conventions, add example, tighten type, add lint rule.
Convert corrections into permanent signals.

_CI config, lint rules, pre-commit hooks in dedicated spec._

## 11. Codebase is the Prompt

This is the principle unique to agent-led development. Every file an agent
reads is context. Every function signature, type definition, test file, and
naming choice shapes the agent's understanding of the system and the code it
produces. Vague conventions produce vague output. Inconsistent examples
produce inconsistent output.

**What this means in practice:**

- **Function signatures and types are self-documenting.** An agent should
  be able to understand a package's contract from its public API without
  needing external documentation. If a function requires explanation beyond
  its signature and a brief doc comment, the API is too clever.
- **Test files are executable examples.** Tests show how a package is
  intended to be used -- valid inputs, expected outputs, error cases. An
  agent reading tests should learn the contract faster than reading docs.
- **Consistent patterns across packages.** When an agent learns how one
  package is structured, that knowledge should transfer. If config merging
  works one way and mount resolution works a completely different way for
  no good reason, the agent can't generalize.
- **Specs are part of the codebase.** These spec files exist in the repo
  specifically so agents read them as context. They're not external docs --
  they're first-class inputs that guide agent behavior.
- **Colocation.** Related things together. Tests beside the code they test.
  Non-obvious behavior documented on the function, not in a separate doc
  that drifts. An agent looking at a function should find everything it
  needs nearby.

**When the codebase sends conflicting signals.** Inconsistencies will
exist -- naming drift, structural differences between packages, patterns
followed in some places and broken in others. When an agent encounters
conflicting patterns:

1. **Follow the majority pattern.** The most common approach in the
   codebase is the de facto convention. Don't follow outliers unless a
   spec, doc comment, or explicit instruction justifies the deviation.
2. **File the inconsistency as a bug.** Drift is a defect -- it degrades
   the codebase's ability to guide future work. Flag it so it gets fixed,
   don't silently propagate either pattern without addressing the conflict.

An agent should never guess which pattern is "newer" or "better." Count
occurrences, follow the majority, and make the inconsistency visible.

**The feedback loop:** when a human reviews agent output and finds a
mistake, the question is not "what did the agent do wrong?" but "what did
the codebase fail to communicate?" Fix the signal -- tighten a type, add a
test case, rename a function, update a spec -- so the next agent gets it
right.

## 12. Incremental Complexity

Add abstraction only when cost of not having it > cost of introducing it.
Premature abstraction compounds -- agents build on it because it exists, not
because it's right.

- Start concrete. Introduce abstraction when a real need emerges -- a second
  implementation, a testing requirement, or repeated patterns that clearly
  warrant it. But never compromise dependency isolation
  ([Section 4](#4-dependency-isolation)) in the name of concreteness.
- Three similar lines > premature abstraction. No helpers for one-time ops.
- Build what specs define, not a framework that could do anything.
- Prefer reversible decisions. Function before package. Promoting cheap;
  demoting expensive.

## Applying

Every principle above exists because the codebase is the prompt. Agents
read what you write -- code, types, tests, specs, naming, structure. Make
it clear, and agents produce clear output. Leave it vague, and agents
fill the gaps with assumptions.

When principles tension, the right answer is contextual -- but some
tensions recur and have clear resolutions.

**Common tensions:**

1. **Dependency isolation (4) vs. incremental complexity (12).**
   Isolation wins. A wrapper feels like premature abstraction when there's
   only one implementation, but isolation is structural -- the cost of not
   having it compounds silently. Section 12 already states this
   explicitly: never compromise isolation in the name of concreteness.

2. **Spec says Y, codebase does X everywhere.** The spec wins -- it
   represents deliberate design intent. But following the spec means
   migrating toward it, not just applying Y to new code while X persists.
   File the inconsistency so existing code gets updated.

3. **Fail-fast (5) vs. best-effort (6).** Not a real tension -- they
   operate at different levels. Fail fast within a single unit. Best-effort
   across independent units in the same operation. Section 6 resolves this.

4. **One nameable job (1) vs. practical convenience.** If splitting a unit
   produces a piece with no independently testable contract, the split
   isn't earning its keep. "One job" means one meaningful responsibility,
   not one line of code. Use the decomposition questions in Section 1 to
   judge.

For tensions not listed here: document the tradeoff in a comment or commit
message so the reasoning is visible to the next agent.

**Decision checklist:**

1. One nameable job? If it needs "and," split it.
2. Can this be pure? If yes, make it one.
3. Build or depend? Ownership < dependency cost?
4. What's the blast radius? Change here breaks there = wrong boundary.
5. Error contract explicit? Caller knows how this fails?
6. Dependency isolated? Swappable without touching domain?
7. Would an agent find this? Discoverable name, explicit behavior, clear
   contract?
8. Simplest thing that works? Abstraction not earning keep → remove it.
