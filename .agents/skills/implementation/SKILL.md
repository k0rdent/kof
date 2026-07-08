---
name: implementation
description: Use when implementing features, fixing bugs, or making architectural changes in the KOF (k0rdent Observability and FinOps) repository. Provides high-level context on how the major components — charts, operator controllers, CRDs, and multi-cluster policies — relate to each other so that changes in one area are made with awareness of downstream effects.
---

# KOF Implementation Guide

Architectural context for the KOF repository to help reason about cross-component impact before making changes.

---

## Repository Layout

```
charts/             All Helm charts
kof-operator/       Single Go module: operator + supporting binaries
  api/v1beta1/      KOF-native CRD type definitions
  cmd/              Binary entry point — registers all controllers
  internal/
    controller/     Reconciler implementations
    env/            Operator runtime config (env vars)
    models/labels/  Label/annotation key constants
    names/          Stable name hashing
    server/         HTTP API server + UI handlers
    vmuser/         VMUser lifecycle helpers
  webapp/           Embedded React+TypeScript UI
config/             Kind cluster configs (local dev/CI)
docs/               Feature documentation
tests/              Integration tests and reference manifests
scripts/            Smoke tests and CI utilities
.github/workflows/  CI/CD pipelines
```

---

## Cluster Roles and Topology

Every `ClusterDeployment` must carry `k0rdent.mirantis.com/kof-cluster-role`.

| Role | Label Value | What runs there |
|---|---|---|
| Mothership | *(management cluster itself)* | KCM, kof-operator, VMCluster, promxy, Grafana, VMAlert |
| Regional | `regional` | kof-storage (VMCluster + VLCluster + VTCluster + VMAuth), kof-collectors |
| Child | `child` | kof-collectors only |

**Data path**: Child → Regional VMAuth → Regional storage → Mothership (promxy federation for metrics; VMStorageConnection for logs/traces).

**Regionless mode** (`regionless.enabled: true` in `charts/kof/values.yaml`) collapses regional into mothership; child clusters write directly to the mothership.

---

## The Five Controllers (`kof-operator/internal/controller/`)

### 1. ClusterDeploymentReconciler
Watches `ClusterDeployment` (KCM).
- **Regional**: creates/updates regional ConfigMap with endpoint metadata (metrics, logs, traces, audit-logs URLs) derived from annotations. In Istio mode, only read endpoints are stored; write endpoints are baked into the child MCS.
- **Child**: discovers parent regional cluster, creates child ConfigMap, generates VMUser credentials, creates propagation MCS.

**Impact**: renaming ConfigMap fields or annotation keys breaks MCS `templateResourceRefs` downstream.

### 2. RegionalClusterConfigMapReconciler
Watches regional ConfigMaps. On change, creates/updates: vmrules ConfigMap, VMUser + propagation MCS, `PromxyServerGroup` CR, `VMStorageConnection` CRs for logs and traces.

**Impact**: regional ConfigMap field changes cascade to all of the above.

### 3. PromxyServerGroupReconciler
Watches `PromxyServerGroup` CRs. Aggregates them by `secret-name` label, renders promxy config, updates the promxy Secret, and POSTs `/-/reload`.

**Impact**: spec changes must be reflected in the config template under `kof-operator/internal/controller/`.

### 4. AlertsConfigMapReconciler
Watches alert/record rule ConfigMaps. Merges `PrometheusRule` CRs and per-cluster ConfigMaps into the mothership promxy rules Secret and per-cluster vmrules ConfigMaps (injected into `kof-storage` via MCS).

**Impact**: `defaultRules.*` in `charts/kof-mothership/values.yaml` controls which rule groups are enabled.

### 5. VMStorageConnectionReconciler
Watches `VMStorageConnection` CRs. Patches `VTCluster`/`VLCluster` extraArgs to add storage node addresses. Uses a finalizer for clean deletion.

**Impact**: mothership VLCluster/VTCluster must exist before regional clusters register — `kof-mothership` must deploy first.

---

## Helm Chart Dependency Chain

