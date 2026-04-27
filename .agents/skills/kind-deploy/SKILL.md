---
name: kind-deploy
description: Deploy KCM and KOF on local kind clusters for development and testing. Covers the full workflow from prerequisites through mothership setup, local registry, chart publishing, optional Istio service mesh, and optional adopted regional/child cluster deployment.
license: Apache-2.0
compatibility: opencode
metadata:
  audience: developers
  workflow: local-dev
---

## What I do

Guide developers through a complete local KCM + KOF deployment using kind clusters. I cover:

- Prerequisites and sibling repo layout (`../kcm` next to `../kof`)
- Optional squid proxy and Docker registry auth setup
- KCM bootstrap on the mothership kind cluster (`kcm-dev`)
- Optional Istio service mesh setup (`../istio`)
- Local OCI registry deployment and Helm chart packaging/pushing
- KOF mothership deployment via `dev-deploy`
- Optional adopted regional and child kind cluster setup (with or without Istio)
- CoreDNS patching for cross-cluster name resolution (non-Istio path)
- Monitoring and verification commands

## When to use me

Use this skill when a developer asks to:
- Set up a local dev environment for KOF or KCM
- Deploy KOF on kind clusters
- Deploy KOF with Istio service mesh
- Troubleshoot or re-run the local dev deployment workflow
- Deploy adopted regional/child clusters locally (with or without Istio)
- Understand what `make kcm-dev-apply`, `make dev-deploy`, `make dev-istio-regional-deploy-adopted`, etc. actually do

---

## Auto mode

When the user asks to deploy with **`auto`** (e.g. "deploy KOF auto", "run the full setup automatically"), execute all **non-optional** steps sequentially without asking for confirmation at each step. Optional steps are skipped unless the user explicitly requests them.

**Non-optional steps run in auto mode (in order):**

1. `make kcm-dev-apply` — Deploy KCM on the mothership kind cluster
2. `make helm-push` — Build and push KOF Helm charts
3. Add `/etc/hosts` entry for `dex.example.com`
4. Start `cloud-provider-kind` container
5. `make dev-deploy` — Deploy KOF mothership

**Steps skipped in auto mode (optional, run only on explicit request):**

- Step 1: `make cli-install` (run manually if tools are missing)
- Step 2: Docker pull-limit workaround
- Step 3: Squid proxy setup
- Step 5: Istio service mesh setup
- Step 10: Adopted regional cluster deployment
- Step 11: Adopted child cluster deployment

**Behavior in auto mode:**

- Run each non-optional step using the Bash tool immediately, without prompting.
- If a step fails, stop and report the error with the relevant logs/output.
- After all steps complete, print a summary of what was done and verification commands.
- If the user says `auto` together with optional steps (e.g. "auto with Istio", "auto with regional cluster"), include those optional steps in the sequence.
- If Istio is requested, run Step 5 after Step 4 and before Step 6. When deploying adopted clusters, use the Istio variants (`make dev-istio-regional-deploy-adopted`, `make dev-istio-child-deploy-adopted`).

---

## Prerequisites

1. **Go** installed (used by `make cli-install` to download `yq`, `kind`)
2. **Docker** running
3. **kubectl** on `$PATH`
4. **htpasswd** available (part of `apache2-utils` / `httpd-tools`)
5. Repos cloned as siblings:
   ```
   ~/work/
   ├── kcm/    # https://github.com/k0rdent/kcm
   ├── kof/    # this repo (or your fork)
   └── istio/  # https://github.com/k0rdent/istio (optional, only for Istio setup)
   ```
   The Makefile variable `KCM_REPO_PATH` defaults to `../kcm`.

---

## Step 1 — Optional: Install CLI tools

From the `kof/` repo root:

```bash
make cli-install
```

Installs into `./bin/`:
- `yq-v*`
- `helm-v*`
- `kind-v*`

Versions are defined in `Makefile`

---

## Step 2 — Optional: Docker pull-limit workaround

Copy your real Docker Hub credentials so the kind node can pull without hitting rate limits:

```bash
cp -r config/docker dev/
# Edit dev/docker/config.json with your real Docker Hub username and token/password
```

