---
name: troubleshoot
description: Troubleshoot KOF and KCM deployments by analyzing custom resources across k0rdent.mirantis.com, kof.k0rdent.mirantis.com, config.projectsveltos.io, and lib.projectsveltos.io API groups. Supports both live Kubernetes cluster inspection and offline support bundle directory analysis.
license: Apache-2.0
compatibility: opencode
metadata:
  audience: operators, support engineers
  workflow: troubleshooting
---

## What I do

Systematically inspect and report failures in KOF/KCM custom resources by walking the
ownership and reference graph described in `docs/relationships.md`. I cover:

- All `k0rdent.mirantis.com` resources: `Management`, `Release`, `ClusterDeployment`,
  `Credential`, `ServiceSet`, `MultiClusterService`, `ProviderTemplate`, `ClusterTemplate`,
  `ServiceTemplate`
- `kof.k0rdent.mirantis.com` resources: `PromxyServerGroup`
- `config.projectsveltos.io` resources: `Profile`, `ClusterSummary`, `ClusterConfiguration`
- `lib.projectsveltos.io` resources: `SveltosCluster`
- Supporting Flux resources: `HelmRelease`
- Supporting core resources: `Pod`, `Deployment`, `StatefulSet`, `Secret`, `ConfigMap`

## When to use me

Use this skill when asked to:
- Diagnose why a `ClusterDeployment` is not ready
- Find why services are not being deployed to a regional or child cluster
- Investigate HelmRelease or Profile failures
- Analyse a support bundle or exported object snapshots
- Produce a structured health report of the KOF stack

---

## Input mode

At the start of every session **ask the user which mode to use**:

> "Should I analyse a **live Kubernetes cluster** or a **support bundle directory**?"

### Mode A — Live cluster

Use `kubectl` commands to query the cluster directly. The current kubeconfig context is used
unless the user specifies `--context` or a kubeconfig path.

### Mode B — Support bundle directory

The user provides a path to a directory that contains exported Kubernetes object snapshots.
A bundle produced by the `support-bundle` tool has the following structure:

```
<bundle-dir>/                          # e.g. support-bundle-2026-04-10T05_51_50/
  analysis.json                        # bundle-level analysis summary
  version.yaml                         # tool version metadata
  cluster-info/                        # kubectl cluster-info dump
  cluster-resources/
    nodes.json                         # cluster-scoped resources as JSON
    namespaces.json
    pvs.json
    storage-classes.json
    clusterroles.json
    clusterrolebindings.json
    custom-resource-definitions.json
    pods/
      <namespace>.json                 # all pods in that namespace (JSON list)
    deployments/
      <namespace>.json
    statefulsets/
      <namespace>.json
    configmaps/
      <namespace>.json
    services/
      <namespace>.json
    <other-core-kind>/
      <namespace>.json
    custom-resources/
      <group>.json                     # cluster-scoped CRs (JSON list)
      <group>.yaml                     # same, YAML list (multi-document)
      <group>/                         # namespaced CRs split by namespace
        <namespace>.json
        <namespace>.yaml
  execution-data/                      # output of any exec collectors
  logs/                                # pod logs
    <namespace>/
      <pod-name>/
        <container-name>.log
```

Key conventions:
- Core namespaced resources (pods, deployments, statefulsets, configmaps, …) are stored as
  **JSON lists** at `cluster-resources/<kind>/<namespace>.json`.
- Custom resources are under `cluster-resources/custom-resources/`. Cluster-scoped CRs are
  stored as `<group>.yaml` (YAML multi-document) and `<group>.json`. Namespaced CRs have a
  subdirectory `<group>/` with one `<namespace>.yaml` and `<namespace>.json` per namespace.
- Each JSON file is a Kubernetes list object `{"kind":"List","items":[...]}` — use Python's
  `json.load` and iterate `.get('items', [])` rather than assuming it is an array.
- YAML files are multi-document streams (multiple `---` separated objects) — use
  `yaml.safe_load_all` or `grep`/`sed` for targeted field extraction.
- A bundle directory may contain **multiple** timestamped bundle subdirectories from
  different collection runs; always confirm with the user which one(s) to analyse.

