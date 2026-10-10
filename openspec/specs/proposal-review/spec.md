# proposal-review Specification

## Purpose
Reviews a finished OpenSpec proposal as a whole before implementation, so that gaps in requirements, contradictions between artifacts and needless complexity are found before code is written against them.

## Requirements

### Requirement: The full proposal review starts once the proposal is written
The hook that runs when the agent finishes a turn, in Claude Code and in Codex, SHALL start the full proposal review of an active change when all of these hold: the change has every artifact its schema requires, its task list has tasks and none is checked, the change's files changed during this turn, and their state differs from the last state the full proposal review accepted for this change. The turn SHALL be recognized by the change's files, not by the request text: whether the agent started through a command, a skill or a plain request MUST NOT matter. The accepted state SHALL be stored per change, not per session. Archived changes are not reviewed.

#### Scenario: The agent writes every artifact after a plain request
- **WHEN** the user writes "write a spec for this", and the agent creates the proposal, design, delta specs and tasks and finishes the turn
- **THEN** the hook starts the full proposal review of that change

#### Scenario: Artifacts are not finished
- **WHEN** a change has a proposal but no tasks, and the agent finishes the turn
- **THEN** the full proposal review does not start and the quick checks run

#### Scenario: Implementation has started
- **WHEN** at least one task of the change is checked, and the agent edits a delta spec
- **THEN** the full proposal review does not start

#### Scenario: New session without edits
- **WHEN** the full proposal review accepted a proposal in one session, and in a new session the agent finishes a turn without touching the change's files
- **THEN** the full proposal review does not start

#### Scenario: Edit after acceptance
- **WHEN** after an accepted review the agent edits the design of a change whose tasks are not started
- **THEN** the full proposal review starts again

### Requirement: OpenSpec validation runs first
Before starting reviewers the hook SHALL run strict OpenSpec validation of the change. When validation fails, the hook SHALL block the turn and pass the agent the end of its output. Reviewers MUST NOT start in that case.

#### Scenario: A requirement without a scenario
- **WHEN** a delta spec has a requirement without a scenario and validation fails
- **THEN** the turn is blocked, the agent gets the validation output, and no reviewer starts

### Requirement: What the full proposal review checks
The full proposal review SHALL run several read-only reviewers in parallel, on the same models as the full implementation review: one per delta spec of the change, one for the proposal and design, and one for the task list.

The delta spec reviewer SHALL check that requirements are complete and testable, do not contradict the main specs, the proposal or the design, and that claims about existing code are true. The proposal and design reviewer SHALL look for a noticeably simpler way to reach the same result, following the repository rule that the best code is unwritten, check the design decisions against the module architecture rules and accepted ADRs, and check that decisions agree with each other. The task reviewer SHALL check that a task with a test covers every scenario, that every task has a verification, and that the task order is feasible.

#### Scenario: A change with two delta specs
- **WHEN** the full proposal review starts for a change with two delta specs
- **THEN** the hook starts four reviewers in parallel: one per delta spec, one for proposal and design, and one for tasks

#### Scenario: The task reviewer's role
- **WHEN** the hook starts the task reviewer
- **THEN** its prompt names the role and requires checking that a task with a test covers every scenario, every task has a verification, and the task order is feasible

### Requirement: The agent resolves findings itself, except cutting the feature
A reviewer SHALL return findings, each with a category, a path and a description. The categories are `gap` (a requirement is incomplete or a case is missing), `mismatch` (artifacts contradict each other, the main specs, the code or the architecture rules), `tasks` (a scenario without a task with a test, a task without verification, an infeasible order), `simplify` (the same result for the user, noticeably simpler), `scope` (an option that cuts the feature) and `minor`. The hook SHALL block the turn when there is a `gap`, `mismatch`, `tasks` or `simplify` finding, and require the agent to resolve each one without asking the user. `scope` and `minor` findings MUST NOT block the turn. The agent MUST NOT apply a `scope` finding itself: cutting the task is the user's decision.

#### Scenario: A simplification without loss
- **WHEN** a reviewer finds that a field in an existing table replaces a new table, and the user loses nothing
- **THEN** the turn is blocked, and the agent gets the finding with the instruction to resolve it without asking the user

#### Scenario: A proposal to cut the feature
- **WHEN** a reviewer returns only a `scope` finding saying the settings screen can be dropped
- **THEN** the hook does not require a fix and once asks the agent for a summary that shows this option to the user

### Requirement: Repeat and summary of the full proposal review
After a blocked review the next run SHALL check the previous findings and the change's files that changed after that review, not the whole proposal again. There SHALL be at most two repeats per user request, after which the hook SHALL stop the agent and show the remaining findings. When the review accepts the proposal after at least one block in this request, or returns `scope` findings, the hook SHALL block the turn once and ask the agent to write the user a summary of the whole proposal: what is proposed, what each review round found and how it was resolved, and separately each option to cut the feature with what the user loses and saves. The next turn without edits to the change's files MUST NOT be blocked.

#### Scenario: Accepted after fixes
- **WHEN** a repeat review accepts a proposal the previous review blocked
- **THEN** the hook once asks the agent for a summary of the whole proposal, and the next turn without edits is not blocked

#### Scenario: Accepted at once without cuts
- **WHEN** the first review returns only `minor` findings
- **THEN** the turn is not blocked and the user sees the findings

#### Scenario: Repeat limit
- **WHEN** the third proposal review in a row within one user request blocks the turn again
- **THEN** the hook stops the agent and shows the remaining findings

### Requirement: Quick checks yield to the full proposal review
When only files of changes that have every artifact and no checked task changed during the turn, the `necessity`, `architecture`, `diff` and `naming` checks SHALL skip the turn. This way the quick checks duplicate neither the review turn nor the summary turn after acceptance. When other files changed too, the quick checks SHALL run as usual.

#### Scenario: A turn with only artifacts
- **WHEN** the agent finished every artifact of a change and changed nothing else
- **THEN** the quick checks skip the turn and only the full proposal review runs

#### Scenario: An unfinished proposal
- **WHEN** the agent wrote only the proposal and finishes the turn with a question to the user
- **THEN** the full proposal review does not start, and the quick necessity check reviews the proposal
