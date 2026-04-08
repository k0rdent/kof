# Copilot Coding Agent Instructions for k0rdent/kof

### Common Commands

```bash
# Operator validation commands
cd kof-operator && make build|run|docker-build|test|lint

# Web UI validation commands
cd kof-operator/webapp/collector
npm install && npm run build
npm run lint   # max-warnings=0 enforced
npm test       # vitest + jsdom

# Helm validation command
make helm-package
```

### CI/CD Requirements

All PRs must pass:
- ✓ Conventional commits (`feat`, `fix`, `docs`, `test`, `ci`, `refactor`, `perf`, `chore`, `revert`)
- ✓ Go tests (`make test`), React lint + tests, `npm audit` (no moderate+)
- ✓ Helm docs generated and current
- ✓ PRs touching charts: deploy to kind, both `dev` and `dev-istio` scenarios

**PR Title Format:** `<type>(<scope>): <description>`

---

## PR Review: Context-Awareness Rules

> These rules are derived from patterns where reviewers pushed back on Copilot comments. Read context before flagging.

1. **`helm upgrade -i --reset-values` is used in this project.** Don't flag missing default values as "will break upgrades" — reset-values means the chart always starts fresh.
2. **Check `values.yaml` for existing docs before flagging missing documentation.** Reviewers have rejected "this is undocumented" comments when the docs were already present.
3. **Global Helm values applying to multiple resources is intentional by design.** Don't suggest splitting `global.helmRepo.*` or similar shared config blocks into per-resource blocks unless there is a concrete conflict.
4. **For HTTP POST endpoints, consuming all parameters from the request body (and not re-adding them to the URL) is intentional** — URLs have a ~4 KB limit; this is not a bug.
5. **CI `paths:` trigger restrictions are often intentional.** Don't flag a workflow that only triggers on `charts/**` as "missing coverage" — that may be a deliberate design to limit CI scope.
6. **Infrastructure-specific values (e.g., `gatewayClassName: cloud-provider-kind`) map to actually installed components** — don't flag them as "unlikely to resolve" without verifying the deployment environment.
7. **The mothership cluster has no collectors** (`DISABLE_KOF_COLLECTORS=true`). Don't assume a multi-cluster step should run against mothership context; check which clusters actually have the target resource.
8. **Helm `mergeOverwrite` order is intentional.** When you see `mergeOverwrite $a $b`, the author wants `$b` to win. Don't suggest swapping order unless you can prove correctness requires it.
9. **Avoid making the same comment multiple times in one PR.** If duplicate files contain the same pattern, note it once and reference the other locations.
10. **Prefer concise over verbose.** The team rejects suggestions that replace a working one-liner with a multi-line refactor for style reasons alone.

---

## PR Review Focus Areas

### Go

**Real issues to catch**

