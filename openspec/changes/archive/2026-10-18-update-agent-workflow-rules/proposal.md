## Why

The root `AGENTS.md` gives agents two opposite instructions about refactoring. "After each run: suggest refactorings" (`AGENTS.md:87-89`) tells the agent to list refactorings and not apply them, while "Refactoring within a feature" (`AGENTS.md:113-115`) makes the refactoring part of the same task. Agents follow the first one, so touched code stays unrefactored and the summary ends with a list the user has to turn into work.

The summary an agent writes after a full review also retells every review round, because the hook asks for it (`.agent-hooks/review_gate.py:1037-1039`, `:1062-1066`). The user wants the result, not the history of findings and fixes. The sibling project the same team works in has already fixed both problems. It also has rules this repository lacks: a "pack" command that archives a finished change and commits it in one step, foreground checks, stopping dev processes an agent started, and keeping lasting rules in the repository instead of an agent's local memory. This change brings those rules here.

## Governing docs

This change answers to the root [`AGENTS.md`](../../../AGENTS.md). It touches no code under `backend/`, `agent/`, `frontend/` or `website/`, so the module docs do not govern it.

## What Changes

- Delete "After each run: suggest refactorings" from the root `AGENTS.md`. "Refactoring within a feature" gains the rule that the agent refactors touched code while implementing, without asking, and names in its summary only refactoring that goes beyond the change and needs the user's agreement. `openspec/config.yaml` points to that section.
- After a blocked full review the agent finishes with one short line about its fixes and names the findings it rejected, because the repeat review reads that reply. The summary for the user describes the final result as if there had been no review. The quick checks ask for the same kind of reply after a block. The hook texts, the "Review hooks" section of `AGENTS.md` and both full review specs change accordingly.
- Add the "pack" command: sync the change's delta specs into the main specs, archive the change, and commit the code, the main specs and the archive in one commit in the format of the Git commits skill. The command itself is the user's consent to that commit. It does not push.
- Add a rule that the agent runs tests, linters and builds as ordinary foreground commands, and stops any backend or frontend process it started for a check before the task ends.
- Add a rule that the agent writes nothing to its local memory outside the repository. A rule or preference that must outlive the session goes into `AGENTS.md` or a skill, with the user's consent.
- Expose the `git-commits` skill to Claude Code through a symlink in `.claude/skills`, the way `humanizer` already is.

### Out of scope

- Moving the skills into a single `.skills/` directory, upgrading the OpenSpec CLI and regenerating the `openspec-*` skills. Those files are generated, and regenerating them drops the hand-added Humanizer preflight.
- Moving the user's existing local memory entries into the repository. The user decides which of them become rules.
- The user's global `~/.claude/CLAUDE.md`, which also tells agents to list refactorings rather than apply them. It lives outside the repository.
- Running tests from the review hook and a test reviewer. The change `run-change-tests-in-review` covers them.

## Capabilities

### New Capabilities

- `agent-workflow`: how the agent refactors touched code, what its summary after a review contains, the "pack" command, foreground checks and dev processes, where lasting rules live, and which repository skills each agent client can load.

### Modified Capabilities

- `implementation-review`: the reply after a block is one short line for the repeat review, and the summary after acceptance describes the result without review rounds.
- `proposal-review`: the same for the summary of a proposal; options to cut the feature are still listed separately.

## Impact

- `AGENTS.md`: one section deleted, rules added to "Refactoring within a feature", "Review hooks" and "OpenSpec", new short sections for running commands and agent memory.
- `openspec/config.yaml`: a context line about refactoring, and the apply guidance on commits names the "pack" command.
- `.agent-hooks/review_gate.py` and `review_gate_test.py`: block and summary texts.
- `.claude/skills/git-commits/SKILL.md`: new symlink.
