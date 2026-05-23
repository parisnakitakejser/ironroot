# Distributing Trust

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status">Status: Draft</span>
</div>

Distribute the Root CA public certificate to clients, servers, browsers, Kubernetes workloads, and container runtimes that must trust IronRoot certificates.

The trust bundle is public certificate material. It is still security-sensitive because installing it grants trust in certificates issued under that Root CA.

Use controlled package channels, GitOps repositories, golden images, MDM, or offline media depending on your environment.
