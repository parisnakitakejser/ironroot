# Root CA Handling

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status">Status: Draft</span>
</div>

The Root CA should be generated and stored offline. It should not run continuously and should only sign Intermediate CA certificates.

Do not place the Root CA private key:

- On the IronRoot server.
- In Kubernetes.
- In a container image.
- In CI.
- In a shared file server.

Back up the encrypted Root CA key to at least two offline locations. Test recovery before production use.
