{{/* vim: set filetype=mustache: */}}
{{/*
ACL labels
*/}}
{{- define "acl.labels" -}}
helm.sh/chart: {{ include "operator.chart" . }}
{{ include "acl.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/component: acl
{{- end -}}

{{/*
ACL Selector labels
*/}}
{{- define "acl.selectorLabels" -}}
app.kubernetes.io/name: {{ include "operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}-acl
{{- end -}}
