# Databasus — Agent Rules and Guidelines

This document contains project-wide coding standards and best practices for Databasus.
This is NOT a strict set of rules — it is a set of recommendations to help write better, more consistent code.

Per-folder rules live next to the code they govern:

- [`backend/CLAUDE.md`](backend/CLAUDE.md) — Go + Gin + GORM + PostgreSQL backend (controllers, migrations, CRUD, DI, testing, logging)
- [`agent/verification/CLAUDE.md`](agent/verification/CLAUDE.md) — Go verification agent CLI (self-update + capacity heartbeat; restore logic deferred)
- [`frontend/CLAUDE.md`](frontend/CLAUDE.md) — React 19 + TypeScript + Vite + Ant Design + Tailwind

This root file holds the engineering philosophy that applies everywhere.

---

## Development environment

Work happens inside the repo's [Dev Container](.devcontainer/devcontainer.json). The container ships Go, Node.js + pnpm, Docker-in-Docker, linters and matching VS Code extensions, so the toolchain is identical for every contributor. Ports `4005` (backend) and `5173` (Vite) are forwarded automatically. Don't install or rely on host-level SDKs — run `make`, `pnpm` and `docker` commands from inside the container.

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

Name variables and functions for **intent**, not mechanism. Naming is the biggest readability lever — avoid generic placeholders (`data`, `handle`, `process`, `tmp`, `helper`, `manager`), type-suffix noise (`nameStr`, `agentList`, `tokenObj`), and mechanism-flavored names (`tickNow`, `hbResp`, `dataObj`).

Booleans take an `is` / `can` / `has` / `should` prefix (`isAllowed`, `canAccess`, `hasItems`, `shouldRetry`) — never bare nouns/verbs like `allowed` or `touches`.

State that holds "the entity currently being X-ed" must include the entity: `createdAgent`, not `created`; `listedAgents`, not `listed`; `deletingAgentId`, not `deletingId`. Pair related state explicitly: `revealedToken` + `revealedTokenAgentName`, not `tokenAgentName`. Loop variables get the domain word — `for _, agent := range listedAgents` (Go) — single letters only inside trivial one-line lambdas (`.map((a) => a.id)`).

Test-mechanism names (`got`, `want`, `expected`) are not acceptable — use the domain noun: `persistedAgent`, `firstRotation`, `secondRotation`. Match domain language across the wire — if the API says `agent`, don't rename to `worker` or `node` in client code.

Idiomatic short names stay where the convention is well-established: Go receivers (`s *AgentService`, `r *AgentRepository`, `ctx *gin.Context`) and JS/TS error catches (`} catch (e) {`). When no good name exists, the abstraction is wrong — extract or rename it, don't reach for `helper` / `tmp` / `manager`.

Examples:

```go
// bad
var hbResp HeartbeatResponse
var got *Agent
for _, a := range listed { ... }

// good
var heartbeatResponse HeartbeatResponse
var persistedAgent *Agent
for _, agent := range listedAgents { ... }
```

```ts
// bad
const [tickNow, setTickNow] = useState(Date.now());
const [deletingId, setDeletingId] = useState<string | null>(null);
const [tokenAgentName, setTokenAgentName] = useState('');

// good
const [currentTimeMs, setCurrentTimeMs] = useState(Date.now());
const [deletingAgentId, setDeletingAgentId] = useState<string | null>(null);
const [revealedTokenAgentName, setRevealedTokenAgentName] = useState('');
```

### Functions, methods and types

The intent rule applies to callables and types too — this is where it's violated most:

