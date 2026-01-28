{{/*
Expand the name of the chart.
*/}}
{{- define "kof.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "kof.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "kof.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "kof.labels" -}}
helm.sh/chart: {{ include "kof.chart" . }}
{{ include "kof.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
k0rdent.mirantis.com/managed: "true"
{{- end }}

{{/*
Selector labels
*/}}
{{- define "kof.selectorLabels" -}}
app.kubernetes.io/name: {{ include "kof.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
HelmRelease namespace
*/}}
{{- define "kof.namespace" -}}
{{- .Values.global.namespace | default "kof" }}
{{- end }}

{{/*
HelmRelease namespace
*/}}
{{- define "helmchart.namespace" -}}
{{- .Values.global.helmRepo.enabled | ternary (include "kof.namespace" . ) .Values.global.helmRepo.namespace }}
{{- end }}
