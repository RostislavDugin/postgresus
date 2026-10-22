# verification-container-resources Specification

## Purpose
Defines the resource limits the verification agent applies to each container that restores a backup, and how the agent adapts those limits to what the Docker host can enforce.

## Requirements

### Requirement: Restore containers are CPU-limited when the host supports it

The verification agent SHALL limit each restore container to the per-job CPU budget derived from the configured CPU total and the configured number of concurrent jobs, unless the Docker host reports that it cannot enforce a CPU limit.

#### Scenario: Host supports CPU limits

- **WHEN** the Docker host reports at agent startup that it can enforce a CPU limit
- **AND** the agent runs with a CPU total of 4 and 2 concurrent jobs
- **THEN** each restore container is limited to 2 CPU cores

#### Scenario: Host support cannot be determined

- **WHEN** the agent cannot read the Docker host's CPU limit support at startup
- **THEN** the agent limits each restore container to the per-job CPU budget, as on a host that supports the limit

### Requirement: Verification runs on hosts that cannot enforce a CPU limit

The verification agent SHALL create restore containers without a CPU limit when the Docker host reports at agent startup that it cannot enforce one. Verification on such a host SHALL proceed instead of failing at container creation.

#### Scenario: Host lacks CPU limit support

- **WHEN** the Docker host reports at agent startup that it cannot enforce a CPU limit
- **AND** the agent claims a verification job
- **THEN** the agent creates the restore container without a CPU limit
- **AND** the restore and verification proceed to their normal terminal status

#### Scenario: CPU budget still sizes the restore

- **WHEN** the Docker host cannot enforce a CPU limit
- **THEN** the agent still accepts its configured CPU total, reports it to the server, and uses the per-job CPU budget to size restore parallelism

### Requirement: A missing CPU limit is announced and leaves other isolation intact

A restore runs code taken from the backup with database superuser rights, so a restore container without a CPU limit can use every CPU core on the host. The verification agent SHALL log a warning at startup when it will run restore containers without a CPU limit, and SHALL keep every other container isolation control in force.

#### Scenario: Startup warning on an unsupported host

- **WHEN** the Docker host reports at agent startup that it cannot enforce a CPU limit
- **THEN** the agent logs a warning that restore containers will run without a CPU limit and can use every CPU core on the host

#### Scenario: Other limits remain applied

- **WHEN** the agent creates a restore container without a CPU limit
- **THEN** the container still has the per-job memory limit, the process count limit, all capabilities dropped except the minimal set, and privilege escalation disabled

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

### Requirement: An unreachable restore container is reported with its cause

When the agent cannot connect to the database in a restore container before the readiness timeout, the verification SHALL fail, and its failure message SHALL include the error of the agent's last connection attempt that ended before the timeout, such as a refused connection, a connection that got no answer or a rejected login. The failure message SHALL NOT contain the database password.

#### Scenario: Published port refuses connections

- **WHEN** the database in a restore container stays unreachable because its published port refuses every connection
- **THEN** the agent reports the verification as failed
- **AND** the failure message says that the readiness timeout passed and that the last connection attempt was refused
