## Architecture review

The standard for this review is the architecture rules of the module docs and the accepted ADRs. Read the sections that apply to the paths in scope:

- `backend/AGENTS.md`: "File organization", "Controllers", "Dependency injection (DI)", "Background services", "Stored files", "Migrations";
- `frontend/AGENTS.md`: "FSD (Feature-Sliced Design)";
- `agent/verification/AGENTS.md`: "File organization", "Background services";
- the ADRs in `adr/` that cover the area of the change.

If the candidate is a plan, check that the proposed structure follows these rules. If the candidate is a finished change, use `git status`, the full `git diff HEAD` and the untracked files, and check the changed code.

Check only architecture:

- backend feature boundaries: a feature reaches another feature through its service, never through its repository, and dependencies are wired through `SetupDependencies()`;
- layers inside a backend feature: controllers call services, services hold the logic, repositories hold the database access, and files sit in the roles the module doc names;
- schema changes go through migrations in the form the module doc requires;
- frontend FSD layers and slices: import direction, slice placement, no cross-imports between slices of one layer, the public API of a slice;
- a decision that contradicts an accepted ADR without a new ADR that supersedes it.

If the change edits an ADR or an architecture section of a module doc, check that the edit agrees with the code and with the latest user request.

Do not check names, style, prose or general correctness. Other reviewers do that.
