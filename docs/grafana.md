# Grafana

Grafana installation and automatic configuration is now disabled in KOF by default.

## Use KOF data without Grafana

If you want to keep Grafana disabled,
you can use KOF data without Grafana as described below.

### Metrics and alerts

* [Prometheus UI](https://docs.k0rdent.io/next/admin/kof/kof-alerts/#prometheus-ui):
  * Run in the management cluster:
    ```bash
    kubectl port-forward -n kof svc/kof-mothership-promxy 8082:8082
    ```
  * Graph: http://127.0.0.1:8082/graph?g0.expr=up&g0.tab=0
  * Alerts: http://127.0.0.1:8082/alerts
  * CLI queries for automation:
    ```bash
    curl http://localhost:8082/api/v1/query?query=up \
      | jq '.data.result | map(.metric.cluster) | unique'

    curl http://localhost:8082/api/v1/query?query=up \
      | jq '.data.result | map(.metric.job) | unique'

    curl http://localhost:8082/api/v1/query \
      -d 'query=up{cluster="mothership", job="kube-controller-manager"}' \
      | jq
    ```
* [Alertmanager UI](https://docs.k0rdent.io/next/admin/kof/kof-alerts/#alertmanager-ui):
  * Run in the management cluster:
    ```bash
    kubectl port-forward -n kof svc/vmalertmanager-cluster 9093:9093
    ```
  * Open http://127.0.0.1:9093/
* [VictoriaMetrics UI](https://docs.victoriametrics.com/victoriametrics/cluster-victoriametrics/#vmui):
  * Run in the regional cluster:
    ```bash
    KUBECONFIG=regional-kubeconfig kubectl port-forward \
      -n kof svc/vmselect-cluster 8481:8481
    ```
    To get metrics stored [from Management to Management](https://docs.k0rdent.io/next/admin/kof/kof-storing/#from-management-to-management) (if any),
    do this port-forward in the management cluster.
  * Open http://127.0.0.1:8481/select/0/vmui/#/dashboards

### Logs

* [VictoriaLogs UI](https://docs.victoriametrics.com/victorialogs/querying/#web-ui):
  * Run in the regional cluster:
    ```bash
    KUBECONFIG=regional-kubeconfig kubectl port-forward \
      -n kof svc/kof-storage-victoria-logs-cluster-vlselect 9471:9471
    ```
    We're using port 9471, not 9428.
  * Open http://127.0.0.1:9471/select/vmui/
  * CLI queries for automation:
    ```bash
    curl http://127.0.0.1:9471/select/logsql/query \
      -d 'query=_time:1h' \
      -d 'limit=10'
    ```
* Inside of Istio mesh:
  ```bash
  curl http://$REGIONAL_CLUSTER_NAME-logs-select:9471/select/logsql/query \
    -d 'query=_time:1h' \
    -d 'limit=10'
  ```
* Without Istio and port-forwarding:
  ```bash
  VM_USER=$(
    kubectl get secret -n kof storage-vmuser-credentials -o yaml \
    | yq .data.username | base64 -d
  )
  VM_PASS=$(
    kubectl get secret -n kof storage-vmuser-credentials -o yaml \
    | yq .data.password | base64 -d
  )
  curl https://vmauth.$REGIONAL_DOMAIN/vls/select/logsql/query \
    -u "$VM_USER":"$VM_PASS" \
    -d 'query=_time:1h' \
    -d 'limit=10'
  ```

### Traces

* Use [Jaeger UI](https://docs.k0rdent.io/v1.6.0/admin/kof/kof-using/#access-to-jaeger) for now.
* Jaeger will be replaced with VictoriaTraces soon:
  * https://github.com/k0rdent/kof/pull/679
  * [VictoriaTraces UI](https://docs.victoriametrics.com/victoriatraces/querying/#web-ui)

## Install and enable Grafana

If you want to install Grafana manually and enable its support in KOF, apply the next steps:

* If you had `kof-operators` chart version less than 1.6.0 installed, run:
  ```bash
  kubectl apply --server-side --force-conflicts \
    -f https://github.com/grafana/grafana-operator/releases/download/v5.18.0/crds.yaml
  ```
* Apply step 1 of the [Management Cluster](https://docs.k0rdent.io/next/admin/kof/kof-install/#management-cluster) section
  to install or upgrade `kof-operators` chart,
  adding `--set grafana-operator.enabled=true`
* Apply step 6 to install or upgrade `kof-mothership` chart,
  adding `--set grafana.enabled=true`
* Install Grafana manually, for example:
  ```bash
  kubectl apply -f - <<EOF
  apiVersion: grafana.integreatly.org/v1beta1
  kind: Grafana
  metadata:
    name: grafana-vm
    namespace: kof
    labels:
      dashboards: grafana
  spec:
    version: 10.4.18-security-01
    disableDefaultAdminSecret: true
    persistentVolumeClaim:
      spec:
        accessModes:
          - ReadWriteOnce
        resources:
          requests:
            storage: 200Mi
        # storageClassName: openebs-hostpath
    deployment:
      spec:
        template:
          spec:
            securityContext:
              fsGroup: 472
            volumes:
              - name: grafana-data
                persistentVolumeClaim:
                  claimName: grafana-vm-pvc
            containers:
              - name: grafana
                env:
                  - name: GF_SECURITY_ADMIN_USER
                    valueFrom:
                      secretKeyRef:
                        key: GF_SECURITY_ADMIN_USER
                        name: grafana-admin-credentials
                  - name: GF_SECURITY_ADMIN_PASSWORD
                    valueFrom:
                      secretKeyRef:
                        key: GF_SECURITY_ADMIN_PASSWORD
                        name: grafana-admin-credentials
                  - name: GF_INSTALL_PLUGINS
                    value: "victoriametrics-logs-datasource 0.21.0,victoriametrics-metrics-datasource 0.19.4"
  EOF
  ```
* You may optionally add features like `dex` and `ingress` from [this example](https://github.com/k0rdent/kof/blob/main/charts/kof-mothership/templates/grafana/grafana.yaml).
* Wait for Grafana installation to complete successfully:
  ```bash
  kubectl wait grafana -n kof grafana-vm \
    --for='jsonpath={.status.stage}=complete' \
    --for='jsonpath={.status.stageStatus}=success' \
    --timeout=5m
  ```
* Get access to Grafana:
  ```bash
  kubectl get secret -n kof grafana-admin-credentials -o yaml | yq '{
    "user": .data.GF_SECURITY_ADMIN_USER | @base64d,
    "pass": .data.GF_SECURITY_ADMIN_PASSWORD | @base64d
  }'

  kubectl port-forward -n kof svc/grafana-vm-service 3000:3000
  ```
* Login to http://127.0.0.1:3000/dashboards with the username/password printed above.
* Check the [Dashboards - Cluster Monitoring - Kubernetes / Views / Global](http://127.0.0.1:3000/d/k8s_views_global/kubernetes-views-global),
  it should show all clusters you collect metrics from.
* If you want to uninstall Grafana:
  ```bash
  kubectl delete --wait grafana -n kof grafana-vm
  ```
