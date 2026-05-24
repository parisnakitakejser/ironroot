# Monitoring With irtop

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status">Status: Draft</span>
</div>

`irtop` gives operators a live terminal view of IronRoot. It is useful over SSH, during local development, and while operating a Kubernetes deployment through port-forwarding.

## Local Server

Start IronRoot with the default local development config, then run:

```bash
irtop --server http://localhost:8443
```

The local development config serves HTTP because API TLS files are empty. Production should use HTTPS and a trusted CA bundle:

```bash
irtop --server https://ironroot.example.com:8443 --ca-file ./root-ca.crt
```

`irtop` does not silently downgrade HTTPS to HTTP. If the scheme does not match the server, it prints a remediation hint.

## Kubernetes Port-Forward

```bash
kubectl -n ironroot port-forward svc/ironroot 8443:8443
irtop --server https://localhost:8443 --ca-file ./root-ca.crt
```

## Operator Questions

Use `irtop` to answer:

- Is the API healthy and ready?
- Is the CA chain configured?
- Are certificates expiring soon?
- Are enrollments failing?
- Are bootstrap tokens being reused or expired?
- Is telemetry configured?
- Are recent audit events suspicious?

## Views

The overview screen summarizes server, certificate, enrollment, security, and telemetry posture. Use number keys to switch into deeper views.

```text
1 Overview      2 Certificates  3 Enrollments  4 Tokens
5 CA Health     6 Security      7 Telemetry    8 Audit
9 Server
```

## Troubleshooting

If `irtop` cannot connect:

- Verify the server URL and scheme.
- Check whether the API server is listening.
- Check whether the server is using HTTP or HTTPS.
- Check TLS trust or use `--ca-file`.
- Use `--insecure-skip-verify` only for deliberate TLS debugging.
- Verify any admin token is valid.
- Use `--output text` for a simpler one-shot diagnostic.

```bash
irtop --server http://localhost:8443 --output text
```
