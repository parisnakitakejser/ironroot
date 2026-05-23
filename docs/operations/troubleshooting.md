# Troubleshooting

<span class="ironroot-page-status ironroot-page-status--in-progress">Status: In progress</span>

Start with `/healthz`, `/readyz`, server logs, and traces. Enrollment failures usually involve expired tokens, hostname mismatch, or machine-id mismatch. TLS failures usually involve trust bundle installation or missing intermediates.
