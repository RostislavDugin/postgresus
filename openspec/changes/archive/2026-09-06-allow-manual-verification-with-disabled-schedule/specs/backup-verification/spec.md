## Purpose

Defines the relationship between on-demand backup verification and the configuration that controls automatic verification runs.

## ADDED Requirements

### Requirement: Manual verification is independent of automatic scheduling

The system SHALL allow an authorized user to enqueue and run a manual backup verification while automatic verification is disabled.

#### Scenario: Enqueue manual verification while automatic verification is disabled

- **WHEN** an authorized user requests manual verification of an eligible backup whose automatic verification is disabled and no other manual verification for that database is pending or running
- **THEN** the system accepts the request and keeps the manual verification pending for an agent

#### Scenario: Disable automatic verification while a manual verification is pending

- **WHEN** automatic verification is disabled for a database with a pending manual verification
- **THEN** the manual verification remains pending

#### Scenario: Disable automatic verification while a manual verification is running

- **WHEN** automatic verification is disabled for a database with a running manual verification
- **THEN** the manual verification remains running and can reach its normal terminal status

### Requirement: Disabling automatic verification cancels only automatic runs

The system SHALL cancel non-terminal automatic verification runs when automatic verification is disabled without canceling manual verification runs for the same database.

#### Scenario: Disable automatic verification with a time-based run

- **WHEN** automatic verification is disabled for a database with a pending or running time-based verification
- **THEN** the system cancels that automatic verification

#### Scenario: Disable automatic verification with an after-backup run

- **WHEN** automatic verification is disabled for a database with a pending or running after-backup verification
- **THEN** the system cancels that automatic verification

#### Scenario: Manual and automatic runs coexist when scheduling is disabled

- **WHEN** automatic verification is disabled for a database with both manual and automatic non-terminal verifications
- **THEN** the system cancels the automatic verifications and leaves the manual verifications unchanged
