# ironroot-admin

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status">Status: Draft</span>
</div>

`ironroot-admin` is the trusted operator CLI.

Common commands:

```bash
ironroot-admin ca create-root --out ./pki/root
ironroot-admin ca create-intermediate --root-cert ./pki/root/root-ca.crt --root-key ./pki/root/root-ca.key --out ./pki/intermediate
ironroot-admin bootstrap --config ./examples/config.local.yaml
ironroot-admin --config ./examples/config.local.yaml create-token --host local-demo --ttl 24h
ironroot-admin security-check --config ./examples/config.local.yaml
```
