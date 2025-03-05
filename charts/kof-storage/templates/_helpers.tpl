{{- /* VMRule key */ -}}
{{- define "victoria-metrics-k8s-stack.rulegroup.key" -}}
  {{- without (regexSplit "[-_.]" .name -1) "exporter" "rules" | join "-" | camelcase | untitle -}}
{{- end -}}

{{- define "cert-manager.cluster-issuer.name" -}}
{{- with index .Values "cert-manager" }}
{{- (index . "cluster-issuer" "name" ) | default (printf "%s-%s" (index . "cluster-issuer" "provider") "prod") }}
{{- end -}}
{{- end -}}

{{- define "cert-manager.acme-annotation" -}}
{{- if and (index .Values "cert-manager" "enabled") (eq (index .Values "cert-manager" "cluster-issuer" "provider") "letsencrypt") }}
kubernetes.io/tls-acme: "true"
{{- end -}}
{{- end -}}
