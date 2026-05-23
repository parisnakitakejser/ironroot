# Setup Guide

<span class="ironroot-page-status ironroot-page-status--in-progress">Status: In progress</span>

Create the offline Root CA outside the server runtime. Keep the root private key disconnected from the online environment.

Create or import an Intermediate CA signed by the Root CA, then mount the following files into the server:

- `root-ca.crt`
- `ca-chain.crt`
- `intermediate.crt`
- `intermediate.key` or encrypted equivalent

The default config expects local development paths under `./pki`. Container and Kubernetes deployments mount CA material at `/pki`.

