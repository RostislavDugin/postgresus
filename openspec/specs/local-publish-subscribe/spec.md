# Local Publish-Subscribe Specification

## Purpose

Defines reliable process-local message delivery for features that coordinate work inside the single Databasus application process.

## Requirements

### Requirement: Publications stay inside the application process

The publish-subscribe provider SHALL deliver messages only to active subscriptions in the same application process. It SHALL NOT persist messages or replay publications made before a subscription became active.

#### Scenario: Message is published before subscription

- **WHEN** a message is published before a matching subscription becomes active
- **THEN** the later subscription does not receive that message

#### Scenario: Another process uses the same channel

- **WHEN** separate application processes use the same channel name
- **THEN** neither process receives the other process's publications

### Requirement: Subscription registration has no readiness race

Subscription creation SHALL return only after the subscription can accept publications. Canceling its context or closing the subscription SHALL prevent delivery of later publications.

#### Scenario: Publish follows subscription immediately

- **WHEN** a caller publishes immediately after subscription creation returns
- **THEN** the new subscription accepts the message without an additional readiness delay

#### Scenario: Subscription is canceled

- **WHEN** a subscription's context is canceled before a later publication
- **THEN** the canceled subscription does not receive that publication

### Requirement: Successful publication does not silently drop messages

A successful publication SHALL submit one copy of the message to every matching subscription that was active when publication began. Messages submitted to one subscription SHALL preserve publication order. If the publication context ends before all active subscriptions accept the message, the operation SHALL return an error instead of reporting success.

#### Scenario: Channel has multiple subscribers

- **WHEN** a caller successfully publishes to a channel with multiple active subscriptions
- **THEN** each subscription receives one copy of the message

#### Scenario: Subscriber receives consecutive publications

- **WHEN** a caller successfully publishes two messages to the same channel in sequence
- **THEN** each active subscription receives them in the same order

#### Scenario: Publication is canceled during delivery

- **WHEN** the publication context ends before every active subscription accepts the message
- **THEN** the publication returns an error

### Requirement: Subscriber failures are isolated

A subscriber failure SHALL NOT stop other subscriptions, corrupt broker state or crash the application process.

#### Scenario: Message handler panics

- **WHEN** one subscription handler panics while processing a message
- **THEN** other active subscriptions can continue receiving messages
- **AND** later publications can still use the broker

### Requirement: Broker shutdown releases subscriptions

Closing the publish-subscribe provider SHALL stop its subscriptions and reject later publications. Repeated close operations SHALL be safe.

#### Scenario: Provider closes

- **WHEN** the provider is closed while subscriptions are active
- **THEN** those subscriptions stop
- **AND** a later publication returns an error
