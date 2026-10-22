# verification-container-cleanup Specification

## Purpose
Defines how the verification agent clears the Docker containers and networks that earlier verification attempts and earlier agent processes left on the host, so that such leftovers never block a new verification attempt.

## Requirements

### Requirement: A retried verification is not blocked by leftovers of its earlier attempt

Before the verification agent creates the container and network of a verification attempt, it SHALL remove every container and network that it labeled for the same verification in an earlier attempt. A container or network that disappears before the agent removes it SHALL count as removed.

#### Scenario: The previous attempt left its network behind

- **WHEN** an attempt of a verification failed and left its network on the Docker host without a container attached
- **AND** the backend assigns the same verification to the agent again
- **THEN** the agent removes the leftover network
- **AND** the new attempt creates its network and proceeds to the restore

#### Scenario: The previous attempt left its container and network behind

- **WHEN** an earlier attempt of the verification left both its container and its network on the Docker host
- **AND** the agent starts a new attempt of the same verification
- **THEN** the agent removes the leftover container and then the leftover network
- **AND** the new attempt proceeds to the restore

#### Scenario: The leftover is removed concurrently

- **WHEN** the teardown of the previous attempt removes its network while the new attempt is clearing the same network
- **THEN** the new attempt treats the network as removed and proceeds to the restore

#### Scenario: No leftovers exist

- **WHEN** the agent starts the first attempt of a verification
- **THEN** the attempt proceeds to the restore without removing anything

### Requirement: A leftover that cannot be removed is reported as such

When the verification agent cannot list or cannot remove the leftovers of an earlier attempt of the same verification, it SHALL fail the new attempt before the restore and before it creates the attempt's network. The failure message SHALL say that the leftovers of an earlier attempt could not be removed, rather than that the network name is taken.

#### Scenario: Docker refuses to remove the leftover

- **WHEN** the Docker host refuses to remove a leftover container or network of an earlier attempt of the verification
- **THEN** the agent reports the attempt as failed before the restore starts
- **AND** the failure message says that the leftovers of an earlier attempt could not be removed
- **AND** the agent creates no network for the attempt

#### Scenario: Docker cannot list the leftovers

- **WHEN** the Docker host fails to list the containers or the networks of the verification before an attempt
- **THEN** the agent reports the attempt as failed before the restore starts
- **AND** the failure message says that the leftovers of an earlier attempt could not be removed
- **AND** the agent creates no network for the attempt

### Requirement: Agent startup removes orphaned networks

At startup, before it claims any verification, the verification agent SHALL remove every container and then every network labeled with its agent ID, including networks that no container is attached to. A container or network that disappears before the agent removes it SHALL count as removed. As before, a failure to list or remove a leftover SHALL be logged and SHALL NOT prevent the agent from starting. A failure with one kind of resource SHALL NOT stop the agent from clearing the other kind or the remaining resources.

#### Scenario: A network without a container remains from an earlier process

- **WHEN** the agent starts and the Docker host holds a network labeled with this agent's ID that no container is attached to
- **THEN** the agent removes that network before it claims a verification

#### Scenario: A container is still attached to the agent's network

- **WHEN** the agent starts and the Docker host holds a container labeled with this agent's ID that is attached to a network with the same label
- **THEN** the agent removes the container and then the network

#### Scenario: A leftover disappears during the purge

- **WHEN** the agent starts and a container or network labeled with this agent's ID disappears before the agent removes it
- **THEN** the agent treats it as removed and goes on with the remaining resources

#### Scenario: Docker cannot list the networks

- **WHEN** the agent starts and the Docker host fails to list networks
- **THEN** the agent logs a warning
- **AND** the agent still removes the containers labeled with its agent ID
- **AND** the agent starts normally

#### Scenario: Docker cannot list the containers

- **WHEN** the agent starts and the Docker host fails to list containers
- **THEN** the agent logs a warning
- **AND** the agent still tries to remove the networks labeled with its agent ID, logging each network that Docker refuses to remove
- **AND** the agent starts normally

### Requirement: Cleanup touches only the agent's own resources

The verification agent SHALL remove only containers and networks labeled with its own agent ID. The cleanup before an attempt SHALL additionally limit itself to resources labeled with the verification of that attempt. Containers and networks of other agents, of other verifications and of unrelated workloads on the same Docker host SHALL stay untouched.

#### Scenario: Another verification runs concurrently on the same agent

- **WHEN** the agent starts a retried attempt of one verification while a different verification runs in another slot
- **THEN** the container and network of the running verification stay untouched

#### Scenario: Another agent left resources for the same verification

- **WHEN** the agent starts an attempt of a verification and the Docker host holds a container and a network labeled with the same verification but a different agent ID
- **THEN** the agent leaves that container and network in place

#### Scenario: Another agent shares the Docker host

- **WHEN** the agent starts and the Docker host holds a network labeled with a different agent ID
- **THEN** the agent leaves that network in place

#### Scenario: An unrelated network shares the Docker host

- **WHEN** the agent starts and the Docker host holds a network without the agent's labels
- **THEN** the agent leaves that network in place
