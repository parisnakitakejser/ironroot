# Local Development: Troubleshooting

<div class="ironroot-doc-meta" markdown>
  <span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
  <span class="ironroot-badge ironroot-badge--status ironroot-badge--in-progress">Status: In Progress</span>
</div>

This page lists common local failures and the fastest checks for each one.

## Binary Not Found

```bash
export PATH="$HOME/.local/bin:$PATH"
just install-local
```

Verify:

```bash
which ironroot-server
which ironroot-admin
which ironroot-client
which irtop
```

## Port 8443 Already In Use

```bash
lsof -i :8443
```

Stop the existing process or use another port:

```bash
IRONROOT_SERVER_ADDRESS=localhost:9443 \
  ironroot-server --config .localdev/config/config.yaml
```

Then use the same port everywhere:

```bash
irtop --server http://localhost:9443
ironroot-client enroll --server http://localhost:9443 --hostname demo.local --token <token>
```

## Invalid Bootstrap Token

If enrollment returns `401 Unauthorized` with `invalid bootstrap token`, the most common cause is a server using a different SQLite database than the admin command that created the token.

Create/list tokens with the same config the server uses:

```bash
ironroot-admin create-token --config .localdev/config/config.yaml --host demo.local --ttl 24h
ironroot-admin list-tokens --config .localdev/config/config.yaml
```

Start the server with that same config:

```bash
ironroot-server --config .localdev/config/config.yaml
```

Also confirm the enrollment hostname matches the token host:

```bash
ironroot-client enroll \
  --server http://localhost:8443 \
  --hostname demo.local \
  --token <token>
```

## Wrong HTTP/HTTPS Scheme

Local development uses HTTP unless TLS cert/key files are configured:

```bash
irtop --server http://localhost:8443
curl http://localhost:8443/healthz
```

Do not use `https://localhost:8443` against the default local server.

## Missing PKI Files

Check generated material:

```bash
find .localdev/pki -maxdepth 3 -type f -print
```

Expected files include:

```text
.localdev/pki/root/root-ca.crt
.localdev/pki/root/root-ca.key
.localdev/pki/intermediate/intermediate-ca.crt
.localdev/pki/intermediate/intermediate-ca.key
.localdev/pki/intermediate/ca-chain.crt
```

If they are missing, rerun the Root and Intermediate creation commands from [Up And Running](local-development-up-and-running.md).

## Bootstrap Not Completed

```bash
ironroot-admin bootstrap \
  --config .localdev/config/config.yaml \
  --non-interactive \
  --acknowledge-risk
```

## SQLite Permission Problems

```bash
ls -ld .localdev/data
sqlite3 .localdev/data/ironroot.db ".tables"
```

If the DB was created by another user, remove `.localdev` or fix ownership before restarting the server.

## Client Cannot Connect

```bash
curl http://localhost:8443/healthz
curl http://localhost:8443/readyz
```

If health checks fail, verify the server process is running and using the expected config:

```bash
ironroot-server --config .localdev/config/config.yaml
```

## TLS Trust Errors

Install the Root CA public certificate, not the private key:

```text
.localdev/pki/root/root-ca.crt
```

Confirm the requested certificate DNS name matches the URL you are testing.

Inspect a certificate:

```bash
openssl x509 -in .localdev/certs/demo.local/tls.crt -text -noout
```

Verify the chain:

```bash
openssl verify \
  -CAfile .localdev/pki/root/root-ca.crt \
  -untrusted .localdev/pki/intermediate/intermediate-ca.crt \
  .localdev/certs/demo.local/tls.crt
```

## MkDocs Missing Dependencies

```bash
just docs-install
just docs-build
```

## Clean Local State

This removes generated local state:

```bash
just dev-clean
```

Then rebuild with [Up And Running](local-development-up-and-running.md).
