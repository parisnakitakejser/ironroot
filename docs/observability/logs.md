# Logs

<span class="ironroot-page-status ironroot-page-status--in-progress">Status: In progress</span>

IronRoot uses structured JSON logs. Logs include trace correlation fields when a span is active:

- `timestamp`
- `level`
- `msg`
- `trace_id`
- `span_id`
- `component`
- `operation`
- `hostname`
- `serial`
- `request_id`

Logs should help security auditing without exposing secrets. IronRoot must not log private keys, bootstrap token values, passphrases, or sensitive environment values.

Use logs with traces:

1. Open a failed trace in Tempo or another tracing backend.
2. Copy the `trace_id`.
3. Search logs in Loki for that `trace_id`.
4. Review API, CA, DB, and audit events around the failure.

Example Loki query:

```logql
{app="ironroot"} | json | trace_id="TRACE_ID"
```
