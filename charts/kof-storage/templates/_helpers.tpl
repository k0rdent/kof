{{- /* VMRule key */ -}}
{{- define "victoria-metrics-k8s-stack.rulegroup.key" -}}
  {{- without (regexSplit "[-_.]" .name -1) "exporter" "rules" | join "-" | camelcase | untitle -}}
{{- end -}}

{{- define "cert-manager.cluster-issuer.name" -}}
{{- with index . "cert-manager" }}
{{- if index . "enabled" | default false }}
{{- (index . "cluster-issuer" "name" ) | default (printf "%s-%s" (index . "cluster-issuer" "provider") "prod") -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "cert-manager.acme-annotation" -}}
{{- if and (index . "cert-manager" "enabled" | default false) (eq (index . "cert-manager" "cluster-issuer" "provider") "letsencrypt") }}
kubernetes.io/tls-acme: "true"
{{- end -}}
{{- end -}}
