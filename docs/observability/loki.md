# Loki

Loki stores structured IronRoot logs. Logs are most useful when indexed by stable deployment labels such as:

- `app=ironroot`
- `component=pki-server`
- `environment=production`
- `namespace=ironroot`

Do not index certificate serial numbers, hostnames, or request IDs as labels unless volume is known to be small. Keep those as JSON fields to avoid high cardinality.

Example query:

```logql
{app="ironroot"} | json | level="ERROR"
```
