{{- define "dragon-cmk.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "dragon-cmk.fullname" -}}
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

{{- define "dragon-cmk.labels" -}}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}
app.kubernetes.io/name: {{ include "dragon-cmk.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "dragon-cmk.namespace" -}}
{{- default .Release.Namespace .Values.namespace.name -}}
{{- end -}}

{{- define "dragon-cmk.selectorLabels" -}}
app.kubernetes.io/name: {{ include "dragon-cmk.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "dragon-cmk.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "dragon-cmk.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{- define "dragon-cmk.postgresqlSecretName" -}}
{{- if .Values.postgresql.existingSecret -}}
{{- .Values.postgresql.existingSecret -}}
{{- else -}}
{{- printf "%s-postgresql" (include "dragon-cmk.fullname" .) -}}
{{- end -}}
{{- end -}}

{{- define "dragon-cmk.tlsSecretName" -}}
{{- if .Values.tls.existingSecret -}}
{{- .Values.tls.existingSecret -}}
{{- else if .Values.certManager.server.secretName -}}
{{- .Values.certManager.server.secretName -}}
{{- else -}}
{{- printf "%s-server-tls" (include "dragon-cmk.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "dragon-cmk.clientTlsSecretName" -}}
{{- if .Values.certManager.client.secretName -}}
{{- .Values.certManager.client.secretName -}}
{{- else -}}
{{- printf "%s-client-tls" (include "dragon-cmk.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "dragon-cmk.postgresqlName" -}}
{{- printf "%s-postgresql" (include "dragon-cmk.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "dragon-cmk.postgresqlPersistentVolumeName" -}}
{{- printf "%s-pv" (include "dragon-cmk.postgresqlName" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "dragon-cmk.postgresqlPersistentVolumeClaimName" -}}
{{- printf "%s-pvc" (include "dragon-cmk.postgresqlName" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "dragon-cmk.postgresqlSelectorLabels" -}}
app.kubernetes.io/name: {{ include "dragon-cmk.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: postgresql
{{- end -}}
