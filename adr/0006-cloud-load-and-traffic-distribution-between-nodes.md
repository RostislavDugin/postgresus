# ADR-0006: Distribute cloud load across primary and processing nodes

- **Status:** Deprecated (cloud edition discontinued, 2026-06; multi-node deployment vars `IS_PRIMARY_NODE` / `IS_PROCESSING_NODE` remain valid)
- **Date:** 2026-02-01
- **Tags:** cloud, deployment, networking, multi-node

## Context

The project required funding so we need to launch Cloud to invest in development.

In cloud mode Databasus faces two pressures that a single-node deployment cannot answer.

**Source-IP exposure.** Every backup opens an outbound connection from the Databasus host to the customer's external database. A single-node cloud install would expose one fixed IP to every tenant — a public target for attack.

**Bandwidth contention.** Backups can be arbitrarily large. One tenant pulling 20 backups of 1 TB each will saturate even a 10 Gbit/s uplink for a significant amount of time and starve every other tenant's backup or download down to a crawl. The instance-mode design assumes a single operator with predictable load; the cloud has many tenants with adversarial timing.

The instance (self-hosted) deployment does not have these pressures: one operator runs one node against their own databases on their own network.

## Decision

Cloud deployments run as a cluster of nodes built from the **same binary**, differentiated by two orthogonal axes.

### Application roles

Roles are controlled by environment flags:

- `IS_PRIMARY_NODE=true` — runs schedulers, migrations, the backup and restore registries that distribute work over Valkey pub/sub. One node per Databasus installation.
- `IS_PROCESSING_NODE=true` — runs the backup and restore worker loops that pick up jobs from Valkey and stream bytes between customer databases and storage. N nodes per Databasus installation.

Both flags can be set on the same node. Instance mode is exactly this case: `IS_MANY_NODES_MODE=false` defaults both to `true` so one container handles everything.

### Network exposure

Network exposure is controlled by infrastructure, not by the binary:

- **API-facing nodes** sit behind CloudFlare with port 4005 published through a reverse proxy. CloudFlare absorbs DDoS on the API surface.
- **Processing nodes** do not publish port 4005 in their Docker config and are not added to any reverse-proxy upstream. They reach the rest of the cluster only through the internal Valkey channel. They are not reachable from the public internet so they cannot be DDoSed on the API surface.

Inter-node coordination flows through Valkey pub/sub — primary publishes assignments on `backup:submit` and `restore:submit`; processing nodes subscribe and report completions on the corresponding completion channels. Nodes never call each other over HTTP.

### Bandwidth fairness

Each node has a configured `NODE_NETWORK_THROUGHPUT_MBPS` budget (default 125 MB/s ≈ 1 Gbit/s). The `BandwidthManager` reserves 75% of the budget for data-moving operations and splits it equally across the active ones in real time. Ten parallel operations on a 1 Gbit/s node get ~100 Mbit/s each. Starting or finishing an operation recomputes every active rate so no single user can hog the pipe.

A per-user lock caps each user to one in-flight operation of each kind so one tenant cannot fan out an unbounded number of parallel jobs against the node.

## Consequences

### Positive

- **Many source IPs instead of one.** Processing nodes are deployed across multiple hosts and regions. A customer sees one of N IPs per backup so no single IP carries the whole fleet's reputation.
- **API and backup capacity scale independently.** Adding API capacity costs a small node behind CloudFlare. Adding backup capacity costs a gigabit-class processing node with no inbound exposure.
- **Fair sharing under load.** No tenant can starve another by starting a huge download. The split is computed per active session so the worst case for any user is a slow download, not a failed one.
- **One binary, one image.** Roles are environment configuration so deploys, tests and rollbacks operate on a single artifact.

### Negative

- **The deployment contract lives outside the code.** "Processing node has no port mapping and no proxy upstream" is enforced by ops, not by the binary. A misconfigured deploy can expose a processing node's API. The contract belongs in the deployment runbook and infra-as-code.
- **CloudFlare protects only inbound API traffic.** Backup throughput is outbound from processing nodes directly to customer databases and storage so it never traverses CloudFlare. Flooding a processing node's uplink is still possible; the mitigation is gigabit-class nodes and the per-node bandwidth budget.

## References

- [ADR-0004](./0004-focus-on-streaming-chunk-by-chunk-with-backpressure.md) — chunk-streaming with backpressure (the memory-safety foundation this scheduling sits on top of).
