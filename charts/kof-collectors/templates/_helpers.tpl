{{/*
Expand the name of the chart.
*/}}
{{- define "opentelemetry-kube-stack.name" -}}
{{- default .Chart.Name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "opentelemetry-kube-stack.fullname" -}}
{{- if .fullnameOverride }}
{{- .fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{/*
Allow the release namespace to be overridden
*/}}
{{- define "opentelemetry-kube-stack.namespace" -}}
  {{- if .Values.namespaceOverride -}}
    {{- .Values.namespaceOverride -}}
  {{- else -}}
    {{- .Release.Namespace -}}
  {{- end -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "opentelemetry-kube-stack.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "opentelemetry-kube-stack.labels" -}}
helm.sh/chart: {{ include "opentelemetry-kube-stack.chart" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
release: {{ .Release.Name | quote }}
{{- end }}

{{/* Sets default scrape limits for servicemonitor */}}
{{- define "opentelemetry-kube-stack.servicemonitor.scrapeLimits" -}}
{{- with .sampleLimit }}
sampleLimit: {{ . }}
{{- end }}
{{- with .targetLimit }}
targetLimit: {{ . }}
{{- end }}
{{- with .labelLimit }}
labelLimit: {{ . }}
{{- end }}
{{- with .labelNameLengthLimit }}
labelNameLengthLimit: {{ . }}
{{- end }}
{{- with .labelValueLengthLimit }}
labelValueLengthLimit: {{ . }}
{{- end }}
{{- end -}}

{{- define "cluster_processors" }}
  resource/k8sobjects:
    attributes:
      - action: insert
        value: {{ .Values.global.clusterName }}
        key: k8s.cluster.name
{{- end }}

{{- define "cluster_exporters" }}
{{- if .kof.metrics.endpoint }}
prometheusremotewrite:
  endpoint: {{ .kof.metrics.endpoint }}
  {{- include "kof-collectors.helper.tls_options" .kof.metrics | indent 2 }}
  {{- if .kof.basic_auth }}
  auth:
    authenticator: basicauth/metrics
  {{- end }}
{{- end }}
{{- if .kof.logs.endpoint }}
otlphttp:
  {{- if .kof.basic_auth }}
  auth:
    authenticator: basicauth/logs
  {{- end }}
  {{- include "kof-collectors.helper.tls_options" .kof.logs | indent 2 }}
  logs_endpoint: {{ .kof.logs.endpoint }}
{{- end }}
{{- end }}

{{- define "node_processors" }}
  resource/k8sobjects:
    attributes:
      - action: insert
        value: {{ .Values.global.clusterName }}
        key: k8s.cluster.name
{{- end }}

{{- define "node_receivers" }}
{{- if .Values.collectors.node.receivers.prometheus }}
prometheus:
  config:
    global:
      external_labels:
        {{ .Values.global.clusterLabel }}: {{ .Values.global.clusterName }}
{{- end }}
{{- end }}

{{- define "node_exporters" }}
{{- if .kof.traces.endpoint }}
otlphttp/traces:
  endpoint: {{ .kof.traces.endpoint }}
  {{- include "kof-collectors.helper.tls_options" .kof.traces | indent 2 }}
{{- end }}
{{- if .kof.metrics.endpoint }}
prometheusremotewrite:
  endpoint: {{ .kof.metrics.endpoint }}
  {{- include "kof-collectors.helper.tls_options" .kof.metrics | indent 2 }}
  {{- if .kof.basic_auth }}
  auth:
    authenticator: basicauth/metrics
  {{- end }}
{{- end }}
{{- if .kof.logs.endpoint }}
otlphttp/logs:
  {{- if .kof.basic_auth }}
  auth:
    authenticator: basicauth/logs
  {{- end }}
  logs_endpoint: {{ .kof.logs.endpoint }}
  {{- include "kof-collectors.helper.tls_options" .kof.logs | indent 2 }}
{{- end }}
{{- end }}

{{- define "service" }}
{{- if .Values.kof.basic_auth }}
extensions:
  - basicauth/metrics
  - basicauth/logs
{{- end }}
{{- end }}

{{- /* Basic auth extensions */ -}}
{{- define "basic_auth_extensions" -}}
{{- if .Values.kof.basic_auth }}
{{- range tuple "metrics" "logs" }}
{{- $secret := (lookup "v1" "Secret" $.Release.Namespace (index $.Values "kof" . "credentials_secret_name")) }}
{{- if not (empty $secret) }}
{{- if not $.Values.global.lint }}
basicauth/{{ . }}:
  client_auth:
    username: {{ index $secret.data (index $.Values "kof" . "username_key") | b64dec | quote }}
    password: {{ index $secret.data (index $.Values "kof" . "password_key") | b64dec | quote }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}

{{- define "kof-collectors.helper.tls_options" -}}
{{- $parsedEndpoint := urlParse .endpoint }} 
{{- if eq $parsedEndpoint.scheme "http" }}
tls:
  insecure: true
{{- else if eq $parsedEndpoint.scheme "https" }}
  {{- with .tls_options }}
tls: {{ . | toYaml | nindent 2 }}
  {{- end }}
{{- end }}
{{- end }}
