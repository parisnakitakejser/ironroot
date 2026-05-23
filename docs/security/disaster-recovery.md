# Disaster Recovery

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status">Status: Draft</span>
</div>

Restore database and Intermediate CA material first. If the Intermediate CA is lost, create and import a new Intermediate signed by the offline Root. If the Root CA is lost, recover from offline backups or perform Root migration.
