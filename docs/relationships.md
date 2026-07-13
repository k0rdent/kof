# k0rdent.mirantis.com — Resource Relationships

This document describes the relationships between custom resource kinds in the
`k0rdent.mirantis.com`, `kof.k0rdent.mirantis.com`, `config.projectsveltos.io`, and
`lib.projectsveltos.io` API groups, discovered from a live cluster using
`metadata.ownerReferences` and `spec`-level field references.

## Relationship types

| Arrow style | Meaning |
|---|---|
| `-.->` dashed | `ownerReference` — child is garbage-collected when owner is deleted |
| `-->` solid | `spec` field reference by name |
| Labels on arrows | the field or relationship type |

---

## Diagram

```mermaid
graph TD

  %% ── k0rdent: cluster-scoped core ────────────────────────────────

  subgraph k0rdent_cluster["k0rdent.mirantis.com — cluster-scoped"]
    Management["Management"]
    Release["Release"]
    AccessManagement["AccessManagement"]
    StateManagementProvider["StateManagementProvider"]
    ProviderTemplate["ProviderTemplate"]
    ProviderInterface["ProviderInterface"]
    MultiClusterService["MultiClusterService"]
  end

  Management -->|"spec.release"| Release
  Management -.->|"owns"| AccessManagement
  Management -.->|"owns"| StateManagementProvider
  Release -.->|"owns"| ProviderTemplate

  %% ── k0rdent: namespaced ──────────────────────────────────────────

  subgraph k0rdent_ns["k0rdent.mirantis.com — namespaced"]
    ClusterDeployment["ClusterDeployment"]
    Credential["Credential"]
    ServiceSet["ServiceSet"]
    ClusterTemplate["ClusterTemplate"]
    ClusterTemplateChain["ClusterTemplateChain"]
    ServiceTemplate["ServiceTemplate"]
    ServiceTemplateChain["ServiceTemplateChain"]
  end

  ClusterTemplateChain -->|"spec.supportedTemplates[]"| ClusterTemplate
  ServiceTemplateChain -->|"spec.supportedTemplates[]"| ServiceTemplate

  ClusterDeployment -->|"spec.template"| ClusterTemplate
  ClusterDeployment -->|"spec.credential"| Credential
  ClusterDeployment -.->|"owns"| ServiceSet

  Credential -->|"spec.identityRef"| Secret

  ServiceSet -->|"spec.multiClusterService"| MultiClusterService
  ServiceSet -->|"spec.provider"| StateManagementProvider

  MultiClusterService -->|"spec.services[].template\n(label-selector driven)"| ServiceTemplate

  %% ── core/v1 ──────────────────────────────────────────────────────

  Secret["Secret\n(core/v1)"]
  ConfigMap["ConfigMap\n(core/v1)"]

  %% ── kof.k0rdent.mirantis.com ─────────────────────────────────────

  subgraph kof["kof.k0rdent.mirantis.com — namespaced"]
    VMStorageConnection["VMStorageConnection"]
  end

  %% kof-operator path: regional ClusterDeployment → ConfigMap → VMStorageConnection
  ClusterDeployment -.->|"owns\n(regional clusters)"| ConfigMap
  ConfigMap -.->|"owns"| VMStorageConnection

  %% Release of kof-storage helm chart creates standalone VMStorageConnection in management cluster
  HelmRelease -.->|"owns\n(mothership)"| VMStorageConnection

  %% VMStorageConnection is consumed by the VMStorageConnection controller, which patches
  %% the referenced VMCluster/VLCluster/VTCluster's select component `-storageNode` ExtraArgs
  VMStorageConnection -->|"spec.cluster_ref\n(patches -storageNode ExtraArgs)"| MultilevelSelectCluster["VMCluster / VLCluster / VTCluster\n(operator.victoriametrics.com)"]

  %% ── flux/helm ────────────────────────────────────────────────────

  HelmRelease["HelmRelease\n(helm.toolkit.fluxcd.io)"]

  %% ── lib.projectsveltos.io ────────────────────────────────────────

  subgraph sveltos_lib["lib.projectsveltos.io — cluster/namespaced"]
    SveltosCluster["SveltosCluster\n(namespaced)"]
    Classifier["Classifier\n(cluster-scoped)"]
    DebuggingConfiguration["DebuggingConfiguration\n(cluster-scoped)"]
  end

  SveltosCluster -->|"spec.kubeconfigSecret"| Secret

  %% ── config.projectsveltos.io ─────────────────────────────────────

  subgraph sveltos_config["config.projectsveltos.io — namespaced"]
    Profile["Profile"]
    ClusterSummary["ClusterSummary"]
    ClusterConfiguration["ClusterConfiguration"]
    ClusterProfile["ClusterProfile\n(cluster-scoped)"]
  end

  %% ── Cross-group: k0rdent → projectsveltos ────────────────────────

  ServiceSet -.->|"owns"| Profile

  Profile -->|"spec.clusterRefs[]"| SveltosCluster
  Profile -.->|"owns"| ClusterSummary
  Profile -.->|"owns"| ClusterConfiguration
```

