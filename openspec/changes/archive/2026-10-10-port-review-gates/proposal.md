## Why

The review hooks run one reviewer that judges a plan or a finished turn against every rule at once (`.agent-hooks/review_gate.py:158-168`, `.agent-hooks/review-prompt.md`). A single low-effort pass over all rules misses whole classes of problems: nobody asks whether the task is needed in this form, architecture and naming get a line each among twenty other checks, and a finished OpenSpec proposal or implementation is never reviewed as a whole against its own specs. Those gaps surface later, when the user reviews the work in a new session.

The sibling project the same team works in already splits this review into narrow gates with their own prompts, plus two full reviews of an OpenSpec change. It also has a rule for how an agent presents a task in explore mode, so that the user understands the problem before the details. This change brings both here.

## Governing docs

This change answers to the root [`AGENTS.md`](../../../AGENTS.md). It touches no code under `backend/`, `agent/`, `frontend/` or `website/`, so the module docs do not govern it. The hook prompts name every module doc, because the reviewers apply them to future changes.

## What Changes

- Replace the single reviewer with separate gates, each with its own prompt: `plan` (the plan solves the request at the right layer), `necessity` (whether the task is needed in this form, or something much smaller covers the same need), `architecture` (module architecture rules and accepted ADRs), `diff` (correctness, tests, documentation consistency), and `naming` (names the diff added or changed, run only when code changed).
- Add the full proposal review. When every artifact of an active change is written and no task is checked, the hook runs `openspec validate --strict`, then parallel reviewers: one per delta spec, one for proposal and design, one for tasks.
- Add the full implementation review. When every task of an active change is checked, the hook runs the read-only linters of the touched modules and the hook tests, then parallel reviewers: one per delta spec, one for documentation and tasks.
- Add a session start hook that tells the agent to load the communication skill first.
- Run the Codex reviewer on Codex itself instead of Claude.
- Add sections to the root `AGENTS.md`: what the agent does with review findings, how it presents a task in explore mode, "The best code is unwritten", documentation consistency, refactoring within a feature, and other agents' changes in the working tree. Remove the parts of the root `AGENTS.md` that the new `agent-instructions` rule forbids: hook internals, inventories, port numbers and command internals.
- **BREAKING** for anyone invoking the `reviewer` subagent: it is deleted. The hooks are the review now. Its path-to-doc table moves into the shared hook prompt and its website build check into the implementation review checks.

### Out of scope

- Upgrading the OpenSpec CLI or regenerating the OpenSpec skills.
- Cleaning the module `AGENTS.md` files against the new `agent-instructions` rule. They are not edited here.
- Running the hook tests in CI.
- Running the Go test suites from the hook. `make test` resets the shared test containers and slows every turn, so tests stay with the agent, and the reviewers check its claims about them.

## Capabilities

### New Capabilities

- `proposal-review`: the full review of a finished OpenSpec proposal before implementation.
- `implementation-review`: the full review of a change whose tasks are all checked.
- `agent-instructions`: what belongs in `AGENTS.md` and the repository's own skills, and how the diff reviewer enforces it.

### Modified Capabilities

None.

## Impact

- `.agent-hooks/`: the gate script, prompts, result schemas and new tests.
- `.gitignore`: Python bytecode caches that running the hook tests leaves behind.
- `.claude/settings.json`, `.codex/hooks.json`: new hook wiring.
- `.claude/agents/reviewer.md`: deleted.
- `AGENTS.md`, `openspec/config.yaml`: references to the reviewer subagent replaced, new sections.
