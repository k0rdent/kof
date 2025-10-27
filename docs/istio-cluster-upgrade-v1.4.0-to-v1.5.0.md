# Upgrade Guide (Istio only): v1.4.0 → v1.5.0

This document describes the steps required to upgrade from version v1.4.0 to v1.5.0.

**Important!!!**
Starting from **v1.5.0** release, Istio has been moved to a separate repository, which changes the installation and upgrade process. Upgrading KOF **will trigger a complete uninstallation** of all components across all clusters. To prevent data loss and ensure a smooth migration, carefully follow the steps in this document in the given order.

## 1. Back Up Data from Istio Clusters

> **Note:** Perform all backup steps in this section **for each Istio cluster** individually. Each cluster must have its own backup before proceeding with the upgrade.

### Create a PersistentVolumeClaim

Create a `PersistentVolumeClaim` named `backup-pvc` in the kof namespace:

```yaml
kubectl apply -f - <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: backup-pvc
  namespace: kof
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi # Adjust according to your data size
  storageClassName: standard
EOF
```

> **Note:** To estimate how much storage you need for the backup, open the [kof-ui](./ui.md), navigate to the `VictoriaMetrics/Logs` section, and click the `VictoriaMetrics storage` pod name on the cluster you want to back up. Find the `Data Size` metric and multiply this value by at least two (or by at least five for VictoriaLogs). This is because the metric shows data compressed using the VictoriaMetrics algorithm, while the backup will be stored in a simple gzip format, which does not compress as efficiently.

### Create a Backup Pod with `curl`

Deploy a temporary pod for performing backups:

```yaml
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: backup-deployment
  namespace: kof
spec:
  replicas: 1
  selector:
    matchLabels:
      app: backup-deployment
  template:
    metadata:
      labels:
        app: backup-deployment
    spec:
      containers:
      - name: backup
        image: curlimages/curl:latest
        command: ["/bin/sleep", "infinity"]
        volumeMounts:
        - mountPath: /home/curl_user
          name: backup-volume
      volumes:
      - name: backup-volume
        persistentVolumeClaim:
          claimName: backup-pvc
EOF
```

### Retrieve Cluster Endpoints

Locate the regional cluster ConfigMap named `kof-<cluster-name>` in the `kcm-system` namespace of the management cluster.

Record the following endpoint values:

* read_logs_endpoint
* read_metrics_endpoint

### Connect to the Backup Pod

```bash
kubectl exec -it -n kof <BACKUP_DEPLOYMENT_POD_NAME> -- /bin/sh
```

### Back Up VictoriaMetrics Data

```bash
curl -H 'Accept-Encoding: gzip' -sSN \
  <read_metrics_endpoint>/api/v1/export \
  -d 'match[]={__name__!=""}' \
  > victoria-metrics-backup.gz
```

### Back Up VictoriaLogs Data

```bash
curl -H 'Accept-Encoding: gzip' -sSN \
  <read_logs_endpoint>/select/logsql/query \
  -d 'query=*' > victoria-logs-backup.gz
```

### (Optional) Download Backups Locally

```bash
kubectl cp -n kof <BACKUP_DEPLOYMENT_POD_NAME>:victoria-metrics-backup.gz ./victoria-metrics-backup.gz
kubectl cp -n kof <BACKUP_DEPLOYMENT_POD_NAME>:victoria-logs-backup.gz ./victoria-logs-backup.gz
```

## 2. Cleanup the Old Istio Clusters

**Important**: Ensure that both VictoriaMetrics and VictoriaLogs backup files have been successfully created and verified before the cluster cleanup.

### Pause Synchronization on Managed Clusters

Before starting the cleanup, execute the following command on the management cluster:

```bash
kubectl scale --replicas=0 deployment/addon-controller -n projectsveltos
```

This command temporarily pauses synchronization between the management cluster and all managed clusters. It ensures that no old configurations are applied during the cleanup process.

### Cleanup Old KOF Components

Use the `kof-nuke.bash` script (located in the `kof/script` directory) to completely remove all old `kof` resources from the remote cluster.

Run the script for each **regional** and **child** cluster, for example:

```bash
export KUBECONFIG=regional-kubeconfig && ls $KUBECONFIG && scripts/kof-nuke.bash
```

