{{- define "cluster_exporters" }}
prometheusremotewrite:
  endpoint: {{ .kof.metrics.endpoint }}
  {{- include "kof-collectors.helper.tls_options" .kof.metrics | indent 2 }}
  {{- if .kof.basic_auth }}
  auth:
    authenticator: basicauth/metrics
  {{- end }}
otlphttp:
  {{- if .kof.basic_auth }}
  auth:
    authenticator: basicauth/logs
  {{- end }}
  {{- include "kof-collectors.helper.tls_options" .kof.logs | indent 2 }}
  logs_endpoint: {{ .kof.logs.endpoint }}
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
otlphttp/traces:
  endpoint: {{ .kof.traces.endpoint }}
  {{- include "kof-collectors.helper.tls_options" .kof.traces | indent 2 }}
prometheusremotewrite:
  endpoint: {{ .kof.metrics.endpoint }}
  {{- include "kof-collectors.helper.tls_options" .kof.metrics | indent 2 }}
  {{- if .kof.basic_auth }}
  auth:
    authenticator: basicauth/metrics
  {{- end }}
otlphttp/logs:
  {{- if .kof.basic_auth }}
  auth:
    authenticator: basicauth/logs
  {{- end }}
  logs_endpoint: {{ .kof.logs.endpoint }}
  {{- include "kof-collectors.helper.tls_options" .kof.logs | indent 2 }}
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
