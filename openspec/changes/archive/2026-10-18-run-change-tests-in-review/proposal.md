## Why

The full implementation review never sees a test run. The hook runs only the linters of the touched modules, and its prompt tells the reviewers to "check the agent's claims about tests against the repository" (`.agent-hooks/implementation-prompt.md:3`). A reviewer can read a test, but it cannot tell whether the test passes, and nobody reviews the tests themselves as a whole: the delta spec reviewer only matches each scenario to a test. Test rules are also scattered across the module docs, and some of them are duplicated there, so a reviewer has no single standard to check tests against.

The hook can also fail silently. It starts through `python3` from the user's shell (`.claude/settings.json`), and it finds `openspec` the same way. When `openspec` is missing, the proposal review quietly stops running (`.agent-hooks/review_gate.py:660-674`). A check that times out leaves the processes it spawned running, because only the direct child is killed (`review_gate.py:738-758`).

The sibling project the same team works in solved the launch and timeout problems and lets a change name test commands the hook runs. This change brings that here. It keeps this repository's rule that agents never run the full backend suite, because that suite resets the test containers other agents share.

## Governing docs

This change answers to the root [`AGENTS.md`](../../../AGENTS.md). It edits the testing sections of [`backend/AGENTS.md`](../../../backend/AGENTS.md) and [`agent/verification/AGENTS.md`](../../../agent/verification/AGENTS.md), so it answers to those as well. It touches no product code.

## What Changes

- A change's `tasks.md` can end with a "Change verification" section: a shell code block with one test command per line. The full implementation review runs these commands after the linters and before the reviewers, and blocks the turn with the end of the output when one fails. Only three command forms are allowed, and their arguments are checked against a safe character set: backend package tests, the verification agent's tests and the frontend tests. The full backend suite is not among them. The full proposal review checks the section's format before validation.
- The full implementation review gains a tests reviewer. It checks the tests the change added or changed against the testing skill and the module docs. A test must fail without the behavior it pins, and a bug fix needs a test that reproduces the bug. It also checks that the change verification commands cover the packages whose tests the change touched.
- The task reviewer of the full proposal review checks that a change whose tasks add or change tests that an allowed form runs has the "Change verification" section, and that the section covers the packages named in the tasks' verifications. The hook tests are exempt, because the module check already runs them.
- A new `testing` skill holds the test rules that apply to every module: run what you wrote, start a bug fix with a failing test, wait for conditions instead of sleeping, clean up only your own leftovers after an interrupted run. The module docs keep their stack-specific rules and link to the skill. Reviewers load the skill when the work involves tests.
- The hooks start through a wrapper script that runs the hook with the Python and OpenSpec versions pinned in `mise.toml`. When the wrapper or a tool is missing, the hook stops the agent with the reason instead of skipping the review. A new `just doctor` command checks the tools the hooks need.
- A check that times out is stopped together with every process it spawned. Each full review runs inside a time budget that ends before the hook timeout: the checks share one total limit, and a reviewer starts only when its whole limit still fits, otherwise the hook reports it. Both full review hook timeouts grow from 3600 to 7200 seconds.

### Out of scope

- Choosing tests automatically from the changed files. The change names its commands.
- Running the full backend suite (`just test`) or the Docker filesystem suite from the hook or from a task.
- Upgrading the OpenSpec CLI or regenerating the `openspec-*` skills. `mise.toml` pins the version already in use.
- Moving the backend naming rules, controller test rules or the canonical controller test out of `backend/AGENTS.md`. They are stack-specific.

## Capabilities

### New Capabilities

- `review-hook-launch`: how the hooks start with the pinned tools, what happens when a tool is missing, and how the user checks the environment.

### Modified Capabilities

- `implementation-review`: the checks before the reviewers also run the change's verification commands, under a shared time limit and with process groups stopped on timeout. A tests reviewer joins the parallel reviewers. New requirements cover which verification commands are allowed and the time budget that keeps the review inside the hook timeout.
- `proposal-review`: the checks before the reviewers also validate the format of the "Change verification" section. The task reviewer checks that the section exists when the tasks add or change tests an allowed form runs. A new requirement covers the same time budget.

## Impact

- `.agent-hooks/`: new `run.sh`; `review_gate.py`, `review_gate_test.py`, `common-prompt.md`, `diff-prompt.md`, `implementation-prompt.md` and `proposal-prompt.md` change.
- `.claude/settings.json`, `.codex/hooks.json`: every hook runs through `run.sh`, and both full review hook timeouts become 7200 seconds.
- `mise.toml`: pins `python` and `npm:@fission-ai/openspec`. Every worktree needs `mise install` before its first turn after the change lands.
- `Justfile`: a `doctor` recipe.
- `.agents/skills/testing/SKILL.md` and a symlink to it in `.claude/skills/testing/`.
- `AGENTS.md`, `backend/AGENTS.md`, `agent/verification/AGENTS.md`: the command table gains `just doctor`, a pointer to the testing skill, and duplicated test rules replaced by links.
- `openspec/config.yaml`: a tasks rule describing the "Change verification" section.