Cleanup typically takes 1–5 minutes, depending on the cluster size and network speed.

### Cleanup Old Istio Components

Remove old Istio components for each **regional** and **child** cluster

```bash
export KUBECONFIG=regional-kubeconfig

helm uninstall --wait -n istio-system kof-istio-gateway
helm uninstall --wait -n istio-system kof-istio
helm uninstall --wait -n istio-system cert-manager
kubectl delete namespace istio-system --wait
kubectl get crd -o name | grep --color=never 'istio.io' | xargs kubectl delete
```

Once the cleanup of all regional and child clusters is complete,
run `unset KUBECONFIG` to use the **management** cluster in the next steps.

## 3. Remove Old Istio Chart

Remove the old Istio release and related resources from the management cluster:

```bash
helm un -n istio-system kof-istio
kubectl delete namespace istio-system --wait
kubectl get crd -o name | grep --color=never 'istio.io' | xargs kubectl delete
```

## 4. Deploy New Istio Release

Install new Istio release to the management cluster.

### Install `k0rdent-istio-base` Chart

```bash
helm upgrade -i --wait \
  --create-namespace -n istio-system k0rdent-istio-base \
  --set cert-manager-service-template.enabled=false \
  --set injectionNamespaces="{kof}" \
  oci://ghcr.io/k0rdent/istio/charts/k0rdent-istio-base --version 0.1.0
```

**Notes:**

* `cert-manager-service-template.enabled=false` disables the deployment of the cert-manager service template. It should already be deployed as part of KOF.
* `injectionNamespaces="{kof}"` ensures Istio sidecars are injected only into the `kof` namespace. To inject sidecars into additional namespaces, list them comma-separated: `{kof,<YOUR_NAMESPACE>}`.

### Install `k0rdent-istio` Chart

```bash
helm upgrade -i --wait -n istio-system k0rdent-istio \
  --set cert-manager-service-template.enabled=false \
  --set "istiod.meshConfig.extensionProviders[0].name=otel-tracing" \
  --set "istiod.meshConfig.extensionProviders[0].opentelemetry.port=4317" \
  --set "istiod.meshConfig.extensionProviders[0].opentelemetry.service=kof-collectors-daemon-collector.kof.svc.cluster.local" \
  oci://ghcr.io/k0rdent/istio/charts/k0rdent-istio --version 0.1.0
```

## 5. Upgrade KOF Version

Upgrade KOF to the target version following standard upgrade procedures.

## 6. Restart KOF Pods

Restarting KOF pods on the management cluster is required to update the `istio-proxy` sidecar. Without restarting, you may encounter connection errors or Istio pod crashes.

```bash
kubectl delete pod --all -n kof
```

**Notes:**

* If you have additional namespaces with Istio sidecar injection, make sure to restart pods in those namespaces as well.
* Restarting ensures all pods run with the updated Istio configuration and sidecar versions.

## 7. Resume Sveltos Synchronization

Resume synchronization between the management cluster and all managed (regional and child) clusters.

Run the following command on the management cluster:

```bash
kubectl scale --replicas=1 deployment/addon-controller -n projectsveltos
```

## 8. Add New Labels to ClusterDeployment

To enable Istio on your clusters, you need to add two new labels to the corresponding ClusterDeployment resources.

* `k0rdent.mirantis.com/istio-role: member` - Apply this label to СlustersDeployment where Istio should be installed.
* `k0rdent.mirantis.com/istio-gateway: "true"` - Apply this label only to regional clusters to install Istio gateway.

## 9. Restore Data

Use the backup created in step 2 to restore VictoriaMetrics and VictoriaLogs data.

### Connect to the Backup Pod

```bash
kubectl exec -it -n kof <BACKUP_DEPLOYMENT_POD_NAME> -- /bin/sh
```

### Restore VictoriaMetrics Data

```bash
curl -H 'Content-Encoding: gzip' -sSX POST \
  -T victoria-metrics-backup.gz \
  http://<CLUSTER_NAME>-vminsert:8480/insert/0/prometheus/api/v1/import
```

### Restore VictoriaLogs Data

```bash
curl -H 'Content-Encoding: gzip' -sSX POST \
  -T victoria-logs-backup.gz \
  http://<CLUSTER_NAME>-logs-insert:9481/insert/jsonline
```