---

## Key relationships explained

### Management → Release → ProviderTemplate

`Management` is the cluster root object. It references a `Release` via `spec.release`.
The `Release` defines which Helm chart versions to use for every provider and **owns** all
`ProviderTemplate` instances — when the Release is deleted, all ProviderTemplates are
garbage-collected.

### ClusterDeployment — the primary user object

`ClusterDeployment` is the main user-facing resource. It wires together:

- **`spec.template`** → `ClusterTemplate` — what cluster topology/provider to use
- **`spec.credential`** → `Credential` → `Secret` — kubeconfig or cloud credentials
- **`ownerReferences`** → `ServiceSet` children — which services to install on the cluster

### MultiClusterService — standing policy via label selectors

`MultiClusterService` is a cluster-scoped policy object. It selects `ClusterDeployments` by
label and declares which `ServiceTemplates` to apply. When a `ClusterDeployment` matching the
selector is created, the KCM controller automatically creates a `ServiceSet` child linking
the two — so services are applied without any per-cluster configuration.

### ServiceSet → Profile (cross-group boundary)

`ServiceSet` is an **owned** child of `ClusterDeployment`. When `spec.provider` names a
`StateManagementProvider` backed by Sveltos, the KSM controller translates each `ServiceSet`
into a `config.projectsveltos.io/v1beta1 Profile`, also owned by the `ServiceSet`. This is
the key bridge between the k0rdent and Sveltos APIs.

### Profile — Sveltos execution unit

`Profile` is the Sveltos resource that performs the actual Helm/kustomize rendering onto a
target cluster. It targets clusters via:

- **`spec.clusterRefs[]`** → `SveltosCluster` — direct cluster reference
- **`spec.templateResourceRefs[]`** → management-cluster resources used during template
  instantiation — not a cluster-targeting reference; cluster selection belongs under
  `spec.clusterRefs[]` / `spec.clusterSelector`

It **owns** two status/audit resources:

- `ClusterSummary` — per-cluster summary of deployed resources
- `ClusterConfiguration` — per-cluster record of applied configuration

### SveltosCluster — cluster representation in Sveltos

`SveltosCluster` (`lib.projectsveltos.io`) is the Sveltos-side representation of a registered
cluster. It holds a reference to the kubeconfig `Secret` used to reach that cluster. Both
`Profile` and `MultiClusterService` resolve their target clusters through `SveltosCluster`.

### Template chains

`ClusterTemplateChain` and `ServiceTemplateChain` declare which template versions are supported
and available for upgrade, grouping related `ClusterTemplate` / `ServiceTemplate` versions
under a single chain object.

### VMStorageConnection — metrics, logs and traces aggregation routing (kof.k0rdent.mirantis.com)

`VMStorageConnection` is the only CRD in the `kof.k0rdent.mirantis.com` group. It registers a
remote storage read endpoint (target address, auth secret, TLS config) as a `-storageNode` on
a select-only multilevel-select cluster (`VMCluster` for metrics, `VLCluster` for logs/audit-logs,
`VTCluster` for traces) running on the mothership. Instances are created via two paths:

- **kof-operator** (for regional clusters): watches `ClusterDeployment` resources labelled
  `kof-cluster-role=regional`, creates an intermediate `ConfigMap` (owned by the
  `ClusterDeployment`) and then creates `VMStorageConnection` instances owned by that ConfigMap —
  one each for metrics, logs, audit-logs and traces.
- **kof-storage** installed to the management cluster creates standalone
  `VMStorageConnection` instances pointing at the local mothership VictoriaMetrics/VictoriaLogs/
  VictoriaTraces.

The `VMStorageConnectionReconciler` in kof-operator watches all `VMStorageConnection` objects
and rebuilds the referenced cluster's `-storageNode`/`-storageNode.usernameFile`/
`-storageNode.tls` ExtraArgs from every active connection pointing at it, giving the mothership
a single VictoriaMetrics-native query endpoint (`vmselect`/`vlselect`/`vtselect`) fanning out
across all regions. This replaced the previous `promxy`-based metrics aggregation.
