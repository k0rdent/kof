{{- define "cert-manager.cluster-issuer.name" -}}
{{- with index .Values "cert-manager" }}
{{- (index . "cluster-issuer" "name" ) | default (printf "%s-%s" (index . "cluster-issuer" "provider") "prod") }}
{{- end -}}
{{- end -}}

{{- define "cert-manager.acme-annotation" -}}
{{- if and (index .Values "cert-manager" "enabled") (eq (index .Values "cert-manager" "cluster-issuer" "provider") "letsencrypt") }}
kubernetes.io/tls-acme: "true"
{{- end -}}
{{- end -}}

{{/*
Expand the name of the chart.
*/}}
{{- define "mothership.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "mothership.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if .Release.Name | contains $name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "mothership.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "mothership.labels" -}}
helm.sh/chart: {{ include "mothership.chart" . }}
{{ include "mothership.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Selector labels
*/}}
{{- define "mothership.selectorLabels" -}}
app.kubernetes.io/name: {{ include "mothership.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

