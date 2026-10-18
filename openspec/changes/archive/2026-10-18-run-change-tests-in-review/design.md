## Context

See proposal.md for the motivation and the delta specs for the behavior contract.

How the hooks start and check today:

- `.claude/settings.json` and `.codex/hooks.json` run `python3 .../.agent-hooks/review_gate.py <action>` with whatever `python3` the client's `PATH` holds. `mise.toml` pins neither Python nor the OpenSpec CLI. On this machine `python3` resolves through the global mise shims to 3.14.7. `openspec --version` reports 1.11.0, but that copy is `/usr/local/bin/openspec`, outside mise: mise itself has only 1.13.2 and 1.14.1 installed, and the generated `openspec-*` skills were written by 1.11.0.
- `is_planning_complete` (`.agent-hooks/review_gate.py:660-674`) treats a missing `openspec` like any other error and returns `False`. The proposal review then never starts, and the quick checks do. The comment at `:658-659` states this as intended.
- `MODULE_CHECKS` (`review_gate.py:112-128`) lists read-only checks per touched module. The comment above it says `just test` "never runs here" because it resets the shared test containers. `run_checks` (`:738-758`) runs each check with `subprocess.run(timeout=900)`. On timeout that kills only the direct child, and there is no limit across checks.
- The implementation review reviewers are one per delta spec plus `documentation and tasks` (`review_gate.py:1027`). They run at most six at a time (`MAX_PARALLEL_REVIEWERS`), each with a 1800-second limit (`review_gate.py:87`). Both full review hooks have a 3600-second timeout in both client configs, so two waves of reviewers already reach it, and nothing stops a third wave.
- `implementation-prompt.md:3` tells reviewers the hook does not run the Go test suites. `proposal-prompt.md` has the task reviewer reject any task that runs `just test` or `just test-filesystem`.
- `just test-package` (`backend/Justfile`) is a shebang recipe that interpolates its arguments into `go test -count=1 {{ packages }}` inside bash. `pnpm --dir frontend test` runs the `test` script of `frontend/package.json` (`vitest run`) through a shell. `just test` in `agent/verification/Justfile` runs `go test -race -count=1 -failfast ./internal/...`.
- Test rules live in `backend/AGENTS.md` ("Testing") and `agent/verification/AGENTS.md` ("Testing"). Both open with the same sentence, "Always run tests after writing them and verify they pass."

Constraints from the root `AGENTS.md`: agents never run `just test` ("Several agents, one set of containers"), the hook never modifies the working tree, and one rule is written in one place ("Documentation consistency"). From `backend/AGENTS.md`: test names follow `Test_WhatWeDo_WhichConditions_WhatWeExpect`, controller tests are preferred, and `just test-package` is the way to run backend packages.

## Goals / Non-Goals

**Goals:**

- The implementation review sees real test results for the packages the change names before any reviewer starts.
- Nothing in a task list can make the hook run an arbitrary command.
- A hook that cannot run is visible to the user.

**Non-Goals:**

- Choosing tests from the changed paths automatically.
- Making the full backend suite safe for parallel agents.

## Decisions

### The change names its test commands in a "Change verification" section

`tasks.md` may end with a section `## Change verification` that holds one `sh` code block with one command per line. The section has no checkboxes, so the apply workflow does not track it as a task. The implementation review reads it, de-duplicates commands across the changes it reviews, and runs them after `MODULE_CHECKS`. The proposal review parses it before `openspec validate --strict` and runs nothing from it. `openspec/config.yaml` gets a tasks rule that describes the section and its allowed forms.

The section is optional for the parser. Whether a change needs it is a judgment about the change, so the reviewers decide it. The proposal task reviewer asks for the section when the tasks add or change tests that an allowed form runs: backend packages, verification agent packages and frontend tests. The implementation tests reviewer checks that it covers every such package. The hook tests are exempt: no allowed form runs them, and the module check runs them whenever `.agent-hooks/` changes. The website has no test runner, only its lint and build module checks.

Rejected: selecting tests from the changed files, as the sibling project's `just test-affected` does. It needs an import-graph selector, and its fallback for unmapped paths is the full suite, which agents here must not run. Rejected: parsing the "Verify with" sentence of each task. Those sentences are prose, and a parser for them would guess. Rejected: requiring the section in the parser. A documentation-only change would need a fake command.

### Three allowed command forms, run without a shell, with a safe character set for arguments

