## Context

See proposal.md for the motivation and the delta specs for the behavior contract.

The rules sit in these places today:

- `AGENTS.md:87-89` "After each run: suggest refactorings" tells the agent to list refactorings and not apply them. `AGENTS.md:113-115` "Refactoring within a feature" makes the refactoring part of the task, and `AGENTS.md:111` asks the agent to simplify code within the task itself. The module docs already lean the second way: `backend/AGENTS.md` "Refactor tests as you touch them" and `frontend/AGENTS.md:248`.
- The reply after a block comes from hook texts. The quick checks block with "Fix the result and finish the turn again" (`.agent-hooks/review_gate.py:637`). A failed check before a full review uses "Fix the failure and finish the turn again" (`:918`). The full reviews block with `block_instruction` (`:1031`, `:1056`) and ask for a summary with `summary_instruction` (`:1037-1039`, `:1062-1066`). That summary covers "what the full review found in each round and how you resolved it". The hook tests pin the old wording at `review_gate_test.py:807`, `:1057` and `:1123`.
- "Review hooks" in `AGENTS.md:91-102` says what the agent does with findings, and nothing about the shape of the reply.
- Commits follow `.agents/skills/git-commits/SKILL.md`, and `openspec/config.yaml` apply guidance says "Never commit or push without explicit approval". Nothing describes packing a finished change.
- `.claude/skills/humanizer/SKILL.md` is a relative symlink to `.agents/skills/humanizer/SKILL.md`. `git-commits` has no entry under `.claude/skills`, so Claude Code does not list it as a skill.

The rules this change adds come from the sibling project's `AGENTS.md`, its hook texts and its `openspec/config.yaml`, translated and fitted to this repository's sections.

## Goals / Non-Goals

**Goals:**

- One rule for refactoring touched code, with no contradicting section left.
- Every hook text that asks for a reply asks for the reply shape the `agent-workflow` spec defines.
- Each new rule written once, in the section an agent reads for that situation.

**Non-Goals:**

- New hook behavior. Only texts change in the hook script; triggers, gates and state stay as they are.
- Rewriting the module docs. Their refactoring lines already agree with the new rule.

## Decisions

### The refactoring rule lives in "Refactoring within a feature", and the listing section goes

"After each run: suggest refactorings" is deleted. "Refactoring within a feature" gains two sentences. The agent refactors the touched code while implementing, without asking. Its summary names only refactoring beyond the change, with the cost, because that needs agreement. `openspec/config.yaml` gets one context line that points to the section, the way its last context line already points to "Review hooks".

Rejected: narrowing the listing section to out-of-scope code and keeping it. The two sections would still describe one decision from two places, and "One rule is written in one place" (`AGENTS.md`, "Documentation consistency") forbids that. Rejected: putting the rule only in `openspec/config.yaml`, as the sibling project did. Agents read that file only inside the OpenSpec workflow, and plain requests would keep the old behavior.

### The reply shape is written in "Review hooks" and repeated in every hook text that asks for a reply

"Review hooks" gets two bullets. After a full review blocks the turn, the agent replies with one short line about its fixes and names the rejected findings with reasons, because the repeat review reads that reply. The summary for the user describes the final state as if there had been no review. The hook texts carry the same instruction at the moment it applies:

- the quick check block at `review_gate.py:637` is shared by every stop gate, and its subject comes from `get_stop_subject` (`review_gate.py:569-589`). For the subject "finished change" it asks for a new reply written for the user, as a summary of the final state, without retelling the findings or fixes. For the subject "plan", which Codex shows in its final message, it keeps asking the agent to revise the plan and show it again, as the ExitPlanMode check does at `review_gate.py:555`. The "OpenSpec proposal" subject of the necessity check gets the summary wording, because its reply reports artifacts the agent wrote;
- the check failure at `:918` and both `block_instruction` strings ask for one short line about the fixes and the rejected findings with reasons, and say that the summary comes after acceptance;
- both `summary_instruction` strings ask for the summary as if there had been no review. The implementation summary covers what was done, decisions, checks passed, `minor` findings left and the next step. The proposal summary covers what is proposed and the decisions, and still lists every `scope` finding with what the user loses and saves.