- **A name that needs a "what" comment is a naming bug.** Doc comments are for *why*, ordering, and hidden constraints — never to restate behavior. If you wrote `// Foo does X` above `func Foo`, rename `Foo` until that comment is redundant, then delete it. Keep only the sentence a name *cannot* carry (e.g. "must be called before the container exists").
- **Predicate methods read as a question**, same as boolean vars: `IsAborted(id)`, not `AbortedContains(id)`; `HasCapacity()`, not `CapacityCheck()`.
- **The name states everything the unit does.** A function that records *and* cancels is `recordAndCancelAborts` — or it's two functions. A name that hides a second effect is a lie.
- **Getters take a `Get` prefix and name the entity.** `GetRunningVerificationIDs()`, not `Active()`, `RunningVerificationIDs()`, or `DiskUsageBytes()`. This is a deliberate house style: prefer the explicit `Get` *even though vanilla Go idiom omits it* — it keeps accessors visually distinct from actions and matches the backend (`GetAuditLogService`, `GetGlobalAuditLogs`). Constructors stay `New...`; a getter that returns a bool is still a predicate, so it stays `Is/Has...`, not `Get...`.
- **Long positional parameter lists become a struct.** ~4+ params, or two same-typed params adjacent → a named parameter/DTO struct (the input counterpart to the result type, e.g. `spawnPlan` → `SpawnSpec`). Kills call-site ordering bugs and the comment explaining argument order.
- **Type names: avoid generic *and* avoid package stutter.** Not `Manager` / `Provisioner` / `Handler` (generic), not `container.ContainerManager` (stutters — `revive` fails CI). If the only honest name stutters, put the noun on the *variable*, not the type: keep the type `container.Manager`, name the variable `containerManager`. If no precise type name exists, the abstraction is wrong — split it.

```go
// bad — comment restates the name; name hides the second effect; predicate isn't a question
// applyAborts records the abort set and cancels each registered job.
func (h *Heartbeater) applyAborts(ids []uuid.UUID) { ... }
func (h *Heartbeater) AbortedContains(id uuid.UUID) bool { ... }

// good — the names carry it; no comment needed
func (h *Heartbeater) recordAndCancelAborts(ids []uuid.UUID) { ... }
func (h *Heartbeater) IsAborted(id uuid.UUID) bool { ... }
```

### Linting and formatting

After each change run linting and formatting depending on folder you are working it.
- backend and agent has `make lint` commands
- frontend has `pnpm lint` and `pnpm format` commands

### Comments are a last resort

The default is **no comment**. Before writing one, reach for a clearer name or a smaller function first — self-explanatory code beats commented code every time. A comment is justified only when it carries something the code cannot: a *why* (business rule, hidden cross-system constraint, non-obvious algorithm or optimisation, ADR reference). Never write a comment that restates *what* the code does or narrates the obvious — if a `// Foo does X` comment sits above the code, that's a naming bug: rename until the comment is redundant, then delete it. This applies everywhere, tests included — a well-named test needs no header comment explaining what it checks.

### No "how it was" comments, no unrequested backward compatibility

Don't write comments that explain previous behavior ("used to be X", "was renamed from Y", "kept for legacy callers"). Code shows the current state; history lives in git.

Don't preserve backward compatibility unless the user asks for it. No deprecation shims, no aliases, no fallbacks for the old shape. When planning a change that would break existing callers, schemas, configs, or APIs, call out the break explicitly in the plan. If the user approves it, delete the old code outright — do not leave a transition layer behind.

### Security

Databasus handles sensitive data, so security is a layered defence. CodeQL, CodeRabbit, Codex Security, Trivy, Dependabot, gitleaks and semgrep all run on every PR. When working in the repo:

- Don't disable or weaken security checks to make a build pass. For genuine false positives, suppress explicitly with a documented reason (e.g. `.trivyignore`) and explain why in the PR.
- Pin every new GitHub Action to a full commit SHA with a `# vX.Y.Z` tag comment. No floating tags like `@v4` or `@main`.
- Workflows default to top-level `permissions: contents: read`; elevate per-job only when justified.
- Keep Dockerfiles free of secrets, floating base-image tags and unjustified root. If root-at-start is required (PUID/PGID remap, volume chown, initdb), drop privileges with `gosu` before `exec`-ing the app.
- Never log secrets, tokens or credentials. Redact at the logger layer, not at call sites.

The README's `🛡️ Security & reliability engineering` section is the public-facing version of these practices — keep both consistent if substance changes.