### The `charts/kof` Umbrella Chart

`charts/kof` is the single entry point — it creates FluxCD HelmRelease objects for all components. No workloads run in this chart itself.

`charts/kof/values.yaml` is the **top-level config for an entire KOF installation**. Values flow down to each component chart through HelmReleases. Every key under a component's `values` block must exist in that component's own `values.yaml`.

The HelmRelease templates in `charts/kof/templates/` merge additional values based on top-level flags (`regionless.*`, `tls.*`) at Helm render time — this is how feature flags propagate without duplicating values.

**Install order** is defined by `dependsOn` per component in `charts/kof/values.yaml`. When adding a new chart, add it to `global.components` and set its `dependsOn`.

### Chart-to-Cluster Deployment Map

| Chart | Deployed Where | How |
|---|---|---|
| `kof-operators` | Mothership | FluxCD HelmRelease |
| `victoria-metrics-operator` | Mothership | FluxCD HelmRelease |
| `kof-mothership` | Mothership | FluxCD HelmRelease |
| `kof-dashboards` | Mothership | Bundled dependency of kof-mothership and kof-storage |
| `kof-regional` | Mothership | HelmRelease — creates MCS policy objects only |
| `kof-child` | Mothership | HelmRelease — creates MCS policy objects only |
| `kof-storage` | Regional clusters | MCS rendered by kof-regional |
| `kof-collectors` | Regional and child clusters | MCS rendered by kof-regional and kof-child |
| `kof-propagation` | Mothership | ServiceTemplate for ConfigMap propagation |
| `audit-logs-exporter` | Regional clusters | Dependency of kof-storage |
| `cold-storage-exporter` | Regional/mothership | Dependency of kof-storage |

### Two-Layer Helm Templating in MCS Values

`kof-regional` and `kof-child` each render a `MultiClusterService`. Their `services[].values` use **Helm-within-Helm templates**: the outer layer runs at mothership install time (bakes in umbrella values and feature flags), the inner layer runs at Sveltos apply time (reads live cluster annotations and referenced Secrets).

**Merge priority** (last wins):
```
global defaults  <  computed values  <  Helm values (kof umbrella)  <  per-cluster annotation overrides
```

Operators can override any collector or storage value per cluster via a JSON annotation on `ClusterDeployment` (e.g., `k0rdent.mirantis.com/kof-collectors-values`).

### Values Consistency

A CI script in `scripts/` validates that every default in `charts/kof/values.yaml` exists at the same path in the component's own chart. Run it after changing defaults in either location.

---

## KOF-Native CRDs (`kof-operator/api/v1beta1/`, group `kof.k0rdent.mirantis.com/v1beta1`)

**PromxyServerGroup** — describes one remote Prometheus-compatible backend for promxy. Key fields: `targets`, `pathPrefix`, `scheme`, `clusterName`, `basicAuth`, `tlsConfig`. The `k0rdent.mirantis.com/secret-name` label groups multiple objects into one promxy config Secret.

**VMStorageConnection** — registers a remote VictoriaMetrics storage node with a `VTCluster` or `VLCluster`. Key fields: `clusterRef`, `address`, `authSecret`, `tlsInsecureSkipVerify`.

---

## Multi-Cluster Propagation Pattern

1. Operator creates a `MultiClusterService` in `kcm-system`.
2. MCS carries a cluster selector and `services` list with `ServiceTemplate` references.
3. KCM/Sveltos renders and applies templates to matching clusters.
4. `templateResourceRefs` injects ConfigMap/Secret values from the mothership at render time.

**Naming conventions**:
- Regional ConfigMap: `kof-{clusterName}`
- Child ConfigMap: `kof-cluster-config-{clusterName}`
- vmrules ConfigMap: `kof-record-vmrules-{clusterName}`
- VMUser Secret: `storage-vm-user-credentials-{hashedName}`
- Propagation MCS: `kof-config-{clusterName}` / `kof-vmrules-{clusterName}`

