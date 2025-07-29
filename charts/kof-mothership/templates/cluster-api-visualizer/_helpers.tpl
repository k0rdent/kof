{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "cluster-api-visualizer.name" -}}
{{- default .Chart.Name (index .Values "cluster-api-visualizer").nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
NOTE that despite the naming `cluster-api-visualizer.fullname` it does not include the "cluster-api-visualizer" string,
it is just "kof-mothership" at the moment.
*/}}
{{- define "cluster-api-visualizer.fullname" -}}
{{- if (index .Values "cluster-api-visualizer").fullnameOverride -}}
{{- (index .Values "cluster-api-visualizer").fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name (index .Values "cluster-api-visualizer").nameOverride -}}
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
{{- define "cluster-api-visualizer.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "cluster-api-visualizer.labels" -}}
helm.sh/chart: {{ include "cluster-api-visualizer.chart" . }}
{{ include "cluster-api-visualizer.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Selector labels
*/}}
{{- define "cluster-api-visualizer.selectorLabels" -}}
app.kubernetes.io/name: {{ include "cluster-api-visualizer.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}