The `kind-deploy` Makefile target auto-detects `dev/docker/config.json` and mounts it into the kind node at `/var/lib/kubelet/config.json`.

---

## Step 3 — Optional: Squid proxy

Run a TLS-intercepting Squid proxy to simulate a proxied environment:

```bash
make squid-deploy
```

This generates a self-signed cert at `dev/squid-proxy.crt`, starts the `squid-proxy` Docker container on `127.0.0.1:3128`, and connects it to the `kind` Docker network. The `kind-deploy` target auto-detects the running container and mounts the cert into the kind node.

---

## Step 4 — Deploy KCM on the mothership kind cluster

```bash
make kcm-dev-apply
```

What this does:
1. Runs `make cli-install` (idempotent)
2. Creates `dev/kind-local.yaml` from `config/kind-local.yaml`, injecting any docker auth / squid cert / registry cert mounts
3. Creates kind cluster `kcm-dev` (skips if it already exists) with port mappings `32000` (Dex HTTPS NodePort)
4. Deploys `kubelet-csr-approver` via Helm (auto-approves kubelet serving cert CSRs)
5. Runs `make dev-apply` inside `../kcm` to bootstrap KCM
6. Waits for `mgmt/kcm` to exist (1 min) and become `Ready` (10 min)
7. Waits for `deployment/kcm-controller-manager` to be available (1 min) in `kcm-system`

**To verify:**

```bash
kubectl get pod -n kcm-system
kubectl get mgmt kcm
```

---

## Step 5 — Optional: Set up Istio service mesh

Skip this step if you do not need Istio. When Istio is used, some later steps differ — see Steps 10 and 11.

This step must be run **before** `make dev-deploy` (Step 9) because the `kof` namespace must be pre-created with the Istio injection label before KOF is installed. If Helm creates the namespace via `--create-namespace`, the injection label will not be set and sidecar injection will not work.

### 5a — Pre-create the KOF namespace with injection enabled

```bash
kubectl create namespace kof
kubectl label namespace kof istio-injection=enabled
```

### 5b — Clone the istio repo (if not already done)

```bash
cd ..
git clone https://github.com/k0rdent/istio.git
cd istio
```

### 5c — Build and push Istio charts to the local registry

```bash
make cli-install
make helm-push
```

This publishes Istio Helm charts to the shared local OCI registry at `oci://127.0.0.1:5001/charts` (the same registry used by KOF).

### 5d — Build the istio-operator Docker image

```bash
make istio-operator-docker-build
```

### 5e — Install the `k0rdent-istio` Helm chart

```bash
helm upgrade --create-namespace --install --wait k0rdent-istio ./charts/k0rdent-istio \
  -n istio-system \
  --set k0rdent-istio.repo.spec.url=oci://kcm-local-registry:5000/charts \
  --set k0rdent-istio.repo.spec.type=oci \
  --set k0rdent-istio.repo.spec.insecure=true \
  --set operator.image.registry=docker.io/library \
  --set operator.image.repository=istio-operator-controller \
  --set "istiod.meshConfig.extensionProviders[0].name=otel-tracing" \
  --set "istiod.meshConfig.extensionProviders[0].opentelemetry.port=4317" \
  --set "istiod.meshConfig.extensionProviders[0].opentelemetry.service=kof-collectors-daemon-collector.kof.svc.cluster.local" \
  --set-json 'gateway.resource.spec.servers[0]={"port":{"number":15443,"name":"tls","protocol":"TLS"},"tls":{"mode":"AUTO_PASSTHROUGH"},"hosts":["{clusterName}-vmauth.kof.svc.cluster.local"]}'
```

The OTel tracing extension provider points to the KOF collectors daemon so that Istio can export traces directly via the OpenTelemetry protocol.

**To verify:**

```bash
kubectl get pod -n istio-system
kubectl get helmrelease -n istio-system
```

---

## Step 6 — Build and push KOF Helm charts

From the `kof/` repo root:

```bash
make helm-push
```