Use the `Read`, `Glob`, `Grep`, and `Bash` (`cat … | python3`) tools to parse files from
disk instead of `kubectl`. Extract `kind`, `metadata.name`, `metadata.namespace`,
`metadata.ownerReferences`, `metadata.labels`, `metadata.annotations`, `spec`, and `status`
from each file.

**Python analysis scripts** are available in the `scripts/` directory (relative to this skill
file). They automate all 12 analysis steps and can be run directly or imported as modules.

| Script | Purpose |
|--------|---------|
| `scripts/analyze_bundle.py` | **Full analysis runner** — executes all 12 steps against a single bundle dir |
| `scripts/lib.py` | Shared helpers: `load_yaml_list`, `load_json_list`, `cr_path`, `core_path` |
| `scripts/step1_management_release.py` | Management and Release health |
| `scripts/step2_templates.py` | ProviderTemplate / ClusterTemplate / ServiceTemplate validity |
| `scripts/step3_credentials.py` | Credential readiness |
| `scripts/step4_clusterdeployments.py` | ClusterDeployment conditions and service states |
| `scripts/step5_servicesets.py` | ServiceSet deployed/provider.ready conditions |
| `scripts/step6_multiclusterservices.py` | MultiClusterService Ready conditions and dependencies |
| `scripts/step7_sveltos_clusters.py` | SveltosCluster connectionStatus / connectionFailures |
| `scripts/step8_profiles.py` | Profile matchingClusters |
| `scripts/step9_clustersummaries.py` | ClusterSummary featureSummaries / helmReleaseSummaries |
| `scripts/step10_helmreleases.py` | HelmRelease Ready conditions |
| `scripts/step11_promxyservergroups.py` | PromxyServerGroup presence, labels, and targets |
| `scripts/step12_workloads.py` | Pod / Deployment / StatefulSet health |

**Quick full analysis** (requires `pyyaml` — install once with
`pip3 install pyyaml --break-system-packages`):

```bash
# Analyse a single bundle
python3 scripts/analyze_bundle.py /path/to/support-bundle-<timestamp>

# Analyse a child/regional bundle whose kof namespace differs from default
python3 scripts/analyze_bundle.py /path/to/support-bundle-<timestamp> kof

# Analyse every bundle in an archive directory
for B in /path/to/bundle-archive/support-bundle-*/; do
  echo "=== $(basename $B) ==="
  python3 scripts/analyze_bundle.py "$B"
done
```

Each per-step script can also be run individually:

```bash
python3 scripts/step9_clustersummaries.py /path/to/support-bundle-<timestamp>
```

All scripts exit 0 when healthy, 1 when failures are found — suitable for use in CI or shell
pipelines.  Missing files (steps not applicable to a given bundle type, e.g. no Management on a
child cluster) are reported as `[WARN]` rather than `[FAIL]` and do not affect the exit code.

**Bundle path shortcuts for the standard layout:**

| Resource | Path pattern |
|---|---|
| Management | `cluster-resources/custom-resources/managements.k0rdent.mirantis.com.yaml` |
| Release | `cluster-resources/custom-resources/releases.k0rdent.mirantis.com.yaml` |
| ProviderTemplate | `cluster-resources/custom-resources/providertemplates.k0rdent.mirantis.com.yaml` |
| ClusterTemplate | `cluster-resources/custom-resources/clustertemplates.k0rdent.mirantis.com/kcm-system.yaml` |
| ServiceTemplate | `cluster-resources/custom-resources/servicetemplates.k0rdent.mirantis.com/kcm-system.yaml` |
| Credential | `cluster-resources/custom-resources/credentials.k0rdent.mirantis.com/kcm-system.yaml` |
| ClusterDeployment | `cluster-resources/custom-resources/clusterdeployments.k0rdent.mirantis.com/kcm-system.yaml` |
| ServiceSet | `cluster-resources/custom-resources/servicesets.k0rdent.mirantis.com/kcm-system.yaml` |
| MultiClusterService | `cluster-resources/custom-resources/multiclusterservices.k0rdent.mirantis.com.yaml` |
| PromxyServerGroup | `cluster-resources/custom-resources/promxyservergroups.kof.k0rdent.mirantis.com/kcm-system.yaml` or `cluster-resources/custom-resources/promxyservergroups.kof.k0rdent.mirantis.com/kof.yaml` |
| SveltosCluster | `cluster-resources/custom-resources/sveltosclusters.lib.projectsveltos.io/kcm-system.yaml` |
| Profile | `cluster-resources/custom-resources/profiles.config.projectsveltos.io/kcm-system.yaml` |
| ClusterSummary | `cluster-resources/custom-resources/clustersummaries.config.projectsveltos.io/kcm-system.yaml` |
| HelmRelease (kof) | `cluster-resources/custom-resources/helmreleases.helm.toolkit.fluxcd.io/kof.yaml` |
| HelmRelease (kcm) | `cluster-resources/custom-resources/helmreleases.helm.toolkit.fluxcd.io/kcm-system.yaml` |
| Pod | `cluster-resources/pods/<namespace>.json` |
| Deployment | `cluster-resources/deployments/<namespace>.json` |
| StatefulSet | `cluster-resources/statefulsets/<namespace>.json` |

