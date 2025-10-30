### Prerequisites

1. Read docs in /docs folder, README.md in /backend and /frontend folders
2. Run both backend and frontend following the instructions in their respective README.md files (for development)
3. Read this file till the end

### How to create a pull request?

We use gitflow approach.

1. Create a new branch from main
2. Make changes
3. Create a pull request to main
4. Wait for review
5. Merge pull request

Commits should be named in the following format depending on the type of change:

- `FEATURE (area): What was done`
- `FIX (area): What was fixed`
- `REFACTOR (area): What was refactored`

To see examples, look at commit history in main branch.

Branches should be named in the following format:

- `feature/what_was_done`
- `fix/what_was_fixed`
- `refactor/what_was_refactored`

Example:

- `feature/add_support_of_kubernetes_helm`
- `fix/make_healthcheck_optional`
- `refactor/refactor_navbar`

Before any commit, make sure:

1. You created critical tests for your changes
2. `make lint` is passing (for backend) and `npm run lint` is passing (for frontend)
3. All tests are passing
4. Project is building successfully
5. All your commits should be squashed into one commit with proper message (or to meaningful parts)
6. Code do really refactored and production ready
7. You have one single PR per one feature (at least, if features not connected)

### Automated Versioning

This project uses automated versioning based on commit messages:

- **FEATURE (area)**: Creates a **minor** version bump (e.g., 1.0.0 → 1.1.0)
- **FIX (area)**: Creates a **patch** version bump (e.g., 1.0.0 → 1.0.1)
- **REFACTOR (area)**: Creates a **patch** version bump (e.g., 1.0.0 → 1.0.1)
- **BREAKING CHANGE**: Creates a **major** version bump (e.g., 1.0.0 → 2.0.0)

The system automatically:

- Analyzes commits since the last release
- Determines the appropriate version bump
- Generates a changelog grouped by area (frontend/backend/etc.)
- Creates GitHub releases with detailed release notes
- Updates package.json version numbers

To skip automated release (for documentation updates, etc.), add `[skip-release]` to your commit message.

### Docs

If you need to add some explanation, do it in appropriate place in the code. Or in the /docs folder if it is something general. For charts, use Mermaid.

### Priorities

Before taking anything more than a couple of lines of code, please write Rostislav via Telegram (@rostislav_dugin) and confirm priority. It is possible that we already have something in the works, it is not needed or it's not project priority.

Nearsest features:
- add API keys and API actions

Backups flow:

- check Neon backups flow
- check AWS S3 support
- when testing connection with S3 - verify files can be really uploaded
- do not remove old backups on backups disable
- add FTP
- add Dropbox
- add OneDrive
- add NAS
- add Yandex Drive
- think about pg_dumpall / pg_basebackup / WAL backup / incremental backups
- add encryption for backups
- add support of PgBouncer

Notifications flow:

- add Mattermost

Extra:

- add HTTPS for Postgresus
- add simple SQL queries via UI
- add support of Kubernetes Helm
- create pretty website like rybbit.io with demo

Monitoring flow:

- add queries stats (slowest, most frequent, etc. via pg_stat_statements)
- add DB size distribution chart (tables, indexes, etc.)
- add chart of connections (from IPs, apps names, etc.)
