# Offline Root CA

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status">Status: Draft</span>
</div>

The Root CA is the trust anchor. It should not be an always-on service. It should ideally live on an offline or air-gapped machine and only be powered on or mounted when signing a new Intermediate CA or performing a planned root migration.

## Responsibilities

- Create the top-level trust anchor.
- Sign Intermediate CA certificates.
- Support planned Root CA migration.
- Stay offline and protected from routine operational access.

## Lifecycle

```mermaid
flowchart TD
  Create[Create Root CA offline] --> Encrypt[Encrypt Root CA private key]
  Encrypt --> Backup[Create two or more encrypted offline backups]
  Backup --> Sign[Sign Intermediate CA certificate]
  Sign --> Store[Return Root key to offline storage]
  Store --> Rotate[Use only for Intermediate rotation or Root migration]
```

Recommended Root CA lifetime: 20 years.

## Signing Workflow

```mermaid
sequenceDiagram
  participant ServerHost as Secure build/admin host
  participant Offline as Offline Root CA host
  participant IronRoot as IronRoot Server
  ServerHost->>ServerHost: generate Intermediate key and CSR
  ServerHost->>Offline: transfer Intermediate CSR only
  Offline->>Offline: sign Intermediate certificate
  Offline-->>ServerHost: return Intermediate certificate and Root certificate
  ServerHost->>IronRoot: mount Intermediate cert/key and CA chain
```

The Root CA private key does not leave the offline machine. The object transferred to the online environment is the Intermediate certificate, not the Root key.

## Offline Storage Architecture

```mermaid
flowchart LR
  RootKey[Encrypted Root CA private key] --> SafeA[Offline backup A]
  RootKey --> SafeB[Offline backup B]
  RootCert[Root CA certificate] --> PublicBundle[Trust bundle distribution]
  CSR[Intermediate CSR] --> OfflineHost[Offline signing host]
  OfflineHost --> IntermediateCert[Signed Intermediate CA certificate]
```

Root CA backups should be encrypted, offline, and stored in separate physical or administrative locations.

## Recovery Guidance

Recovery should be tested before production. A recovery runbook should cover:

- where encrypted Root CA backups are stored
- who can access decryption material
- how to verify backup integrity
- how to sign a replacement Intermediate CA
- how to publish a new trust bundle without exposing Root private key material

## Migration Guidance

```mermaid
timeline
  title Root rotation flow
  Create new Root offline : do not expose private key
  Sign new Intermediate : import Intermediate into IronRoot
  Serve both roots : clients trust old and new chains
  Disable old issuer : no new certificates from old chain
  Retire old root : after old certificates expire
```

IronRoot's CA generation schema is designed for this phased model. Existing certificates can continue until expiry while new certificates are issued from the new Intermediate.
