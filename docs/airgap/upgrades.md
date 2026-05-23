# Airgap Upgrades

<span class="ironroot-page-status ironroot-page-status--in-progress">Status: In progress</span>

Airgap upgrades should be staged:

1. Mirror binaries, images, charts, and documentation.
2. Verify checksums.
3. Test in an isolated staging environment.
4. Back up the database and Intermediate CA material.
5. Upgrade IronRoot.
6. Run `ironroot-admin security-check`.
7. Validate enrollment, issuance, renewal, and telemetry.
