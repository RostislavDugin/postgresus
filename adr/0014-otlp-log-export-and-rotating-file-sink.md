# ADR-0014: Export logs over OTLP and keep a copy on disk

- **Status:** Accepted
- **Date:** 2026-08-14
- **Tags:** backend, observability

## Context

Databasus logs to stdout, which is fine for `docker logs` and useless for everything else.
Operators want their logs in whatever they already run (Grafana, VictoriaLogs, Graylog, SigNoz)
and when a container dies or the database gets corrupted, the logs about it should still be
readable.

## Decision

Every log record goes through `log/slog` and fans out to three sinks: stdout, a rotating JSON file,
and an OTLP exporter.

**Open Telemetry** because it is the one protocol every log backend speaks. We ship to a single endpoint
configured by the operator and let the OpenTelemetry Collector handle routing from there. If the
remote is down, the exporter drops records instead of blocking a request.

**The file** because it survives what the database does not. Audit records are written to it as
well as to the audit table, so a corrupted or wiped database still leaves a trail on disk. The file
sink waits a moment before dropping anything, since it is the forensic copy.

**Request IDs** so the three sinks can be joined. A middleware puts a fresh UUID on every request,
returns it as `X-Request-Id`, and stores it in the request context. Controllers pass that context
into every service call, so any log written while handling a request carries the same ID without a
single call site threading a logger by hand. An inbound `X-Request-Id` is ignored, otherwise a
client could stitch its actions into someone else's trace.

Audit records get a `log_type=audit` marker and ignore `LOG_LEVEL`, so raising the level never
silences the trail. Redaction happens in the handler, not at call sites, because audit messages
carry user emails off the machine.

## Consequences

### Positive

- Logs land in any backend the operator already runs, with no Databasus-side integration per vendor.
- An audit record, its access log line, and any error in between share a `request_id` everywhere.
- The audit trail outlives the database.