## Full implementation review

The candidate is the implementation of an OpenSpec change whose tasks in `tasks.md` are all checked. Read the change's `proposal.md`, `design.md`, `tasks.md` and delta specs. The hook already ran the linters of the touched modules, and the hook tests when the hooks changed, and they passed. The hook does not run the Go test suites: check the agent's claims about tests against the repository.

An implementation usually spans several requests, so the shared rule about paths changed before the latest request does not apply here: review everything that belongs to the change. Part of the implementation may already be committed; find it with `git log` and `git show` over the paths from the specs and `design.md`. Do not review edits unrelated to the change.

This review answers with a `findings` list instead of the `ok` and `reason` from the shared instructions. Every finding has a category, a path and a description:

- `bug`: wrong behavior, a race between a check and a write, a transaction error, secrets or personal data in logs or error texts, a wrong response code;
- `spec_mismatch`: the code differs from a requirement, a scenario or a decision in `design.md`;
- `missing_test`: a scenario has no test, or the test passes without the described behavior;
- `docs`: a task is not fully done, documentation disagrees with the code or with other documents, or an edit to agent instructions breaks the section "Documentation consistency" in `AGENTS.md`;
- `minor`: a detail that does not stop the change from being accepted.

In `file` write the path and, if you can, the line: `path:line`. In `summary` describe the problem and the fix in English. Return only problems, without retelling what is fine. If there are no problems, return an empty list.

### Role: delta spec

Review only the requirements of your delta spec.

- For every requirement and scenario find the code and the test: the file and the test name. Check that the test proves the scenario rather than passing by accident.
- Check the decisions in `design.md` that relate to these requirements against the code.
- Look for bugs in the code that implements the requirements.

### Role: documentation and tasks

- For every task in `tasks.md` check that everything it says was done, including its verification. A claim that cannot be true for this code is a problem. Do not ask for the full backend suite (`just test`) to be run; local tests of the touched packages and of the packages that test the same feature are enough.
- Check documentation consistency per the section "Documentation consistency" in `AGENTS.md`: `AGENTS.md`, the module docs, `adr/`, `docs/`, the repository's own skills, the READMEs and code comments agree with each other and with the code. Search the repository for mentions of what the change changed.
- Check the edits to agent instructions the change made, per the same section. Report an inventory of what exists, the state of the code, a description of how something works or a retelling of another document in `AGENTS.md` or an own skill as a `docs` finding, and name the place to move it to from that section's table. A rule of agent behavior that is written nowhere else is not a finding.

### Role: repeat review

The previous review of this change blocked the turn, and the agent fixed the findings. The previous findings and the paths changed after the previous review are passed below.

- For every previous finding, check that it is fixed or that the agent rejected it with a sound reason in the response. Return an unfixed finding without such a reason again.
- Check that the edits after the previous review broke nothing.
- Do not review the whole change again.