Name hashing in `kof-operator/internal/names/` uses FNV/Adler-32. The Helm `adler32sum` helper in charts must produce matching names — **keep them in sync**.

---

## Istio Mode

Replaces Envoy Gateway, cert-manager, and external-dns with Istio service mesh. Enabled by labelling the `kof` namespace with `istio-injection: enabled`.

**Cluster labels**: `k0rdent.mirantis.com/istio-role: member` (all Istio clusters); `k0rdent.mirantis.com/istio-gateway: "true"` (regional east-west gateways).

### Parallel MCS Tracks

Both `kof-regional` and `kof-child` render two MCS objects — standard and Istio — with non-overlapping selectors.

| | Standard | Istio |
|---|---|---|
| Regional MCS | `kof-regional-cluster` | `kof-istio-regional-cluster` |
| Child MCS | `kof-child-cluster` | `kof-istio-child-cluster` |

### What Changes in Istio Mode

- `cert-manager`, `envoy-gateway`, `external-dns` are not installed on regional clusters.
- Write endpoints are not stored in the regional ConfigMap; they are baked into the child MCS using the cluster name. Only read endpoints are stored.
- `kof-storage/templates/istio/` creates an in-mesh `Service` (`{clusterName}-vmauth`) instead of an HTTPRoute + TLS cert.
- Collector DaemonSet: `hostNetwork: false`, OTLP receivers removed, `controller-k0s` disabled.
- Operator OTLP export target: `http://kof-collectors-daemon-collector:4317` instead of `http://$(NODE_IP):4317`.
- Namespace bootstrapping: `charts/kof-mothership/templates/istio/` creates the `kof-namespace-template` ConfigMap, a ServiceTemplate, and MCS objects to label the `kof` namespace on all Istio-member clusters.
- VMUser propagation MCS `dependsOn` references `kof-istio-regional-cluster` instead of `kof-regional-cluster`.

### Istio Impact

| Change | Affected Areas |
|---|---|
| Add a new storage endpoint | `defaultEndpoints` + `istioEndpoints` in `kof-operator/internal/controller/`; both standard and Istio MCS templates |
| Modify collector config | Both standard and Istio value blocks in `kof-regional` and `kof-child` MCS templates |
| Add a new kof-storage service | May need a corresponding in-mesh Service in `kof-storage/templates/istio/` |
| Change endpoint URL pattern | Controller endpoint derivation, `PromxyServerGroup` target construction, both MCS templates |

---

## Environment Variables (`kof-operator/internal/env/`)

Injected by the operator Deployment template in `charts/kof-mothership/templates/kof-operator/`. Key variables:

- `RELEASE_NAMESPACE`, `RELEASE_NAME`
- `KOF_REGIONLESS_ENABLED`, `KOF_REGIONLESS_DOMAIN`
- `KOF_PROPAGATION_TEMPLATE` — ServiceTemplate name for ConfigMap propagation
- `KOF_REGIONAL_CLUSTER_MCS_NAME`, `KOF_CHILD_CLUSTER_MCS_NAME`
- `KOF_VL_CLUSTER_NAME`, `KOF_VT_CLUSTER_NAME` — enable VMStorageConnection creation
- `KOF_VL_SELECT_URL`, `KOF_VL_INSERT_URL`, `KOF_VT_SELECT_URL`, `KOF_VT_INSERT_URL`
- `KOF_GRAFANA_ENABLED`

**When adding a new env var**: add to `kof-operator/internal/env/` first, then add value + template entry in `charts/kof-mothership/templates/kof-operator/`.

---

## Label and Annotation Conventions

Constants in `kof-operator/internal/models/labels/`; controller-specific keys defined near their usage in `kof-operator/internal/controller/`.

**ClusterDeployment labels**: `kof-cluster-role` (regional/child), `kof-regional-cluster-name` (explicit parent override), `kof-cluster-name`, `istio-role: member`, `istio-gateway: "true"`.

