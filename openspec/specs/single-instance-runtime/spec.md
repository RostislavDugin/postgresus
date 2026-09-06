# Single-Instance Runtime Specification

## Purpose

Defines one application process as the supported installation topology so transient coordination remains consistent without a shared service.

## Requirements

### Requirement: Supplied deployments run one application process

Databasus installation artifacts SHALL start exactly one application process for each installation. They SHALL NOT expose a setting that increases the application replica count.

#### Scenario: Operator deploys the Helm chart

- **WHEN** an operator renders or installs the supplied Helm chart
- **THEN** the resulting workload contains exactly one application replica

#### Scenario: Operator uses the Docker image

- **WHEN** an operator starts one instance of the supplied Docker image
- **THEN** that instance runs exactly one Databasus application process

### Requirement: One process owns transient runtime state

The single application process SHALL own cache entries, publish-subscribe subscriptions and rate-limit counters for its installation. The supplied deployment SHALL NOT require another service to share that state.

#### Scenario: Installation starts normally

- **WHEN** the application process starts
- **THEN** its transient runtime providers are available inside that process
- **AND** startup does not wait for a shared transient-state service
