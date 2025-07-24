# kof-mothership

![Version: 1.1.0](https://img.shields.io/badge/Version-1.1.0-informational?style=flat-square) ![AppVersion: 1.1.0](https://img.shields.io/badge/AppVersion-1.1.0-informational?style=flat-square)

A Helm chart that deploys Grafana, Promxy, and VictoriaMetrics.

## Requirements

| Repository | Name | Version |
|------------|------|---------|
| https://charts.dexidp.io | dex | 0.23.0 |
| https://projectsveltos.github.io/helm-charts | sveltos-dashboard | 0.56.0 |
| https://victoriametrics.github.io/helm-charts/ | victoria-metrics-operator | 0.40.5 |
| oci://ghcr.io/grafana/helm-charts | grafana-operator | v5.18.0 |
| oci://ghcr.io/k0rdent/catalog/charts | cert-manager-service-template(kgst) | 1.0.0 |
| oci://ghcr.io/k0rdent/catalog/charts | ingress-nginx-service-template(kgst) | 1.0.0 |
| oci://ghcr.io/k0rdent/cluster-api-visualizer/charts | cluster-api-visualizer | 1.4.0 |

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| cert-manager-service-template | object | `{"chart":"cert-manager:v1.16.4",`<br>`"namespace":"kcm-system",`<br>`"repo":{"name":"cert-manager",`<br>`"url":"https://charts.jetstack.io"}}` | Config of `ServiceTemplate` to use `cert-manager` in `MultiClusterService`. |
| cert-manager<br>.cluster-issuer<br>.create | bool | `false` | Whether to create a default clusterissuer |
| cert-manager<br>.cluster-issuer<br>.provider | string | `"letsencrypt"` | Default clusterissuer provider |
| cert-manager<br>.email | string | `"mail@example.net"` | If we use letsencrypt (or similar) which email to use |
| cert-manager<br>.enabled | bool | `true` | Whether cert-manager is present in the cluster |
| cluster-api-visualizer | object | `{"enabled":true,`<br>`"image":{"repository":"ghcr.io/k0rdent"}}` | [Docs](https://github.com/Jont828/cluster-api-visualizer/tree/main/helm#configurable-values) |
| cluster-api-visualizer<br>.image<br>.repository | string | `"ghcr.io/k0rdent"` | Custom `cluster-api-visualizer` image repository. |
| clusterAlertRules | object | `{}` | Cluster-specific patch of Prometheus alerting rules, e.g. `cluster1.alertgroup1.alert1.expr` overriding the threshold `> ( 25 / 100 )` and adding `{cluster="cluster1"}` filter, or just adding whole new rules |
| clusterRecordRules | object | `{}` | Cluster-specific patch of Prometheus recording rules, e.g. `regionalCluster1.recordGroup1` overriding whole group of rules (because `record` is not unique), or adding new groups |
| defaultAlertRules | object | `{"docker-containers":{"ContainerHighMemoryUsage":{"annotations":{"description":"Container Memory usage is above 80%\n  VALUE = {{ $value }}\n  LABELS = {{ $labels }}",`<br>`"summary":"Container High Memory usage ({{ $labels.cluster }}/{{ $labels.namespace }}/{{ $labels.pod }})"},`<br>`"expr":"sum(container_memory_working_set_bytes{pod!=\"\"}) by (cluster,`<br>` namespace,`<br>` pod)\n/ sum(container_spec_memory_limit_bytes > 0) by (cluster,`<br>` namespace,`<br>` pod) * 100\n> 80",`<br>`"for":"2m",`<br>`"labels":{"severity":"warning"}}}}` | Patch of default Prometheus alerting rules, e.g. `alertgroup1.alert1` overriding `for` field and adding `{cluster!~"^cluster1$|^cluster10$"}` for rules overridden in `clusterRulesPatch`, or just adding whole new rules |
| defaultRecordRules | object | `{}` | Patch of default Prometheus recording rules, e.g. `recordgroup1` overriding whole group of rules (`record` is not unique), or adding new groups |
| dex<br>.config<br>.connectors[0]<br>.config<br>.clientID | string | `""` |  |
| dex<br>.config<br>.connectors[0]<br>.config<br>.clientSecret | string | `""` |  |
| dex<br>.config<br>.connectors[0]<br>.config<br>.redirectURI | string | `"https://dex.example.com:32000/callback"` |  |
| dex<br>.config<br>.connectors[0]<br>.id | string | `"google"` |  |
| dex<br>.config<br>.connectors[0]<br>.name | string | `"Google"` |  |
| dex<br>.config<br>.connectors[0]<br>.type | string | `"google"` |  |
| dex<br>.config<br>.issuer | string | `"https://dex.example.com:32000"` | The identifier (issuer) URL for Dex. |
| dex<br>.config<br>.staticClients[0]<br>.id | string | `"grafana-id"` |  |
| dex<br>.config<br>.staticClients[0]<br>.name | string | `"Grafana"` |  |
| dex<br>.config<br>.staticClients[0]<br>.redirectURIs[0] | string | `"http://localhost:3000/login/generic_oauth"` |  |
| dex<br>.config<br>.staticClients[0]<br>.secret | string | `"grafana-secret"` |  |
| dex<br>.config<br>.storage<br>.type | string | `"memory"` | Specifies the storage type used by Dex. |
| dex<br>.config<br>.web<br>.https | string | `"0.0.0.0:5554"` | Address and port for the HTTPS endpoint. |
| dex<br>.config<br>.web<br>.tlsCert | string | `"/etc/dex/tls/tls.crt"` | Path to the TLS certificate file. |
| dex<br>.config<br>.web<br>.tlsKey | string | `"/etc/dex/tls/tls.key"` | Path to the TLS private key file. |
| dex<br>.enabled | bool | `false` | Enables Dex. |
| dex<br>.https | object | `{"enabled":true}` | Enables the HTTPS endpoint. |
| dex<br>.image<br>.tag | string | `"v2.42.1"` | Version of Dex to use. |
| dex<br>.service<br>.ports<br>.http<br>.port | int | `5556` |  |
| dex<br>.service<br>.ports<br>.https<br>.nodePort | int | `32000` |  |
| dex<br>.service<br>.ports<br>.https<br>.port | int | `5554` |  |
| dex<br>.service<br>.type | string | `"NodePort"` |  |
| dex<br>.volumeMounts[0]<br>.mountPath | string | `"/etc/dex/tls"` |  |
| dex<br>.volumeMounts[0]<br>.name | string | `"tls"` |  |
| dex<br>.volumeMounts[0]<br>.readOnly | bool | `true` |  |
| dex<br>.volumes[0]<br>.name | string | `"tls"` |  |
| dex<br>.volumes[0]<br>.secret<br>.secretName | string | `"dex-tls"` |  |
| global<br>.clusterLabel | string | `"cluster"` | Name of the label identifying where the time series data points come from. |
| global<br>.clusterName | string | `"mothership"` | Value of this label. |
| global<br>.random_password_length | int | `12` | Length of the auto-generated passwords for Grafana and VictoriaMetrics. |
| global<br>.random_username_length | int | `8` | Length of the auto-generated usernames for Grafana and VictoriaMetrics. |
| global<br>.registry | string | `"docker.io"` | Custom image registry, `sveltos-dashboard` requires not empty value. |
| global<br>.storageClass | string | `""` | Name of the storage class used by Grafana, `vmstorage` (long-term storage of raw time series data), and `vmselect` (cache of query results). Keep it unset or empty to leverage the advantages of [default storage class](https://kubernetes.io/docs/concepts/storage/storage-classes/#default-storageclass). |
| grafana-operator<br>.image<br>.repository | string | `"ghcr.io/grafana/grafana-operator"` | Custom `grafana-operator` image repository. |
| grafana<br>.dashboard<br>.datasource<br>.current | object | `{"text":"promxy",`<br>`"value":"promxy"}` | Values of current datasource |
| grafana<br>.dashboard<br>.datasource<br>.regex | string | `"/promxy/"` | Regex pattern to filter datasources. |
| grafana<br>.dashboard<br>.filters | object | `{"clusterName":"mothership"}` | Values of filters to apply. |
| grafana<br>.dashboard<br>.istio_dashboard_enabled | bool | `true` | Enables istio dashboards |
| grafana<br>.enabled | bool | `true` | Enables Grafana. |
| grafana<br>.ingress<br>.enabled | bool | `false` | Enables an ingress to access Grafana without port-forwarding. |
| grafana<br>.ingress<br>.host | string | `"grafana.example.net"` | Domain name Grafana will be available at. |
| grafana<br>.logSources | list | `[]` | Old option to add `GrafanaDatasource`-s. |
| grafana<br>.pvc<br>.resources<br>.requests<br>.storage | string | `"200Mi"` | Size of storage for Grafana. |
| grafana<br>.security<br>.create_secret | bool | `true` | Enables auto-creation of Grafana username/password. |
| grafana<br>.security<br>.credentials_secret_name | string | `"grafana-admin-credentials"` | Name of secret for Grafana username/password. |
| grafana<br>.version | string | `"10.4.18-security-01"` | Version of Grafana to use. |
| ingress-nginx-service-template | object | `{"chart":"ingress-nginx:4.12.1",`<br>`"namespace":"kcm-system",`<br>`"repo":{"name":"ingress-nginx",`<br>`"url":"https://kubernetes.github.io/ingress-nginx"}}` | Config of `ServiceTemplate` to use `ingress-nginx` in `MultiClusterService`. |
| kcm<br>.installTemplates | bool | `false` | Installs `ServiceTemplates` to use charts like `kof-storage` in `MultiClusterService`. |
| kcm<br>.kof<br>.clusterProfiles | object | `{"kof-storage-secrets":{"create_secrets":true,`<br>`"matchLabels":{"k0rdent.mirantis.com/kof-storage-secrets":"true"},`<br>`"secrets":["storage-vmuser-credentials"]}}` | Names of secrets auto-distributed to clusters with matching labels. |
| kcm<br>.kof<br>.operator<br>.autoinstrumentation<br>.enabled | bool | `true` | Enable autoinstrumentation to collect metrics and traces from the operator. |
| kcm<br>.kof<br>.operator<br>.enabled | bool | `true` |  |
| kcm<br>.kof<br>.operator<br>.image | object | `{"pullPolicy":"IfNotPresent",`<br>`"registry":"ghcr.io/k0rdent",`<br>`"repository":"kof/kof-operator-controller"}` | Image of the kof operator. |
| kcm<br>.kof<br>.operator<br>.rbac<br>.create | bool | `true` | Creates the `kof-mothership-kof-operator` cluster role and binds it to the service account of operator. |
| kcm<br>.kof<br>.operator<br>.replicaCount | int | `1` |  |
| kcm<br>.kof<br>.operator<br>.resources<br>.limits | object | `{"cpu":"100m",`<br>`"memory":"128Mi"}` | Maximum resources available for operator. |
| kcm<br>.kof<br>.operator<br>.resources<br>.requests | object | `{"cpu":"100m",`<br>`"memory":"128Mi"}` | Minimum resources required for operator. |
| kcm<br>.kof<br>.operator<br>.serviceAccount<br>.annotations | object | `{}` | Annotations for the service account of operator. |
| kcm<br>.kof<br>.operator<br>.serviceAccount<br>.create | bool | `true` | Creates a service account for operator. |
| kcm<br>.kof<br>.operator<br>.serviceAccount<br>.name | string | `nil` | Name for the service account of operator. If not set, it is generated as `kof-mothership-kof-operator`. |
| kcm<br>.kof<br>.operator<br>.ui<br>.port | int | `9090` | Port for the web UI server. |
| kcm<br>.kof<br>.operator<br>.ui<br>.receiverPort | int | `9090` | Port for Prometheus metrics receiver. |
| kcm<br>.kof<br>.repo | object | `{"name":"kof",`<br>`"spec":{"type":"oci",`<br>`"url":"oci://ghcr.io/k0rdent/kof/charts"}}` | Repo of `kof-*` helm charts. |
| kcm<br>.namespace | string | `"kcm-system"` | K8s namespace created on installation of k0rdent/kcm. |
| kcm<br>.serviceMonitor<br>.enabled | bool | `true` | Enables the "KCM Controller Manager" Grafana dashboard. |
| promxy<br>.configmapReload<br>.resources<br>.limits | object | `{"cpu":0.02,`<br>`"memory":"20Mi"}` | Maximum resources available for the `promxy-server-configmap-reload` container in the pods of `kof-mothership-promxy` deployment. |
| promxy<br>.configmapReload<br>.resources<br>.requests | object | `{"cpu":0.02,`<br>`"memory":"20Mi"}` | Minimum resources required for the `promxy-server-configmap-reload` container in the pods of `kof-mothership-promxy` deployment. |
| promxy<br>.enabled | bool | `true` | Enables `kof-mothership-promxy` deployment. |
| promxy<br>.extraArgs | object | `{"log-level":"info",`<br>`"web.external-url":"http://127.0.0.1:8082"}` | Extra command line arguments passed as `--key=value` to the `/bin/promxy`. |
| promxy<br>.image | object | `{"pullPolicy":"IfNotPresent",`<br>`"registry":"quay.io",`<br>`"repository":"jacksontj/promxy",`<br>`"tag":"v0.0.93"}` | Promxy image to use. |
| promxy<br>.ingress | object | `{"annotations":{},`<br>`"enabled":false,`<br>`"extraLabels":{},`<br>`"hosts":["example.com"],`<br>`"ingressClassName":"nginx",`<br>`"path":"/",`<br>`"pathType":"Prefix",`<br>`"tls":[]}` | Config of `kof-mothership-promxy` [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/). |
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
| victoria-metrics-operator | object | `{"crds":{"cleanup":{"enabled":true},`<br>`"plain":true},`<br>`"enabled":true,`<br>`"operator":{"disable_prometheus_converter":true}}` | [Docs](https://github.com/VictoriaMetrics/helm-charts/tree/master/charts/victoria-metrics-operator#parameters) |
| victoriametrics<br>.enabled | bool | `true` | Enables VictoriaMetrics. |
| victoriametrics<br>.vmalert<br>.enabled | bool | `true` | Enables VMAlertManager only, as VMAlert is replaced with promxy in kof-mothership. |
| victoriametrics<br>.vmalert<br>.vmalertmanager<br>.config | string | `""` | `configRawYaml` of [VMAlertmanagerSpec](https://docs.victoriametrics.com/operator/api/#vmalertmanagerspec). Check examples [here](https://docs.k0rdent.io/next/admin/kof/kof-alerts/#alertmanager-demo). |
| victoriametrics<br>.vmcluster<br>.enabled | bool | `true` | Enables high-available and fault-tolerant version of VictoriaMetrics database. |
| victoriametrics<br>.vmcluster<br>.replicaCount | int | `1` | The number of replicas for components of cluster. |
| victoriametrics<br>.vmcluster<br>.replicationFactor | int | `1` | The number of replicas for each metric. |
| victoriametrics<br>.vmcluster<br>.retentionPeriod | string | `"1"` | Days to retain the data. |
| victoriametrics<br>.vmcluster<br>.vminsert<br>.labels<br>."k0rdent<br>.mirantis<br>.com/istio-mtls-enabled" | string | `"true"` | Label to enable mtls |
| victoriametrics<br>.vmcluster<br>.vmselect<br>.storage<br>.size | string | `"2Gi"` | Query results cache size. |
| victoriametrics<br>.vmcluster<br>.vmstorage<br>.storage<br>.size | string | `"10Gi"` | Long-term storage size of raw time series data. |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs v1.14.2](https://github.com/norwoodj/helm-docs/releases/v1.14.2)
