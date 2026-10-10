## Necessity review

The candidate is a plan or an OpenSpec proposal the user has not seen yet. Whether the solution is correct is not checked here. Check whether the task should be done in this form. The standard is the section "The best code is unwritten" in `AGENTS.md`.

If the candidate is an OpenSpec proposal, read the change's artifacts under `openspec/changes`, first of all `proposal.md` and `design.md`.

- Product: which user need the task covers. Check whether an existing feature, a manual process or an external service already covers it, and whether a much smaller option covers the same need.
- Code: search the repository and its dependencies for something that already does what is needed, or nearly. Check whether a small refactoring with reuse replaces the new code, table, endpoint, package or dependency.
- If such an option exists and the candidate silently bypasses it, return `ok: false`. In the reason name the option, explain what makes it simpler and what is lost with it, and point to where it lives in the repository. The candidate passes if it shows the user this option with its pros and cons, or explains why the option does not fit.
- Do not demand dropping the task: the user decides. Do not suggest again an option the user already rejected in the request.
- Count only a noticeably simpler option: less code, fewer entities, screens or maintenance. The option must be verified against the repository. Small simplifications and guesses do not count.
- Plan correctness, prose and architecture are checked by other reviewers.
