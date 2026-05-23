# Threat Model

<span class="ironroot-page-status ironroot-page-status--in-progress">Status: In progress</span>

Primary threats include Root CA key exposure, Intermediate CA key exposure, unauthorized enrollment, stolen bootstrap tokens, weak filesystem permissions, missing audit logs, and unmonitored certificate issuance spikes.

The main security boundary is between the offline Root CA and the online IronRoot server.