Runs `helm dependency update` + `helm lint` + `helm package` for every chart under `charts/`, then pushes all `.tgz` files to `oci://127.0.0.1:5001/charts` (plain HTTP). Charts are pulled inside kind at `oci://kcm-local-registry:5000/charts`.

---

## Step 7 — Update local DNS for Dex

```bash
grep -qxF "127.0.0.1 dex.example.com" /etc/hosts || \
  echo "127.0.0.1 dex.example.com" | sudo tee -a /etc/hosts
```

---

## Step 8 — Run cloud-provider-kind (external IP for gateways)

```bash
docker start cloud-provider-kind 2>/dev/null || \
  docker run -d --name cloud-provider-kind --network kind \
    -v /var/run/docker.sock:/var/run/docker.sock \
    registry.k8s.io/cloud-provider-kind/cloud-controller-manager:v0.10.0 --gateway-channel="disabled"
```

This allocates external IPs for `LoadBalancer` services inside kind clusters. The command is idempotent: if the container already exists (running or stopped) it is restarted via `docker start`; otherwise a new container is created. To stop it: `docker stop cloud-provider-kind`.

---

## Step 9 — Deploy KOF mothership

```bash
make dev-deploy
```

What this does:
1. Builds `kof-operator-controller`, `kof-opentelemetry-collector-contrib`, and `kof-acl-server` Docker images; loads them into the `kcm-dev` kind cluster
2. Copies `charts/kof/values-local.yaml` → `dev/values-local.yaml`
3. Reads `git config user.email` for the Dex admin email (falls back to `admin@example.com`)
4. Generates a bcrypt hash for the default `admin` password and patches it into `dev/values-local.yaml`
5. Runs `scripts/generate-dex-secret.bash` to create the Dex TLS secret
6. Gets the kind control-plane container IP and runs `scripts/patch-coredns.bash` to add `dex.example.com → <host-ip>` to CoreDNS
7. Optionally reads `dev/dex.env` for `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET`
8. Sets `global.helmRepo.kofManaged.url = oci://kcm-local-registry:5000/charts` in values
9. Runs `helm upgrade -i --reset-values --wait -n kof --create-namespace kof ./charts/kof -f dev/values-local.yaml`
10. Waits for all HelmReleases in `kof` namespace to be `Ready` (10 min)
11. Restarts `kof-mothership-kof-operator` and `kof-mothership-dex` deployments

> **Note (Istio):** If you ran Step 5, the `kof` namespace already exists with the injection label. Helm's `--create-namespace` is a no-op in that case and the label is preserved.

**Optional env vars for `dev-deploy`:**

| Var | Effect |
|---|---|
| `HELM_CHART_NAME=kof-mothership` | Redeploy only that subchart |
| `DISABLE_KOF_COLLECTORS=true` | Skip kof-collectors |
| `DISABLE_KOF_STORAGE=true` | Skip kof-storage |
| `SKIP_WAIT=true` | Skip HelmRelease readiness wait |

**Monitor progress for Mothership cluster:**

```bash
kubectl wait --for=condition=Ready --all helmrelease -n kof --timeout=20m
kubectl get helmreleases -n kof
kubectl get pod -n kof
```

---

## Step 10 (Optional) — Deploy adopted regional cluster

```bash
make dev-adopted-deploy KIND_CLUSTER_NAME=regional-adopted
```

Creates a second kind cluster `regional-adopted` using `config/kind-adopted.yaml`, registers it as an adopted cluster in KCM by generating credentials from `demo/creds/adopted-credentials.yaml`, and loads the OTel collector image into it.

Then deploy the regional ClusterDeployment — choose the variant that matches your setup:

### Without Istio

```bash
make dev-regional-deploy-adopted
```

This:
1. Patches `demo/cluster/adopted-cluster-regional.yaml` → `dev/adopted-cluster-regional.yaml` with `kof-regional-domain=adopted-cluster-regional` and `kof-cert-email=<your git email>`
2. Applies the ClusterDeployment to KCM
3. Waits for `cert-manager` and `envoy-gateway` to deploy, then `kof-operators`, `kof-storage`, `kof-collectors` to upgrade on the regional cluster

### With Istio

```bash
make dev-istio-regional-deploy-adopted
```

