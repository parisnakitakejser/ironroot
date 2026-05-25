# 10. Troubleshooting

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status ironroot-badge--in-progress">Status: In Progress</span>
</div>

Use this page when the onboarding path does not behave as expected.

## Prerequisites

- Know which config file you are using.
- Keep the server terminal open so you can read startup errors.

## Fast Checks

```bash
ironroot-server --config .localdev/config/config.yaml
curl http://localhost:8443/healthz
curl -s http://localhost:8443/v1/status/ca-hierarchy
irtop --server http://localhost:8443
```

## Common Startup Issues

| Error | Cause | Fix |
| --- | --- | --- |
| `address already in use` | Port `8443` is occupied. | Stop the other process or change `server.address`. |
| missing PKI file | Config points at a file that does not exist. | Recreate Root/Intermediate files or fix `pki.*` paths. |
| invalid Intermediate password | `pki.intermediate_key_pass` does not decrypt the key. | Use the password from Intermediate creation or a password file/secret source. |
| RBAC validation error | A manifest is malformed or references a missing resource. | Fix `.localdev/config/rbac/*.yaml` and restart. |

## Token And Enrollment Issues

| Symptom | Check |
| --- | --- |
| `invalid bootstrap token` | Token expired, already used, revoked, host mismatch, or admin/server use different databases. |
| Hostname mismatch | `create-token --host` must match `ironroot-client enroll --hostname`. |
| Token secret missing | Token secrets are printed once. Create a new token if lost. |
| Certificate request rejects enrollment | Use the returned `enrollment_id`, not the bootstrap token. |

## Certificate Chain Issues

Validate the chain:

```bash
ironroot-admin ca verify-chain \
  --root-cert .localdev/pki/root/root-ca.crt \
  --intermediate-cert .localdev/pki/intermediate/intermediate-ca.crt \
  --cert .localdev/certs/local-demo/tls.crt
```

If browsers do not trust the certificate, install the Root CA trust bundle into the operating system or browser trust store. The local API being HTTP on `localhost:8443` is separate from certificates issued for workloads.

## RBAC Mistakes

- Use exact `metadata.name` values in role references.
- Keep `kind` values capitalized as documented.
- Use explicit `intermediateRef` for CA-scoped roles and token policies.
- Restart the server after changing RBAC files.

## Debugging Recommendations

```bash
ironroot-admin --config .localdev/config/config.yaml list-tokens --wide
ironroot-admin --server http://localhost:8443 list-certs
curl -s http://localhost:8443/v1/status/tokens
curl -s http://localhost:8443/v1/status/ca-hierarchy
```

Run `irtop --server http://localhost:8443` for a live read-only view.

## Expected Outcome

You can identify whether a problem is caused by config paths, PKI files, RBAC validation, token state, or certificate chain trust.

## Still Blocked

Collect:

- command run.
- full error text.
- relevant `config.yaml` section with secrets removed.
- server startup logs.
- `curl http://localhost:8443/healthz` result.

Then compare with [Contributing Local Development Troubleshooting](../contributing/local-development-troubleshooting.md) for deeper local development checks.
