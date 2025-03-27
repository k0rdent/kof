{{- define "catalog.serviceTemplate" }}
  {{- if .Values.kcm.installTemplates }}
    {{- $template_name := printf "%s-%s" .templateChart (.templateVersion | replace "." "-") }}
    {{- $template := lookup "k0rdent.mirantis.com/v1alpha1" "ServiceTemplate" .Values.kcm.namespace $template_name }}
    {{- if (not $template) }}
---
apiVersion: k0rdent.mirantis.com/v1alpha1
kind: ServiceTemplate
metadata:
  name: {{ $template_name }}
  namespace: {{ .Values.kcm.namespace }}
  annotations:
    helm.sh/resource-policy: keep
    # To avoid `ServiceTemplate not found` in `MultiClusterService/ClusterDeployment`:
    helm.sh/hook: pre-install
spec:
  helm:
    chartSpec:
      chart: {{ .templateChart }}
      version: {{ .templateVersion }}
      interval: 10m0s
      sourceRef:
        kind: HelmRepository
        name: k0rdent-catalog
    {{- end }}
  {{- end }}
{{- end }}
