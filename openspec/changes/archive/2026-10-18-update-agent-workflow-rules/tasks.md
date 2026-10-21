The hook test command below runs from the repository root: `python3 -m unittest discover -s .agent-hooks -p '*_test.py'`.

## 1. Reply shape in the hook texts

- [x] 1.1 Rewrite the quick check block text at `.agent-hooks/review_gate.py:637` per design.md. For the subjects "finished change" and "OpenSpec proposal", the agent fixes the result and finishes the turn again with a new reply written for the user, as a summary of the final state, without retelling the findings or the fixes. For the subject "plan", the agent revises the plan and shows it again. Add these tests:
  - extend `test_blocks_turn_with_reviewer_reason_when_reviewer_rejects` in `DiffGateTest` to assert that the block reason asks for a summary of the final state and no longer says "Fix the result" (`agent-workflow` scenario "The diff check blocks a turn");
  - a test where a Codex plan turn is blocked by the necessity check, asserting that the reason asks to revise the plan and show it again and does not ask for a summary ("A quick check blocks a plan in the final message").

  Verify with the hook test command.
- [x] 1.2 Rewrite the check failure text at `review_gate.py:918` and both `block_instruction` strings (`:1031`, `:1056`) per design.md: finish the turn again with one short line about the fixes that names every rejected finding with its reason, and leave the summary for the user until the review accepts the result. Extend these tests to assert the "one short line" instruction, the rejected findings with reasons, and that no summary is asked for yet:
  - `test_blocks_turn_and_requires_agent_to_resolve_findings_when_reviewer_finds_bug`, covering the `implementation-review` scenario "A bug is found" and the `agent-workflow` scenario "The full implementation review blocks a turn";
  - `test_blocks_turn_with_output_tail_when_linter_fails`, covering the `agent-workflow` scenario "A linter fails before the full review";
  - `test_blocks_turn_and_requires_agent_to_resolve_simplify_finding_without_asking_user`, plus a new test where a `gap` finding blocks the proposal review, covering the `proposal-review` scenario "A gap blocks the turn".

  Verify with the hook test command.
- [x] 1.3 Rewrite both `summary_instruction` strings (`review_gate.py:1037-1039`, `:1062-1066`) to ask for a summary as if there had been no review, with the contents design.md lists for each. The proposal summary still lists every `scope` finding with what the user loses and saves. Rename and update the tests that pin the old wording at `review_gate_test.py:807`, `:1057` and `:1123`, so they assert the "as if there had been no review" wording and that "each round" is gone. They cover the scenarios "Accepted after fixes" of both `implementation-review` and `proposal-review`. Add a test where a repeat proposal review accepts with a `scope` finding, asserting the summary request lists the option with what the user loses and saves and asks to leave out review rounds ("Accepted after fixes with an option to cut the feature"). Verify with the hook test command, and with `grep -n "each round\|whole implementation, not only\|whole proposal, not only" .agent-hooks/review_gate.py` returning nothing.

## 2. Agent instructions

- [x] 2.1 Delete "After each run: suggest refactorings" (`AGENTS.md:87-89`). Add to "Refactoring within a feature" the two sentences from design.md: the agent refactors the touched code while implementing, without asking the user, and its summary names only refactoring that goes beyond the change, together with its cost. Add the context line to `openspec/config.yaml` that points to that section. Verify with these searches:
  - `grep -n "suggest refactorings\|don't silently apply" AGENTS.md` returns nothing;
  - `grep -n "while implementing, without asking" AGENTS.md` and `grep -n "goes beyond the change, together with its cost" AGENTS.md` each find the new sentence;
  - `grep -n "Refactoring within a feature" openspec/config.yaml` finds the new line.
- [x] 2.2 Add the two bullets about the reply after a block and the summary without review history to "Review hooks" in `AGENTS.md`, per design.md. Verify with `grep -n "one short line" AGENTS.md` and `grep -n "as if there had been no review" AGENTS.md` each finding its bullet.
- [x] 2.3 Add the "pack" paragraph to the OpenSpec section of `AGENTS.md` per design.md. It covers:
  - refusing a change with unchecked tasks;
  - `openspec-archive-change` with the pack request as the answer to its sync prompt;
  - one commit per the Git commits skill;
  - "Other agents' changes in the working tree";
  - no push.

  In `openspec/config.yaml` apply guidance, state that a "pack" request is approval to commit, not to push. Verify with `grep -n 'pack' AGENTS.md | grep openspec-archive-change` finding the paragraph and `grep -n "approval to commit, not to push" openspec/config.yaml` finding the guidance.
- [x] 2.4 Add the "Running commands" section under "Development environment" (foreground checks; stop long-lived processes the agent started for a check) and the "Agent memory" section under "Engineering philosophy" (no local memory outside the repository; lasting rules go into `AGENTS.md` or a skill with the user's consent), per design.md. Verify with `grep -n "^### Running commands\|^### Agent memory" AGENTS.md` finding both headings.
- [x] 2.5 Search the repository for other mentions of the changed rules and bring them in line per "Documentation consistency": the module docs, `adr/`, the repository's own skills and the hook prompts. Verify with `grep -rn -i "suggest refactor\|list refactor\|each review round\|whole implementation, not only" --include='*.md' --include='*.py' --include='*.yaml' . | grep -v '^./openspec/'` returning nothing.

## 3. Skill exposure

- [x] 3.1 Create `.claude/skills/git-commits/SKILL.md` as a relative symlink to `../../../.agents/skills/git-commits/SKILL.md`, the same shape as the `humanizer` link. Verify with `readlink .claude/skills/git-commits/SKILL.md` printing that target and `head -3 .claude/skills/git-commits/SKILL.md` printing the skill's front matter. This covers the `agent-workflow` scenario "The commit skill in Claude Code".

## 4. Change validation

- [x] 4.1 Verify with `openspec validate update-agent-workflow-rules --strict` passing and the hook test command passing.
