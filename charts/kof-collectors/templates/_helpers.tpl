{{- /* Basic auth extensions */ -}}
{{- define "basic_auth_extensions" -}}
{{- range tuple "metrics" "logs" }}
{{- $secret := (lookup "v1" "Secret" $.Release.Namespace (index $.Values "kof" . "credentials_secret_name")) }}
{{- if not $.Values.global.lint }}
basicauth/{{ . }}:
  client_auth:
    username: {{ index $secret.data (index $.Values "kof" . "username_key") | b64dec | quote }}
    password: {{ index $secret.data (index $.Values "kof" . "password_key") | b64dec | quote }}
{{- end }}
{{- end }}
{{- end }}


{{- define "kof-collectors.helper.tls_options" -}}
{{- $parsedEndpoint := urlParse .endpoint }} 
{{- $nonEmptyOptions :=  or (eq $parsedEndpoint.scheme "http") (not (empty .tls_options |  default dict )) }}
{{- if $nonEmptyOptions }}
tls:
{{- if or (eq $parsedEndpoint.scheme "http") }} 
  insecure: true
{{- else if (eq $parsedEndpoint.scheme "https") }}
{{- range $k, $v := .tls_options }}
  {{ $k }}: {{ $v }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
