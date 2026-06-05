{{/*
Expand the name of the chart.
*/}}
{{- define "cold-storage-exporter.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "cold-storage-exporter.fullname" -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels.
*/}}
{{- define "cold-storage-exporter.labels" -}}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
{{ include "cold-storage-exporter.selectorLabels" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels.
*/}}
{{- define "cold-storage-exporter.selectorLabels" -}}
app.kubernetes.io/name: {{ include "cold-storage-exporter.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
ServiceAccount name.
*/}}
{{- define "cold-storage-exporter.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
  {{- default (include "cold-storage-exporter.fullname" .) .Values.serviceAccount.name }}
{{- else }}
  {{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Name of the Secret holding S3 credentials.
*/}}
{{- define "cold-storage-exporter.secretName" -}}
{{- if .Values.s3.existingSecret }}
  {{- .Values.s3.existingSecret }}
{{- else }}
  {{- .Values.s3.secretName | default (printf "%s-s3-credentials" (include "cold-storage-exporter.fullname" .)) }}
{{- end }}
{{- end }}
