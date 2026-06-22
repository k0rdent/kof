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

{{/*
Service addresses - host:port without scheme. Single source of truth for all VM/VL/VT service addresses.
*/}}

{{- define "kof-storage.vlSelectAddress" -}}
{{ .Release.Name }}-victoria-logs-cluster-vlselect.{{ .Release.Namespace }}.svc:9471
{{- end -}}

{{- define "kof-storage.vlInsertAddress" -}}
{{ .Release.Name }}-victoria-logs-cluster-vlinsert.{{ .Release.Namespace }}.svc:9481
{{- end -}}

{{- define "kof-storage.vlAuditSelectAddress" -}}
vlselect-audit-logs.{{ .Release.Namespace }}.svc:9471
{{- end -}}

{{- define "kof-storage.vlAuditInsertAddress" -}}
vlinsert-audit-logs.{{ .Release.Namespace }}.svc:9481
{{- end -}}

{{- define "kof-storage.vmSelectAddress" -}}
vmselect-cluster.{{ .Release.Namespace }}.svc:8481
{{- end -}}

{{- define "kof-storage.vmInsertAddress" -}}
vminsert-cluster.{{ .Release.Namespace }}.svc:8480
{{- end -}}

{{- define "kof-storage.vtSelectAddress" -}}
{{ .Release.Name }}-vt-cluster-vtselect.{{ .Release.Namespace }}.svc:10471
{{- end -}}

{{- define "kof-storage.vtInsertAddress" -}}
{{ .Release.Name }}-vt-cluster-vtinsert.{{ .Release.Namespace }}.svc:10481
{{- end -}}

{{/* URL helpers — http:// prepended to the corresponding address helper. */}}

{{- define "kof-storage.vlSelectLocalUrl" -}}
http://{{ include "kof-storage.vlSelectAddress" . }}
{{- end -}}

{{- define "kof-storage.vlInsertLocalUrl" -}}
http://{{ include "kof-storage.vlInsertAddress" . }}
{{- end -}}

{{- define "kof-storage.vlAuditSelectLocalUrl" -}}
http://{{ include "kof-storage.vlAuditSelectAddress" . }}
{{- end -}}

{{- define "kof-storage.vlAuditInsertLocalUrl" -}}
http://{{ include "kof-storage.vlAuditInsertAddress" . }}
{{- end -}}

{{- define "kof-storage.vmSelectLocalUrl" -}}
http://{{ include "kof-storage.vmSelectAddress" . }}
{{- end -}}

{{- define "kof-storage.vmInsertLocalUrl" -}}
http://{{ include "kof-storage.vmInsertAddress" . }}
{{- end -}}

{{- define "kof-storage.vtSelectLocalUrl" -}}
http://{{ include "kof-storage.vtSelectAddress" . }}
{{- end -}}

{{- define "kof-storage.vtInsertLocalUrl" -}}
http://{{ include "kof-storage.vtInsertAddress" . }}
{{- end -}}
