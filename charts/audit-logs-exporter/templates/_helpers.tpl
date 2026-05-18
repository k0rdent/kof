{{/*
Expand the name of the chart.
*/}}
{{- define "audit-logs-exporter.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "audit-logs-exporter.fullname" -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels.
*/}}
{{- define "audit-logs-exporter.labels" -}}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
{{ include "audit-logs-exporter.selectorLabels" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels.
*/}}
{{- define "audit-logs-exporter.selectorLabels" -}}
app.kubernetes.io/name: {{ include "audit-logs-exporter.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
ServiceAccount name.
*/}}
{{- define "audit-logs-exporter.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
  {{- default (include "audit-logs-exporter.fullname" .) .Values.serviceAccount.name }}
{{- else }}
  {{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Name of the Secret holding S3 credentials.
*/}}
{{- define "audit-logs-exporter.secretName" -}}
{{- if .Values.s3.existingSecret }}
  {{- .Values.s3.existingSecret }}
{{- else }}
  {{- .Values.s3.secretName | default (printf "%s-s3-credentials" (include "audit-logs-exporter.fullname" .)) }}
{{- end }}
{{- end }}

{{/*
Producer version — falls back to chart appVersion.
*/}}
{{- define "audit-logs-exporter.producerVersion" -}}
{{- .Values.producerVersion | default .Chart.AppVersion }}
{{- end }}
