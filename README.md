# databasus-operator

A Kubernetes operator that manages database backups declaratively via Custom Resource Definitions (CRDs). Instead of configuring databases, schedules, storages, and notifiers through the [Databasus](www.databasus.com) UI, define them as Kubernetes resources and let the operator reconcile them against the databasus API.

_Forked from [databasus/databasus](https://github.com/databasus/databasus). The operator is in an early, but usable, stage of development. Please direct your feedback to [the proposal upstream](https://github.com/databasus/databasus/pull/534)._

<img width="1181" height="665" alt="Screenshot 2026-04-28 at 19 08 55" src="https://github.com/user-attachments/assets/e5d3c6c0-c039-4711-b20c-d468106fc8b8" />

## Features

Ideal for multi-service, multi-environment configuration management scenarios.

  - Easily define a 3-2-1 backup strategy, with customizable retention policies.
  - Keep your backup config close to your service codebase.
  - Integrate seemlessly with your alerting infrastructure.

## How it works

The operator watches three CRDs and syncs their state to a running databasus instance via its REST API:

- **`Storage`** -- storage backends where backups are saved (S3, SFTP, Azure Blob, etc.)
- **`Notifier`** -- notification channels for alerts (Discord, Slack, Telegram, etc.)
- **`DatabaseBackup`** -- database connection + backup schedule + healthcheck config, referencing Storage and Notifier resources by name

On create/update, the operator calls the databasus API to upsert resources. On delete, a finalizer ensures cleanup via the API before the Kubernetes resource is removed.

## Examples and Further Reading

For configuration examples see [operator/config/samples/](./operator/config/samples)

For more information and set up instructions see [operator/README.md](./operator/README.md)