The parser reads the first code block fenced as `sh` under the heading; a `bash`, `shell` or bare fence does not count, so there is one spelling to parse and test. It accepts:

| Form | Runs in |
|---|---|
| `just test-package <package>...`, at least one package starting with `./` | `backend/` |
| `just --justfile agent/verification/Justfile test`, nothing after it | the repository root; `just` switches to the justfile's directory |
| `pnpm --dir frontend test [<argument>...]` | the repository root |

The hook splits a line with `shlex` and runs the word list directly, without a shell. Every word after the fixed prefix must match `[A-Za-z0-9._/=:@-]+`. That set leaves out every shell operator, redirection and substitution character, so no separate operator check is needed.

The character check exists because the hook runs outside the agent's permission prompts, and both runners pass arguments through a shell. `just test-package ./internal/a;rm` contains no separate operator word, but the recipe would expand it to `go test -count=1 ./internal/a;rm` inside bash. The sibling project checks operator words only, which misses this case.

Rejected: `just test` with a directory rule for the verification agent. At the root the same text means the full backend suite, and both the parser and the reviewers would have to tell the two apart by context. Rejected: allowing `go test` directly. It would bypass the libedit shim that `just test-package` applies on Fedora.

### One time budget per full review, with limits for checks and process groups

Both full review hooks get a 7200-second timeout in both client configs. A full review runs inside a budget of 6600 seconds from the hook's start, which leaves 600 seconds for Git, `openspec status`, prompt building and result handling.

- `run_checks` starts each check with `Popen(start_new_session=True)`. On timeout it kills the whole process group with `os.killpg`. A check keeps its 900-second limit, and all checks of one review share a total limit of 1800 seconds. The failure message names the limit that was exceeded.
- A reviewer job starts only when the budget still holds its whole 1800-second limit. A job that cannot start returns the error "the review time budget ran out before this reviewer started", which blocks the turn like any reviewer that could not run.

The worst case that still runs is 1800 seconds of checks plus two waves of reviewers, 5400 seconds in all. Two waves happen when a change has more than four delta specs, because the docs and tests roles bring the job count above six. A third wave could not start after slow checks, and it is reported instead of being cut off by the client.

Rejected: keeping `subprocess.run(timeout=...)`. `just` starts bash, bash starts `go test`, and `go test` starts test binaries that hold a test slot database and testcontainers. Killing only `just` leaves the rest running. Rejected: no total limit for checks. Several 900-second checks would outlast the hook timeout, and the client would kill the hook with no message. Rejected: raising the hook timeout without a reviewer budget. A change with many delta specs would still outgrow any fixed timeout.

### The tests reviewer is a role of the existing full review

`IMPLEMENTATION_REVIEW.first_roles` gains `tests`, and `implementation-prompt.md` gets a "Role: tests" section. The role reads the testing skill and the module testing rules, reviews every test the change added or changed, and checks that the change verification commands cover the packages of those tests that an allowed form runs. Hook tests count as covered by their module check. It reports with the existing categories:

- `missing_test` when a test is missing or proves nothing;
- `bug` when a test is wrong or flaky, for example a fixed pause or leaked data;
- `minor` for the rest.

The delta spec role keeps matching scenarios to tests. The documentation role keeps the rule not to ask for the full suite.

Rejected: a separate Stop gate for tests. It would run on every turn and repeat the diff check's test point. Rejected: a new finding category. `missing_test` and `bug` already block the turn, and a new category would change the result schema and the block logic for no new behavior.

### A `testing` skill holds the rules shared by every module

`.agents/skills/testing/SKILL.md` holds the cross-module rules:

- run every test you wrote or changed and see it pass, and in the report tell a compile-only check from a test run;
- a bug fix starts with a test that reproduces the bug and fails;
- test what a caller observes, at the level the module doc prefers;
- mock only external systems and failures unsafe to reproduce;
- wait for a condition up to a limit instead of a fixed pause, and let a negative check first wait for a sign that the action was processed;
- run the tests the change touched with the commands from the root `AGENTS.md` table, and never `just test`;
- when package tests fail because the shared fixtures are down, tell the user;
- after an interrupted run clean up only your own leftovers, never all containers;
- a checklist before handing work over.

