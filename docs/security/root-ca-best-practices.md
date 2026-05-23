# Root CA Best Practices

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status">Status: Draft</span>
</div>

Treat the Root CA as offline security infrastructure, not as an application runtime dependency.

## Recommended Practice

- Generate the Root CA on an offline or air-gapped machine.
- Encrypt `root-ca.key` at rest.
- Store at least two encrypted offline backups.
- Use the Root CA only to sign Intermediate CAs.
- Keep normal server and client certificate issuance on the Intermediate CA.
- Record fingerprints and recovery instructions.
- Test recovery before production use.

## What Not To Do

- Do not copy `root-ca.key` to the IronRoot server.
- Do not mount `root-ca.key` into containers.
- Do not store `root-ca.key` in Kubernetes Secrets.
- Do not commit Root CA material to Git.
- Do not put Root CA passwords in CI logs or shell history.

## Inspect Root CA Material

```bash
ironroot-admin ca inspect --output table ./pki/root/root-ca.crt
ironroot-admin ca inspect --output markdown ./pki/root/root-ca.crt
ironroot-admin ca inspect --output json ./pki/root/root-ca.crt
```

Use inspection output during reviews and recovery drills. It shows subject, issuer, algorithm, curve or key size, validity, key usages, path length, and fingerprints.
