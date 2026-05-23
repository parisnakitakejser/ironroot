# Security Principles

<span class="ironroot-page-status ironroot-page-status--in-progress">Status: In progress</span>

- Keep the Root CA offline.
- Store the Intermediate CA private key encrypted at rest where possible.
- Mount CA material from controlled files or encrypted local storage.
- Hash bootstrap tokens in the database.
- Use time-limited bootstrap tokens.
- Treat MAC address as metadata only.
- Validate bootstrap token, hostname, and machine ID during enrollment.
- Generate client private keys locally.
- Audit every important action.
- Serve the API over TLS in production.
- Keep room for future mTLS authentication.

Report vulnerabilities using the process in `SECURITY.md`.

Operational security pages:

- [First-run bootstrap guide](bootstrap-guide.md)
- [Security check command](security-check.md)
- [Offline Root CA handling](offline-root-handling.md)
- [Host hardening](host-hardening.md)