- `sync.WaitGroup` has no `.Go()` method is a ligit error only if go version below 1.25
- Don't mutate cache objects — always `DeepCopy()`
- Set `OwnerReferences` for garbage collection
- Use proper requeueing (exponential backoff, not immediate requeue on error). This is [enabled by default](https://github.com/kubernetes-sigs/controller-runtime/blob/6210f847b2c1df3f28e5be34a4b1458f03896c73/pkg/controller/controller.go#L252-L254).
- Use typed clients, predicates, finalizers correctly
- Go naming conventions: `tenantID` not `tenantId`, `userID` not `userId`
- `nil` pointer dereference risk on optional config fields — guard before dereference

**Consistency with existing codebase:**

- Use `res.Logger` and `res.Fail(...)` for error handling in HTTP handlers — not `logrus` + `http.Error`
- Avoid naming packages `utils` — use domain-specific names (`labels`, `k8s`, `handlers`, etc.)

### React / TypeScript

**General:**
- No `any` without justification (TypeScript strict mode)
- Memoize expensive computations (`useMemo`, `useCallback`)
- Handle loading and error states
- Clean up effects (return cleanup from `useEffect`)

### Kubernetes Manifests

- Include resource requests/limits; don't use `latest` tags
- Run as non-root, read-only filesystem; least-privilege RBAC
- `EnvVar.value` must be a **string** — `value: true` (boolean) will fail schema validation
- Placeholder substitutions in YAML templates (e.g., `{clusterName}`) must be quoted/escaped to stay YAML-safe
- Use `.Release.Namespace`, not hardcoded namespace values

### Helm Charts

**Version/Metadata:**
- Keep `Chart.lock` in sync (`helm dependency update`) after changing `Chart.yaml` dependencies

**Templates:**
- Dynamic key access for names with hyphens: use `index .Values "kof-mothership"` or `["kof-mothership"]` — dot notation breaks on hyphens in `yq` and Helm
- Handle nil/missing values: `dig "annotations" "key" "default"` instead of `index .Cluster.metadata.annotations "key"` — the map may not exist
- `now | unixEpoch` in templates makes renders **non-deterministic** — every upgrade triggers a diff; avoid unless intentional
- Add an `else` branch with `fail` for `if/else if` provider selectors — an unsupported value should error, not silently render a broken resource
- Sveltos template expressions mixed with Helm `{{ }}` should be explicit about which layer evaluates each expression by using the unified opening/closing tags:
  ```
  key: Helm {{`still Helm`}} Helm {{`{{`}} Sveltos {{`}}`}} Helm
  ```

### CI / GitHub Actions

**Real issues to catch:**

- OCI chart references should use `${repo_lower}` (derived from `github.repository`) for fork compatibility, not hardcoded `k0rdent/kof`
- Validate API responses before using them: check SHA is non-empty and not `"null"` before writing to `$GITHUB_OUTPUT`
- Different repos (`k0rdent/istio` vs `k0rdent/kcm`) may need separate release env vars — don't assume one ref applies to all
- Align `actions/setup-python` version with other workflows in the repo; enable pip caching for speed

### Shell / Makefile

**Real issues to catch:**

- **Each recipe line in a Makefile runs in a separate shell** — multi-step logic (start background process, capture PID, trap) must use line continuation `; \` or be grouped in `{ ...; }` or a script file
- Always use `$(KUBECTL)` and `$(HELM)` tool variables — never raw `kubectl` or `helm` in Makefile targets
- Add `KUBECTL_CONTEXT` parameter support to targets that operate on specific clusters, consistent with other targets in the Makefile (e.g., `support-bundle`)
- Guard `kubectl patch ... --type json -p '[{"op":"replace"...}]'` with an existence check — `op: replace` fails when the field is absent

### Python Scripts

**Real issues to catch:**

- Use `except ImportError` for optional-import guards — `except Exception` silently hides real runtime errors
- Use `tempfile.TemporaryDirectory()` context manager — `mkdtemp()` leaks temp dirs across invocations
- Validate `tarfile` member paths before `extractall()` — untrusted tarballs can write outside the target directory (path traversal)
- Fail loudly: use `assert` or `sys.exit(1)` with a clear message on unexpected input — don't silently skip
- `for x in generator` over multiple patterns is O(n) per pattern; consider combining into a single traversal

---

## Architecture Awareness

### Multi-Cluster Hierarchy

- **Mothership:** Central management — **collectors and storage are disabled here**
- **Regional:** Mid-tier, aggregates from child clusters
- **Child:** Workload clusters being monitored

**Data Flow:** Child → Regional → Mothership

**Important:** The mothership runs with `DISABLE_KOF_COLLECTORS=true` and `DISABLE_KOF_STORAGE=true`. CI steps that wait for `OpenTelemetryCollector` resources must target regional or child contexts, not mothership.

### Helm Charts Hierarchy

- `charts/kof` is an umbrella Helm chart that deploys FluxCD Helm releases for other Helm charts, so its `values.yaml` contains value sections for those charts
- `charts/kof-child` and `charts/kof-regional` deploy a MultiClusterService template that uses k0rdent automation to deploy services via Helm to Child and Regional KOF clusters, with service values defined as a Helm template that renders another Helm template.

### Observability Stack

- VictoriaMetrics for storage, Promxy for aggregation, Prometheus format at `/metrics`
- OpenTelemetry for tracing
- Grafana Operator manages dashboards and datasources as Kubernetes resources
- Minimize metric cardinality: avoid pod names, IPs, UUIDs as labels
- Optional Istio integration

### FinOps

- OpenCost provides resource cost data; customPricing stub prevents parse errors on LoadBalancer fields
- Accurate resource metrics are critical for cost calculations

### Istio

- Istio is **optional** — always test with and without (`dev` vs `dev-istio`)
- Istio-specific functionality (remote secrets, mesh topology) belongs in `k0rdent/istio`, not `kof`

---

## Security Checklist

- No hardcoded credentials, API keys, tokens, or secrets anywhere in code or logs
- Proper RBAC (least privilege); avoid `cluster-admin`
- `npm audit` must pass (no moderate+ vulnerabilities); avoid pre-release npm packages
- Sanitize user inputs; validate webhook inputs thoroughly
- Don't use `tlsInsecureSkipVerify: true` in production paths

*When in doubt, check `docs/` and the Makefile for context. Prefer asking for clarification over making incorrect assumptions about intentional design choices.*
