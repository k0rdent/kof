{{- define "repo_chart_name" -}}
{{- if eq .type "oci" }}
chartName: {{ .name }}
{{- else }}
chartName: {{ .repo }}/{{ .name }}
{{- end }}
{{- end -}}
