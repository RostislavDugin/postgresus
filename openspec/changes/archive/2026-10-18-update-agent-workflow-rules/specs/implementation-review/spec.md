## MODIFIED Requirements

### Requirement: Which findings block the turn
A reviewer SHALL return findings, each with a category (`bug`, `spec_mismatch`, `missing_test`, `docs`, `minor`), a path and a description. The hook SHALL block the turn when at least one finding is not `minor`, and pass the agent every finding with `minor` marked optional. When the turn is not blocked, the user SHALL see the `minor` findings. The block text SHALL require the agent to resolve every finding without asking the user: fix the code or, when the implementation is right, the change's delta spec. The block text SHALL also require the agent to finish the turn again with one short line about its fixes that names every rejected finding with the reason, and to leave the summary for the user until the review accepts the change. With no findings or only `minor` ones, the review SHALL count as accepted. When the review accepts the change after at least one block within this user request, the hook SHALL block the turn once and ask the agent to write the user a summary of the implementation as if there had been no review: what was done for the change's tasks, which decisions were made, which checks passed, which `minor` findings were left, and the next step, without review rounds, findings or the fixes made for them. The hook SHALL pass the `minor` findings in that request.

#### Scenario: Only minor findings
- **WHEN** every finding is `minor`, and no earlier review in this request blocked the turn
- **THEN** the state is stored as accepted and the user sees the findings

#### Scenario: Accepted after fixes
- **WHEN** a repeat review accepts a change the previous review blocked
- **THEN** the hook blocks the turn once and asks the agent for a summary of the implementation that leaves out review rounds and the fixes made for them, and the next turn without edits is not blocked

#### Scenario: A bug is found
- **WHEN** a reviewer returns a `bug` finding
- **THEN** the turn is blocked, and the agent gets every finding with the instruction to resolve them itself and to finish the turn with one short line about the fixes and the rejected findings

#### Scenario: The reviewer cannot run
- **WHEN** the reviewer process crashes or times out
- **THEN** the review is not accepted and the agent gets the failure reason