---

## Analysis procedure

Run the full procedure top-down following the ownership graph. Collect all findings into a
structured report at the end. Do **not** stop at the first failure — continue through all
resources to build a complete picture.

**Preferred approach for bundle mode:** run the full analysis script first, then use
individual step scripts or ad-hoc reads to investigate specific failures:

```bash
python3 scripts/analyze_bundle.py <bundle-dir>
```

The steps below document what each script checks and the failure signals to look for.

### Step 1 — Management and Release

**Script:** `scripts/step1_management_release.py <bundle-dir>`

**Live:**
```bash
kubectl get management -o json | jq '{name: .items[0].metadata.name, status: .items[0].status}'
kubectl get release -o json | jq '[.items[] | {name: .metadata.name, status: .status}]'
```

**Bundle:** read `cluster-resources/custom-resources/managements.k0rdent.mirantis.com.yaml` and `cluster-resources/custom-resources/releases.k0rdent.mirantis.com.yaml`; extract `.status`.

**Failure signals:**
- `Management.status.conditions[type=Ready].status != "True"`
- `Management.status.components[*].success == false` — check `.template` field to identify which ProviderTemplate failed
- `Release.status.ready != true`
- `Release.status.conditions[type=TemplatesValid].status != "True"`

---

### Step 2 — ProviderTemplates, ClusterTemplates, ServiceTemplates

**Script:** `scripts/step2_templates.py <bundle-dir>`

**Live:**
```bash
kubectl get providertemplate -A -o json | jq '[.items[] | select(.status.valid != true) | {name: .metadata.name, valid: .status.valid}]'
kubectl get clustertemplate -A -o json | jq '[.items[] | select(.status.valid != true) | {name: .metadata.name, valid: .status.valid}]'
kubectl get servicetemplate -A -o json | jq '[.items[] | select(.status.valid != true) | {name: .metadata.name, valid: .status.valid}]'
```

**Bundle:** read `cluster-resources/custom-resources/providertemplates.k0rdent.mirantis.com.yaml`, `cluster-resources/custom-resources/clustertemplates.k0rdent.mirantis.com/kcm-system.yaml`, and `cluster-resources/custom-resources/servicetemplates.k0rdent.mirantis.com/kcm-system.yaml`.

**Failure signals:**
- `status.valid != true` on any template — the chart could not be resolved or linted

---

### Step 3 — Credentials

**Script:** `scripts/step3_credentials.py <bundle-dir>`

**Live:**
```bash
kubectl get credential -A -o json | jq '[.items[] | {name: .metadata.name, ns: .metadata.namespace, ready: .status.ready, conditions: .status.conditions}]'
```

**Bundle:** read `cluster-resources/custom-resources/credentials.k0rdent.mirantis.com/kcm-system.yaml`.

**Failure signals:**
- `status.ready != true`
- `status.conditions[type=CredentialReady].status != "True"`
- Referenced `Secret` (via `spec.identityRef`) does not exist

---

### Step 4 — ClusterDeployments

**Script:** `scripts/step4_clusterdeployments.py <bundle-dir>`

**Live:**
```bash
kubectl get clusterdeployment -A -o json | jq '[.items[] | {
  name: .metadata.name, ns: .metadata.namespace,
  conditions: .status.conditions,
  services: .status.services
}]'
```

**Bundle:** read `cluster-resources/custom-resources/clusterdeployments.k0rdent.mirantis.com/kcm-system.yaml`.

