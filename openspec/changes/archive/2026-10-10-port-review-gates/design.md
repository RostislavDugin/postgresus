## Context

See proposal.md for the motivation and the three delta specs for the behavior contract.

Today the hook script has four actions: `snapshot`, `plan-gate`, `claude` and `codex` (`.agent-hooks/review_gate.py:373-391`). Every review runs `claude-opus-5` at low effort with one prompt, `.agent-hooks/review-prompt.md`, which delegates the path-to-doc mapping and the most-violated rules to `.claude/agents/reviewer.md`. Codex turns are reviewed by Claude too (`review_gate.py:210-256`). The state lives under the git directory in `agent-review-hooks/<session>.json` (`review_gate.py:112-118`), so worktrees keep separate state.

The sibling project's gate script is the source of this change. Its structure carries over unchanged: a request snapshot with workspace fingerprint and per-file hashes, one state file per gate, per-change state for the full reviews, and quick gates that skip a turn which only wrote a finished proposal.

Constraints from the root [`AGENTS.md`](../../../AGENTS.md): English in every committed file, no backward compatibility the user did not ask for, and several agents sharing one working tree and one set of test containers.

## Goals / Non-Goals

**Goals:**

- Each gate reviews one concern with a prompt that names its standard.
- A finished proposal and a finished implementation are each reviewed once as a whole, in parallel by role, with a repeat pass that checks only the previous findings.
- The hook never modifies the working tree.

**Non-Goals:**

- Faster reviews. The full reviews add minutes to the turn that finishes a change, by design.
- A hook-level test run.

## Decisions

### Port the sibling script whole and adapt only project facts

The script is taken as is and translated. The adapted parts are the communication skill path, the code directories that trigger the naming gate, and the check commands of the implementation review.

Rejected: extending the current script gate by gate. The sibling script already has tests for the interactions between gates, such as quick gates yielding to the proposal review and the necessity gate ignoring fixes to a change under review. Re-deriving those would repeat the work and its bugs.

### Models and reasoning effort match the sibling project exactly

Quick gates run `opus` at `low` effort on Claude and `gpt-6.1-sol` at `low` on Codex. Both full reviews run `opus` at `medium` on Claude and `gpt-6.1-sol` at `high` on Codex. The current pinned `claude-opus-5` and `--max-turns 30` go away.

Rejected: keeping Claude as the reviewer for Codex turns. The user chose the sibling project's setup, where each client reviews its own turns.

### The implementation review runs read-only checks of the touched modules

The check list is built from the paths changed relative to `HEAD`: `golangci-lint run` in `backend/` and in `agent/verification/`, `pnpm --dir frontend lint`, `npm --prefix website run lint` and `npm --prefix website run build`, and the hook unit tests through the hook's own Python interpreter when `.agent-hooks/` changed.

Rejected: `make lint`, because it starts with `golangci-lint fmt` and would rewrite files while a parallel agent works in the same tree. Rejected: `make test`, because it removes every test container and recreates the slot databases that other agents use. Rejected: one fixed command list as in the sibling project, because this repository has no root command that lints every module.

The hook tests run through `sys.executable` rather than `python3` from `PATH`. Under test, `PATH` holds stubs, and a stubbed `python3` would not be the interpreter the test expects.

### The architecture gate reads the module docs and ADRs

This repository has no single architecture document. The architecture prompt names the architectural sections of the module docs: file organization, controllers, dependency injection, background services, stored files and migrations for the backend, FSD for the frontend, file organization and background services for the verification agent. It adds the accepted ADRs in `adr/`.

Rejected: writing a new architecture document first. That is a separate piece of documentation work with its own review, and the module docs already state the rules.

### The reviewer subagent is deleted

The hooks run the same audit on every turn. The subagent duplicated the rules in a third place, and its path-to-doc table moves into `common-prompt.md`.

Rejected: keeping it for sessions with hooks disabled. The user chose to delete it.

### The root `AGENTS.md` is cleaned in the same change

The new `agent-instructions` requirement forbids inventories, statements about the current state of code and descriptions of tooling internals in agent instructions. The root `AGENTS.md` breaks it in a few places, starting with the hook paragraph this change rewrites anyway. Leaving them would make the diff gate reject this change.

## Risks / Trade-offs

- [Full reviews take minutes on the turn that finishes a change] → They run once per proposal and once per implementation, and an accepted state is remembered per change, so a new session does not repeat them.
- [The Codex reviewer depends on the `codex` CLI and its model name] → A reviewer that cannot run never accepts the turn, so a broken setup is visible at once.
- [Linters for the touched modules are slow on a large diff] → They run once per full review, not on every turn.
