## MODIFIED Requirements

### Requirement: Restore containers run with reduced privileges

A restore runs code taken from the backup with database superuser rights. The verification agent SHALL start every restore container as the image's unprivileged database user, SHALL run its own commands inside the container as that user, SHALL leave the container's capability bounding set empty, so that even a process that reaches user ID 0 inside the container holds no capabilities, and SHALL mount no directory of the host into it. The agent SHALL request these settings from the container engine when it creates the container. It SHALL NOT fail a job because the engine describes the applied settings in a different form than the agent requested them.

#### Scenario: Container is created with reduced privileges

- **WHEN** the agent creates a restore container for a verification job
- **THEN** the container engine receives a request that starts the container as the image's unprivileged database user, drops all capabilities, adds none back and mounts no host directory

#### Scenario: Agent's commands run as the unprivileged user

- **WHEN** the agent runs a command inside a restore container, such as writing the backup into it, restoring it or measuring its disk usage
- **THEN** the agent's request names no user, so the command runs as the container's unprivileged database user

#### Scenario: Processes in the container hold no privileges

- **WHEN** a restore container created by the agent is running
- **THEN** its first process runs as a non-root user
- **AND** the capability bounding set of its processes is empty, so a process that gains user ID 0 inside the container still holds no capabilities

#### Scenario: Engine reports dropped capabilities as an explicit list

- **WHEN** the container engine reports the dropped capabilities of a restore container as a list of capability names instead of `ALL`
- **THEN** the agent proceeds with the restore and the verification reaches its normal terminal status

### Requirement: Verification runs on Podman

The verification agent SHALL verify backups when its Docker endpoint is a rootful Podman engine reached through Podman's Docker-compatible API, including on a host that confines containers with AppArmor. A restore of a valid backup SHALL complete, a restore of a broken backup SHALL fail with the restore error, and in both cases the agent SHALL remove the containers and networks it created for the job.

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

#### Scenario: Podman on a host with AppArmor

- **WHEN** the agent's Docker endpoint is a rootful Podman engine on a host where AppArmor confines containers and the container runtime has its own AppArmor profile
- **AND** the agent claims a verification job for a valid PostgreSQL backup
- **THEN** the database in the restore container starts and accepts the agent's connection
- **AND** the agent reports the verification as completed

### Requirement: A missing CPU limit is announced and leaves other isolation intact

A restore runs code taken from the backup with database superuser rights, so a restore container without a CPU limit can use every CPU core on the host. The verification agent SHALL log a warning at startup when it will run restore containers without a CPU limit, and SHALL keep every other container isolation control in force.

#### Scenario: Startup warning on an unsupported host

- **WHEN** the Docker host reports at agent startup that it cannot enforce a CPU limit
- **THEN** the agent logs a warning that restore containers will run without a CPU limit and can use every CPU core on the host

#### Scenario: Other limits remain applied

- **WHEN** the agent creates a restore container without a CPU limit
- **THEN** the container still has the per-job memory limit, the process count limit, no capabilities, and runs as the image's unprivileged database user