**Failure signals:**
- Any condition `status != "True"` — pay special attention to:
  - `CredentialReady` — credential or its Secret missing/invalid
  - `TemplateReady` — referenced ClusterTemplate invalid
  - `HelmChartReady` — chart could not be sourced
  - `HelmReleaseReady` — Helm install/upgrade failed; `reason != "InstallSucceeded"` and `reason != "UpgradeSucceeded"`
  - `ServicesInReadyState` — message format is `"N/M"` where N < M means some services not ready
- `status.services[*].state != "Deployed"` — individual service deployment failures

---

### Step 5 — ServiceSets

**Script:** `scripts/step5_servicesets.py <bundle-dir>`

**Live:**
```bash
kubectl get serviceset -A -o json | jq '[.items[] | select(
  .status.deployed != true or
  (.status.conditions[]? | select(.status != "True")) != null
) | {name: .metadata.name, ns: .metadata.namespace, status: .status}]'
```

**Bundle:** read `cluster-resources/custom-resources/servicesets.k0rdent.mirantis.com/kcm-system.yaml`.

**Failure signals:**
- `status.deployed != true`
- `status.provider.ready != true`
- `status.conditions[type=ServiceSetProfile].status != "True"` — Profile was not created
- `status.conditions[type=ServiceSetStatusesCollected].status != "True"` — Sveltos not reporting back

---

### Step 6 — MultiClusterServices

**Script:** `scripts/step6_multiclusterservices.py <bundle-dir>`

**Live:**
```bash
kubectl get multiclusterservice -o json | jq '[.items[] | select(
  (.status.conditions[]? | select(.status != "True")) != null
) | {name: .metadata.name, conditions: .status.conditions}]'
```

**Bundle:** read `cluster-resources/custom-resources/multiclusterservices.k0rdent.mirantis.com.yaml`.

**Failure signals:**
- `conditions[type=Ready].status != "True"`
- `conditions[type=ServicesReferencesValidation].status != "True"` — a referenced ServiceTemplate does not exist
- `conditions[type=ClusterInReadyState]` message shows `0/N` — selector matches clusters but none are ready

---

### Step 7 — SveltosClusters (lib.projectsveltos.io)

**Script:** `scripts/step7_sveltos_clusters.py <bundle-dir>`

**Live:**
```bash
kubectl get sveltoscluster -A -o json | jq '[.items[] | {
  name: .metadata.name, ns: .metadata.namespace,
  connectionStatus: .status.connectionStatus,
  ready: .status.ready,
  connectionFailures: .status.connectionFailures,
  version: .status.version
}]'
```

**Bundle:** read `cluster-resources/custom-resources/sveltosclusters.lib.projectsveltos.io/kcm-system.yaml` (and `mgmt.yaml` if present).

**Failure signals:**
- `status.connectionStatus != "Healthy"`
- `status.ready != true`
- `status.connectionFailures > 0` — incrementing counter indicates recurring connectivity issues
- Referenced `Secret` (kubeconfig) missing

---

### Step 8 — Profiles (config.projectsveltos.io)

**Script:** `scripts/step8_profiles.py <bundle-dir>`

**Live:**
```bash
kubectl get profile -A -o json | jq '[.items[] | {
  name: .metadata.name, ns: .metadata.namespace,
  matchingClusters: .status.matchingClusters
}]'
```

**Bundle:** read `cluster-resources/custom-resources/profiles.config.projectsveltos.io/kcm-system.yaml`.

**Failure signals:**
- `status.matchingClusters` is empty or null — no SveltosCluster matched the selector;
  check that the target cluster's labels match `spec.clusterSelector`

---

### Step 9 — ClusterSummaries (config.projectsveltos.io)

**Script:** `scripts/step9_clustersummaries.py <bundle-dir>`

**Live:**
```bash
kubectl get clustersummary -A -o json | jq '[.items[] | {
  name: .metadata.name, ns: .metadata.namespace,
  featureSummaries: .status.featureSummaries,
  helmReleaseSummaries: .status.helmReleaseSummaries
}]'
```

**Bundle:** read `cluster-resources/custom-resources/clustersummaries.config.projectsveltos.io/kcm-system.yaml`.

