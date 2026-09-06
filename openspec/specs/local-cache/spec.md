# Local Cache Specification

## Purpose

Defines the transient key-value behavior that cache consumers can rely on when Databasus runs as one application process.

## Requirements

### Requirement: Cache state is process-local and volatile

The cache SHALL keep entries inside the running application process. A new process SHALL start with an empty cache and SHALL NOT require a remote cache service.

#### Scenario: Application restarts

- **WHEN** the application process stops and starts again
- **THEN** entries created by the previous process are absent

#### Scenario: Application starts without a cache service

- **WHEN** an operator starts Databasus without a remote cache endpoint
- **THEN** cache-backed features remain available

### Requirement: Namespaced entries expire and can be invalidated

Each cache entry SHALL belong to a caller-selected namespace and SHALL have a positive lifetime. A read before expiration SHALL return the stored value. A read at or after expiration SHALL report a miss and make the expired entry eligible for reclamation. Callers SHALL be able to delete one entry or clear all entries owned by the process.

#### Scenario: Entry is read before expiration

- **WHEN** a caller reads a stored entry before its lifetime ends
- **THEN** the cache returns the stored value

#### Scenario: Entry reaches its expiration time

- **WHEN** a caller reads an entry at or after its expiration time
- **THEN** the cache reports a miss

#### Scenario: Namespaces use the same key

- **WHEN** two callers store different values under the same key in different namespaces
- **THEN** each caller reads the value from its own namespace

#### Scenario: Entry is invalidated

- **WHEN** a caller invalidates a stored entry
- **THEN** subsequent reads report a miss

### Requirement: Conditional storage operations are atomic

The cache SHALL provide atomic read-and-delete and create-if-absent operations. Expired entries SHALL be treated as absent by both operations.

#### Scenario: Concurrent callers consume one value

- **WHEN** multiple callers concurrently read and delete the same live entry
- **THEN** exactly one caller receives the value
- **AND** every other caller receives a miss

#### Scenario: Concurrent callers reserve one key

- **WHEN** multiple callers concurrently create the same absent key
- **THEN** exactly one caller stores its value
- **AND** every other caller is told that a live entry already exists

#### Scenario: Caller reserves an expired key

- **WHEN** a caller conditionally creates a key whose previous entry has expired
- **THEN** the new value is stored

### Requirement: Cache memory is bounded without evicting live entries

The cache SHALL enforce its configured payload budget. Before rejecting a write, it SHALL reclaim expired entries. It SHALL NOT evict an unexpired entry to admit another value.

#### Scenario: Expired entries occupy the available budget

- **WHEN** a write needs space held by expired entries
- **THEN** the cache reclaims those entries and admits the write

#### Scenario: Live entries occupy the available budget

- **WHEN** a write would exceed the budget after expired entries are reclaimed
- **THEN** the write fails
- **AND** every unexpired entry remains available

#### Scenario: One value exceeds the budget

- **WHEN** a caller tries to store a value larger than the entire cache budget
- **THEN** the write fails without changing existing entries

### Requirement: Concurrent cache access preserves the contract

Cache operations SHALL remain safe when called concurrently. A failed or canceled operation SHALL NOT leave a partially written value.

#### Scenario: Reads and writes overlap

- **WHEN** callers read, write, invalidate, reserve and consume entries concurrently
- **THEN** every completed operation has the same result as one valid serial ordering
- **AND** no caller observes a partial value

### Requirement: Security-sensitive cache failures stop protected operations

Restore-token issuance and consumption and stream-lock acquisition SHALL fail closed when their required cache operation fails. A cache failure SHALL NOT be interpreted as an absent token or an available stream lock.

#### Scenario: Restore-token issuance cannot store the token

- **WHEN** the cache rejects or fails the write required to issue a restore token
- **THEN** token issuance returns an error without returning a usable token

#### Scenario: Restore-token consumption cannot read and delete the token

- **WHEN** the cache cannot atomically read and delete a submitted restore token
- **THEN** token consumption returns an error
- **AND** the restore operation does not start

#### Scenario: Stream-lock acquisition cannot reserve the lock

- **WHEN** the cache cannot atomically reserve the required stream lock
- **THEN** lock acquisition returns an error
- **AND** the download or restore stream does not start
