# Tempo

<span class="ironroot-page-status ironroot-page-status--in-progress">Status: In progress</span>

Tempo stores traces from IronRoot. Use it to inspect enrollment, certificate request, renewal, revocation, bootstrap, and security-check workflows.

Look for:

- Missing `traceparent` propagation between CLI and server.
- Slow `db.*` spans.
- `ca.sign_csr` errors.
- Audit write failures.
- API spans with non-2xx status codes.

Tempo is most useful when logs include `trace_id` and metrics link to exemplars or trace drilldowns.