**Failure signals:**
- `status.featureSummaries[*].status != "Provisioned"` — possible values: `Failed`, `Processing` (stuck)
- `status.helmReleaseSummaries[*].status != "Managing"` — possible values: `Failed`, `Conflicts`
- Missing `helmReleaseSummaries` entirely when services should have been deployed

---

### Step 10 — HelmReleases (helm.toolkit.fluxcd.io)

**Script:** `scripts/step10_helmreleases.py <bundle-dir>`

**Live:**
```bash
kubectl get helmrelease -A -o json | jq '[.items[] | select(
  (.status.conditions[]? | select(.type == "Ready" and .status != "True")) != null
) | {name: .metadata.name, ns: .metadata.namespace, conditions: .status.conditions, history: .status.history}]'
```

**Bundle:** read `cluster-resources/custom-resources/helmreleases.helm.toolkit.fluxcd.io/kof.yaml` and `cluster-resources/custom-resources/helmreleases.helm.toolkit.fluxcd.io/kcm-system.yaml`.

**Failure signals:**
- `conditions[type=Ready].status != "True"`
- `conditions[type=Ready].reason` contains `Failed` (e.g. `InstallFailed`, `UpgradeFailed`, `ReconciliationFailed`)
- `history[0].status != "deployed"`
- Check `conditions[type=Ready].message` for the actual Helm error

---

### Step 11 — PromxyServerGroups (kof.k0rdent.mirantis.com)

**Script:** `scripts/step11_promxyservergroups.py <bundle-dir>`

**Live:**
```bash
kubectl get promxyservergroup -A -o json | jq '[.items[] | {
  name: .metadata.name, ns: .metadata.namespace,
  labels: .metadata.labels,
  ownerRefs: .metadata.ownerReferences,
  targets: .spec.targets,
  clusterName: .spec.cluster_name
}]'
```

**Bundle:** read `cluster-resources/custom-resources/promxyservergroups.kof.k0rdent.mirantis.com/kcm-system.yaml`.

**Failure signals (indirect — no `.status` field):**
- Missing expected instances: every regional `ClusterDeployment` should produce two
  `PromxyServerGroup` objects (one for metrics, one for logs) in the same namespace; if absent,
  check the `kof-operator` pod logs
- Missing label `k0rdent.mirantis.com/secret-name` — the promxy/vlogxy aggregator will not
  discover this group
- `spec.targets` is empty or points to an unreachable host
- Owning `ConfigMap` does not exist (orphaned group or controller crash)

---

### Step 12 — Pods, Deployments, StatefulSets

**Script:** `scripts/step12_workloads.py <bundle-dir> [<namespace> ...]`

Check core workloads in `kof` and `kcm-system` namespaces.

**Live:**
```bash
# Not-ready pods
kubectl get pod -A -o json | jq '[.items[] | select(
  .metadata.namespace == "kof" or .metadata.namespace == "kcm-system"
) | select(.status.phase != "Running" or
  (.status.conditions[]? | select(.type == "Ready" and .status != "True")) != null
) | {name: .metadata.name, ns: .metadata.namespace, phase: .status.phase,
     conditions: .status.conditions, containerStatuses: .status.containerStatuses}]'

# Not-available deployments
kubectl get deployment -A -o json | jq '[.items[] | select(
  .metadata.namespace == "kof" or .metadata.namespace == "kcm-system"
) | select(.status.availableReplicas < .spec.replicas) |
{name: .metadata.name, ns: .metadata.namespace, desired: .spec.replicas,
 available: .status.availableReplicas, conditions: .status.conditions}]'

# Not-ready statefulsets
kubectl get statefulset -A -o json | jq '[.items[] | select(
  .metadata.namespace == "kof" or .metadata.namespace == "kcm-system"
) | select(.status.readyReplicas < .spec.replicas) |
{name: .metadata.name, ns: .metadata.namespace, desired: .spec.replicas, ready: .status.readyReplicas}]'
```

**Bundle:** parse `cluster-resources/pods/kof.json`, `cluster-resources/pods/kcm-system.json`,
`cluster-resources/deployments/kof.json`, `cluster-resources/deployments/kcm-system.json`,
`cluster-resources/statefulsets/kof.json`, `cluster-resources/statefulsets/kcm-system.json`
using `scripts/step12_workloads.py` or the shared `load_json_list()` helper in `scripts/lib.py`.
Note: `phase=Succeeded` pods with `ready=False` container state are **completed Jobs** — not failures.

