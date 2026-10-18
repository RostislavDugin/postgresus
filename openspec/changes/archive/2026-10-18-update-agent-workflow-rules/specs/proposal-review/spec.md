## MODIFIED Requirements

### Requirement: Repeat and summary of the full proposal review
After a blocked review the next run SHALL check the previous findings and the change's files that changed after that review, not the whole proposal again. The block text SHALL require the agent to finish the turn again with one short line about its fixes that names every rejected finding with the reason, and to leave the summary for the user until the review accepts the proposal. There SHALL be at most two repeats per user request, after which the hook SHALL stop the agent and show the remaining findings. When the review accepts the proposal after at least one block in this request, or returns `scope` findings, the hook SHALL block the turn once and ask the agent to write the user a summary of the proposal as if there had been no review: what is proposed and which decisions were made, without review rounds, findings or the fixes made for them, and separately each option to cut the feature with what the user loses and saves. The next turn without edits to the change's files MUST NOT be blocked.

#### Scenario: A gap blocks the turn
- **WHEN** the proposal review blocks the turn with a `gap` finding
- **THEN** the block text asks the agent to resolve the finding without asking the user and to finish the turn with one short line about the fixes that names each rejected finding with its reason, and not to write the user summary yet

#### Scenario: Accepted after fixes
- **WHEN** a repeat review accepts a proposal the previous review blocked
- **THEN** the hook once asks the agent for a summary of the proposal that leaves out review rounds and the fixes made for them, and the next turn without edits is not blocked

#### Scenario: Accepted at once without cuts
- **WHEN** the first review returns only `minor` findings
- **THEN** the turn is not blocked and the user sees the findings

#### Scenario: Accepted after fixes with an option to cut the feature
- **WHEN** a repeat review accepts a proposal the previous review blocked, and returns a `scope` finding
- **THEN** the summary request asks to list that option with what the user loses and saves, and to leave out the review rounds and the fixes made for them

#### Scenario: Repeat limit
- **WHEN** the third proposal review in a row within one user request blocks the turn again
- **THEN** the hook stops the agent and shows the remaining findings
