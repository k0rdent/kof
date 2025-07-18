{{- define "repo_chart_name" -}}
{{- if eq .type "oci" }}
chartName: {{ .name }}
{{- else }}
chartName: {{ .repo }}/{{ .name }}
{{- end }}
{{- end -}}

{{- define "collectors_values_format" -}}
        opentelemetry-kube-stack:
          collectors:
            collector-k0s:
              enabled: false
          clusterName: %s
          defaultCRConfig:
            config:
              processors:
                resource/k8sclustername:
                  attributes:
                  - action: insert
                    key: k8s.cluster.name
                    value: %s
              exporters:
                debug: {}
                otlphttp/traces:
                  endpoint: http://%s-jaeger-collector:4318
                otlphttp/logs:
                  logs_endpoint: http://%s-logs-insert:9481/insert/opentelemetry/v1/logs
                prometheusremotewrite:
                  external_labels:
                    cluster: %s
                  endpoint: http://%s-vminsert:8480/insert/0/prometheus/api/v1/write
        global:
          clusterName: %s
        opencost:
          opencost:
            prometheus:
              existingSecretName: ""
              external:
                url: http://%s-vmselect:8481/select/0/prometheus
            exporter:
              defaultClusterId: %s
{{- end }}
