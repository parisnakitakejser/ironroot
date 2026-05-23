# PKI Concepts

<span class="ironroot-page-status ironroot-page-status--in-progress">Status: In progress</span>

## Offline Root CA

The Root CA is the long-lived trust anchor. IronRoot recommends a 20 year Root CA lifetime and keeping the root private key offline.

## Online Intermediate CA

The Intermediate CA is online and signs workload CSRs. IronRoot recommends a 5 year Intermediate CA lifetime.

## Issued certificates

Server and workload certificates default to 90 days. Renewal is supported before expiry, with a default renewal window of 30 days.

## Private keys

Clients generate private keys locally. IronRoot servers sign CSRs and must not generate normal client or server private keys.

