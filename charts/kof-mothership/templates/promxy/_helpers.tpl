{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "promxy.name" -}}
{{- default .Chart.Name .Values.promxy.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "promxy.fullname" -}}
{{- if .Values.promxy.fullnameOverride -}}
{{- .Values.promxy.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.promxy.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "promxy.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "promxy.labels" -}}
helm.sh/chart: {{ include "promxy.chart" . }}
{{ include "promxy.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Selector labels
*/}}
{{- define "promxy.selectorLabels" -}}
app.kubernetes.io/name: {{ include "promxy.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/*
Create the name of the service account to use
*/}}
{{- define "promxy.serviceAccountName" -}}
{{- if .Values.promxy.serviceAccount.create -}}
    {{ default (printf "%s-promxy" (include "promxy.fullname" .)) .Values.promxy.serviceAccount.name }}
{{- else -}}
    {{ default "default" .Values.promxy.serviceAccount.name }}
{{- end -}}
{{- end -}}

{{/*
Defins the name of secret
*/}}
{{- define "promxy.secretname" -}}
{{- if .Values.promxy.secret -}}
{{- .Values.promxy.secret -}}
{{- else -}}
{{- include "promxy.fullname" . -}}-promxy-config
{{- end -}}
{{- end -}}

{{- define "split-host-port" -}}
{{- $hp := split ":" . -}}
{{- printf "%s" $hp._1 -}}
{{- end -}}
