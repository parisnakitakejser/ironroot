# Helm Releases

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status">Status: Draft</span>
</div>

Release tags publish both a container image and Helm chart.

RC tag behavior for `v0.1.0-rc.1`:

- pushes `ghcr.io/OWNER/ironroot:v0.1.0-rc.1`
- packages chart version `0.1.0-rc.1`
- sets chart `appVersion` to `v0.1.0-rc.1`
- pushes `oci://ghcr.io/OWNER/charts/ironroot`
- uploads the chart `.tgz` to a prerelease GitHub Release

Stable tag behavior for `v0.1.0`:

- pushes `ghcr.io/OWNER/ironroot:v0.1.0`
- pushes `ghcr.io/OWNER/ironroot:latest`
- packages chart version `0.1.0`
- pushes the Helm OCI artifact
- uploads the chart `.tgz` to a stable GitHub Release

Supported tag patterns:

- `v*.*.*-rc.*`
- `v*.*.*`
- `chart-v*.*.*`
