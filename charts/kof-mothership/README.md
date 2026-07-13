# kof-mothership

![Version: 1.11.0-rc0](https://img.shields.io/badge/Version-1.11.0--rc0-informational?style=flat-square) ![AppVersion: 1.11.0-rc0](https://img.shields.io/badge/AppVersion-1.11.0--rc0-informational?style=flat-square)

KOF Helm chart for KOF Management cluster

## Requirements

| Repository | Name | Version |
|------------|------|---------|
| file://../kof-dashboards/ | kof-dashboards | 1.11.0-rc0 |
| https://charts.dexidp.io | dex | 0.24.1 |
| https://kubernetes-sigs.github.io/external-dns/ | external-dns | 1.20.0 |
| https://kubernetes-sigs.github.io/metrics-server/ | metrics-server | 3.13.0 |
| oci://ghcr.io/k0rdent/catalog/charts | cert-manager-service-template(kgst) | 2.0.1 |
| oci://ghcr.io/k0rdent/catalog/charts | ingress-nginx-service-template(kgst) | 2.0.1 |
| oci://ghcr.io/k0rdent/catalog/charts | envoy-gateway-service-template(kgst) | 2.0.1 |
| oci://ghcr.io/k0rdent/catalog/charts | victoria-metrics-operator-service-template(kgst) | 2.0.1 |

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| cert-manager-service-template | object | `{"chart":"cert-manager:v1.19.3",`<br>`"enabled":true,`<br>`"namespace":"kcm-system",`<br>`"repo":{"name":"cert-manager",`<br>`"spec":{"url":"oci://quay.io/jetstack/charts"}}}` | Config of `ServiceTemplate` to use `cert-manager` in `MultiClusterService`. |
| cert-manager<br>.cluster-issuer<br>.create | bool | `false` | Whether to create a default clusterissuer |
| cert-manager<br>.cluster-issuer<br>.name | string | `"letsencrypt-prod"` |  |
| cert-manager<br>.cluster-issuer<br>.provider | string | `"letsencrypt"` | Default clusterissuer provider |
| cert-manager<br>.email | string | `"mail@example.net"` | If we use letsencrypt (or similar) which email to use |
| cert-manager<br>.enabled | bool | `true` | Whether cert-manager is present in the cluster |
| cert-manager<br>.solvers[0]<br>.http01<br>.gatewayHTTPRoute<br>.parentRefs[0]<br>.kind | string | `"Gateway"` |  |
| cert-manager<br>.solvers[0]<br>.http01<br>.gatewayHTTPRoute<br>.parentRefs[0]<br>.name | string | `"mothership-gateway"` |  |
| cert-manager<br>.solvers[0]<br>.http01<br>.gatewayHTTPRoute<br>.parentRefs[0]<br>.namespace | string | `"kof"` |  |
| clusterAlertRules | object | `{}` | Cluster-specific patch of Prometheus alerting rules, e.g. `cluster1.alertgroup1.alert1.expr` overriding the threshold `> ( 25 / 100 )` and adding `{cluster="cluster1"}` filter, or just adding whole new rules |
| clusterRecordRules | object | `{}` | Cluster-specific patch of Prometheus recording rules, e.g. `regionalCluster1.recordGroup1` overriding whole group of rules (because `record` is not unique), or adding new groups |
| defaultAlertRules | object | `{"docker-containers":{"ContainerHighMemoryUsage":{"annotations":{"description":"Container Memory usage is above 80%\n  VALUE = {{ $value }}\n  LABELS = {{ $labels }}",`<br>`"summary":"Container High Memory usage ({{ $labels.cluster }}/{{ $labels.namespace }}/{{ $labels.pod }}/{{ $labels.container }})"},`<br>`"expr":"sum(container_memory_working_set_bytes{pod!=\"\",`<br>` container!=\"\",`<br>` metrics_path=\"/metrics/cadvisor\"}) by (tenant,`<br>` cluster,`<br>` namespace,`<br>` pod,`<br>` container)\n/ sum(container_spec_memory_limit_bytes > 0) by (tenant,`<br>` cluster,`<br>` namespace,`<br>` pod,`<br>` container) * 100\n> 80",`<br>`"for":"2m",`<br>`"labels":{"severity":"warning"}}},`<br>`"kube-state-metrics":{"ConditionStatusFailed":{"annotations":{"description":"LABELS = {{ $labels }}",`<br>`"summary":"k0rdent custom resource condition status failed ({{ $labels.cluster }}/{{ $labels.name }})"},`<br>`"expr":"{customresource_group=\"k0rdent.mirantis.com\",`<br>` job=\"kube-state-metrics\"} == 0",`<br>`"for":"10m",`<br>`"labels":{"severity":"error"}}}}` | Patch of default Prometheus alerting rules, e.g. `alertgroup1.alert1` overriding `for` field and adding `{cluster!~"^cluster1$|^cluster10$"}` for rules overridden in `clusterRulesPatch`, or just adding whole new rules |
| defaultRecordRules | object | `{}` | Patch of default Prometheus recording rules, e.g. `recordgroup1` overriding whole group of rules (`record` is not unique), or adding new groups |
| dex<br>.config<br>.connectors | list | `[]` |  |
| dex<br>.config<br>.issuer | string | `"https://dex.example.com"` | The identifier (issuer) URL for Dex. |
| dex<br>.config<br>.staticClients | list | `[]` |  |
| dex<br>.config<br>.storage<br>.type | string | `"memory"` | Specifies the storage type used by Dex. |
| dex<br>.enabled | bool | `false` | Enables Dex. |
| dex<br>.httpRoute<br>.enabled | bool | `true` | Enables creation of the Dex HTTPRoute. |
| dex<br>.httpRoute<br>.hostnames | list | `["dex.example.com"]` | Hostname at which Dex will be exposed via the Gateway. |
| dex<br>.https | object | `{"enabled":false}` | Enables the HTTPS endpoint. |
| envoy-gateway-service-template | object | `{"chart":"gateway-helm:v1.7.2",`<br>`"namespace":"kcm-system",`<br>`"repo":{"name":"envoy-gateway",`<br>`"spec":{"url":"oci://docker.io/envoyproxy"}}}` | Config of `ServiceTemplate` to use `envoy-gateway` in `MultiClusterService`. |
| external-dns | object | `{"enabled":false,`<br>`"provider":{"name":"aws"},`<br>`"sources":["service",`<br>`"ingress",`<br>`"gateway-httproute"]}` | [Docs](https://kubernetes-sigs.github.io/external-dns/) Installs ExternalDNS on the mothership cluster. |
| external-dns<br>.enabled | bool | `false` | Enables ExternalDNS deployment. |
| external-dns<br>.provider | object | `{"name":"aws"}` | DNS provider to use (e.g. aws, azure, cloudflare, google). |
| external-dns<br>.sources | list | `["service",`<br>`"ingress",`<br>`"gateway-httproute"]` | Sources to watch for DNS records. |
| gateway | object | `{"annotations":{"cert-manager.io/cluster-issuer":"letsencrypt-prod"},`<br>`"createGatewayClass":true,`<br>`"enabled":false,`<br>`"gatewayClassControllerName":"gateway.envoyproxy.io/gatewayclass-controller",`<br>`"name":"mothership-gateway",`<br>`"spec":{"gatewayClassName":"mothership-eg",`<br>`"listeners":[{"name":"http",`<br>`"port":80,`<br>`"protocol":"HTTP"},`<br>`{"hostname":"*.example.com",`<br>`"name":"https",`<br>`"port":443,`<br>`"protocol":"HTTPS",`<br>`"tls":{"certificateRefs":[{"kind":"Secret",`<br>`"name":"kof-https"}],`<br>`"mode":"Terminate"}}]}}` | Optional Gateway infrastructure resources (GatewayClass, Gateway, ClusterIssuer, HTTPRoute for Dex). Requires Envoy Gateway (envoy-gateway.enabled=true) and cert-manager (cert-manager.enabled=true) to be present in the cluster. |
| gateway<br>.annotations | object | `{"cert-manager.io/cluster-issuer":"letsencrypt-prod"}` | Annotations applied to the Gateway resource. Typically used to reference the cert-manager ClusterIssuer. |
| gateway<br>.createGatewayClass | bool | `true` | Whether to create a GatewayClass resource. Requires Envoy Gateway to be installed in the cluster. |
| gateway<br>.enabled | bool | `false` | Enables creation of Gateway-related resources. |
| gateway<br>.gatewayClassControllerName | string | `"gateway.envoyproxy.io/gatewayclass-controller"` | Controller name used in the GatewayClass spec. |
| gateway<br>.name | string | `"mothership-gateway"` | Name of the Gateway resource to create. |
| gateway<br>.spec | object | `{"gatewayClassName":"mothership-eg",`<br>`"listeners":[{"name":"http",`<br>`"port":80,`<br>`"protocol":"HTTP"},`<br>`{"hostname":"*.example.com",`<br>`"name":"https",`<br>`"port":443,`<br>`"protocol":"HTTPS",`<br>`"tls":{"certificateRefs":[{"kind":"Secret",`<br>`"name":"kof-https"}],`<br>`"mode":"Terminate"}}]}` | Spec of the Gateway resource. |
| global<br>.clusterLabel | string | `"cluster"` | Name of the label identifying where the time series data points come from. |
| global<br>.clusterName | string | `"mothership"` | Value of clusterName usually identical to cluster used in some subcharts (e.g. otel) |
| global<br>.random_password_length | int | `12` | Length of the auto-generated passwords for Grafana (if enabled) and VictoriaMetrics. |
| global<br>.random_username_length | int | `8` | Length of the auto-generated usernames for Grafana (if enabled) and VictoriaMetrics. |
| global<br>.registry | string | `""` | Custom image registry. |
| global<br>.storageClass | string | `""` | Name of the storage class used by Grafana (if enabled), `vmstorage` (long-term storage of raw time series data), and `vmselect` (cache of query results). Keep it unset or empty to leverage the advantages of [default storage class](https://kubernetes.io/docs/concepts/storage/storage-classes/#default-storageclass). |
| grafana<br>.config | object | `{}` | Custom configuration for Grafana. |
| grafana<br>.dashboard<br>.resyncPeriod | string | `"0m"` | How often the operator should reconcile dashboards with Grafana. Set to "0m" to disable periodic reconciliation (recommended for production to allow operators to adjust dashboards without losing changes on resync). |
| grafana<br>.enabled | bool | `false` | Enables Grafana. |
| grafana<br>.gateway<br>.enabled | bool | `false` | Use gateway to access Grafana without port-forwarding. |
| grafana<br>.gateway<br>.httpRoute<br>.spec<br>.hostnames[0] | string | `"grafana.example.net"` |  |
| grafana<br>.gateway<br>.httpRoute<br>.spec<br>.parentRefs[0]<br>.name | string | `"mothership-gateway"` |  |
| grafana<br>.gateway<br>.httpRoute<br>.spec<br>.parentRefs[0]<br>.namespace | string | `"kof"` |  |
| grafana<br>.gateway<br>.httpRoute<br>.spec<br>.rules[0]<br>.backendRefs[0]<br>.name | string | `"grafana-vm-service"` |  |
| grafana<br>.gateway<br>.httpRoute<br>.spec<br>.rules[0]<br>.backendRefs[0]<br>.port | int | `3000` |  |
| grafana<br>.ingress<br>.enabled | bool | `false` | Enables an ingress to access Grafana without port-forwarding. |
| grafana<br>.ingress<br>.host | string | `"grafana.example.com"` | Domain name Grafana will be available at. |
| grafana<br>.logSources | list | `[]` | Old option to add `GrafanaDatasource`-s. |
| grafana<br>.pvc<br>.resources<br>.requests<br>.storage | string | `"200Mi"` | Size of storage for Grafana. |
| grafana<br>.security<br>.create_secret | bool | `true` | Enables auto-creation of Grafana username/password. |
| grafana<br>.security<br>.credentials_secret_name | string | `"grafana-admin-credentials"` | Name of secret for Grafana username/password. |
| grafana<br>.version | string | `"10.4.18-security-01"` | Version of Grafana to use. |
| ingress-nginx-service-template | object | `{"chart":"ingress-nginx:4.14.3",`<br>`"namespace":"kcm-system",`<br>`"repo":{"name":"ingress-nginx",`<br>`"spec":{"type":"default",`<br>`"url":"https://kubernetes.github.io/ingress-nginx"}}}` | Config of `ServiceTemplate` to use `ingress-nginx` in `MultiClusterService`. |
| istio<br>.enabled | bool | `true` | Installs resources required for the KOF to work properly with the main Istio chart. |
| kcm<br>.installTemplates | bool | `true` | Installs `ServiceTemplates` to use charts like `kof-storage` in `MultiClusterService`. |
| kcm<br>.kof<br>.acl<br>.developmentMode | bool | `false` | Enables development mode. Disables token verification and bypasses authentication, granting admin access to the ACL server. |
| kcm<br>.kof<br>.acl<br>.enabled | bool | `false` | Enables the ACL server. |
| kcm<br>.kof<br>.acl<br>.extraArgs | object | `{}` | Extra arguments for ACL server as key-value pairs (e.g., log-level: debug) |
| kcm<br>.kof<br>.acl<br>.image | object | `{"pullPolicy":"IfNotPresent",`<br>`"registry":"ghcr.io/k0rdent",`<br>`"repository":"kof/kof-acl-server"}` | Image of the kof ACL server. |
| kcm<br>.kof<br>.acl<br>.port | int | `9091` | Port for ACL server. |
| kcm<br>.kof<br>.acl<br>.replicaCount | int | `1` | Number of the ACL deployment replicas. |
| kcm<br>.kof<br>.acl<br>.resources<br>.limits | object | `{"cpu":"100m",`<br>`"memory":"256Mi"}` | Maximum resources available for ACL. |
| kcm<br>.kof<br>.acl<br>.resources<br>.requests | object | `{"cpu":"100m",`<br>`"memory":"256Mi"}` | Minimum resources required for ACL. |
| kcm<br>.kof<br>.acl<br>.service | object | `{"annotations":{},`<br>`"enabled":true,`<br>`"type":"ClusterIP"}` | Config of `kof-acl` Service. |
| kcm<br>.kof<br>.acl<br>.service<br>.annotations | object | `{}` | Service annotations. |
| kcm<br>.kof<br>.acl<br>.service<br>.enabled | bool | `true` | Enables the Service for ACL server. |
| kcm<br>.kof<br>.acl<br>.service<br>.type | string | `"ClusterIP"` | Service type. |
| kcm<br>.kof<br>.ingress | object | `{"annotations":{},`<br>`"enabled":false,`<br>`"extraLabels":{},`<br>`"hosts":["example.com"],`<br>`"ingressClassName":"nginx",`<br>`"path":"/",`<br>`"pathType":"Prefix",`<br>`"tls":[]}` | Config of `kof-mothership-kof-operator-ui` [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/). |
| kcm<br>.kof<br>.mcs | string | `nil` | Names of secrets auto-distributed to clusters with matching labels. |
| kcm<br>.kof<br>.operator<br>.autoUpgrade | bool | `false` | Governed by `autoUpgrade` in the `kof` umbrella chart. |
| kcm<br>.kof<br>.operator<br>.crossNamespace | bool | `false` | Allows regional cluster to be in another namespace than the child cluster. |
| kcm<br>.kof<br>.operator<br>.enabled | bool | `true` | Enables the `kof-operator`. |
| kcm<br>.kof<br>.operator<br>.image | object | `{"pullPolicy":"IfNotPresent",`<br>`"registry":"ghcr.io/k0rdent",`<br>`"repository":"kof/kof-operator-controller"}` | Image of the kof operator. |
| kcm<br>.kof<br>.operator<br>.otlp<br>.enabled | bool | `true` | Enables OTel SDK. |
| kcm<br>.kof<br>.operator<br>.otlp<br>.endpoint | string | `""` | OTLP endpoint override for the operator's OTel SDK. When set, used regardless of Istio injection. When empty, defaults to http://kof-collectors-daemon-collector:4317 (Istio) or http://$(NODE_IP):4317 (non-Istio). |
| kcm<br>.kof<br>.operator<br>.rbac<br>.create | bool | `true` | Creates the `kof-mothership-kof-operator` cluster role and binds it to the service account of operator. |
| kcm<br>.kof<br>.operator<br>.replicaCount | int | `1` | Number of the `kof-operator` deployment replicas. |
| kcm<br>.kof<br>.operator<br>.resources<br>.limits | object | `{"cpu":"500m",`<br>`"memory":"512Mi"}` | Maximum resources available for operator. |
| kcm<br>.kof<br>.operator<br>.resources<br>.requests | object | `{"cpu":"200m",`<br>`"memory":"256Mi"}` | Minimum resources required for operator. |
| kcm<br>.kof<br>.operator<br>.serviceAccount<br>.annotations | object | `{}` | Annotations for the service account of operator. |
| kcm<br>.kof<br>.operator<br>.serviceAccount<br>.create | bool | `true` | Creates a service account for operator. |
| kcm<br>.kof<br>.operator<br>.serviceAccount<br>.name | string | `nil` | Name for the service account of operator. If not set, it is generated as `kof-mothership-kof-operator`. |
| kcm<br>.kof<br>.operator<br>.storageURLs | object | `{"vlAuditInsert":"http://vlinsert-audit-logs.kof.svc:9481",`<br>`"vlAuditSelect":"http://vlselect-audit-logs.kof.svc:9471",`<br>`"vlInsert":"http://kof-storage-victoria-logs-cluster-vlinsert.kof.svc:9481",`<br>`"vlSelect":"http://kof-storage-victoria-logs-cluster-vlselect.kof.svc:9471",`<br>`"vmInsert":"http://vminsert-cluster.kof.svc:8480",`<br>`"vmSelect":"http://vmselect-cluster.kof.svc:8481",`<br>`"vtInsert":"http://kof-storage-vt-cluster-vtinsert.kof.svc:10481",`<br>`"vtSelect":"http://kof-storage-vt-cluster-vtselect.kof.svc:10471"}` | URLs of the VM/VL/VT storage services used to configure VMUser for regional and child clusters. NOTE: These are the default URLs for kof-storage. Change them if you make changes to kof-storage in kof-regional chart. |
| kcm<br>.kof<br>.operator<br>.ui<br>.gateway | object | `{"enabled":false,`<br>`"httpRoute":{"spec":{"hostnames":["kof-ui.example.net"],`<br>`"parentRefs":[{"name":"mothership-gateway",`<br>`"namespace":"kof"}],`<br>`"rules":[{"backendRefs":[{"name":"kof-mothership-kof-operator",`<br>`"port":9090}]}]}}}` | Config of `kof-mothership-kof-operator-ui` [Gateway](https://kubernetes.io/docs/concepts/services-networking/gateway/). |
| kcm<br>.kof<br>.operator<br>.ui<br>.port | int | `9090` | Port for the web UI server. |
| kcm<br>.kof<br>.operator<br>.ui<br>.receiverPort | int | `9090` | Port for Prometheus metrics receiver. |
| kcm<br>.kof<br>.repo<br>.name | string | `"oci-registry"` | Name of existing helm repo with `kof-*` helm charts. This name is set by `kof` umbrella chart which creates such helm repo. |
| kcm<br>.kof<br>.secrets | object | `{"kof-storage-secrets":{"secrets":["storage-vmuser-credentials"]}}` | Generation of secrets used by kof components. Generate random username/password if secret not found. |
| kcm<br>.kof<br>.service | object | `{"annotations":{},`<br>`"clusterIP":"",`<br>`"enabled":true,`<br>`"externalIPs":[],`<br>`"extraLabels":{},`<br>`"loadBalancerIP":"",`<br>`"loadBalancerSourceRanges":[],`<br>`"type":"ClusterIP"}` | Config of `kof-mothership-kof-operator` [Service](https://kubernetes.io/docs/concepts/services-networking/service/). |
| kcm<br>.namespace | string | `"kcm-system"` | K8s namespace created on installation of k0rdent/kcm. |
| kcm<br>.serviceMonitor<br>.enabled | bool | `true` | Enables the "KCM Controller Manager" Grafana dashboard. |
| kof-dashboards<br>.grafana<br>.dashboard<br>.datasource<br>.current | object | `{"text":"kof-metrics",`<br>`"value":"kof-metrics"}` | Values of current datasource |
| kof-dashboards<br>.grafana<br>.dashboard<br>.datasource<br>.regex | string | `"/kof-metrics/"` | Regex pattern to filter datasources. |
| kof-dashboards<br>.grafana<br>.dashboard<br>.filters | object | `{"cluster":"mothership"}` | Values of filters to apply. |
| kof-dashboards<br>.grafana<br>.dashboard<br>.istio_dashboard_enabled | bool | `true` | Enables istio dashboards |
| metrics-server | object | `{"enabled":false}` | [Docs](https://github.com/kubernetes-sigs/metrics-server/blob/main/charts/metrics-server/README.md) |
| metrics-server<br>.enabled | bool | `false` | Enables Metrics Server. |
| promxy<br>.configmapReload<br>.resources<br>.limits | object | `{"cpu":0.02,`<br>`"memory":"20Mi"}` | Maximum resources available for the `promxy-server-configmap-reload` container in the pods of `kof-mothership-promxy` deployment. |
| promxy<br>.configmapReload<br>.resources<br>.requests | object | `{"cpu":0.02,`<br>`"memory":"20Mi"}` | Minimum resources required for the `promxy-server-configmap-reload` container in the pods of `kof-mothership-promxy` deployment. |
| promxy<br>.enabled | bool | `true` | Enables `kof-mothership-promxy` deployment. |
| promxy<br>.extraArgs | object | `{"log-level":"info",`<br>`"web.external-url":"http://127.0.0.1:8082"}` | Extra command line arguments passed as `--key=value` to the `/bin/promxy`. |
| promxy<br>.gateway | object | `{"enabled":false,`<br>`"httpRoute":{"spec":{"hostnames":["kof-promxy.example.net"],`<br>`"parentRefs":[{"name":"mothership-gateway",`<br>`"namespace":"kof"}],`<br>`"rules":[{"backendRefs":[{"name":"kof-mothership-promxy",`<br>`"port":8082}]}]}}}` | Config of `kof-mothership-promxy` [Gateway](https://kubernetes.io/docs/concepts/services-networking/gateway/). |
| promxy<br>.image | object | `{"pullPolicy":"IfNotPresent",`<br>`"registry":"quay.io",`<br>`"repository":"jacksontj/promxy",`<br>`"tag":"v0.0.95"}` | Promxy image to use. |
| promxy<br>.ingress | object | `{"annotations":{},`<br>`"enabled":false,`<br>`"extraLabels":{},`<br>`"hosts":["example.com"],`<br>`"ingressClassName":"nginx",`<br>`"path":"/",`<br>`"pathType":"Prefix",`<br>`"tls":[]}` | Config of `kof-mothership-promxy` [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/). |
| promxy<br>.replicaCount | int | `3` | Number of replicated promxy pods. |
| promxy<br>.resources<br>.limits | object | `{"cpu":"2000m",`<br>`"memory":"2Gi"}` | Maximum resources available for the `promxy` container in the pods of `kof-mothership-promxy` deployment. |
| promxy<br>.resources<br>.requests | object | `{"cpu":"200m",`<br>`"memory":"1Gi"}` | Minimum resources required for the `promxy` container in the pods of `kof-mothership-promxy` deployment. |
| promxy<br>.service | object | `{"annotations":{},`<br>`"clusterIP":"",`<br>`"enabled":true,`<br>`"externalIPs":[],`<br>`"extraLabels":{},`<br>`"loadBalancerIP":"",`<br>`"loadBalancerSourceRanges":[],`<br>`"servicePort":8082,`<br>`"type":"ClusterIP"}` | Config of `kof-mothership-promxy` [Service](https://kubernetes.io/docs/concepts/services-networking/service/). |
| promxy<br>.serviceAccount<br>.annotations | object | `{}` | Annotations for the service account of promxy. |
| promxy<br>.serviceAccount<br>.create | bool | `true` | Creates a service account for promxy. |
| promxy<br>.serviceAccount<br>.name | string | `nil` | Name for the service account of promxy. If not set, it is generated as `kof-mothership-promxy`. |
| regionless | object | `{"domain":"mothership.example.com",`<br>`"enabled":false}` | Part of the regionless setup managed by the `kof` umbrella chart. |
| sveltos<br>.serviceMonitors | bool | `true` | Creates `ServiceMonitor`-s for Sveltos `sc-manager` and `addon-controller`. |
| victoria-logs-multilevel-select<br>.allowPartialResponse | bool | `true` | Allow returning partial results when some of the storage nodes are down or unreachable. |
| victoria-logs-multilevel-select<br>.enabled | bool | `true` | Enables VictoriaMetrics Logs aggregation. |
| victoria-logs-multilevel-select<br>.extraArgs | object | `{}` |  |
| victoria-logs-multilevel-select<br>.image<br>.repository | string | `"victoriametrics/victoria-logs"` |  |
| victoria-logs-multilevel-select<br>.image<br>.tag | string | `"v1.50.0"` |  |
| victoria-logs-multilevel-select<br>.replicaCount | int | `2` |  |
| victoria-logs-multilevel-select<br>.resources<br>.limits<br>.cpu | string | `"2000m"` |  |
| victoria-logs-multilevel-select<br>.resources<br>.limits<br>.memory | string | `"512Mi"` |  |
| victoria-logs-multilevel-select<br>.resources<br>.requests<br>.cpu | string | `"200m"` |  |
| victoria-logs-multilevel-select<br>.resources<br>.requests<br>.memory | string | `"512Mi"` |  |
| victoria-metrics-operator-service-template | object | `{"chart":"victoria-metrics-operator:0.58.1",`<br>`"namespace":"kcm-system",`<br>`"repo":{"name":"victoria-metrics",`<br>`"spec":{"type":"default",`<br>`"url":"https://victoriametrics.github.io/helm-charts"}},`<br>`"skipVerifyJob":true}` | Config of `ServiceTemplate` to use `victoria-metrics-operator` in `MultiClusterService`. |
| victoria-traces-multilevel-select<br>.allowPartialResponse | bool | `true` | Allow returning partial results when some of the storage nodes are down or unreachable. |
| victoria-traces-multilevel-select<br>.enabled | bool | `true` | Enables VictoriaMetrics Traces aggregation. |
| victoria-traces-multilevel-select<br>.extraArgs | object | `{}` |  |
| victoria-traces-multilevel-select<br>.image<br>.repository | string | `"victoriametrics/victoria-traces"` |  |
| victoria-traces-multilevel-select<br>.image<br>.tag | string | `"v0.8.1"` |  |
| victoria-traces-multilevel-select<br>.replicaCount | int | `2` |  |
| victoria-traces-multilevel-select<br>.resources<br>.limits<br>.cpu | string | `"2000m"` |  |
| victoria-traces-multilevel-select<br>.resources<br>.limits<br>.memory | string | `"512Mi"` |  |
| victoria-traces-multilevel-select<br>.resources<br>.requests<br>.cpu | string | `"200m"` |  |
| victoria-traces-multilevel-select<br>.resources<br>.requests<br>.memory | string | `"512Mi"` |  |
| victoriametrics<br>.enabled | bool | `true` | Enables VictoriaMetrics. |
| victoriametrics<br>.vmalert<br>.enabled | bool | `true` | Enables VMAlertManager only, as VMAlert is replaced with promxy in kof-mothership. |
| victoriametrics<br>.vmalert<br>.manager<br>.spec | object | `{"image":{"repository":"prom/alertmanager",`<br>`"tag":"v0.27.0"},`<br>`"port":"9093"}` | [VMAlertmanagerSpec](https://docs.victoriametrics.com/operator/api/#vmalertmanagerspec). |
| victoriametrics<br>.vmalert<br>.spec | object | `{"datasource":{"url":"http://vmselect-cluster:8481/select/0/prometheus"},`<br>`"evaluationInterval":"15s",`<br>`"extraArgs":{"http.pathPrefix":"/",`<br>`"notifier.blackhole":"true",`<br>`"remoteWrite.disablePathAppend":"true"},`<br>`"image":{"tag":"v1.105.0"},`<br>`"port":"8080",`<br>`"remoteRead":{"url":"http://vmselect-cluster:8481/select/0/prometheus"},`<br>`"remoteWrite":{"url":"http://vminsert-cluster:8480/insert/0/prometheus/api/v1/write"},`<br>`"selectAllByDefault":true}` | [VMAlertSpec](https://docs.victoriametrics.com/operator/api/#vmalertspec) |
| victoriametrics<br>.vmcluster<br>.enabled | bool | `true` | Enables high-available and fault-tolerant version of VictoriaMetrics database. |
| victoriametrics<br>.vmcluster<br>.spec | object | `{"license":{},`<br>`"replicationFactor":2,`<br>`"retentionPeriod":"30d",`<br>`"vminsert":{"extraArgs":{"maxLabelsPerTimeseries":"60"},`<br>`"image":{"tag":"v1.105.0-cluster"},`<br>`"podMetadata":{"labels":{"k0rdent.mirantis.com/istio-mtls-enabled":"true",`<br>`"k0rdent.mirantis.com/kof-victoria-metrics":"true"}},`<br>`"port":"8480",`<br>`"replicaCount":2,`<br>`"resources":{"limits":{"cpu":"1000m",`<br>`"memory":"1Gi"},`<br>`"requests":{"cpu":"200m",`<br>`"memory":"512Mi"}}},`<br>`"vmselect":{"cacheMountPath":"/select-cache",`<br>`"image":{"tag":"v1.105.0-cluster"},`<br>`"podMetadata":{"labels":{"k0rdent.mirantis.com/kof-victoria-metrics":"true"}},`<br>`"port":"8481",`<br>`"replicaCount":2,`<br>`"resources":{"limits":{"cpu":"1000m",`<br>`"memory":"2Gi"},`<br>`"requests":{"cpu":"200m",`<br>`"memory":"1Gi"}},`<br>`"storage":{"volumeClaimTemplate":{"spec":{"resources":{"requests":{"storage":"10Gi"}}}}}},`<br>`"vmstorage":{"image":{"tag":"v1.105.0-cluster"},`<br>`"replicaCount":3,`<br>`"resources":{"limits":{"cpu":"2000m",`<br>`"memory":"4Gi"},`<br>`"requests":{"cpu":"500m",`<br>`"memory":"2Gi"}},`<br>`"storage":{"volumeClaimTemplate":{"spec":{"resources":{"requests":{"storage":"100Gi"}}}}},`<br>`"storageDataPath":"/vm-data"}}` | VMCluster object spec |
| victoriametrics<br>.vmcluster<br>.spec<br>.replicationFactor | int | `2` | The number of replicas for each metric. |
| victoriametrics<br>.vmcluster<br>.spec<br>.retentionPeriod | string | `"30d"` | Days to retain the data |
| victoriametrics<br>.vmcluster<br>.spec<br>.vminsert<br>.podMetadata<br>.labels<br>."k0rdent<br>.mirantis<br>.com/istio-mtls-enabled" | string | `"true"` | Label to enable mtls |
| victoriametrics<br>.vmcluster<br>.spec<br>.vminsert<br>.podMetadata<br>.labels<br>."k0rdent<br>.mirantis<br>.com/kof-victoria-metrics" | string | `"true"` | Allows KOF UI to fetch internal metrics |
| victoriametrics<br>.vmcluster<br>.spec<br>.vminsert<br>.replicaCount | int | `2` | The number of replicas for vminsert |
| victoriametrics<br>.vmcluster<br>.spec<br>.vmselect<br>.podMetadata<br>.labels<br>."k0rdent<br>.mirantis<br>.com/kof-victoria-metrics" | string | `"true"` | Allows KOF UI to fetch internal metrics |
| victoriametrics<br>.vmcluster<br>.spec<br>.vmselect<br>.replicaCount | int | `2` | The number of replicas for vmselect |
| victoriametrics<br>.vmcluster<br>.spec<br>.vmselect<br>.storage<br>.volumeClaimTemplate<br>.spec<br>.resources<br>.requests<br>.storage | string | `"10Gi"` | Query results cache size. |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs v1.14.2](https://github.com/norwoodj/helm-docs/releases/v1.14.2)
