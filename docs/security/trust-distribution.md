# Trust Distribution

<span class="ironroot-page-status ironroot-page-status--in-progress">Status: In progress</span>

Trust distribution installs the Root CA certificate into systems that need to validate IronRoot-issued certificates. It does not install private keys.

For service configuration, `ironroot-client request-cert` writes both `tls.crt` and `fullchain.crt`. Browsers trust the service only when the Root CA is installed in the OS or browser trust store and the service presents a chain from the leaf certificate to the Intermediate CA.

## Which File Is The Trust Anchor?

Install the public Root CA certificate as trust material:

```text
root-ca.crt
trust-bundle/root-ca.crt
```

Do not install private keys into trust stores. Do not copy `root-ca.key` or `intermediate-ca.key` to client machines.

The Intermediate certificate is normally served as part of the certificate chain, not installed as the primary trust anchor. Use `fullchain.crt` or `ca-chain.crt` for services that need to present or validate the chain.

Linux:

```bash
sudo cp root-ca.crt /usr/local/share/ca-certificates/ironroot.crt
sudo update-ca-certificates
```

Fedora/RHEL:

```bash
sudo trust anchor root-ca.crt
sudo update-ca-trust
```

macOS:

```bash
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain root-ca.crt
```

Windows:

Use MMC Certificates for the local computer and import the Root CA into **Trusted Root Certification Authorities**.

Firefox:

Import the Root CA under **Settings → Privacy & Security → Certificates → Authorities**.

Kubernetes:

Store trust bundles in ConfigMaps for workloads and mount them into containers. Keep Intermediate CA private keys in Secrets with strict RBAC.

Podman containers:

Mount the trust bundle read-only and update the image trust store at startup or build a base image that contains only the public Root CA certificate.
