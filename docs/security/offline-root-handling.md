# Offline Root Handling

The Root CA is the trust anchor. Treat it as offline infrastructure, not an online service dependency.

Recommended posture:

- create the Root CA on an offline or air-gapped machine
- never copy the Root CA private key to the online IronRoot server
- encrypt the Root CA private key at rest
- keep at least two offline backups in separate secure locations
- use the Root CA only to sign Intermediate CA certificates
- never sign normal server certificates directly with the Root CA
- use an approximate 20 year Root CA lifetime

During root migration, serve both old and new roots while issuing from the new Intermediate CA generation. Retire the old root only after old certificates have expired.

