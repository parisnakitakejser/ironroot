# CLI Examples

<span class="ironroot-page-status ironroot-page-status--in-progress">Status: In progress</span>

See the local quick start for a complete flow from Root CA creation to browser-trusted HTTPS.

```bash
ironroot-admin ca verify-chain --root-cert ./pki/root/root-ca.crt --intermediate-cert ./pki/intermediate/intermediate-ca.crt
ironroot-client trust install --server http://localhost:8443 --out ./trust
```
