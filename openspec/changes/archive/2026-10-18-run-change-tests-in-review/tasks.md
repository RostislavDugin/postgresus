The hook test command below runs from the repository root: `python3 -m unittest discover -s .agent-hooks -p '*_test.py'`.

## 1. Hook launch

- [x] 1.1 Pin `python = "3.14.7"` and `"npm:@fission-ai/openspec" = "1.11.0"` in `mise.toml`. Verify with `mise install`, then `mise exec -- python3 --version` printing `Python 3.14.7` and `mise exec -- openspec --version` printing `1.11.0`.
- [x] 1.2 Add `.agent-hooks/run.sh` per design.md. It finds `mise` on `PATH` or in `$HOME/.local/bin`, refuses to start when `mise ls --current --missing` reports the pinned `python` or `npm:@fission-ai/openspec` missing, sets `MISE_EXEC_AUTO_INSTALL=false` and runs `review_gate.py` through `mise exec`. When `mise` is missing, a pinned tool is missing or the hook fails, it prints an escaped stop JSON with exit code 0. For `doctor` it passes the exit code through and exits with code 1 when `mise` or a pinned tool is missing. Add `LauncherTest` to `review_gate_test.py`. It runs the script with a stub `mise`, which records its arguments and environment, and a temporary `HOME`:
  - with no `mise` on `PATH` but a stub in `$HOME/.local/bin`, the hook runs through the stub (`review-hook-launch` scenario "A client started from an IDE");
  - the stub sees `MISE_EXEC_AUTO_INSTALL=false`, and when its `ls --missing` reports a missing pinned tool while its `exec` still succeeds, like real `mise`, the output is valid JSON with `continue: false` that names `mise install` and `just doctor`, and `exec` never runs ("A pinned version is not installed");
  - with no `mise` anywhere, the output is valid JSON with `continue: false` that names the version manager and `just doctor` ("The version manager is missing");
  - with a stub that exits with code 3, the output is valid JSON that names the action, the code and `just doctor` ("The hook crashes");
  - a message containing a double quote stays valid JSON;
  - `run.sh doctor` exits with the code of the stub, and with code 1 when no `mise` is found ("No version manager").

  Verify with the hook test command.
- [x] 1.3 In `review_gate.py`, raise `MissingToolError` when `openspec` cannot start in `is_planning_complete`, and turn it into a stop message that names the tool and `just doctor`. Add the `doctor` action, which reports Python, `git`, `openspec` and the reviewer CLIs, and keep `--client` required for every other action. Update the module docstring and the comment above `is_planning_complete`. Add these tests:
  - `MissingToolTest`, covering "OpenSpec is missing" with `openspec` removed from the test `PATH`;
  - `DoctorTest`, covering "Every tool is present" and "No reviewer CLI".

  Verify with the hook test command.
- [x] 1.4 Add the `doctor` recipe (`.agent-hooks/run.sh doctor`) to the root `Justfile` and its row to the command table in the root `AGENTS.md`. Verify with `just doctor` exiting with code 0, and with `PATH=/usr/bin HOME=$(mktemp -d) .agent-hooks/run.sh doctor` exiting with a non-zero code.
- [x] 1.5 Switch every hook in `.claude/settings.json` and `.codex/hooks.json` to `run.sh`, and set the timeouts of the `implementation` and `proposal` hooks to 7200 in both files. Verify with `python3 -m json.tool` on both files, and with `echo '{"hook_event_name":"Stop","session_id":"manual-check","cwd":"'$PWD'"}' | .agent-hooks/run.sh naming --client claude` printing `{}`.

## 2. Checks, time budget and change verification commands

- [x] 2.1 Rewrite `run_checks` per design.md: start every check in its own process group, kill the group on timeout, and enforce the shared 1800-second total limit on top of the 900-second limit per check. The failure message names the exceeded limit. Make the limits parameters so tests can shrink them. Add `RunChecksTest`:
  - a check whose child process outlives the timeout leaves no process behind (`implementation-review` scenario "A test hangs");
  - two checks that each stay under their own limit but together pass the total limit fail with a message naming the total limit, and the second check's process group is gone ("The checks together exceed the total limit").

  Verify with the hook test command.
- [x] 2.2 Add the time budget of both full reviews per design.md: 6600 seconds from the hook's start, with checks inside it, and a reviewer job that starts only when the budget still holds its whole limit. A job that cannot start returns the "review time budget ran out" error. Make the budget a parameter. Add tests to `ImplementationGateReviewTest` and `ProposalGateReviewTest` with a shrunk budget and reviewer limit:
  - a third wave of reviewers does not start, and the turn is blocked with the budget reason (both "Too many reviewers for the budget" scenarios);
  - two waves that fit both run ("Two waves of reviewers after slow checks").

  Verify with the hook test command.
