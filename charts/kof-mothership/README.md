# kof-mothership

![Version: 0.1.1](https://img.shields.io/badge/Version-0.1.1-informational?style=flat-square) ![AppVersion: 0.1.1](https://img.shields.io/badge/AppVersion-0.1.1-informational?style=flat-square)

A Helm chart that deploys Grafana, Promxy, and VictoriaMetrics.

## Requirements

| Repository | Name | Version |
|------------|------|---------|
| https://projectsveltos.github.io/dashboard-helm-chart | sveltos-dashboard | 0.44.* |
| https://victoriametrics.github.io/helm-charts/ | victoria-metrics-operator | 0.36.* |
| oci://ghcr.io/grafana/helm-charts | grafana-operator | v5.13.0 |
| oci://ghcr.io/k0rdent/cluster-api-visualizer/charts | cluster-api-visualizer | 1.4.0 |

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| cluster-api-visualizer | object | `{"enabled":true}` | [Docs](https://github.com/Jont828/cluster-api-visualizer/tree/main/helm#configurable-values) |
| global<br>.clusterLabel | string | `"clusterName"` | Name of the label identifying where the data point comes from. |
| global<br>.clusterName | string | `"mothership"` | Value of this label. |
| global<br>.random_password_length | int | `12` | Length of the auto-generated passwords for Grafana and VictoriaMetrics. |
| global<br>.random_username_length | int | `8` | Length of the auto-generated usernames for Grafana and VictoriaMetrics. |
| global<br>.storageClass | string | `""` | Name of the storage class used by Grafana, `vmstorage` (long-term storage of raw time series data), and `vmselect` (cache of query results). Keep it unset or empty to leverage the advantages of [default storage class](https://kubernetes.io/docs/concepts/storage/storage-classes/#default-storageclass). |
| grafana<br>.alerts<br>.enabled | bool | `true` | Creates [VMRule](https://docs.victoriametrics.com/operator/resources/vmrule/)-s based on [files/rules/](files/rules/). |
| grafana<br>.enabled | bool | `true` | Enables Grafana. |
| grafana<br>.ingress<br>.enabled | bool | `false` | Enables an ingress to access Grafana without port-forwarding. |
| grafana<br>.ingress<br>.host | string | `"grafana.example.net"` | Domain name Grafana will be available at. |
| grafana<br>.logSources | list | `[]` | Old option to add `GrafanaDatasource`-s. Please add them as in [the docs](https://docs.k0rdent.io/head/admin-kof/#regional-cluster) instead. |
| grafana<br>.security<br>.create_secret | bool | `true` | Enables auto-creation of Grafana username/password. |
| grafana<br>.security<br>.credentials_secret_name | string | `"grafana-admin-credentials"` | Name of secret for Grafana username/password. |
| grafana<br>.storage<br>.size | string | `"200Mi"` | Size of storage for Grafana. |
| grafana<br>.version | string | `"10.4.7"` | Version of Grafana to use. |
| kcm<br>.installTemplates | bool | `false` | Auto-installs `ServiceTemplate`-s like `cert-manager` and `kof-storage` to reference them from Regional and Child `ClusterDeployment`-s. |
| kcm<br>.kof<br>.charts | object | `{"collectors":{"version":"0.1.1"},`<br>`"operators":{"version":"0.1.1"},`<br>`"storage":{"version":"0.1.1"}}` | Versions of `kof-*` helm charts and related `ServiceTemplate`-s auto-installed by `kcm.installTemplates`. |
| kcm<br>.kof<br>.clusterProfiles | object | `{"kof-storage-secrets":{"create_secrets":true,`<br>`"matchLabels":{"k0rdent.mirantis.com/kof-storage-secrets":"true"},`<br>`"secrets":["storage-vmuser-credentials"]}}` | Names of secrets auto-distributed to clusters with matching labels. |
| kcm<br>.kof<br>.repo | object | `{"insecure":false,`<br>`"name":"kof",`<br>`"type":"oci",`<br>`"url":"oci://ghcr.io/k0rdent/kof/charts"}` | Repo of `kof-*` helm charts. |
| kcm<br>.namespace | string | `"kcm-system"` | K8s namespace created on installation of k0rdent/kcm. |
| kcm<br>.serviceMonitor<br>.enabled | bool | `true` | Enables the "KCM Controller Manager" Grafana dashboard. |
| promxy<br>.configmapReload<br>.resources<br>.limits | object | `{"cpu":0.02,`<br>`"memory":"20Mi"}` | Maximum resources available for the `promxy-server-configmap-reload` container in the pods of `kof-mothership-promxy` deployment. |
| promxy<br>.configmapReload<br>.resources<br>.requests | object | `{"cpu":0.02,`<br>`"memory":"20Mi"}` | Minimum resources required for the `promxy-server-configmap-reload` container in the pods of `kof-mothership-promxy` deployment. |
| promxy<br>.enabled | bool | `true` | Enables `kof-mothership-promxy` deployment. |
| promxy<br>.extraArgs | object | `{"log-level":"info"}` | Extra command line arguments passed to the `/bin/promxy`. |
| promxy<br>.image | object | `{"pullPolicy":"IfNotPresent",`<br>`"repository":"quay.io/jacksontj/promxy",`<br>`"tag":"latest"}` | Promxy image to use. |
| promxy<br>.ingress | object | `{"annotations":{},`<br>`"enabled":false,`<br>`"extraLabels":{},`<br>`"hosts":["example.com"],`<br>`"ingressClassName":"nginx",`<br>`"path":"/",`<br>`"pathType":"Prefix",`<br>`"tls":[]}` | Config of `kof-mothership-promxy` [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/). |
| promxy<br>.operator<br>.image | object | `{"pullPolicy":"IfNotPresent",`<br>`"repository":"ghcr.io/k0rdent/kof/promxy-operator-controller",`<br>`"tag":"latest"}` | Image of the promxy operator. |
| promxy<br>.operator<br>.rbac<br>.create | bool | `true` | Creates the `kof-mothership-promxy-operator` cluster role and binds it to the service account of promxy. |
| promxy<br>.operator<br>.resources<br>.limits | object | `{"cpu":"100m",`<br>`"memory":"64Mi"}` | Maximum resources available for promxy operator. |
| promxy<br>.operator<br>.resources<br>.requests | object | `{"cpu":"100m",`<br>`"memory":"64Mi"}` | Minimum resources required for promxy operator. |
| promxy<br>.replicaCount | int | `1` | Number of replicated promxy pods. |
| promxy<br>.resources<br>.limits | object | `{"cpu":"100m",`<br>`"memory":"128Mi"}` | Maximum resources available for the `promxy` container in the pods of `kof-mothership-promxy` deployment. |
| promxy<br>.resources<br>.requests | object | `{"cpu":"100m",`<br>`"memory":"128Mi"}` | Minimum resources required for the `promxy` container in the pods of `kof-mothership-promxy` deployment. |
| promxy<br>.service | object | `{"annotations":{},`<br>`"clusterIP":"",`<br>`"enabled":true,`<br>`"externalIPs":[],`<br>`"extraLabels":{},`<br>`"loadBalancerIP":"",`<br>`"loadBalancerSourceRanges":[],`<br>`"servicePort":8082,`<br>`"type":"ClusterIP"}` | Config of `kof-mothership-promxy` [Service](https://kubernetes.io/docs/concepts/services-networking/service/). |
| promxy<br>.serviceAccount<br>.annotations | object | `{}` | Annotations for the service account of promxy. |
| promxy<br>.serviceAccount<br>.create | bool | `true` | Creates a service account for promxy. |
| promxy<br>.serviceAccount<br>.name | string | `nil` | Name for the service account of promxy. If not set, it is generated as `kof-mothership-promxy`. |
| sveltos-dashboard | object | `{"enabled":true}` | [Docs](https://projectsveltos.github.io/dashboard-helm-chart/#values) |
| sveltos<br>.grafanaDashboard | bool | `true` | Adds Sveltos dashboard to Grafana. |
| sveltos<br>.serviceMonitors | bool | `true` | Creates `ServiceMonitor`-s for Sveltos `sc-manager` and `addon-controller`. |
| victoria-metrics-operator | object | `{"crds":{"cleanup":{"enabled":true},`<br>`"plain":true},`<br>`"enabled":true}` | [Docs](https://github.com/VictoriaMetrics/helm-charts/tree/master/charts/victoria-metrics-operator#parameters) |
| victoriametrics<br>.enabled | bool | `true` | Enables VictoriaMetrics. |
| victoriametrics<br>.vmalert<br>.enabled | bool | `true` | Enables VictoriaMetrics alerts. |
| victoriametrics<br>.vmalert<br>.remoteRead | string | `""` | `url` in [VMAlertRemoteReadSpec](https://docs.victoriametrics.com/operator/api/#vmalertremotereadspec). It is auto-configured by kof if you keep it empty. |
| victoriametrics<br>.vmalert<br>.vmalertmanager<br>.config | string | `""` | `configRawYaml` of [VMAlertmanagerSpec](https://docs.victoriametrics.com/operator/api/#vmalertmanagerspec). Check examples [here](https://github.com/k0rdent/kof/blob/main/docs/alerts.md). |
| victoriametrics<br>.vmcluster<br>.enabled | bool | `true` | Enables high-available and fault-tolerant version of VictoriaMetrics database. |
| victoriametrics<br>.vmcluster<br>.replicaCount | int | `1` | The number of replicas for components of cluster. |
| victoriametrics<br>.vmcluster<br>.replicationFactor | int | `1` | The number of replicas for each metric. |
| victoriametrics<br>.vmcluster<br>.retentionPeriod | string | `"1"` | Days to retain the data. |
| victoriametrics<br>.vmcluster<br>.vmselect<br>.storage<br>.size | string | `"2Gi"` | Query results cache size. |
| victoriametrics<br>.vmcluster<br>.vmstorage<br>.storage<br>.size | string | `"10Gi"` | Long-term storage size of raw time series data. |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs v1.14.2](https://github.com/norwoodj/helm-docs/releases/v1.14.2)
