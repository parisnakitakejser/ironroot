# Documentation Standards

<div class="ironroot-doc-meta" markdown>
<span class="ironroot-badge ironroot-badge--stage">Stage: Alpha</span>
<span class="ironroot-badge ironroot-badge--status">Status: Draft</span>
</div>

IronRoot documentation uses metadata badges so readers can quickly understand maturity, completeness, validation, and capability claims.

## Badge Categories

Every page should include stage and status badges directly below the page title.

Validation and capability badges are optional and must be earned by implementation and testing, not added as marketing labels.

## Stage

| Stage | Meaning |
|---|---|
| Concept | Design notes or background material. Behavior may not exist yet. |
| Prototype | Early implementation exists, but workflow may change. |
| Alpha | Implemented enough for active testing and contributor feedback. Breaking changes are expected. |
| Beta | Feature and docs are broadly usable. Interfaces are settling. |
| Stable | Workflow is implemented, tested, documented, and expected to remain compatible. |
| Mature | Long-running, operationally proven, and maintained with compatibility expectations. |

## Status

| Status | Meaning |
|---|---|
| Draft | Page is useful but incomplete or still being shaped. |
| In Progress | Page is actively maintained as part of the current onboarding or feature work. |
| Experimental | Workflow exists but is intentionally exploratory. |
| Stable | Page and workflow are complete and validated for the stated environments. |
| Deprecated | Page describes a workflow that should be avoided or replaced. |

## Validation Badges

Validation badges mean the workflow has been tested end to end in that environment.

| Badge | Requirement |
|---|---|
| Linux Validated | Commands were run successfully on Linux or CI-equivalent Linux. |
| macOS Validated | Commands were run successfully on macOS. |
| Podman Validated | Container flow was tested with Podman. |
| Kubernetes Validated | Manifests or Helm flow were applied to a Kubernetes cluster. |
| Airgap Validated | Offline artifact and trust movement were tested without Internet access. |

Do not add validation badges based only on intent, examples, or partial command checks.

## Capability Badges

Capability badges describe implemented project capabilities that the page uses or explains.

| Badge | Meaning |
|---|---|
| OpenTelemetry Enabled | Workflow emits or configures traces, metrics, or logs. |
| Zero-Trust Ready | Workflow avoids secret sharing and preserves trust boundaries. |
| Helm Supported | Workflow has Helm chart support. |
| GitOps Friendly | Workflow can be managed declaratively. |
| OCI Ready | Workflow uses OCI-compatible artifacts. |
| Security Reviewed | Workflow has had explicit security review. |

## Current Defaults

Most IronRoot pages currently use:

```text
Stage: Alpha
Status: Draft
```

The primary onboarding workflow currently uses:

```text
Stage: Alpha
Status: In Progress
```

## Updating Badges

Badge updates should happen in the same pull request as the implementation, test, or documentation work that justifies the change.

A page should not be marked `Stable` until:

- the implementation exists
- commands have been tested
- examples are accurate
- docs build in strict mode
- expected failure modes are documented
- security implications are explained

Validation badges should include evidence in the pull request summary or linked issue.
