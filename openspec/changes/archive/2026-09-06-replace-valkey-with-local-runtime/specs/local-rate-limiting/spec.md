## Purpose

Defines process-local request limiting that protects authentication and verification endpoints without relying on a separate data service.

## ADDED Requirements

### Requirement: Rate limits use a sliding time window

The rate limiter SHALL record every request attempt for a caller-selected scope and identifier. It SHALL allow no more than the configured number of attempts whose timestamps fall inside the current window. Rejected attempts SHALL also count until they leave that window.

#### Scenario: Attempts remain below the limit

- **WHEN** an identifier makes no more than the configured number of attempts inside the window
- **THEN** every attempt is allowed

#### Scenario: Attempt exceeds the limit

- **WHEN** an identifier makes another attempt while the configured number of earlier attempts remains inside the window
- **THEN** the new attempt is rejected

#### Scenario: Old attempts leave the window

- **WHEN** enough recorded attempts reach the window boundary
- **THEN** those attempts stop contributing to the limit
- **AND** a later request is evaluated against only the remaining attempts

### Requirement: Limit scopes are independent

The rate limiter SHALL keep counts independent for each scope and identifier pair.

#### Scenario: One identifier uses two scopes

- **WHEN** the same identifier reaches the limit in one scope
- **THEN** attempts in another scope retain their own allowance

#### Scenario: Two identifiers use one scope

- **WHEN** one identifier reaches the limit in a scope
- **THEN** another identifier retains its own allowance in that scope

### Requirement: Concurrent attempts cannot exceed the allowance

Recording an attempt and deciding whether it is allowed SHALL be one atomic operation.

#### Scenario: Concurrent attempts meet one remaining allowance

- **WHEN** multiple attempts arrive concurrently and only one allowance remains
- **THEN** exactly one attempt is allowed

### Requirement: Rate-limit state is process-local and reclaimable

Rate-limit counters SHALL remain inside the running application process, start empty after restart and release state after its final recorded attempt leaves the window.

#### Scenario: Application restarts

- **WHEN** the application process restarts
- **THEN** previous rate-limit attempts no longer affect new requests

#### Scenario: Identifier becomes inactive

- **WHEN** every recorded attempt for an identifier has left the window
- **THEN** the limiter can reclaim that identifier's state

### Requirement: Invalid rate-limit settings fail explicitly

The rate limiter SHALL reject a non-positive request limit or a non-positive window without recording an attempt.

#### Scenario: Caller supplies an invalid limit

- **WHEN** a caller supplies a zero or negative request limit or window
- **THEN** the operation returns an error
- **AND** existing counters remain unchanged

### Requirement: Protected endpoints fail closed when a limit cannot be evaluated

An endpoint protected by rate limiting SHALL reject the request when the limiter cannot atomically record and evaluate the attempt. It SHALL NOT treat a limiter error as an allowed request. The endpoint SHALL log the limiter failure without including the caller identifier or other user-derived rate-limit key material.

#### Scenario: Limiter cannot record an attempt

- **WHEN** a protected endpoint receives an error from the rate limiter
- **THEN** the endpoint rejects the request
- **AND** the request does not proceed to the protected operation
- **AND** the failure is logged without the caller identifier or user-derived rate-limit key material
