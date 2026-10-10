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
