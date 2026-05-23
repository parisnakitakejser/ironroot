#!/usr/bin/env sh
set -eu

CHART="${1:-deploy/helm/ironroot}"
TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

helm lint "$CHART"
helm template ironroot "$CHART" > "$TMPDIR/default.yaml"
helm template ironroot "$CHART" --set persistence.enabled=false > "$TMPDIR/no-persistence.yaml"
helm template ironroot "$CHART" -f "$CHART/examples/otel-values.yaml" > "$TMPDIR/otel.yaml"
helm template ironroot "$CHART" -f "$CHART/examples/ingress-values.yaml" > "$TMPDIR/ingress.yaml"
helm template ironroot "$CHART" -f "$CHART/examples/postgres-values.yaml" > "$TMPDIR/postgres.yaml"
helm template ironroot "$CHART" --set networkPolicy.enabled=true > "$TMPDIR/networkpolicy.yaml"
helm template ironroot "$CHART" --set serviceMonitor.enabled=true > "$TMPDIR/servicemonitor.yaml"

grep -q "runAsNonRoot: true" "$TMPDIR/default.yaml"
grep -q "readOnlyRootFilesystem: true" "$TMPDIR/default.yaml"
grep -q "allowPrivilegeEscalation: false" "$TMPDIR/default.yaml"
grep -q "drop:" "$TMPDIR/default.yaml"
grep -q "RuntimeDefault" "$TMPDIR/default.yaml"

if command -v kubeconform >/dev/null 2>&1; then
  kubeconform -strict -ignore-missing-schemas "$TMPDIR/default.yaml"
  kubeconform -strict -ignore-missing-schemas "$TMPDIR/no-persistence.yaml"
  kubeconform -strict -ignore-missing-schemas "$TMPDIR/otel.yaml"
  kubeconform -strict -ignore-missing-schemas "$TMPDIR/ingress.yaml"
  kubeconform -strict -ignore-missing-schemas "$TMPDIR/postgres.yaml"
  kubeconform -strict -ignore-missing-schemas "$TMPDIR/networkpolicy.yaml"
  kubeconform -strict -ignore-missing-schemas "$TMPDIR/servicemonitor.yaml"
fi
