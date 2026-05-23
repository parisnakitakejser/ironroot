# REST API

<span class="ironroot-page-status ironroot-page-status--in-progress">Status: In progress</span>

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/healthz` | Liveness |
| GET | `/readyz` | Readiness |
| GET | `/v1/ca/root` | Root CA certificate |
| GET | `/v1/ca/chain` | CA chain |
| POST | `/v1/enroll` | Bootstrap enrollment |
| POST | `/v1/certificates/request` | Sign CSR |
| POST | `/v1/certificates/renew` | Renew certificate |
| POST | `/v1/certificates/revoke` | Revoke certificate metadata |
| GET | `/v1/certificates/status/{serial}` | Certificate status |
| GET | `/v1/audit` | Audit log entries |

Clients propagate W3C Trace Context headers so the server can continue command traces.

