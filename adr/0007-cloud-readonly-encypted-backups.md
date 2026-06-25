# ADR-0007: Enforce read-only database access and encrypted backups in cloud

- **Status:** Deprecated (cloud edition discontinued, 2026-06)
- **Date:** 2026-02-01
- **Tags:** cloud, security, backups, access-control

## Context

Databasus is a backup tool — by definition it holds the keys to the most valuable data a customer has. We treat that responsibility seriously and invest in security at every layer.

- **Application.** Secret encryption at rest, scoped credentials per tenant, audit logging on every privileged action. Every PR runs through CodeQL, Trivy, Dependabot, gitleaks, semgrep, CodeRabbit and Codex Security. Security review is mandatory on changed code.
- **Infrastructure.** Network isolation per [ADR-0006](./0006-cloud-load-and-traffic-distribution-between-nodes.md), processing nodes with no public ingress, CloudFlare in front of the API, hardened Docker images with non-root runtime, GitHub Actions pinned to commit SHAs, regular base-image refresh.
- **Access.** Least-privilege IAM on storage backends, short-lived tokens, MFA on operator accounts, no shared production credentials.

That stack is genuinely strong and we keep raising it. The chance of a breach against any one layer is small.

It is not zero. A kernel 0-day, a TLS-library CVE, etc. — any of these can land on a Monday morning and cut through layers we trusted on Sunday night. AI offensive tooling like Claude Mythos now chews through systems that were considered hardened a year ago, compressing the gap between disclosure and weaponisation to hours. On a long enough timeline some external dependency we do not own fails.

We add one more layer that does not depend on any of the above holding: we structurally remove anything Databasus could do that would harm a customer. Even in the scenario where every other defence is bypassed, the attacker finds nothing useful at the end of the chain.

## Decision

Cloud mode enforces two hard rules at the application level.

**Read-only database access.** Cloud customers can only connect Databasus to a database role with the read privileges sufficient to run the backup tool for that engine — `SELECT` plus the engine-specific reads needed by `pg_dump` / `mysqldump` / `mongodump`. Roles with write, DDL or superuser privileges are rejected at connection-setup time with a clear error. The check runs before the first backup is scheduled and is repeated on every credential rotation.

**Encrypted backups only.** Every backup artifact produced in cloud mode is encrypted on the processing node before it leaves the process. "Backup without encryption" is not a supported option in cloud — the toggle does not exist in the cloud UI or API.

Self-hosted instance mode keeps both options open. The operator owns the security perimeter and may choose write access or plain backups for their own valid reasons.

## Consequences

### Positive

- **A Databasus compromise cannot corrupt customer data.** Even with full credentials in hand the attacker can only read. There is no write path through Databasus to customer production — the worst case is structurally bounded, not policy-bounded.
- **The 0.0001% case has a defined ceiling.** No single defence has to be perfect because each layer caps what the next failure can do. Weak independent guarantees compose into a strong one.
- **One answer to "can I give a write user?"** Cleaner support story, no edge-case configuration to debug.

## References

- [ADR-0006](./0006-cloud-load-and-traffic-distribution-between-nodes.md) — cluster topology and network isolation for cloud nodes (the perimeter this decision layers on top of).
