# Airgap Upgrades

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status">Status: Draft</span>
</div>

Airgap upgrades should be staged:

1. Mirror binaries, images, charts, and documentation.
2. Verify checksums.
3. Test in an isolated staging environment.
4. Back up the database and Intermediate CA material.
5. Upgrade IronRoot.
6. Run `ironroot-admin security-check`.
7. Validate enrollment, issuance, renewal, and telemetry.
