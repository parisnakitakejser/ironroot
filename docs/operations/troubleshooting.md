# Troubleshooting

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status">Status: Draft</span>
</div>

Start with `/healthz`, `/readyz`, server logs, and traces. Enrollment failures usually involve expired tokens, hostname mismatch, or machine-id mismatch. TLS failures usually involve trust bundle installation or missing intermediates.
