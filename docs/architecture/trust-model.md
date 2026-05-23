# Trust Model

<span class="ironroot-page-status ironroot-page-status--in-progress">Status: In progress</span>

IronRoot trust starts at the offline Root CA. The Root CA signs Intermediate CAs only. The online IronRoot server holds the encrypted Intermediate CA key and uses it to sign workload certificates from validated CSRs.

```mermaid
flowchart TD
  Root[Offline Root CA] --> Intermediate[Online Intermediate CA]
  Intermediate --> Leaf[Client and server certificates]
  Client[Client private key] --> CSR[CSR only]
  CSR --> API[IronRoot API]
  API --> Intermediate
```

The Root CA private key should never exist on the IronRoot server, in Kubernetes, inside containers, or in CI.