Uses the Istio-aware ClusterDeployment template instead. The same wait sequence applies.

**Verify regional cluster:**

```bash
kubectl --context=kind-regional-adopted get pod -A
helm --kube-context=kind-regional-adopted list -A
```

---

## Step 11 (Optional) — Deploy adopted child cluster

Choose the variant that matches your setup:

### Without Istio

```bash
make dev-adopted-deploy KIND_CLUSTER_NAME=child-adopted
make dev-coredns
make dev-child-deploy-adopted
```

`dev-coredns` waits for the gateway in `kind-regional-adopted` to get an external IP, then patches CoreDNS in both the child and mothership clusters with all HTTPRoute hostnames → gateway IP, then restarts CoreDNS on both.

`dev-child-deploy-adopted` applies `demo/cluster/adopted-cluster-child.yaml` and waits for `cert-manager`, `kof-operators`, `kof-collectors` on the child.

### With Istio

```bash
make dev-adopted-deploy KIND_CLUSTER_NAME=child-adopted
make dev-istio-child-deploy-adopted
```

> **Note:** `make dev-coredns` is **not** required when using Istio — the Istio variant handles cross-cluster connectivity without the CoreDNS hostname patching step.

**Optional:** `KOF_TENANT_ID=<id>` adds a tenant label to the ClusterDeployment.

**Verify child cluster:**

```bash
kubectl --context=kind-child-adopted get pod -A
helm --kube-context=kind-child-adopted list -A
```

---

## Useful shortcuts

```bash
# Redeploy only kof-mothership subchart (skips operator image build for other subcharts)
make dev-deploy HELM_CHART_NAME=kof-mothership

# Port-forward promxy for metrics inspection
make dev-promxy-port-forward   # → localhost:8082

# Deploy adopted regional cluster with Istio
make dev-istio-regional-deploy-adopted

# Deploy adopted child cluster with Istio (no dev-coredns needed)
make dev-istio-child-deploy-adopted

# Remove an adopted cluster
make dev-adopted-rm KIND_CLUSTER_NAME=regional-adopted
make dev-adopted-rm KIND_CLUSTER_NAME=child-adopted

# Upgrade KCM controller (increases memory limit to 512Mi first)
make kcm-dev-upgrade
```

---

## Key file locations

| File | Purpose |
|---|---|
| `config/kind-local.yaml` | Mothership kind cluster template (ports 32000) |
| `config/kind-adopted.yaml` | Adopted cluster template (no port mappings) |
| `charts/kof/values-local.yaml` | Source values for `dev-deploy` |
| `dev/values-local.yaml` | Generated/patched working values (git-ignored) |
| `dev/docker/config.json` | Docker Hub credentials for kind node image pulls |
| `dev/dex.env` | Optional Google OIDC connector credentials |
| `demo/cluster/` | ClusterDeployment YAML templates (including Istio variants) |
| `demo/creds/` | Adopted cluster credential templates |
| `scripts/patch-coredns.bash` | CoreDNS hostname patching script (non-Istio path) |
| `scripts/generate-dex-secret.bash` | Dex TLS secret generation script |
| `../istio/charts/k0rdent-istio/` | Istio operator Helm chart (separate repo) |

---

## Post-deployment troubleshooting

After deploying, if the cluster is not healthy use the **troubleshoot** skill scripts to
produce a structured health report. The scripts require `pyyaml`:

```bash
pip3 install pyyaml --break-system-packages
```

### Collect a support bundle

```bash
make support-bundle
```

This produces one or more `support-bundle-<timestamp>.tar.gz` archives in the repo root.
Extract and analyse:

```bash
tar -xzf support-bundle-<timestamp>.tar.gz
```

### Run the full analysis

```bash
# Full 12-step analysis against the mothership bundle
python3 .agents/skills/troubleshoot/scripts/analyze_bundle.py support-bundle-<timestamp>

# If the regional cluster has a different kof namespace, pass it as second arg:
python3 .agents/skills/troubleshoot/scripts/analyze_bundle.py support-bundle-<timestamp> kof

# Analyse all bundles in the repo root at once:
for B in support-bundle-*/; do
  echo "=== $(basename $B) ==="
  python3 .agents/skills/troubleshoot/scripts/analyze_bundle.py "$B"
done
```

