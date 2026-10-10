Hook tests run from the repository root with `python3 -m unittest discover -s .agent-hooks -p '*_test.py'`.

## 1. Gate script and prompts

- [x] 1.1 Replace `.agent-hooks/review_gate.py` with the sibling project's gate script, translated to English, with the communication skill path, the code directories `backend/`, `agent/`, `frontend/src/`, `website/` for the naming gate, the models and effort from design.md, and the implementation review checks built from the changed paths. Verify with `python3 -m py_compile .agent-hooks/review_gate.py`.
- [x] 1.2 Add `common-prompt.md` with the path-to-doc table from `.claude/agents/reviewer.md`, and `plan-prompt.md`, `necessity-prompt.md`, `architecture-prompt.md`, `diff-prompt.md`, `naming-prompt.md`, `proposal-prompt.md`, `implementation-prompt.md`, each citing this repository's docs. Add `proposal-result.schema.json` and `implementation-result.schema.json`. Delete `review-prompt.md`. Verify with `grep -rn "docs/architecture.md\|glossary\|just \|review-prompt" .agent-hooks` returning nothing.
- [x] 1.3 Port `review_gate_test.py`, with `make`, `pnpm`, `npm` and `golangci-lint` stubs instead of `just`, plus tests that a backend change runs only the backend linter, a website change runs lint and build, a failing linter blocks the turn with the end of its output, an `.agent-hooks` change runs the hook tests, and an `agent/` code change starts the naming gate. Verify with the hook test command.

## 2. Hook wiring

- [x] 2.1 Rewrite `.claude/settings.json` and `.codex/hooks.json` with every gate, the sibling project's hook timeouts and the worktree-safe script path. Verify with `python3 -m json.tool` on both files and with `echo '{"hook_event_name":"Stop","session_id":"manual-check","cwd":"'$PWD'"}' | python3 .agent-hooks/review_gate.py naming --client claude` printing `{}`.

## 3. Agent instructions

- [x] 3.1 Delete `.claude/agents/reviewer.md`.
- [x] 3.2 In the root `AGENTS.md`, replace "Mandatory compliance review" with "Review hooks", add the sections "OpenSpec", "Discussing a task in explore", "The best code is unwritten", "Documentation consistency", "Refactoring within a feature" and "Other agents' changes in the working tree", and remove the inventories, port numbers, command internals and the list of security scanners. Verify with `grep -n "reviewer subagent\|reviewer.md\|Opus\|4005\|CodeQL" AGENTS.md` returning nothing.
- [x] 3.3 Replace the reviewer subagent references in `openspec/config.yaml` with the review hooks and add the architecture grounding rule for design. Verify with `grep -n "reviewer" openspec/config.yaml` returning nothing and `openspec validate port-review-gates --strict` passing.
