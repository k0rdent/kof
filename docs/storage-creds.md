# Storage Credentials

## Overview

KOF automatically generates and manages credentials for accessing VictoriaMetrics storage components. When `ClusterDeployments` are deployed with KOF label, the KOF operator creates `VMUser` resources with secure credentials to provide authenticated access to metrics, logs, and traces storage.

## Automatic Credential Creation

For each cluster, the operator automatically creates the following objects:

### 1. Credentials Secret

A Kubernetes Secret containing the username and password:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: kof-vmuser-creds-<cluster-hash>
  namespace: <cluster-namespace>
type: Opaque
data:
  username: <base64-encoded-cluster-hash>
  password: <base64-encoded-random-password>
```

The secret is created in two locations:

- In the cluster's namespace (Used locally and also propagated to regional clusters)
- In the `kof` namespace (Used locally and also propagated to child clusters)

### 2. VMUser Resource

A `VMUser` resource that defines access permissions and target endpoints:

```yaml
apiVersion: operator.victoriametrics.com/v1beta1
kind: VMUser
metadata:
  name: kof-vmuser-<cluster-hash>
  namespace: <cluster-namespace>
  labels:
    app.kubernetes.io/managed-by: kof-operator
    k0rdent.mirantis.com/kof-generated: "true"
spec:
  username: <cluster-hash>
  passwordRef:
    name: kof-vmuser-creds-<cluster-hash>
    key: password
  targetRefs:
    # Access to VictoriaMetrics, VictoriaLogs, and VictoriaTraces
    - paths: ["/vm/select/.*"]
      static:
        url: "http://vmselect-cluster.kof.svc:8481"
    - paths: ["/vm/insert/.*"]
      static:
        url: "http://vminsert-cluster.kof.svc:8480"
    # ... additional endpoints
```

### 3. MultiClusterService (for Regional Clusters)

A `MultiClusterService` is created to propagate the `VMUser` and `Secret` to the regional cluster:

```yaml
apiVersion: k0rdent.mirantis.com/v1beta1
kind: MultiClusterService
metadata:
  name: kof-vmuser-propagation-<cluster-hash>
spec:
  clusterSelector:
    matchLabels:
      k0rdent.mirantis.com/kof-cluster-name: <regional-cluster-name>
  # ... propagation configuration
```

## Multi-Tenancy Support

If a cluster includes the tenant label `k0rdent.mirantis.com/kof-tenant-id`, the operator automatically applies:

- **Extra labels** on written data: `tenantId=<value>`
- **Extra filters** on read queries: `{tenantId="<value>"}`

This ensures data isolation between tenants. See [Multi-Tenancy](./multi-tenancy.md) for more details.