**Failure signals:**
- Pod `phase != "Running"` or `Pending` stuck
- Container `state.waiting.reason` is `CrashLoopBackOff`, `ImagePullBackOff`, `OOMKilled`
- Deployment `availableReplicas < replicas`
- Deployment condition `Available=False` (`reason=MinimumReplicasUnavailable`)
- StatefulSet `readyReplicas < replicas`

---

## Output format

After completing all steps, produce a structured report with these sections:

### 1. Summary table

```
| Resource kind         | Name                  | Namespace   | Status  | Issue                          |
|-----------------------|-----------------------|-------------|---------|--------------------------------|
| ClusterDeployment     | regional-adopted      | kcm-system  | FAILED  | HelmReleaseReady=False: ...    |
| SveltosCluster        | regional-adopted      | kcm-system  | FAILED  | connectionStatus=Unhealthy     |
| HelmRelease           | kof-storage           | kof         | FAILED  | InstallFailed: ...             |
```

Use `OK` for healthy resources, `FAILED` for definitive failures, `WARN` for degraded or
ambiguous states (e.g. `connectionFailures > 0` but still healthy, or `0/0` cluster counts).

### 2. Root cause analysis

For each `FAILED` entry, trace the ownership chain upward to identify whether the failure is
a root cause or a downstream symptom. State clearly which resource is the **root cause**.

Example:
```
ROOT CAUSE: SveltosCluster/regional-adopted — connectionStatus=Unhealthy
  ↳ SYMPTOM: Profile/regional-adopted-f4f250a9 — matchingClusters=[]
  ↳ SYMPTOM: ServiceSet/regional-adopted-f4f250a9 — ServiceSetProfile=False
  ↳ SYMPTOM: ClusterDeployment/regional-adopted — ServicesInReadyState=False (0/4)
```

### 3. Recommended actions

For each root cause, list specific remediation steps. Examples drawn from common failure patterns:

**SveltosCluster unhealthy / connectionFailures > 0:**
```bash
# Check the kubeconfig secret is valid and the cluster API is reachable
kubectl get secret <name>-kubeconf -n kcm-system -o jsonpath='{.data.value}' | base64 -d | kubectl --kubeconfig /dev/stdin cluster-info
```

**Profile has no matchingClusters:**
```bash
# Verify ClusterDeployment labels match MultiClusterService selector
kubectl get clusterdeployment <name> -n kcm-system --show-labels
kubectl get multiclusterservice <mcs-name> -o jsonpath='{.spec.clusterSelector}'
```

**HelmRelease failed:**
```bash
kubectl describe helmrelease <name> -n <namespace>
kubectl logs -n <namespace> -l app.kubernetes.io/name=helm-controller --tail=100
```

**ClusterTemplate / ServiceTemplate invalid:**
```bash
kubectl describe clustertemplate <name> -n kcm-system
# Check the referenced HelmChart source
kubectl get helmchart -n kcm-system
```

**PromxyServerGroup missing for a regional cluster:**
```bash
# Check kof-operator logs for reconciliation errors
kubectl logs -n kof -l app.kubernetes.io/name=kof-operator --tail=200
# Verify the ClusterDeployment has the correct role label
kubectl get clusterdeployment <name> -n kcm-system -o jsonpath='{.metadata.labels}'
```

**Pod CrashLoopBackOff / OOMKilled:**
```bash
kubectl logs -n <namespace> <pod-name> --previous
kubectl describe pod -n <namespace> <pod-name>
```

---

## Quick single-command health check (live cluster)

Run this to get an immediate overview of all non-healthy resources across all watched kinds:

```bash
# All failed/not-ready conditions across k0rdent and Sveltos CRs
for resource in management release clusterdeployment serviceset multiclusterservice credential; do
  echo "=== $resource ==="
  kubectl get $resource -A -o json 2>/dev/null | \
    jq -r '.items[] | select(any(.status.conditions[]?; .status != "True")) |
    "\(.metadata.namespace)/\(.metadata.name): \(([.status.conditions[]? | select(.status != "True") | "\(.type)=\(.status) (\(.reason)): \(.message)"] | join("; ")))"'
done

# Invalid templates
for resource in providertemplate clustertemplate servicetemplate; do
  echo "=== $resource ==="
  kubectl get $resource -A -o json 2>/dev/null | \
    jq -r '.items[] | select(.status.valid != true) | "\(.metadata.namespace)/\(.metadata.name): valid=false"'
done

# SveltosCluster connectivity
kubectl get sveltoscluster -A -o json | \
  jq -r '.items[] | select(.status.ready != true or .status.connectionStatus != "Healthy") |
  "\(.metadata.namespace)/\(.metadata.name): ready=\(.status.ready) connectionStatus=\(.status.connectionStatus) failures=\(.status.connectionFailures)"'

# ClusterSummary feature failures
kubectl get clustersummary -A -o json | \
  jq -r '.items[] | .metadata.name as $n | .status.featureSummaries[]? |
  select(.status != "Provisioned") | "\($n): featureID=\(.featureID) status=\(.status)"'

# HelmRelease failures
kubectl get helmrelease -A -o json | \
  jq -r '.items[] | (.status.conditions[]? | select(.type == "Ready" and .status != "True")) as $ready |
  "\(.metadata.namespace)/\(.metadata.name): \($ready.reason): \($ready.message)"'

# Not-running pods in kof and kcm-system
kubectl get pod -A -o json | \
  jq -r '.items[] | select(.metadata.namespace == "kof" or .metadata.namespace == "kcm-system") |
  select(.status.phase != "Running" and .status.phase != "Succeeded") |
  "\(.metadata.namespace)/\(.metadata.name): phase=\(.status.phase)"'
```

---

## Resource health reference

| Kind | API group | Health field(s) | Healthy value | Failure value |
|------|-----------|-----------------|---------------|---------------|
| Management | k0rdent.mirantis.com | `conditions[Ready].status` + `components[*].success` | `True` + all `true` | `False` or any component `false` |
| Release | k0rdent.mirantis.com | `status.ready` + `conditions[TemplatesValid].status` | `true` + `True` | `false` or `False` |
| ProviderTemplate | k0rdent.mirantis.com | `status.valid` | `true` | `false` |
| ClusterTemplate | k0rdent.mirantis.com | `status.valid` | `true` | `false` |
| ServiceTemplate | k0rdent.mirantis.com | `status.valid` | `true` | `false` |
| Credential | k0rdent.mirantis.com | `status.ready` + `conditions[CredentialReady].status` | `true` + `True` | `false` or `False` |
| ClusterDeployment | k0rdent.mirantis.com | `conditions[*].status` + `services[*].state` | all `True` + all `Deployed` | any `False` or state ≠ `Deployed` |
| ServiceSet | k0rdent.mirantis.com | `status.deployed` + `status.provider.ready` + `conditions[*].status` | all `true` + all `True` | any `false` or `False` |
| MultiClusterService | k0rdent.mirantis.com | `conditions[Ready].status` | `True` | `False` |
| PromxyServerGroup | kof.k0rdent.mirantis.com | presence + owning ConfigMap + labels | exists + owner present + labels set | missing instance or missing label |
| SveltosCluster | lib.projectsveltos.io | `status.connectionStatus` + `status.ready` | `Healthy` + `true` | any other value or `connectionFailures > 0` |
| Profile | config.projectsveltos.io | `status.matchingClusters` | non-empty list | empty or null |
| ClusterSummary | config.projectsveltos.io | `featureSummaries[*].status` + `helmReleaseSummaries[*].status` | `Provisioned` + `Managing` | `Failed` or `Processing` (stuck) |
| HelmRelease | helm.toolkit.fluxcd.io | `conditions[Ready].status` + `conditions[Ready].reason` | `True` + `*Succeeded` | `False` or reason contains `Failed` |
| Deployment | apps/v1 | `conditions[Available].status` | `True` | `False` (`MinimumReplicasUnavailable`) |
| StatefulSet | apps/v1 | `readyReplicas == spec.replicas` | equal | `readyReplicas < replicas` |
| Pod | v1 | `phase` + `conditions[Ready].status` | `Running` + `True` | `Pending` / `Failed` / `CrashLoopBackOff` / `OOMKilled` |