**ClusterDeployment annotations**: `kof-regional-domain` (base domain for endpoint derivation), explicit per-endpoint overrides (`kof-write-metrics-endpoint`, `kof-write-logs-endpoint`, `kof-write-traces-endpoint`, `kof-write-audit-logs-endpoint`, `kof-read-metrics-endpoint`).

**ConfigMap labels** (controller watch triggers): `kof-cluster-role: regional`, `kof-alert-rules-cluster-name`, `kof-record-rules-cluster-name`, `kof-record-vmrules-cluster-name`.

---

## Cross-Component Impact Matrix

| Change | Downstream Impact |
|---|---|
| Add/rename a regional ConfigMap field | `kof-regional` MCS `templateResourceRefs`, ClusterDeploymentReconciler, RegionalClusterConfigMapReconciler |
| Add/rename a child ConfigMap field | `kof-child` MCS `templateResourceRefs`, `kof-collectors` values |
| Change `PromxyServerGroup` spec | PromxyServerGroupReconciler config template, promxy Deployment |
| Change `VMStorageConnection` spec | VMStorageConnectionReconciler, VLCluster/VTCluster ExtraArgs |
| Add a new storage endpoint | Controller endpoint maps, regional ConfigMap fields, both standard and Istio MCS templates |
| Add a new env var | `kof-operator/internal/env/`, `charts/kof-mothership/values.yaml`, operator Deployment template |
| Add a new controller | `kof-operator/cmd/` registration, RBAC in `charts/kof-mothership/templates/kof-operator/` |
| Add a new chart | `charts/kof/templates/` (HelmRelease + HelmChart), `charts/kof/values.yaml` (defaults + dependsOn) |
| Change VMUser credential structure | `kof-operator/internal/controller/vmuser/`, `kof-storage` VMAuth/VMUser templates, `kof-collectors` values, `kof-child` MCS `templateResourceRefs` |
| Change storage endpoint URL scheme | Controller endpoint derivation, `kof-operator/internal/env/`, `kof-storage` VMAuth route config, Istio endpoint map |
| Change recording rules format | AlertsConfigMapReconciler, `kof-storage` `valuesFrom`, `kof-regional` MCS `templateResourceRefs` |
| Change cluster name hashing | `kof-operator/internal/names/`, Helm `adler32sum` helpers (must stay in sync) |

---

## Testing and Validation

```bash
# Unit tests
cd kof-operator && go test ./...

# Helm lint / render
helm lint charts/<chart-name>
helm template test charts/<chart-name> -f charts/<chart-name>/values.yaml

# Values consistency check
python scripts/check_values_consistency.py

# Support bundle
make support-bundle
```

Integration tests run three CI scenarios: `dev`, `dev-istio`, `dev-regionless`. See the `kind-deploy` skill for local setup and the `troubleshoot` skill for bundle analysis.

---

## Key Locations Quick Reference

| Location | Purpose |
|---|---|
| `kof-operator/cmd/` | Controller registration, manager startup |
| `kof-operator/internal/env/` | Operator env var definitions |
| `kof-operator/internal/models/labels/` | Label/annotation key constants |
| `kof-operator/internal/names/` | Stable name hashing (must match Helm `adler32sum`) |
| `kof-operator/api/v1beta1/` | KOF CRD type definitions |
| `kof-operator/internal/controller/` | All reconciler implementations, endpoint maps |
| `charts/kof/values.yaml` | Top-level umbrella config (single source of truth) |
| `charts/kof/templates/` | FluxCD HelmRelease/HelmChart objects, install order |
| `charts/kof-mothership/values.yaml` | Mothership component configuration |
| `charts/kof-mothership/templates/kof-operator/` | Operator RBAC and Deployment |
| `charts/kof-mothership/templates/istio/` | Istio namespace bootstrapping |
| `charts/kof-regional/templates/` | Standard and Istio MCS for regional clusters |
| `charts/kof-child/templates/` | Standard and Istio MCS for child clusters |
| `charts/kof-storage/templates/istio/` | Istio in-mesh VMAuth Service |
| `charts/kof-collectors/values.yaml` | Collector configuration |
