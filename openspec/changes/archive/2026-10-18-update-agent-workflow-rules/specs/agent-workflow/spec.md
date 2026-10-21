## Purpose

Defines how an agent finishes work in this repository: it refactors the code it touches instead of listing suggestions, reports the result rather than the review history, packs a finished change into one commit on request, keeps the user able to see and stop what it runs, and keeps lasting rules where every agent and person can read them.

## ADDED Requirements

### Requirement: The agent refactors touched code within the change
When code the change touches stands in the way of the change or breaks the rules of the module docs or an accepted ADR, the agent SHALL refactor it while implementing the change, without asking the user and without a separate task for later. The agent's summary MUST NOT list refactoring suggestions for code inside the change. The summary SHALL name only a refactoring that goes beyond the change, together with what it would cost, because such a refactoring needs the user's agreement first.

#### Scenario: Duplicated code in a touched function
- **WHEN** the agent edits a function and finds that it duplicates a helper next to it
- **THEN** the agent replaces the duplicate with the helper in the same change, and its summary lists no refactoring suggestion about it

#### Scenario: A rework beyond the change
- **WHEN** the agent finds that a module the change does not touch needs a larger rework
- **THEN** the agent does not do the rework, and its summary names it with its cost for the user to decide

### Requirement: The user summary describes the result, not the review
After a review hook blocked the turn and the agent fixed or rejected the findings, the summary the user reads SHALL describe the final state of the result: what was done, which decisions were made, which checks passed, what remains and the next step. The summary MUST NOT retell review rounds, the findings or the fixes made for them. After a quick check blocks a finished change, the agent's next reply SHALL be such a summary. After a quick check blocks a plan, the agent SHALL revise the plan and show it again. After a full review blocks the turn, whether a reviewer found problems or a check before the reviewers failed, the agent's next reply SHALL be one short line about its fixes that names every rejected finding with the reason, because the repeat review reads that reply; the summary comes once the full review accepts the result.

#### Scenario: The diff check blocks a turn
- **WHEN** the diff check blocks the turn because a test is missing, and the agent adds the test and finishes the turn again
- **THEN** the agent's reply is a summary of the finished work for the user and does not mention the diff check or its finding

#### Scenario: A quick check blocks a plan in the final message
- **WHEN** in Codex the necessity check blocks a plan the agent wrote in its final message
- **THEN** the block text asks the agent to revise the plan and show it again, not to write a summary of a result

#### Scenario: The full implementation review blocks a turn
- **WHEN** the full implementation review blocks the turn with two findings, and the agent fixes one and rejects the other
- **THEN** the agent's reply is one short line that names the fix and the rejected finding with its reason

#### Scenario: A linter fails before the full review
- **WHEN** the backend linter fails before the full implementation review starts its reviewers, and the agent fixes the lint error
- **THEN** the agent's reply is one short line about the fix

### Requirement: The pack command archives and commits a finished change
When the user asks to pack a change, in whatever language the request is written, and every task of the change is checked, the agent SHALL archive the change with its delta specs synced into the main specs, and commit the implementation, the main specs and the archived change in one commit whose message follows the repository's commit format. The request SHALL count as the user's consent to the sync and to that commit, so the agent does not ask about either again. The agent MUST NOT include changes it did not make within the user's requests, and MUST NOT push. When a task of the change is unchecked, the agent MUST NOT sync, archive or commit anything, and SHALL name the unchecked tasks to the user.

#### Scenario: Pack a finished change
- **WHEN** every task of a change is checked and the user writes "pack"
- **THEN** the main specs contain the change's requirements, the change sits in the archive, and one new commit holds the code, the specs and the archive, with nothing pushed

#### Scenario: Pack an unfinished change
- **WHEN** a change has two unchecked tasks and the user writes "pack"
- **THEN** the main specs, the change directory and the Git history stay as they were, and the agent names the two unchecked tasks

#### Scenario: Another agent's edit in the working tree
- **WHEN** the user asks to pack a change while the working tree also holds an edit the agent did not make
- **THEN** the commit leaves that edit out, and it stays uncommitted in the working tree

### Requirement: The user sees and owns what the agent runs
The agent SHALL run tests, linters and builds as ordinary foreground commands, so the user sees which command runs and that the agent is working. When the agent starts the backend, the frontend or another long-lived process to check its work, it SHALL stop that process before it finishes the task. Long-lived processes the user wants running are started by the user.

#### Scenario: The agent runs package tests
- **WHEN** the agent runs the tests of a backend package
- **THEN** the command runs in the foreground of the agent's console, not as a background task

#### Scenario: The agent started the frontend dev server
- **WHEN** the agent started the frontend dev server to check a page
- **THEN** the dev server is no longer running when the agent hands the task over

### Requirement: Lasting rules live in the repository
The agent MUST NOT write rules, preferences or facts to its local memory outside the repository. A rule or preference that must outlive the session SHALL go, with the user's consent, into an `AGENTS.md` file or a repository skill, where every agent and person working in the repository reads it.

#### Scenario: The user states a lasting preference
- **WHEN** the user says that every future change should do something a certain way
- **THEN** the agent proposes the edit to `AGENTS.md` or a skill and writes nothing to its local memory

### Requirement: Every repository skill is available to each agent client
Each of the repository's own skills, that is every skill except the generated `openspec-*` skills, SHALL be loadable as a skill both in Claude Code and in Codex, from one source file, so the two clients never read different versions of it.

#### Scenario: The commit skill in Claude Code
- **WHEN** Claude Code lists the skills it can load in this repository
- **THEN** the list contains the Git commits skill, and it reads the same file Codex reads