- [x] 2.3 Add the change verification parser per the `implementation-review` requirement "Which change verification commands the hook runs" and design.md. It reads the first `sh`-fenced block under `## Change verification`, skips empty and comment lines, accepts the three allowed forms with their directories, and checks every argument against the character set. Every error message carries the list of allowed forms. Add `ChangeVerificationParsingTest` with one test per scenario of that requirement except "Two changes name the same command", plus tests for:
  - `just test-package` with no package;
  - a package path without `./`;
  - extra arguments after the verification agent's form;
  - a `bash`-fenced block, which does not count;
  - an unclosed block;
  - an empty block;
  - a missing section, which yields no commands.

  Verify with the hook test command.
- [x] 2.4 Make the implementation review parse the sections of every change it reviews before any check, run the de-duplicated commands after the module checks, and block on a parse error or a failing command. Update `test_never_runs_formatters_or_just` so that `just` runs only from a change verification block. Add tests to `ImplementationGateChecksTest` for these `implementation-review` scenarios:
  - "A package test fails": no reviewer runs and the output tail is passed;
  - "Backend package tests": the command runs in `backend/`;
  - "Two changes name the same command": the command runs once;
  - "The linters pass and the change has no verification commands": reviewers start and no test runs;
  - a malformed section, after which no check and no reviewer runs.

  Verify with the hook test command.
- [x] 2.5 Make the proposal review parse the section of each change before `openspec validate --strict`, without running its commands, and block on a parse error. Add a test to `ProposalGateValidationTest` for the `proposal-review` scenario "A forbidden command in the verification section": the turn is blocked, the stub `just` is never called and no reviewer runs. Verify with the hook test command.

## 3. Reviewers and prompts

- [x] 3.1 Add the `tests` role to the implementation review's `first_roles` and a "Role: tests" section to `implementation-prompt.md` per design.md, including the exemption for tests that a module check runs. Rewrite the prompt's opening so it says the hook ran the linters and the change verification commands. Raise by one the reviewer counts that the existing implementation review tests assert (`review_gate_test.py:447`, `:493`, `:591`, `:621`, `:632`, `:657`, `:680`). Add tests to `ImplementationGateReviewTest`:
  - a change with two delta specs starts four reviewers (`implementation-review` scenario "A change with two delta specs");
  - the `tests` role's prompt requires checking that a test fails without its behavior, that a bug fix has a reproducing test, that tests wait for conditions instead of fixed pauses, and that the change verification commands cover every package with changed tests that an allowed form runs, with hook tests counted as covered. This covers "A test that waits with a fixed pause", "A bug fix without a reproducing test", "Changed tests outside the verification commands" and "A change with only hook tests".

  Verify with the hook test command.
- [x] 3.2 Extend the tasks role in `proposal-prompt.md`. A change whose tasks add or change tests in a backend package, a verification agent package or the frontend needs a `## Change verification` section that covers every such package the tasks' verifications name. Hook tests need no section. Add a test to `ProposalGateReviewTest` that the task reviewer's prompt requires this and states the exemption. It covers the `proposal-review` scenarios "The task reviewer's role", "Tests without a verification section", "A section that leaves out a package" and "A change with only hook tests". Verify with the hook test command.
- [x] 3.3 Add the testing skill as a reading step to `common-prompt.md` for work that involves tests, and point the tests line of `diff-prompt.md` at the skill and the module docs. Verify with `grep -n "skills/testing/SKILL.md" .agent-hooks/common-prompt.md .agent-hooks/diff-prompt.md` finding both.

## 4. Testing skill and documentation

- [x] 4.1 Write `.agents/skills/testing/SKILL.md` with the rules design.md lists, in the repository's skill format with `name` and `description` front matter, linking the module docs for stack-specific rules. Create `.claude/skills/testing/SKILL.md` as a relative symlink to `../../../.agents/skills/testing/SKILL.md`. Verify with `readlink .claude/skills/testing/SKILL.md` printing that target and `head -4 .claude/skills/testing/SKILL.md` printing the front matter.
- [x] 4.2 Add the "Tests" section to the root `AGENTS.md`. Replace the shared opening sentence of the "Testing" sections in `backend/AGENTS.md` and `agent/verification/AGENTS.md` with a link to the skill. Add the tasks rule for the `## Change verification` section to `openspec/config.yaml`: last section, no checkboxes, one `sh`-fenced block, one command per line, the three allowed forms, and no section needed for hook tests. Rewrite the comment above `MODULE_CHECKS` to match the new behavior. Verify with these searches:
  - `grep -rn "Always run tests after writing them" AGENTS.md backend/AGENTS.md agent/verification/AGENTS.md` returns nothing;
  - `grep -n "Change verification" openspec/config.yaml` finds the rule.
- [x] 4.3 Search the repository for statements the change made untrue and bring them in line per "Documentation consistency". Look for the hook not running tests, the 3600-second full review timeouts, and `python3` as the hook command. Verify with `grep -rn "does not run the Go test suites\|never runs here" .agent-hooks AGENTS.md backend/AGENTS.md` and `grep -n "review_gate.py" .claude/settings.json .codex/hooks.json` both returning nothing.

## 5. Change validation

- [x] 5.1 Verify with `openspec validate run-change-tests-in-review --strict` passing, the hook test command passing and `just doctor` exiting with code 0.
