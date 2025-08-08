{{- define "repo_chart_name" -}}
{{- if eq .type "oci" }}
chartName: {{ .name }}
{{- else }}
chartName: {{ .repo }}/{{ .name }}
{{- end }}
{{- end -}}

{{- define "helm_repo_creds" }}
      registryCredentialsConfig:
        {{- with .secretRef }}
        credentials: {{- . | toYaml | nindent 10 }}
        {{- end }}
        {{- with .certSecretRef }}
        ca: {{- . | toYaml | nindent 10 }}
        {{- end }}
{{- end }}

{{- define "collectors_values_format" -}}
        global:
          clusterName: %s
        kof:
          logs:
            endpoint: http://%s-logs-insert:9481/insert/opentelemetry/v1/logs
          metrics:
            endpoint: http://%s-vminsert:8480/insert/0/prometheus/api/v1/write
          traces:
            endpoint: http://%s-jaeger-collector:4318
          basic_auth: false
        opencost:
          opencost:
            prometheus:
              existingSecretName: ""
              external:
                url: http://%s-vmselect:8481/select/0/prometheus
            exporter:
              defaultClusterId: %s
{{- end }}
