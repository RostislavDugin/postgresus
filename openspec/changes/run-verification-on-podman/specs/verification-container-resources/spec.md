## ADDED Requirements

### Requirement: Restore containers run with reduced privileges

A restore runs code taken from the backup with database superuser rights. The verification agent SHALL create every restore container with all Linux capabilities dropped except `CHOWN`, `DAC_OVERRIDE`, `FOWNER`, `SETGID` and `SETUID`, with privilege escalation disabled, and with no directory of the host mounted into it. The agent SHALL request these settings from the container engine when it creates the container. It SHALL NOT fail a job because the engine describes the applied settings in a different form than the agent requested them.

#### Scenario: Container is created with reduced privileges

- **WHEN** the agent creates a restore container for a verification job
- **THEN** the container engine receives a request that drops all capabilities, adds back only `CHOWN`, `DAC_OVERRIDE`, `FOWNER`, `SETGID` and `SETUID`, disables privilege escalation and mounts no host directory

#### Scenario: Processes in the container hold only the minimal capabilities

- **WHEN** a restore container created by the agent is running
- **THEN** the capability bounding set of its processes contains only `CHOWN`, `DAC_OVERRIDE`, `FOWNER`, `SETGID` and `SETUID`
- **AND** its processes cannot gain privileges through setuid binaries

#### Scenario: Engine reports dropped capabilities as an explicit list

- **WHEN** the container engine reports the dropped capabilities of a restore container as a list of capability names instead of `ALL`
- **THEN** the agent proceeds with the restore and the verification reaches its normal terminal status

### Requirement: Verification runs on Podman

The verification agent SHALL verify backups when its Docker endpoint is a rootful Podman engine reached through Podman's Docker-compatible API. A restore of a valid backup SHALL complete, a restore of a broken backup SHALL fail with the restore error, and in both cases the agent SHALL remove the containers and networks it created for the job.

#### Scenario: Valid backup on Podman

- **WHEN** the agent's Docker endpoint is a rootful Podman engine
- **AND** the agent claims a verification job for a valid PostgreSQL backup
- **THEN** the agent reports the verification as completed with the restored database's statistics
- **AND** no container or network created for the job remains on the engine

#### Scenario: Broken backup on Podman

- **WHEN** the agent's Docker endpoint is a rootful Podman engine
- **AND** the agent claims a verification job for a backup that cannot be restored
- **THEN** the agent reports the verification as failed with the restore error
- **AND** no container or network created for the job remains on the engine
