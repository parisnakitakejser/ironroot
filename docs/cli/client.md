# ironroot-client

`ironroot-client` enrolls machines, requests certificates, renews certificates, checks status, and downloads trust bundles.

```bash
ironroot-client enroll --server http://localhost:8443 --token <token>
ironroot-client request-cert --server http://localhost:8443 --enrollment-id <id> --dns demo.home.arpa --out ./certs
ironroot-client status --server http://localhost:8443 --serial <serial>
```
