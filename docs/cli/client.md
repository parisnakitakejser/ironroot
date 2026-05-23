# ironroot-client

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status">Status: Draft</span>
</div>

`ironroot-client` enrolls machines, requests certificates, renews certificates, checks status, and downloads trust bundles.

```bash
ironroot-client enroll --server http://localhost:8443 --hostname demo.home.arpa --token <token>
ironroot-client request-cert --server http://localhost:8443 --enrollment-id <id> --dns demo.home.arpa --out ./certs
ironroot-client status --server http://localhost:8443 --serial <serial>
```

For enrollment, `--hostname` must match the hostname bound to the bootstrap token. If omitted, IronRoot uses the machine's OS hostname.

`request-cert` and `renew` require the `enrollment_id` returned by `enroll`. Passing the bootstrap token as `--enrollment-id` is invalid.