The rejected findings stay in the short reply, which is the only reply the repeat review reads.

Rejected: keeping review rounds in the summary and only shortening them. The user does not act on a fixed finding, and the block messages already show each round in the transcript. Rejected: dropping the short line and letting the agent write a full summary after every block. The repeat review would then read a user summary and lose the list of rejected findings it must judge.

### "Pack" is a paragraph in the OpenSpec section, and the commit format stays in the Git commits skill

The OpenSpec section of `AGENTS.md` gains the "pack" command. When the user asks to pack a change, in any language, the agent first checks that every task is checked; if not, it stops and names the unchecked tasks. Then it archives the change through `openspec-archive-change`. That skill already syncs the delta specs inline and verifies the main specs before it moves the change (`.agents/skills/openspec-archive-change/SKILL.md:93-132`). The pack request answers the skill's sync prompt: "Sync now" when the specs need it, "Archive now" when they are already synced. Then the agent commits the implementation, the main specs and the archive in one commit per the Git commits skill. The request is the consent to commit. Changes the agent did not make stay out per "Other agents' changes in the working tree", and the agent does not push. The apply guidance in `openspec/config.yaml` gets the same exception: the "pack" request counts as approval to commit, not to push.

Rejected: running `openspec-sync-specs` before the archive skill. The archive skill syncs on its own, and after a separate sync it would still stop at its "Archive now / Sync anyway / Cancel" prompt. Rejected: a separate skill for packing. The command is two steps that existing skills already perform, so a paragraph that links them is enough. Rejected: pushing as part of the command. Pushing publishes the work, and `AGENTS.md` keeps that a separate decision.

### Running commands gets its own short section next to the command table

A new section "Running commands" under "Development environment" holds both rules: checks run as ordinary foreground commands, and a backend, frontend or other long-lived process the agent started for a check is stopped before the task ends. "Several agents, one set of containers" stays about shared test resources.

Rejected: an exception for long runs in the background, as the sibling project has for its full suite. Agents here run only package tests and linters, which are short.

### Agent memory gets a section in "Engineering philosophy"

A section "Agent memory" says the agent writes nothing to local memory outside the repository, such as `~/.claude/projects/*/memory`. A lasting rule or preference goes, with the user's consent, into `AGENTS.md` or a skill. The reason is stated with it: other agents and people cannot read local memory, and it drifts from the repository rules.

Rejected: allowing local memory for personal preferences only. Claude Code and Codex keep separate memories, so the two clients would behave differently in one repository.

### `git-commits` reaches Claude Code through a symlink, like `humanizer`

`.claude/skills/git-commits/SKILL.md` becomes a relative symlink to `../../../.agents/skills/git-commits/SKILL.md`, the same shape as the existing `humanizer` link.

Rejected: moving every skill into one `.skills/` directory with symlinks from both client directories, as the sibling project does. That also needs regenerating the `openspec-*` skills, which are generated files and carry a hand-added Humanizer preflight. It is out of scope.

## Risks / Trade-offs

- [The user's global `~/.claude/CLAUDE.md` still says to list refactorings and not apply them] → Out of the repository's reach. The summary of this change tells the user, and the project rule is the more specific one for this repository.
- [Existing local memory entries keep loading after the memory rule lands] → The rule stops new entries. Moving or deleting the old ones is the user's decision, named in the summary.
- [A summary without review history hides that a rejected finding was argued away] → Rejected findings with reasons stay in the short reply, which the repeat review judges. A finding the repeat review keeps blocks again.
- [An agent packs a change whose implementation review has not accepted it] → Packing refuses a change with unchecked tasks, and the implementation review runs on the turn that checks the last task, before the user can ask to pack.
