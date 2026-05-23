{{- define "ironroot.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "ironroot.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "ironroot.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "ironroot.labels" -}}
helm.sh/chart: {{ include "ironroot.chart" . }}
app.kubernetes.io/name: {{ include "ironroot.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/component: pki-server
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "ironroot.selectorLabels" -}}
app.kubernetes.io/name: {{ include "ironroot.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: pki-server
{{- end -}}

{{- define "ironroot.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "ironroot.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{- define "ironroot.image" -}}
{{- $tag := default .Chart.AppVersion .Values.image.tag -}}
{{- printf "%s:%s" .Values.image.repository $tag -}}
{{- end -}}

{{- define "ironroot.pkiSecretName" -}}
{{- default (printf "%s-ca" (include "ironroot.fullname" .)) .Values.pki.existingSecret -}}
{{- end -}}

{{- define "ironroot.tlsSecretName" -}}
{{- default (printf "%s-tls" (include "ironroot.fullname" .)) .Values.tls.existingSecret -}}
{{- end -}}

{{- define "ironroot.databaseDSN" -}}
{{- if eq .Values.config.database.type "sqlite" -}}
{{- printf "file:%s?_foreign_keys=on" .Values.config.database.sqlite.path -}}
{{- else -}}
{{- printf "postgres://%s@%s:%v/%s" .Values.config.database.postgres.user .Values.config.database.postgres.host .Values.config.database.postgres.port .Values.config.database.postgres.database -}}
{{- end -}}
{{- end -}}
