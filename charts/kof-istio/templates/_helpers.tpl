{{- define "repo_chart_name" -}}
{{- if eq .type "oci" }}
chartName: {{ .name }}
{{- else }}
chartName: {{ .repo }}/{{ .name }}
{{- end }}
{{- end -}}

{{- define "collectors_values_format" -}}
        global:
          clusterName: {childClusterName}
          clusterNamespace: {childClusterNamespace}
        opentelemetry-kube-stack:
          collectors:
            controller-k0s:
              enabled: false
          clusterName: {childClusterName}
          defaultCRConfig:
            config:
              processors:
                resource/k8sclustername:
                  attributes:
                  - action: insert
                    key: k8s.cluster.name
                    value: {childClusterName}
                  - action: insert
                    key: k8s.cluster.namespace
                    value: {childClusterNamespace}
              exporters:
                debug: {}
                otlphttp/traces:
                  endpoint: http://{regionalClusterName}-jaeger-collector:4318
                otlphttp/logs:
                  logs_endpoint: http://{regionalClusterName}-logs-insert:9481/insert/opentelemetry/v1/logs
                prometheusremotewrite:
                  external_labels:
                    cluster: {childClusterName}
                    clusterNamespace: {childClusterNamespace}
                  endpoint: http://{regionalClusterName}-vminsert:8480/insert/0/prometheus/api/v1/write
        opencost:
          opencost:
            prometheus:
              existingSecretName: ""
              external:
                url: http://{regionalClusterName}-vmselect:8481/select/0/prometheus
            exporter:
              defaultClusterId: {childClusterName}
{{- end }}
