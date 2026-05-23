# Threat Model

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status">Status: Draft</span>
</div>

Primary threats include Root CA key exposure, Intermediate CA key exposure, unauthorized enrollment, stolen bootstrap tokens, weak filesystem permissions, missing audit logs, and unmonitored certificate issuance spikes.

The main security boundary is between the offline Root CA and the online IronRoot server.
