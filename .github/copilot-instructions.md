# Copilot Coding Agent Instructions for k0rdent/kof

## High-Level Overview

**k0rdent/kof** is a Kubernetes Observability and FinOps platform combining a Go operator, React/TypeScript UI, Helm charts, and scripts. Supports multi-cluster management, Istio service mesh, VictoriaMetrics, Promxy, Dex SSO, OpenTelemetry, and Grafana Operator.

- **Languages:** Go (operator), TypeScript/React (UI), Bash/Python (scripts), Helm (charts)
- **Documentation:** [Main docs](https://docs.k0rdent.io/next/admin/kof/), additional in `docs/`
- **CODEOWNERS:** @gmlexx @denis-ryzhkov @AndrejsPon00

---

## Build, Test, and Validation

### Quick Start

```bash
# Install CLI tools
make cli-install

# Full local dev setup
make registry-deploy
make helm-push
make dev-deploy

# Verify
kubectl get pod -n kof
```

### Required Tools
- Go >=1.24.0, Node >=18.18.2, Docker >=17.03, kubectl >=1.11.3, Helm >=v3.18.5, yq >=v4.44.2, kind >=v0.27.0

### Common Commands

**Operator:**
```bash
cd kof-operator
make build          # Build binary
make run            # Run locally
make docker-build   # Build image
make test           # Run tests
make lint           # Run linter
```

**Web UI:**
```bash
cd kof-operator/webapp/collector
npm install && npm run build
npm run lint        # Max warnings=0 enforced
npm test            # vitest + jsdom
```

**Helm:**
```bash
make helm-package                 # Package all charts
make lint-chart-<chartname>       # Lint specific chart
```

### CI/CD Requirements

All PRs must pass:
- ✓ Conventional commit validation (`feat`, `fix`, `docs`, `test`, `ci`, `refactor`, `perf`, `chore`, `revert`)
- ✓ Go tests (`make test`)
- ✓ React lint (`npm run lint -- --max-warnings=0`)
- ✓ React tests (`npm test`)
- ✓ npm audit (no moderate+ vulnerabilities)
- ✓ Helm docs generated and current

PRs affecting charts must also:
- ✓ Deploy successfully to kind cluster
- ✓ All pods reach Running state
- ✓ Test both `dev` and `dev-istio` scenarios

**PR Title Format:**
```
<type>(<scope>): <description>

Examples:
feat(operator): add multi-region support
fix(ui): resolve metrics dashboard race condition
docs(helm): update kof-mothership README
```

**Breaking Changes:** Use `!` after type or add `BREAKING CHANGE:` in body:
```
feat!: change cost calculation API
BREAKING CHANGE: values.yaml structure changed, see migration guide
```

---

## PR Review Focus Areas

### Code Quality Essentials

**Go:**
- Proper error wrapping: `fmt.Errorf("context: %w", err)`
- Set `OwnerReferences` for garbage collection
- Don't mutate cache objects; always `DeepCopy()`
- Use proper requeueing (exponential backoff, not immediate)
- Handle finalizers correctly
- Validate webhook inputs thoroughly

**React/TypeScript:**
- TypeScript strict mode, no `any` without justification
- Memoize expensive computations (`useMemo`, `useCallback`)
- Handle loading/error states
- Don't create functions/objects in render
- Clean up effects properly

**Kubernetes Manifests:**
- Include resource requests/limits
- Don't use `latest` tags
- Run as non-root, read-only filesystem
- Set proper probes (avoid overly aggressive liveness)
- Use principle of least privilege for RBAC

### Security Checklist

- No hardcoded credentials, API keys, tokens, or secrets
- No secrets in logs or error messages
- Proper RBAC (least privilege)
- npm audit passes
- Input validation for user data
- Sanitize user inputs in UI

### Testing Requirements

- Unit tests for new functions (>70% coverage target)
- React component tests for UI changes
- Tests pass locally and are deterministic
- Test edge cases and error conditions
- Mock external dependencies

### Documentation

- Update README for new features
- Helm chart `README.md.gotmpl` via helm-docs
- Inline comments for complex logic
- Update `docs/` for architectural/breaking changes
- Include examples for new patterns

### Breaking Changes

**Identifying:**
- CRD field removal/rename
- Default value changes affecting deployments
- Deprecated feature removal
- CLI/Helm value structure changes
- Metric name/label changes

**Protocol:**
1. Mark in commit: `feat!:` or `BREAKING CHANGE:` footer
2. Document migration steps in PR
3. Add upgrade notes to `docs/release.md`
4. Deprecate for 2+ minor versions before removal
5. Test upgrade path

**Note:** KOF version syncs with KCM version. Minor version incremented monthly. Major version has not changed even for major breaking changes.

---

## Helm Chart Reviews

**Version/Metadata:**
- Bump chart version (patch/minor/major)
- Update app version if component changed
- Update dependency versions

**Values:**
- Descriptive comments for new parameters
- Production-ready, secure defaults
- Resource limits/requests for all containers
- Image tags are specific versions, not `latest`
- Use `.Release.Namespace`, not hardcoded

**Templates:**
- Proper YAML indentation (2 spaces)
- No hardcoded namespaces
- Handle nil values (use `default` or `if`)
- Propagate `.global` values in sub-charts
- Move complex logic to `_helpers.tpl`

**Testing:**
- `make lint-chart-<chartname>` passes
- Deploys successfully to kind
- Test mothership/regional/child scenarios
- Test with/without optional features

---

## Architecture Awareness

### Multi-Cluster Hierarchy

- **Mothership:** Central management, aggregates all data
- **Regional:** Mid-tier, aggregates from child clusters in region
- **Child:** Workload clusters being monitored

**Data Flow:** Child → Regional → Mothership

Changes must consider all three tiers. Test with realistic multi-cluster scenarios.

### Observability

**Metrics:**
- VictoriaMetrics for storage, Promxy for aggregation
- Prometheus format at `/metrics`
- Minimize cardinality: avoid IDs, UUIDs, IPs as labels
- Use histograms for latencies
- Follow snake_case naming

**Traces/Logs:**
- OpenTelemetry for tracing (optional Istio integration)
- Structured logging with consistent fields
- Don't log sensitive data
- Correlate logs with traces

### FinOps

- Accurate resource metrics for cost calculations
- Track compute, storage, network costs separately
- Support multiple pricing models
- Changes to cost calculators need careful validation

---

## Common Pitfalls

**Go Operator:**
- ❌ Blocking operations in reconcile loops
- ❌ Mutating cache objects without DeepCopy
- ❌ Immediate requeue on error (use backoff)
- ❌ Logging entire objects (leaks secrets)
- ✓ Use typed clients, predicates, finalizers correctly

**Kubernetes:**
- ❌ Missing resource limits
- ❌ Overly aggressive liveness probes
- ❌ Running as root unnecessarily
- ❌ Granting cluster-admin
- ✓ Specify versions, security contexts, PDBs

**React:**
- ❌ Fetching data in render
- ❌ Mutating state directly
- ❌ Ignoring ESLint warnings
- ❌ Unstable dependencies in hooks
- ✓ Memoize expensive operations, clean up effects

**Istio:**
- ❌ Assuming Istio is always present
- ❌ Hardcoding Istio configs
- ✓ Test with/without Istio (`dev` and `dev-istio`)
- ✓ Configure mTLS, handle sidecar injection
- Mac/arm64: See `docs/workarounds.md` for CoreDNS issues

---

## Dependency Management

**Go:**
1. Run `go mod tidy`
2. Check for vulnerabilities and breaking changes
3. Commit both `go.mod` and `go.sum`

**npm:**
1. Run `npm audit` and fix moderate+ vulnerabilities
2. Document breaking changes
3. Commit both `package.json` and `package-lock.json`

**Helm:**
1. Update `Chart.yaml` dependencies
2. Run `helm dependency update` to update `Chart.lock`
3. Test in kind cluster
4. Commit both files

---

## Performance Considerations

**Resource Guidelines:**
- Operator: 100m/128Mi requests, 1000m/512Mi limits
- Collectors: scale with cluster size
- UI: 50m/64Mi requests, 500m/256Mi limits
- Adjust based on actual measurements

**Metrics:**
- Minimize cardinality (avoid pod names, IPs, user IDs as labels)
- Use aggregation for high-cardinality data
- Sample high-frequency metrics or aggregate before export
- Configure appropriate scrape intervals (15s-60s)

**Multi-Cluster Scale:**
- Plan for 100+ child clusters per regional
- Network: ~1-10 Mbps per child cluster
- Storage: ~1-10 GB per cluster per month
- Use metric relabeling, compression, backpressure

---

## Debugging CI Failures

**Reproduce Locally:**
```bash
# Go tests
cd kof-operator && make test

# React tests
cd kof-operator/webapp/collector && npm test

# Linting
make lint
npm run lint

# Full deployment
make cli-install registry-deploy helm-push dev-deploy
kubectl get pod -n kof -w
kubectl logs -n kof <pod-name>
```

**Support Bundle:** Download from failed workflow artifacts for detailed diagnostics.

---

## Agent Guidance

### When Reviewing PRs

1. Check all required CI validations pass
2. Verify security: no secrets, proper RBAC, npm audit clean
3. Check breaking changes protocol followed
4. Validate test coverage adequate
5. Ensure documentation updated
6. Consider multi-cluster impact
7. Check resource limits in manifests

### When Suggesting Code

1. Provide complete, working code (not pseudocode)
2. Follow existing patterns in codebase
3. Include error handling and tests
4. Update documentation
5. Consider backwards compatibility
6. Explain significant architectural choices

### When Running Commands

1. Use `timeout_ms` for long-running commands
2. Check prerequisites first (Go version, Docker, etc.)
3. Review output for errors/warnings
4. Follow debugging steps on failures

### Red Flags

- Secrets/credentials in code or logs
- Unbounded loops or recursion
- Missing error handling
- Hardcoded values that should be configurable
- Deprecated Kubernetes API versions
- Missing resource limits
- High-cardinality metrics without justification
- Breaking changes without proper marking
- Flaky or environment-dependent tests

---

**If you encounter failures or missing dependencies, check `docs/` and the Makefile for workarounds. When in doubt, ask for clarification rather than making assumptions.**
