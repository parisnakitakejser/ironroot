# Offline Signing

Offline signing keeps the Root CA private key away from the online IronRoot server.

Workflow:

1. Generate the Root CA on the offline machine.
2. Generate an Intermediate CA CSR.
3. Move only the CSR to the offline machine.
4. Sign the Intermediate with the Root CA.
5. Move the signed Intermediate certificate and chain back to the IronRoot server.

```mermaid
sequenceDiagram
  participant Online as Online staging host
  participant Media as Approved transfer media
  participant Offline as Offline Root CA host
  Online->>Media: Intermediate CSR
  Media->>Offline: CSR only
  Offline->>Offline: Root signs Intermediate
  Offline->>Media: signed Intermediate + Root cert
  Media->>Online: CA chain for IronRoot server
```
