# KCM Region

A KCM Region is a cluster that hosts and operates infrastructure objects such as Clusters, Machines, CAPI operators, Cert Manager, Velero, and enabled CAPI providers.

This setup separates responsibilities:

* The management cluster handles orchestration and overall control.
* Regional clusters manage workloads and infrastructure resources.

## Installing a KCM Regional Cluster

When setting up a KCM Regional cluster to work with KOF, make sure that the correct labels are applied for resource propagation:

* `k0rdent.mirantis.com/kcm-region-cluster: "true"` - Enables propagation of templates and required resources to the regional cluster.
* `k0rdent.mirantis.com/kof-aws-dns-secrets: "true"` - Propagates AWS DNS secrets to the regional cluster.
* `k0rdent.mirantis.com/kof-storage-secrets: "true"` - Propagates storage secrets to the regional cluster.

> Note: These labels are required when using KOF. To propagate resources to KCM child clusters, all necessary resources must first exist in the KCM Regional cluster. These labels will be deprecated in the future once KCM automatically propagates all required resources, as currently handled by the MultiClusterService.

### Installing the ClusterDeployment for a KCM Region

> Note: When creating a ClusterDeployment for a KCM Region, use a name shorter than 15 characters. Longer names may cause deployment errors.

To deploy a KCM Regional cluster to cloud, use the following command:

```bash
export KCM_REGION_NAME=region-$USER
make dev-kcm-region-deploy-cloud
```

Alternatively, to deploy it on an adopted cluster, use the following command:

```bash
export KCM_REGION_NAME=kcm-regional-adopted
make dev-kcm-region-deploy-adopted
```

### Registering the Regional Cluster

After the regional cluster is deployed, register it using the following command:

```bash
kubectl apply -f - <<EOF
apiVersion: k0rdent.mirantis.com/v1beta1
kind: Region
metadata:
  name: $KCM_REGION_NAME
spec:
  clusterDeployment:
    name: $KCM_REGION_NAME
    namespace: kcm-system
  providers:
  - name: cluster-api-provider-k0sproject-k0smotron
  - name: cluster-api-provider-aws
  - name: projectsveltos
EOF
```

Full documentation on KCM Regions can be [found here](https://docs.k0rdent.io/v1.5.0/admin/regional-clusters/regional-cluster-registration/).

### Credentials for the Regional Cluster

Create credentials that the KCM Regional cluster will use to deploy its child clusters:

```bash
kubectl apply -f - <<EOF
apiVersion: k0rdent.mirantis.com/v1beta1
kind: Credential
metadata:
  name: $KCM_REGION_NAME
  namespace: kcm-system
spec:
  region: $KCM_REGION_NAME
  description: "Credential for Regional cluster"
  identityRef:
    apiVersion: infrastructure.cluster.x-k8s.io/v1beta2
    kind: AWSClusterStaticIdentity
    name: aws-cluster-identity
EOF
```

Full documentation on Regional cluster credentials can be [found here](https://docs.k0rdent.io/v1.5.0/admin/regional-clusters/creating-credential-in-region/).

## Deploying Child Clusters in a KCM Region

To deploy child clusters within a KCM Region, use the standard cluster templates but reference the regional credentials in the `ClusterDeployment` specification.

Example:

```bash
apiVersion: k0rdent.mirantis.com/v1beta1
kind: ClusterDeployment
metadata:
  name: region-aws-ue2
  namespace: kcm-system
  labels:
    k0rdent.mirantis.com/kof-storage-secrets: "true"
    k0rdent.mirantis.com/kof-aws-dns-secrets: "true"
    k0rdent.mirantis.com/kof-cluster-role: regional
spec:
  template: aws-standalone-cp-1-0-16
  credential: $KCM_REGION_NAME
...
```

> Note: Within a KCM Region, all KOF clusters (both kof-regional and kof-child) operate in isolation from any KOF clusters that are not part of the same Region.

## Extending the KCM Regional Cluster with KOF Role

A KCM Regional cluster can also act as a KOF Regional cluster if it has enough capacity to deploy KOF Regional cluster workloads.
To enable this, add the following label to the KCM Regional ClusterDeployment:

```yaml
k0rdent.mirantis.com/kof-cluster-role: regional
```

This will install and configure the KOF components inside the KCM Region, allowing KOF Child clusters deployed within the same Region to connect to it.

> Note: A KCM Regional cluster cannot be extended with a KOF child role, but you can deploy a separate KOF child cluster within the same KCM Region if needed.
