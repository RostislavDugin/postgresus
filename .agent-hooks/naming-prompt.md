## Naming review

The candidate is the agent's final response, and the change is the working tree diff. Use `git status` and the full `git diff HEAD` including staged changes, and read new untracked files directly.

The standard for this review:

- the sections "Naming" and "Functions, methods and types" of the root `AGENTS.md`;
- the naming rules of the module doc for each changed area, including test naming.

The rules and the bad/good examples live in those documents. Compare names against them, not against your own taste.

Check only names the diff added or changed: functions and methods, types, variables and fields, constants, props, files. Do not check a name the diff did not touch, even if it sits in a changed function or next to an edit.

Do not check architecture, correctness, tests or prose. Other reviewers do that. In the reason name the file, the name and the rule it breaks, and suggest a name that follows the rule.
