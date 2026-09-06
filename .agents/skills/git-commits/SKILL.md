---
name: git-commits
description: Write Databasus commit messages and branch names using the repository's release-compatible format. Use when creating, editing or reviewing a commit message or branch name in this repository.
---

# Git commits

Use one of these subject formats:

- `FEATURE (scope): Summary`
- `FIX (scope): Summary`
- `REFACTOR (scope): Summary`

Never use `!` anywhere in the subject. The release workflow treats it as a breaking change and triggers a major version bump.

Keep the body to a short bullet list of what changed. Do not write long narrative paragraphs or hard-wrap lines at 80 characters. For example:

```text
- validate PostgreSQL targets and quote conninfo values
- verify workspace access before applying saved credentials
- isolate local sockets and rotate internal database credentials
- add regression coverage and OpenSpec documentation
```

Keep the reasons and implementation details in OpenSpec. When a change has related OpenSpec files, commit them together with the implementation.

Name branches `feature/<scope>`, `fix/<scope>` or `refactor/<scope>`.

Do not include co-author attribution unless the user requests it directly.
