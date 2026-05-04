# Databasus — Agent Rules and Guidelines

This document contains project-wide coding standards and best practices for Databasus.
This is NOT a strict set of rules — it is a set of recommendations to help write better, more consistent code.

Per-folder rules live next to the code they govern:

- [`backend/CLAUDE.md`](backend/CLAUDE.md) — Go + Gin + GORM + PostgreSQL backend (controllers, migrations, CRUD, DI, testing, logging)
- [`agent/CLAUDE.md`](agent/CLAUDE.md) — Go agent CLI (no HTTP server, no schema; shares Go conventions with the backend)
- [`frontend/CLAUDE.md`](frontend/CLAUDE.md) — React 19 + TypeScript + Vite + Ant Design + Tailwind

This root file holds the engineering philosophy that applies everywhere.

---

## Language in code

**English only in code, comments, identifiers, log messages, API strings, test assertions, and commit messages.** No other language inside `backend/`, `agent/`, or `frontend/src/` — even for user-facing fallback copy or error messages.

---

## Engineering philosophy

**Think like a skeptical senior engineer and code reviewer. Don't just do what was asked — also think about what should have been asked. Catch real issues, not theoretical ones.**

### Task tiers (scale your response to the task)

- **Trivial** (typos, formatting, single-field adds): apply directly. Steps 5 only.
- **Standard** (CRUD, typical features): steps 1, 5.
- **Complex** (architecture, security, performance-critical): all steps.
- **Unclear** (ambiguous requirements): steps 1 and 4 are mandatory.

### Steps for non-trivial tasks

1. **Restate the objective**, list explicit + inferred assumptions, flag shaky ones.
2. **Propose solutions** — for complex tasks, 2–3 approaches including a simpler baseline; recommend one with tradeoffs (complexity, maintainability, performance, extensibility).
3. **Identify risks** — edge cases, security/privacy, performance, operational concerns (deployment, observability, rollback). Before finalizing, ask "what could go wrong?" and patch.
4. **Handle ambiguity** — pick a reasonable default, label it, note what changes under alternative assumptions.
5. **Deliver quality** — correct, testable, maintainable code with minimal tests/validation. Prefer controller tests over unit tests.
6. **Fix root causes, not symptoms** — ask "why did this happen?" and address the underlying issue.

### After each run: suggest refactorings

Reread the diff with fresh eyes and **list** (don't silently apply) refactor suggestions: unclear names, duplication, dead code, deep nesting, misplaced responsibilities, leaky abstractions. Keep suggestions concrete (file + lines), behavior-preserving, and scoped to the current change. If the diff is already clean, say so in one line.

### Naming

Name variables and functions for **intent**, not mechanism. Naming is the biggest readability lever — avoid generic names like `data`, `handle`, `process`.

Booleans take an `is` / `can` / `has` / `should` prefix (`isAllowed`, `canAccess`, `hasItems`, `shouldRetry`) — never bare nouns/verbs like `allowed` or `touches`.

### Linting and formatting

After each change run linting and formatting depending on folder you are working it.
- backend and agent has `make lint` commands
- frontend has `pnpm lint` and `pnpm format` commands

### No "how it was" comments, no unrequested backward compatibility

Don't write comments that explain previous behavior ("used to be X", "was renamed from Y", "kept for legacy callers"). Code shows the current state; history lives in git.

Don't preserve backward compatibility unless the user asks for it. No deprecation shims, no aliases, no fallbacks for the old shape. When planning a change that would break existing callers, schemas, configs, or APIs, call out the break explicitly in the plan. If the user approves it, delete the old code outright — do not leave a transition layer behind.