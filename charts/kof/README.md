# kof

![Version: 1.8.0-rc0](https://img.shields.io/badge/Version-1.8.0--rc0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 1.8.0-rc0](https://img.shields.io/badge/AppVersion-1.8.0--rc0-informational?style=flat-square)

KOF umbrella Helm chart that uses FluxCD to manage sequential installation of KOF components

**Homepage:** <https://github.com/k0rdent/kof>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| k0rdent |  | <https://github.com/k0rdent> |

## Source Code

* <https://github.com/k0rdent/kof>

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| flux<br>.install<br>.createNamespace | bool | `true` |  |
| flux<br>.install<br>.remediation<br>.retries | int | `5` |  |
| flux<br>.retries | int | `5` |  |
| flux<br>.timeout | string | `"20m"` |  |
| flux<br>.upgrade<br>.retryInterval | string | `"1m"` |  |
| global<br>.components[0] | string | `"kof-operators"` |  |
| global<br>.components[1] | string | `"kof-mothership"` |  |
| global<br>.components[2] | string | `"kof-regional"` |  |
| global<br>.components[3] | string | `"kof-child"` |  |
| global<br>.components[4] | string | `"kof-storage"` |  |
| global<br>.components[5] | string | `"kof-collectors"` |  |
| global<br>.helmRepo<br>.existing<br>.namespace | string | `"kcm-system"` |  |
| global<br>.helmRepo<br>.interval | string | `"10m"` |  |
| global<br>.helmRepo<br>.kofManaged<br>.enabled | bool | `true` |  |
| global<br>.helmRepo<br>.kofManaged<br>.insecure | bool | `false` |  |
| global<br>.helmRepo<br>.kofManaged<br>.type | string | `"oci"` |  |
| global<br>.helmRepo<br>.kofManaged<br>.url | string | `"oci://ghcr.io/k0rdent/kof/charts"` |  |
| global<br>.helmRepo<br>.name | string | `"kof-repo"` |  |
| global<br>.namespace | string | `"kof"` |  |
| global<br>.registry | string | `"ghcr.io/k0rdent"` |  |
| global<br>.storageClass | string | `""` |  |
| kof-child<br>.dependsOn[0] | string | `"kof-mothership"` |  |
| kof-child<br>.enabled | bool | `true` |  |
| kof-child<br>.notes | string | `"Child cluster templates"` |  |
| kof-child<br>.values<br>.collectors<br>.resources<br>.limits<br>.cpu | string | `"500m"` |  |
| kof-child<br>.values<br>.collectors<br>.resources<br>.limits<br>.memory | string | `"512Mi"` |  |
| kof-child<br>.values<br>.collectors<br>.resources<br>.requests<br>.cpu | string | `"100m"` |  |
| kof-child<br>.values<br>.collectors<br>.resources<br>.requests<br>.memory | string | `"256Mi"` |  |
| kof-collectors<br>.dependsOn[0] | string | `"kof-storage"` |  |
| kof-collectors<br>.enabled | bool | `false` |  |
| kof-collectors<br>.notes | string | `"Opentelemetry collectors and OpenCost"` |  |
| kof-collectors<br>.values<br>.collectors<br>.enabled | bool | `true` |  |
| kof-collectors<br>.values<br>.metrics-server<br>.enabled | bool | `false` |  |
| kof-collectors<br>.values<br>.opencost<br>.enabled | bool | `true` |  |
| kof-mothership<br>.dependsOn[0] | string | `"kof-operators"` |  |
| kof-mothership<br>.dependsOn[1] | string | `"victoria-metrics-operator"` |  |
| kof-mothership<br>.enabled | bool | `true` |  |
| kof-mothership<br>.notes | string | `"Management cluster components"` |  |
| kof-mothership<br>.values<br>.cert-manager-service-template<br>.enabled | bool | `true` |  |
| kof-mothership<br>.values<br>.cert-manager<br>.enabled | bool | `true` |  |
| kof-mothership<br>.values<br>.cluster-api-visualizer<br>.enabled | bool | `false` |  |
| kof-mothership<br>.values<br>.defaultRules<br>.create | bool | `true` |  |
| kof-mothership<br>.values<br>.defaultRules<br>.rules<br>.general | bool | `true` |  |
| kof-mothership<br>.values<br>.defaultRules<br>.rules<br>.kubernetesApps | bool | `true` |  |
| kof-mothership<br>.values<br>.defaultRules<br>.rules<br>.kubernetesResources | bool | `true` |  |
| kof-mothership<br>.values<br>.defaultRules<br>.rules<br>.kubernetesStorage | bool | `true` |  |
| kof-mothership<br>.values<br>.defaultRules<br>.rules<br>.kubernetesSystem | bool | `true` |  |
| kof-mothership<br>.values<br>.defaultRules<br>.rules<br>.node | bool | `true` |  |
| kof-mothership<br>.values<br>.dex<br>.config<br>.issuer | string | `"https://dex.example.com"` |  |
| kof-mothership<br>.values<br>.dex<br>.config<br>.staticClients[0]<br>.id | string | `"grafana-id"` |  |
| kof-mothership<br>.values<br>.dex<br>.config<br>.staticClients[0]<br>.name | string | `"Grafana"` |  |
| kof-mothership<br>.values<br>.dex<br>.config<br>.staticClients[0]<br>.redirectURIs[0] | string | `"https://grafana.example.com/login/generic_oauth"` |  |
| kof-mothership<br>.values<br>.dex<br>.config<br>.staticClients[0]<br>.secret | string | `"grafana-secret"` |  |
| kof-mothership<br>.values<br>.dex<br>.enabled | bool | `false` |  |
| kof-mothership<br>.values<br>.global<br>.storageClass | string | `""` |  |
| kof-mothership<br>.values<br>.grafana<br>.enabled | bool | `false` |  |
| kof-mothership<br>.values<br>.grafana<br>.ingress<br>.enabled | bool | `false` |  |
| kof-mothership<br>.values<br>.grafana<br>.ingress<br>.host | string | `"grafana.example.com"` |  |
| kof-mothership<br>.values<br>.grafana<br>.security<br>.create_secret | bool | `true` |  |
| kof-mothership<br>.values<br>.kcm<br>.kof<br>.mcs | object | `{}` |  |
| kof-mothership<br>.values<br>.kcm<br>.kof<br>.operator<br>.crossNamespace | bool | `false` |  |
| kof-mothership<br>.values<br>.kcm<br>.kof<br>.operator<br>.enabled | bool | `true` |  |
| kof-mothership<br>.values<br>.kcm<br>.kof<br>.operator<br>.image<br>.pullPolicy | string | `"IfNotPresent"` |  |
| kof-mothership<br>.values<br>.kcm<br>.kof<br>.operator<br>.image<br>.registry | string | `"ghcr.io/k0rdent"` |  |
| kof-mothership<br>.values<br>.kcm<br>.kof<br>.operator<br>.image<br>.repository | string | `"kof/kof-operator-controller"` |  |
| kof-mothership<br>.values<br>.kcm<br>.kof<br>.operator<br>.resources<br>.limits<br>.cpu | string | `"500m"` |  |
| kof-mothership<br>.values<br>.kcm<br>.kof<br>.operator<br>.resources<br>.limits<br>.memory | string | `"512Mi"` |  |
| kof-mothership<br>.values<br>.kcm<br>.kof<br>.operator<br>.resources<br>.requests<br>.cpu | string | `"200m"` |  |
| kof-mothership<br>.values<br>.kcm<br>.kof<br>.operator<br>.resources<br>.requests<br>.memory | string | `"256Mi"` |  |
| kof-mothership<br>.values<br>.kcm<br>.namespace | string | `"kcm-system"` |  |
| kof-mothership<br>.values<br>.kube-state-metrics<br>.enabled | bool | `true` |  |
| kof-mothership<br>.values<br>.metrics-server<br>.enabled | bool | `false` |  |
| kof-mothership<br>.values<br>.mothershipRules<br>.create | bool | `true` |  |
| kof-mothership<br>.values<br>.promxy<br>.enabled | bool | `true` |  |
| kof-mothership<br>.values<br>.promxy<br>.replicaCount | int | `3` |  |
| kof-mothership<br>.values<br>.promxy<br>.resources<br>.limits<br>.cpu | string | `"2000m"` |  |
| kof-mothership<br>.values<br>.promxy<br>.resources<br>.limits<br>.memory | string | `"2Gi"` |  |
| kof-mothership<br>.values<br>.promxy<br>.resources<br>.requests<br>.cpu | string | `"200m"` |  |
| kof-mothership<br>.values<br>.promxy<br>.resources<br>.requests<br>.memory | string | `"1Gi"` |  |
| kof-mothership<br>.values<br>.victoria-metrics-operator<br>.enabled | bool | `true` |  |
| kof-mothership<br>.values<br>.victoriametrics<br>.enabled | bool | `true` |  |
| kof-mothership<br>.values<br>.victoriametrics<br>.vmcluster<br>.enabled | bool | `true` |  |
| kof-mothership<br>.values<br>.victoriametrics<br>.vmcluster<br>.spec<br>.replicationFactor | int | `2` |  |
| kof-mothership<br>.values<br>.victoriametrics<br>.vmcluster<br>.spec<br>.retentionPeriod | string | `"30d"` |  |
| kof-mothership<br>.values<br>.victoriametrics<br>.vmcluster<br>.spec<br>.vminsert<br>.replicaCount | int | `2` |  |
| kof-mothership<br>.values<br>.victoriametrics<br>.vmcluster<br>.spec<br>.vminsert<br>.resources<br>.limits<br>.cpu | string | `"1000m"` |  |
| kof-mothership<br>.values<br>.victoriametrics<br>.vmcluster<br>.spec<br>.vminsert<br>.resources<br>.limits<br>.memory | string | `"1Gi"` |  |
| kof-mothership<br>.values<br>.victoriametrics<br>.vmcluster<br>.spec<br>.vminsert<br>.resources<br>.requests<br>.cpu | string | `"200m"` |  |
| kof-mothership<br>.values<br>.victoriametrics<br>.vmcluster<br>.spec<br>.vminsert<br>.resources<br>.requests<br>.memory | string | `"512Mi"` |  |
| kof-mothership<br>.values<br>.victoriametrics<br>.vmcluster<br>.spec<br>.vmselect<br>.replicaCount | int | `2` |  |
| kof-mothership<br>.values<br>.victoriametrics<br>.vmcluster<br>.spec<br>.vmselect<br>.resources<br>.limits<br>.cpu | string | `"1000m"` |  |
| kof-mothership<br>.values<br>.victoriametrics<br>.vmcluster<br>.spec<br>.vmselect<br>.resources<br>.limits<br>.memory | string | `"2Gi"` |  |
| kof-mothership<br>.values<br>.victoriametrics<br>.vmcluster<br>.spec<br>.vmselect<br>.resources<br>.requests<br>.cpu | string | `"200m"` |  |
| kof-mothership<br>.values<br>.victoriametrics<br>.vmcluster<br>.spec<br>.vmselect<br>.resources<br>.requests<br>.memory | string | `"1Gi"` |  |
| kof-mothership<br>.values<br>.victoriametrics<br>.vmcluster<br>.spec<br>.vmselect<br>.storage<br>.volumeClaimTemplate<br>.spec<br>.resources<br>.requests<br>.storage | string | `"10Gi"` |  |
| kof-mothership<br>.values<br>.victoriametrics<br>.vmcluster<br>.spec<br>.vmstorage<br>.replicaCount | int | `3` |  |
| kof-mothership<br>.values<br>.victoriametrics<br>.vmcluster<br>.spec<br>.vmstorage<br>.resources<br>.limits<br>.cpu | string | `"2000m"` |  |
| kof-mothership<br>.values<br>.victoriametrics<br>.vmcluster<br>.spec<br>.vmstorage<br>.resources<br>.limits<br>.memory | string | `"4Gi"` |  |
| kof-mothership<br>.values<br>.victoriametrics<br>.vmcluster<br>.spec<br>.vmstorage<br>.resources<br>.requests<br>.cpu | string | `"500m"` |  |
| kof-mothership<br>.values<br>.victoriametrics<br>.vmcluster<br>.spec<br>.vmstorage<br>.resources<br>.requests<br>.memory | string | `"2Gi"` |  |
| kof-mothership<br>.values<br>.victoriametrics<br>.vmcluster<br>.spec<br>.vmstorage<br>.storage<br>.volumeClaimTemplate<br>.spec<br>.resources<br>.requests<br>.storage | string | `"100Gi"` |  |
| kof-mothership<br>.values<br>.vmalert<br>.enabled | bool | `true` |  |
| kof-operators<br>.enabled | bool | `true` |  |
| kof-operators<br>.notes | string | `"CRDs and operators"` |  |
| kof-operators<br>.values<br>.grafana-operator<br>.enabled | bool | `true` |  |
| kof-operators<br>.values<br>.opentelemetry-operator<br>.enabled | bool | `true` |  |
| kof-operators<br>.values<br>.prometheus-operator-crds<br>.enabled | bool | `true` |  |
| kof-regional<br>.dependsOn[0] | string | `"kof-mothership"` |  |
| kof-regional<br>.enabled | bool | `true` |  |
| kof-regional<br>.notes | string | `"Regional cluster templates"` |  |
| kof-regional<br>.values<br>.storage<br>.victoria-logs-cluster<br>.enabled | bool | `true` |  |
| kof-regional<br>.values<br>.storage<br>.victoria-logs-cluster<br>.vlstorage<br>.persistentVolume<br>.size | string | `"100Gi"` |  |
| kof-regional<br>.values<br>.storage<br>.victoria-traces-cluster<br>.enabled | bool | `true` |  |
| kof-regional<br>.values<br>.storage<br>.victoria-traces-cluster<br>.vtstorage<br>.persistentVolume<br>.size | string | `"100Gi"` |  |
| kof-regional<br>.values<br>.storage<br>.victoriametrics<br>.vmcluster<br>.spec<br>.vmstorage<br>.storage<br>.volumeClaimTemplate<br>.spec<br>.resources<br>.requests<br>.storage | string | `"200Gi"` |  |
| kof-storage<br>.dependsOn[0] | string | `"kof-operators"` |  |
| kof-storage<br>.enabled | bool | `false` |  |
| kof-storage<br>.notes | string | `"Storage components"` |  |
| kof-storage<br>.values<br>.dex<br>.enabled | bool | `false` |  |
| kof-storage<br>.values<br>.external-dns<br>.enabled | bool | `false` |  |
| kof-storage<br>.values<br>.global<br>.storageClass | string | `""` |  |
| kof-storage<br>.values<br>.grafana<br>.enabled | bool | `false` |  |
| kof-storage<br>.values<br>.kof-dashboards<br>.enabled | bool | `false` |  |
| kof-storage<br>.values<br>.promxy<br>.enabled | bool | `true` |  |
| kof-storage<br>.values<br>.victoria-logs-cluster<br>.enabled | bool | `true` |  |
| kof-storage<br>.values<br>.victoria-logs-cluster<br>.vlstorage<br>.persistentVolume<br>.size | string | `"50Gi"` |  |
| kof-storage<br>.values<br>.victoria-metrics-operator<br>.enabled | bool | `false` |  |
| kof-storage<br>.values<br>.victoria-traces-cluster<br>.enabled | bool | `true` |  |
| kof-storage<br>.values<br>.victoria-traces-cluster<br>.vtstorage<br>.persistentVolume<br>.size | string | `"50Gi"` |  |
| kof-storage<br>.values<br>.victoriametrics<br>.enabled | bool | `true` |  |
| kof-storage<br>.values<br>.victoriametrics<br>.vmcluster<br>.enabled | bool | `true` |  |
| kof-storage<br>.values<br>.victoriametrics<br>.vmcluster<br>.spec<br>.replicationFactor | int | `2` |  |
| kof-storage<br>.values<br>.victoriametrics<br>.vmcluster<br>.spec<br>.retentionPeriod | string | `"30d"` |  |
| kof-storage<br>.values<br>.victoriametrics<br>.vmcluster<br>.spec<br>.vmstorage<br>.replicaCount | int | `2` |  |
| kof-storage<br>.values<br>.victoriametrics<br>.vmcluster<br>.spec<br>.vmstorage<br>.storage<br>.volumeClaimTemplate<br>.spec<br>.resources<br>.requests<br>.storage | string | `"100Gi"` |  |
| victoria-metrics-operator<br>.dependsOn[0] | string | `"kof-operators"` |  |
| victoria-metrics-operator<br>.enabled | bool | `true` |  |
| victoria-metrics-operator<br>.notes | string | `"VM CRDs and operator"` |  |
| victoria-metrics-operator<br>.repo<br>.url | string | `"https://victoriametrics.github.io/helm-charts/"` |  |
| victoria-metrics-operator<br>.values | object | `{"crds":{"cleanup":{"enabled":false},`<br>`"plain":true},`<br>`"global":{"cluster":{"dnsDomain":"cluster.local"}},`<br>`"operator":{"disable_prometheus_converter":true},`<br>`"serviceMonitor":{"enabled":true,`<br>`"vm":false}}` | [Docs](https://github.com/VictoriaMetrics/helm-charts/tree/master/charts/victoria-metrics-operator#parameters) |
| victoria-metrics-operator<br>.version | string | `"0.43.1"` |  |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs v1.14.2](https://github.com/norwoodj/helm-docs/releases/v1.14.2)
