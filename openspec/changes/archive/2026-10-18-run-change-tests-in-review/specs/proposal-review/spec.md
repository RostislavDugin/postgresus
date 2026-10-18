## MODIFIED Requirements

### Requirement: OpenSpec validation runs first
Before starting reviewers the hook SHALL check the format of the change's "Change verification" section by the rules the full implementation review applies to it, without running its commands, and then run strict OpenSpec validation of the change. When the section's format is wrong or validation fails, the hook SHALL block the turn and pass the agent the reason or the end of the validation output. Reviewers MUST NOT start in that case.

#### Scenario: A requirement without a scenario
- **WHEN** a delta spec has a requirement without a scenario and validation fails
- **THEN** the turn is blocked, the agent gets the validation output, and no reviewer starts

#### Scenario: A forbidden command in the verification section
- **WHEN** the change's "Change verification" section contains `just test`
- **THEN** the turn is blocked with the reason and the list of allowed forms, the command does not run, and no reviewer starts

### Requirement: What the full proposal review checks
The full proposal review SHALL run several read-only reviewers in parallel, on the same models as the full implementation review: one per delta spec of the change, one for the proposal and design, and one for the task list.

The delta spec reviewer SHALL check that requirements are complete and testable, do not contradict the main specs, the proposal or the design, and that claims about existing code are true. The proposal and design reviewer SHALL look for a noticeably simpler way to reach the same result, following the repository rule that the best code is unwritten, check the design decisions against the module architecture rules and accepted ADRs, and check that decisions agree with each other. The task reviewer SHALL check that a task with a test covers every scenario, that every task has a verification, and that the task order is feasible. The task reviewer SHALL also check that a change whose tasks add or change tests in a backend package, a verification agent package or the frontend has a "Change verification" section, and that the section runs the tests of every such package the tasks' verifications name. Tests that no allowed command form runs and that a module check already runs, such as the hook tests, need no section.

#### Scenario: A change with two delta specs
- **WHEN** the full proposal review starts for a change with two delta specs
- **THEN** the hook starts four reviewers in parallel: one per delta spec, one for proposal and design, and one for tasks

#### Scenario: The task reviewer's role
- **WHEN** the hook starts the task reviewer
- **THEN** its prompt names the role and requires checking that a task with a test covers every scenario, every task has a verification, the task order is feasible, and the "Change verification" section covers the tested packages

#### Scenario: Tests without a verification section
- **WHEN** the tasks add tests to a backend package and the change has no "Change verification" section
- **THEN** the task reviewer returns a `tasks` finding that asks for the section with that package

#### Scenario: A section that leaves out a package
- **WHEN** a task's verification runs the tests of two backend packages and the "Change verification" section runs only one of them
- **THEN** the task reviewer returns a `tasks` finding that names the missing package

#### Scenario: A change with only hook tests
- **WHEN** the tasks add tests only to the hook tests and the change has no "Change verification" section
- **THEN** the task reviewer returns no finding about the missing section

## ADDED Requirements

### Requirement: The full proposal review ends within the hook timeout
The full proposal review SHALL work within the same kind of time budget as the full implementation review: the budget ends before the hook timeout with a margin, validation runs inside it, and a reviewer starts only when the budget still holds its whole time limit. A reviewer that cannot start for that reason SHALL count as a reviewer that could not run. The hook MUST NOT be killed by the client timeout without a message.

#### Scenario: Too many reviewers for the budget
- **WHEN** a proposal has so many delta specs that a third wave of reviewers would not fit in the remaining budget
- **THEN** the reviewers of that wave do not start, the turn is blocked, and the agent gets the reason that the review time budget ran out