Exit code 0 = all healthy, 1 = failures found.

### Run individual steps

| Script | What it checks |
|---|---|
| `step1_management_release.py` | Management and Release readiness |
| `step2_templates.py` | ProviderTemplate / ClusterTemplate / ServiceTemplate validity |
| `step3_credentials.py` | Credential readiness and identity Secret existence |
| `step4_clusterdeployments.py` | ClusterDeployment conditions and per-service states |
| `step5_servicesets.py` | ServiceSet deployed / provider.ready conditions |
| `step6_multiclusterservices.py` | MultiClusterService Ready and dependency conditions |
| `step7_sveltos_clusters.py` | SveltosCluster connectionStatus / connectionFailures |
| `step8_profiles.py` | Profile matchingClusters |
| `step9_clustersummaries.py` | ClusterSummary featureSummaries / helmReleaseSummaries |
| `step10_helmreleases.py` | HelmRelease Ready conditions |
| `step11_promxyservergroups.py` | PromxyServerGroup presence, labels, and targets |
| `step12_workloads.py` | Pod / Deployment / StatefulSet health |

Each script takes `<bundle-dir>` as its first argument:

```bash
python3 .agents/skills/troubleshoot/scripts/step4_clusterdeployments.py support-bundle-<timestamp>
python3 .agents/skills/troubleshoot/scripts/step9_clustersummaries.py support-bundle-<timestamp>
python3 .agents/skills/troubleshoot/scripts/step12_workloads.py support-bundle-<timestamp>
```

### Common failure patterns found in CI

| Symptom (ClusterSummary failureMessage) | Root cause | Fix |
|---|---|---|
| `OpenTelemetryCollector … spec.volumeMounts must be of type array: null` | `opentelemetry-kube-stack` collector template emits `volumeMounts:` / `volumes:` as bare keys when no presets are enabled — serialises to `null` | Add `volumeMounts: []` and `volumes: []` to the collector's values, or guard the template with `{{- if … }}` |
| `GVK operator.victoriametrics.com/v1beta1, Kind=VMUser not found on remote cluster` | `vmuser-propagation` MCS applied before `victoria-metrics-operator` CRDs are registered on the remote cluster | Add `dependsOn` so vmuser-propagation waits for kof-storage to be `Deployed` |
| `http: server gave HTTP response to HTTPS client` on `kcm-local-registry:5000` | HelmRepository URL uses `https://` but local registry serves plain HTTP | Use `http://` scheme in the HelmRepository, or configure TLS on the registry |

---

## Common issues

**`make kcm-dev-apply` fails at `wait for mgmt/kcm`**
- Check `kubectl get pod -n kcm-system` for CrashLoopBackOff or ImagePullBackOff
- Ensure `../kcm` is cloned and up to date

**`make dev-deploy` fails building kof-operator image**
- Ensure Docker is running and you have sufficient disk space
- Run `cd kof-operator && make docker-build` manually to see errors

**HelmReleases stuck in `False` / `Progressing`**
- `kubectl describe helmrelease <name> -n kof` for details
- Check if `kcm-local-registry` container is running: `docker ps | grep kcm-local-registry`
- Re-run `make registry-deploy && make helm-push`

**Gateway has no external IP**
- Ensure `cloud-provider-kind` container is running
- `kubectl --context=kind-regional-adopted get gateway -n kof`

**Istio sidecar injection not working after `dev-deploy`**
- The `kof` namespace must have the injection label set **before** `dev-deploy` runs
- If the namespace was created by Helm (without the label), restart all pods in `kof` namespace to enable istio sidecars: `kubectl delete pod -n kof --all`

**`dev-adopted-deploy` fails with `base64: invalid option -- w`** (macOS)
- The Makefile uses `base64 -w 0` (Linux flag). On macOS, `base64` wraps by default but `-w` is not supported. Install GNU coreutils: `brew install coreutils` and ensure `gbase64` or `base64` from coreutils is on `$PATH`, or run on Linux.