`.claude/skills/testing/SKILL.md` is a relative symlink to it. The root `AGENTS.md` gets a short "Tests" section that tells the agent to load the skill when it writes, changes, runs or reviews tests. `backend/AGENTS.md` and `agent/verification/AGENTS.md` replace their shared opening sentence with a link to the skill and keep their stack-specific rules. `common-prompt.md` adds a reading step for the skill when the work involves tests, and `diff-prompt.md` checks tests against it.

Rejected: copying the sibling project's skill whole. Its scenario-prose test names contradict `backend/AGENTS.md` and ADR-0013. Its template database per test does not match the slot databases here. Rejected: putting the shared rules in the root `AGENTS.md`. They matter only for test work, and the root file is read on every turn.

### Hooks start through `run.sh` with `mise exec`, and fail closed

`.agent-hooks/run.sh` finds `mise` on `PATH` or in `$HOME/.local/bin`, where the Fedora install puts it ("Development environment" in `AGENTS.md` names Fedora as the only supported host). It sets `MISE_EXEC_AUTO_INSTALL=false` and runs `mise -C <root> exec -- python3 review_gate.py "$@"`. When `mise` is missing or the hook exits with a non-zero code, it prints `{"continue": false, "stopReason": ..., "systemMessage": ...}` and exits with code 0. The message names the failure and says that `mise install` installs the pinned tools and `just doctor` checks the environment. `mise exec` with `MISE_EXEC_AUTO_INSTALL=false` still runs the command when a pinned version is missing, and `python3` and `openspec` then resolve to whatever else `PATH` holds. So before `mise exec`, `run.sh` runs `mise -C <root> ls --current --missing python npm:@fission-ai/openspec`, and any output or a non-zero exit code lands in the same stop message; for `doctor` it exits with code 1. The message is escaped for JSON before it is printed; the sibling script interpolates it raw.

The `doctor` action is the exception. `run.sh doctor` passes the exit code of `review_gate.py doctor` through with `exec`. When `mise` is missing, it prints the reason and exits with code 1. That way `just doctor` fails exactly when a tool is missing.

`mise.toml` pins `python = "3.14.7"` and `"npm:@fission-ai/openspec" = "1.11.0"`. 1.11.0 is the CLI the generated skills came from, so they keep matching it. `review_gate.py` raises a `MissingToolError` when `openspec` cannot start in `is_planning_complete`, and `main` turns it into the same stop message. A new `doctor` action prints one line per tool and exits non-zero when one is missing. The tools are Python, `git`, `openspec`, and `claude` or `codex`. The root `Justfile` gets `doctor: .agent-hooks/run.sh doctor`, and the root `AGENTS.md` command table gets the row. Every hook in `.claude/settings.json` and `.codex/hooks.json` runs through `run.sh`.

Rejected: searching the macOS Homebrew paths for `mise`, as the sibling script does. Other hosts are on their own here. Rejected: keeping `python3` from `PATH` and checking tools inside the hook. A client started from an IDE may not have the mise shims on `PATH`, so the hook would not start at all and could report nothing. That is how the sibling project lost its full reviews before it moved to `mise exec`. Rejected: upgrading OpenSpec to 1.13.2 while pinning. The upgrade regenerates the `openspec-*` skills, which is out of scope.

## Risks / Trade-offs

- [The shared test fixtures are down, so every `just test-package` run fails] → The turn is blocked with the connection error in the output. The testing skill tells the agent to report it to the user instead of starting the fixtures through `just test`. "Several agents, one set of containers" already says the developer starts the first full run.
- [The turn that checks the last task takes up to 30 more minutes] → Only the commands the change names run, and only on that turn and its repeats.
- [A hook test run claims a test slot database while an agent runs its own package tests] → Same contention as two agents running package tests today. Slots are claimed with advisory locks and wait up to 60 seconds (`backend/internal/config/config.go:46-53`).
- [`run.sh` stops the agent on every turn when mise is broken or a pinned tool is missing] → Intended: a silent skip is worse. The stop message names `mise install` and `just doctor`.
- [A reviewer asks for the section in a change without tests] → The proposal prompt limits the request to tasks that add or change tests.

## Migration Plan

Existing active changes have no "Change verification" section, so the hook runs no tests for them, as today. OpenSpec 1.11.0 is not installed in mise on this machine, so every worktree needs `mise install` before its first turn after this change lands; until then each hook stops the agent with the message that names it. A developer without `mise` on `PATH` or in `$HOME/.local/bin` gets the same stop message. Rollback is a revert of the change. The old hook commands in the client configs come back with it.
