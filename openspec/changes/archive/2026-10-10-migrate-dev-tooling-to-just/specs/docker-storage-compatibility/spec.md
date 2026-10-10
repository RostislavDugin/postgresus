## MODIFIED Requirements

### Requirement: One filesystem suite runs locally and in CI

The repository SHALL provide `just test-filesystem` as the single entry point for the Docker filesystem suite. It SHALL build or accept a local candidate image and run the same cases locally and in a dedicated GitHub Actions job. The job SHALL build and load its own candidate without publishing or transferring it.

Each positive case SHALL start the real container and invoke `databasus --test-storage`, which uses the public local-storage save and delete methods. Each negative case SHALL assert the startup failure and its actionable error. Every case SHALL clean its containers, volumes, networks, mounts, and temporary files.

#### Scenario: Developer runs the complete suite

- **WHEN** a developer runs `just test-filesystem` without an image reference
- **THEN** the recipe builds a local candidate and tests it

#### Scenario: CI tests its candidate

- **WHEN** the filesystem job builds and loads its candidate
- **THEN** the suite tests that local image without pushing or pulling a candidate image

#### Scenario: A case allocates Docker resources

- **WHEN** the case succeeds or fails after resource allocation
- **THEN** the harness removes every resource created by that case
