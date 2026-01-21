# Copilot Coding Agent Instructions for k0rdent/kof

## High-Level Overview

**k0rdent/kof** is a comprehensive platform for Kubernetes Observability and FinOps, designed to automate deployment, monitoring, and cost management across clusters. It combines a Go-based Kubernetes operator, a React/TypeScript web UI, Helm charts, and supporting scripts. The repository is structured for both cloud and local (kind) development, supporting advanced scenarios such as multi-cluster management, Istio service mesh, and integration with external tools (VictoriaMetrics, Promxy, Dex SSO, OpenTelemetry, Grafana Operator).

- **Languages/Frameworks:** Go (operator), TypeScript/React (web UI), Bash/Python (scripts), Helm (charts), YAML (config).
- **Repository Size:** Large, with multiple subdirectories for charts, operators, webapp, scripts, and documentation.
- **Documentation:** [Main docs](https://docs.k0rdent.io/next/admin/kof/), additional docs in `docs/`.

---

## Build, Test, and Validation Instructions

### Environment Setup

- **Required tools:**
  - Go >=1.24.0
  - Node >=18.18.2
  - Docker >=17.03
  - kubectl >=1.11.3
  - Helm >=v3.18.5
  - yq >=v4.44.2
  - kind >=v0.27.0
- **Install all CLI dependencies:**
  ```
  make cli-install
  ```
  This installs `yq`, `helm`, `kind`, and required Helm plugins locally in `bin/`.

### Bootstrap & Build

- **Operator (Go):**
  - Build:
    ```
    cd kof-operator
    make build
    ```
  - Run locally:
    ```
    make run
    ```
  - Build Docker image:
    ```
    make docker-build
    ```
- **Web UI (React/TypeScript):**
  - Build:
    ```
    cd kof-operator/webapp/collector
    npm install
    npm run build
    ```
  - Lint:
    ```
    npm run lint
    ```
  - Test:
    ```
    npm run test
    ```
    (uses `vitest` with `jsdom` environment, see `vite.config.ts` and `tests/setup.js`)

### Helm Charts

- **Package all charts:**
  ```
  make helm-package
  ```
- **Push charts to local registry:**
  ```
  make registry-deploy
  make helm-push
  ```
- **Deploy to cluster (kind/local):**
  - Deploy all required CRDs/operators:
    ```
    make dev-operators-deploy
    ```
  - Deploy storage, collectors, and mothership:
    ```
    make dev-storage-deploy
    make dev-collectors-deploy
    make dev-ms-deploy
    ```

### End-to-End Dev Cluster

- **Full local dev setup:**
  ```
  make cli-install
  make registry-deploy
  make helm-push
  make dev-operators-deploy
  make dev-storage-deploy
  make dev-collectors-deploy
  make dev-ms-deploy
  ```
  Wait for all pods to be `Running`:
  ```
  kubectl get pod -n kof
  ```

### Linting and Validation

- **Go lint:**
  ```
  make lint
  ```
  (uses `golangci-lint` with config in `kof-operator/.golangci.yml`)
- **Helm chart lint:**
  ```
  make lint-chart-<chartname>
  ```
- **Pre-commit (optional):**
  ```
  pre-commit install --install-hooks
  git commit
  ```
  If files are modified by hooks, review, `git add`, and commit again.

### CI/CD and PR Validation

- **GitHub Actions:**
  - PRs are validated for [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/).
  - PRs run unit tests (`make test`), React linter (`npm run lint`), and Helm chart tests.
  - Helm docs are auto-generated and validated.
  - Images and charts are built and pushed on `main` branch.
  - See `.github/workflows/` for all pipelines.

### Common Issues & Workarounds

- **RBAC errors:** Grant yourself cluster-admin or use an admin context.
- **Istio on Mac/arm64:** If Istio sidecars fail to create temp files, patch CoreDNS as described in `docs/workarounds.md`.
- **Helm chart deployment timing:** Use `scripts/wait-helm-charts.bash` to wait for deployments.
- **Grafana Operator upgrades:** See upgrade instructions in `docs/dev.md` and `docs/release.md`.

---

## Project Layout & Key Files

- **Root:**
  - `Makefile` (main entry for all build/test/deploy tasks)
  - `.goreleaser.yml` (GoReleaser config for operator images)
  - `.pre-commit-config.yaml` (optional pre-commit hooks)
  - `.prettierrc` (formatting for JS/TS)
  - `CODEOWNERS` (maintainers: @gmlexx @denis-ryzhkov @AndrejsPon00)
- **kof-operator/**
  - Go operator code, Makefile, `cmd/main.go` (entrypoint), `api/`, `internal/`, `webapp/collector` (React UI)
- **charts/**
  - Helm charts for all components (`kof-mothership`, `kof-operators`, `kof-collectors`, etc.)
- **scripts/**
  - Bash/Python scripts for cluster setup, CoreDNS patching, Dex secret generation, etc.
- **docs/**
  - System requirements, dev instructions, workarounds, release notes, etc.
- **demo/**
  - Example cluster YAMLs for AWS, Azure, kind, etc.

---

## PR Review Guidelines

### What to Review For

#### Code Quality

**Go Code:**
- Adherence to Go best practices and idiomatic patterns
- Proper error handling with context (use `fmt.Errorf` with `%w` for error wrapping)
- Resource cleanup with `defer` statements
- Kubernetes controller best practices (reconciliation loops, finalizers, owner references)
- No infinite loops or blocking operations in reconcile functions
- Proper use of controller-runtime client (cached reads, direct writes to API server)
- Avoid mutating cached objects; always `DeepCopy()` before modification
- Context propagation through function calls
- Consistent logging with structured fields

**TypeScript/React Code:**
- TypeScript strict mode compliance
- Proper React hooks usage (`useState`, `useEffect`, `useMemo`, `useCallback`)
- Component composition and reusability
- Type safety (no `any` types unless absolutely necessary)
- Proper error boundaries and error handling
- Loading and error states for all data fetching
- Accessibility considerations (ARIA labels, keyboard navigation)
- Performance optimizations (memoization for expensive operations)

**Bash/Python Scripts:**
- Proper error handling (`set -euo pipefail` for bash)
- Input validation and sanitization
- Idempotent operations where possible
- Clear usage instructions and comments

#### Testing

**Required Test Coverage:**
- Unit tests for new Go functions/methods (target >70% coverage for new code)
- React component tests using vitest for UI changes
- Test for edge cases, error conditions, and boundary values
- Integration test considerations for multi-cluster scenarios
- Mock external dependencies appropriately

**Testing Checklist:**
- Tests pass locally with `make test` (Go) and `npm test` (React)
- Tests are deterministic (no flaky tests)
- Test names clearly describe what is being tested
- Tests follow Arrange-Act-Assert pattern
- Complex test logic is well-commented

#### Security

**Critical Security Checks:**
- No hardcoded credentials, API keys, tokens, or sensitive data
- No secrets in logs or error messages
- Proper RBAC configurations in Helm charts (principle of least privilege)
- Secure handling of cluster credentials in multi-cluster scenarios
- npm audit must pass with no moderate+ vulnerabilities
- ServiceAccounts, Roles, RoleBindings, and ClusterRoles reviewed carefully
- TLS/mTLS configurations are secure
- Input validation for user-provided data
- No SQL injection, command injection, or path traversal vulnerabilities

**Security Best Practices:**
- Use Kubernetes secrets for sensitive data
- Rotate credentials regularly (document in operational guides)
- Follow principle of least privilege for all RBAC
- Sanitize user inputs in web UI
- Use secure defaults in Helm values

#### Documentation

**Required Documentation:**
- README updates for new features or significant changes
- Helm chart `README.md.gotmpl` updates (auto-generated via helm-docs)
- Inline code comments for complex logic, algorithms, or workarounds
- Update `docs/` for architectural, operational, or breaking changes
- API documentation for new endpoints or CRD fields
- Examples in `demo/` for new deployment scenarios

**Documentation Quality:**
- Clear, concise, and accurate
- Includes examples where appropriate
- Explains "why" not just "what"
- Updated for breaking changes with migration guides
- Links to external documentation where relevant

#### Breaking Changes

**Identifying Breaking Changes:**
- CRD schema changes that remove or rename fields
- API version bumps (e.g., v1alpha1 → v1beta1)
- Changes to default values that affect existing deployments
- Removal of deprecated features
- Changes to CLI flags or command structure
- Helm chart value structure changes
- Changes to exposed metrics or their labels

**Breaking Change Protocol:**
1. Mark in commit message: `feat!:` or `BREAKING CHANGE:` footer
2. Document in PR description with clear migration steps
3. Update version following semver (major bump for breaking changes)
4. Add upgrade notes to `docs/release.md`
5. Consider providing migration scripts or automated upgrade paths
6. Deprecate for at least 2 minor versions before removal (when possible)
7. Add deprecation warnings in logs and documentation

#### Backwards Compatibility

- Maintain compatibility with existing deployments
- Consider impact on multi-cluster synchronization
- Test upgrade path from previous version
- Document any required manual steps for upgrade
- Support graceful degradation when possible

---

## Helm Chart Specific Guidelines

### Required Checks for Chart Changes

**Version and Metadata:**
- Chart version bumped according to semver
  - Patch: bug fixes, no new features
  - Minor: new features, backward compatible
  - Major: breaking changes
- App version updated if operator/component version changed
- Chart dependencies versions updated in `Chart.yaml`

**Values and Configuration:**
- `values.yaml` has descriptive comments for all new parameters
- Default values are production-ready and secure
- Sensitive defaults (passwords, tokens) are empty or clearly marked
- Boolean flags have clear true/false descriptions
- Resource limits and requests defined for all containers
- Namespace configurations use `.Release.Namespace`
- Image tags are specific versions, not `latest`

**Documentation:**
- `README.md` updated via helm-docs (run `make lint-chart-<chartname>`)
- Comments in `values.yaml` are clear and comprehensive
- Examples provided for complex configurations
- Upgrade notes for breaking value changes

**Templates:**
- Proper YAML indentation (2 spaces)
- Whitespace control with `{{- ... -}}` where appropriate
- No hardcoded namespaces (use `.Release.Namespace`)
- Conditional blocks for optional features use consistent patterns
- `.global` values propagated correctly in sub-charts
- Resource names include release name to avoid conflicts
- Labels follow Kubernetes recommended labels

**Testing:**
- Linting passes: `make lint-chart-<chartname>`
- Successfully deploys to kind cluster
- Test installation scenarios:
  - Mothership cluster deployment
  - Regional cluster deployment
  - Child cluster deployment
- Test with and without optional features enabled
- Test upgrades from previous chart version

### Common Helm Issues to Avoid

- Missing `.global` value propagation in sub-charts
- Incorrect template indentation causing YAML parse errors
- Hardcoded namespaces breaking multi-tenant deployments
- Missing conditional blocks causing errors when features disabled
- Resource names without release prefix causing conflicts
- Using `Capabilities.APIVersions` without proper fallbacks
- Not handling nil values in templates (use `default` or `if`)
- Complex logic in templates (move to helper templates in `_helpers.tpl`)

---

## Architecture Patterns

### Multi-Cluster Awareness

**Cluster Hierarchy:**
- **Mothership cluster:** Central management plane, runs mothership components, aggregates all data
- **Regional clusters:** Mid-tier clusters that aggregate data from child clusters in a region
- **Child clusters:** Workload clusters being monitored, run collectors

**Data Flow:**
- Metrics: Child collectors → Regional Promxy → Mothership VictoriaMetrics
- Traces: Child OTEL collectors → Regional OTEL collectors → Mothership
- Logs: Child collectors → Regional aggregators → Mothership

**Design Considerations:**
- Changes must consider data flow through all three tiers
- Test with all three cluster types when modifying collectors or aggregation
- Consider network latency and bandwidth in cross-cluster communication
- Handle cluster connectivity issues gracefully (buffering, retries, backoff)
- Support both push and pull models where appropriate
- Consider eventual consistency in multi-cluster state

### Observability Patterns

**Metrics:**
- Use VictoriaMetrics for efficient long-term storage
- Promxy for multi-cluster metric aggregation
- All components expose Prometheus metrics at `/metrics`
- Follow Prometheus naming conventions (snake_case, descriptive)
- Minimize cardinality (avoid high-cardinality labels like IDs, IPs)
- Use histogram for latencies, not gauge or counter
- Include help text for all metrics

**Traces:**
- OpenTelemetry for distributed tracing
- Optional Istio integration for automatic trace generation
- Configure sampling rates appropriate to scale
- Propagate trace context across service boundaries
- Include relevant attributes (cluster, namespace, pod)

**Logs:**
- Structured logging with consistent fields
- Use appropriate log levels (debug, info, warn, error)
- Don't log sensitive information
- Include context (cluster, component, version)
- Correlate with traces using trace ID

**Dashboards:**
- Grafana dashboards via Grafana Operator
- One dashboard per component/feature
- Include RED metrics (Rate, Errors, Duration) for services
- Include resource utilization (CPU, memory, disk)
- Use template variables for cluster/namespace filtering
- Document dashboard usage in chart README

### FinOps Considerations

**Cost Tracking:**
- Resource metrics collection must be accurate for cost calculations
- Track compute, storage, and network costs separately
- Support multiple pricing models (on-demand, reserved, spot)
- Aggregate costs across cluster hierarchies
- Maintain cost history for trend analysis

**Cost Optimization:**
- Identify overprovisioned resources
- Track resource utilization vs. requests
- Recommend rightsizing based on actual usage
- Identify idle resources for cleanup
- Support cost allocation by team/project/environment

**Financial Domain Review:**
- Changes to cost calculators require careful review
- Validate cost calculation accuracy with test data
- Document cost calculation algorithms
- Consider regional pricing differences
- Handle currency conversions correctly

---

## Common Pitfalls & Best Practices

### Go Operator Code

**Controller Best Practices:**
- **DO** set `OwnerReferences` for all dependent resources (enables garbage collection)
- **DO** use controller-runtime client properly (cached reads, direct writes)
- **DO** use proper requeueing strategies (exponential backoff, rate limiting)
- **DO** handle finalizers correctly (cleanup, removal)
- **DO** use predicates to filter watch events (reduce reconciliation load)
- **DO** validate webhook inputs thoroughly
- **DO** use typed clients for better compile-time safety

**Common Mistakes:**
- **DON'T** perform blocking operations in reconciliation loops (use goroutines with proper lifecycle)
- **DON'T** mutate cache objects; always `DeepCopy()` before modification
- **DON'T** requeue immediately on error (causes tight loop, use backoff)
- **DON'T** ignore context cancellation (respect timeouts and shutdowns)
- **DON'T** fetch resources in a loop (use List with selectors)
- **DON'T** log entire objects (leaks sensitive data, causes log spam)
- **DON'T** forget to handle race conditions (use optimistic locking, retry on conflict)

**Error Handling:**
```go
// Good: wrap errors with context
if err := r.Client.Get(ctx, key, obj); err != nil {
    return ctrl.Result{}, fmt.Errorf("failed to get resource %s: %w", key, err)
}

// Bad: lose error context
if err := r.Client.Get(ctx, key, obj); err != nil {
    return ctrl.Result{}, err
}
```

### Kubernetes Manifests

**Resource Management:**
- **DO** include resource requests/limits for all containers
- **DO** set appropriate probe configurations (liveness, readiness, startup)
- **DO** use PodDisruptionBudgets for high-availability components
- **DO** set `terminationGracePeriodSeconds` appropriately
- **DO** use anti-affinity rules for critical components
- **DO** specify security contexts (run as non-root, read-only filesystem)

**Common Issues:**
- **DON'T** use `latest` tags; always specify versions
- **DON'T** omit resource limits (causes noisy neighbor problems)
- **DON'T** set overly aggressive liveness probes (causes restart loops)
- **DON'T** run as root unless absolutely necessary
- **DON'T** grant cluster-admin unnecessarily (principle of least privilege)
- **DON'T** use deprecated API versions (check with `kubectl api-resources`)

### React/TypeScript UI

**React Best Practices:**
- **DO** use TypeScript strict mode features
- **DO** memoize expensive computations with `useMemo`/`useCallback`
- **DO** handle loading and error states in all components
- **DO** maintain max-warnings=0 for ESLint (CI enforced)
- **DO** use React hooks correctly (follow rules of hooks)
- **DO** split large components into smaller, reusable ones
- **DO** use proper key props in lists

**Common Mistakes:**
- **DON'T** fetch data in render; use proper data fetching patterns (useEffect, React Query)
- **DON'T** create functions/objects in render (causes unnecessary re-renders)
- **DON'T** mutate state directly (use immutable updates)
- **DON'T** ignore ESLint warnings (fix or suppress with justification)
- **DON'T** use inline styles extensively (use CSS modules or styled-components)
- **DON'T** forget to cleanup effects (return cleanup function from useEffect)
- **DON'T** pass unstable references to dependencies (causes infinite loops)

**Performance:**
```typescript
// Good: memoize expensive calculations
const expensiveValue = useMemo(() => {
  return computeExpensiveValue(data);
}, [data]);

// Bad: recalculate on every render
const expensiveValue = computeExpensiveValue(data);
```

### Istio Integration

**Testing Requirements:**
- **DO** test both with and without Istio (`dev` and `dev-istio` make targets)
- **DO** configure mTLS settings appropriately (PERMISSIVE or STRICT)
- **DO** set up OpenTelemetry tracing configuration
- **DO** handle service mesh sidecar injection (enable via namespace labels)

**Common Issues:**
- **DON'T** assume Istio is always present; make it optional
- **DON'T** hardcode Istio-specific configurations
- **DON'T** forget to configure network policies for sidecar communication
- On Mac/arm64: Review CoreDNS workarounds in `docs/workarounds.md` for temp file issues

**Istio Best Practices:**
- Use virtual services for traffic routing
- Configure retry and timeout policies
- Set up circuit breakers for resilience
- Monitor Istio metrics (envoy_*) in addition to application metrics
- Test graceful degradation when mesh is unavailable

---

## Dependency Management

### Go Dependencies

**Adding/Updating Dependencies:**
1. Add import in code
2. Run `go mod tidy` to update `go.mod` and `go.sum`
3. Review license compatibility
4. Check for known vulnerabilities
5. Test thoroughly
6. Commit both `go.mod` and `go.sum`

**Best Practices:**
- Justify major version upgrades in PR description
- Check for breaking changes in dependency release notes
- Run vulnerability scanning: `go list -json -m all | nancy sleuth` (if available)
- Prefer well-maintained dependencies with active communities
- Avoid dependencies with excessive transitive dependencies
- Keep dependencies up-to-date to receive security patches

**Vendor Considerations:**
- Don't commit `vendor/` directory unless explicitly required
- Use `go mod download` in CI for reproducible builds
- Pin versions for stability, update regularly for security

### npm Dependencies

**Adding/Updating Dependencies:**
1. Add to `package.json` with `npm install <package>`
2. Run `npm audit` and address moderate+ vulnerabilities
3. Document breaking changes from dependency updates
4. Lock exact versions for production dependencies
5. Test thoroughly after updates
6. Commit both `package.json` and `package-lock.json`

**Security:**
- Run `npm audit` before every commit
- Fix high/critical vulnerabilities immediately
- Evaluate moderate vulnerabilities case-by-case
- Use `npm audit fix` for automated fixes
- Review audit report for false positives

**Best Practices:**
- Keep devDependencies and dependencies separated
- Don't install unnecessary packages
- Review bundle size impact for frontend dependencies
- Check for TypeScript type definitions availability
- Prefer packages with active maintenance

### Helm Dependencies

**Chart Dependencies:**
1. Add to `Chart.yaml` under `dependencies`
2. Specify version constraints (prefer specific versions)
3. Run `helm dependency update` to update `Chart.lock`
4. Test with updated dependencies in kind cluster
5. Document dependency changes in PR

**Version Management:**
- Use version ranges carefully (prefer exact versions for stability)
- Test compatibility with dependency updates
- Review dependency chart values for breaking changes
- Update both `Chart.yaml` and commit `Chart.lock`

---

## PR-Specific CI Checks

### All PRs Must Pass

**Mandatory Checks:**
- ✓ Conventional commit validation (feat, fix, docs, test, ci, refactor, perf, chore, revert)
- ✓ Go unit tests (`make test`)
- ✓ React linter (`npm run lint -- --max-warnings=0`)
- ✓ React tests (`npm test`)
- ✓ npm security audit (no moderate+ vulnerabilities)
- ✓ Helm docs generated and up-to-date

**PR Title Format:**
```
<type>(<scope>): <description>

Examples:
feat(operator): add support for multi-region deployments
fix(ui): resolve race condition in metrics dashboard
docs(helm): update kof-mothership chart README
test(collector): add integration tests for prometheus scraping
```

**Commit Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Adding or updating tests
- `ci`: CI/CD configuration changes
- `refactor`: Code refactoring without behavior change
- `perf`: Performance improvements
- `chore`: Maintenance tasks, dependency updates
- `revert`: Revert previous commit

**Breaking Changes:**
Use `!` after type or add `BREAKING CHANGE:` in commit body:
```
feat!: change cost calculation API
fix(helm)!: restructure values.yaml format

BREAKING CHANGE: values.yaml structure changed, see migration guide
```

### PRs Affecting Charts Must Also Pass

**Helm Chart CI:**
- ✓ Helm chart linting for all modified charts
- ✓ Successfully deploy to kind cluster
- ✓ All pods reach Running state
- ✓ Validate metrics collection after 10-minute wait
- ✓ Test both `dev` and `dev-istio` scenarios (if relevant)
- ✓ Support-bundle generated on failure (uploaded as artifact)

**Chart Modification Triggers:**
- Changes to `charts/**`
- Changes to `kof-operator/**` (affects operator image)
- NOT triggered by `**.md` changes

### How to Debug CI Failures

**General Approach:**
1. Check workflow logs in GitHub Actions tab
2. Identify the failing step
3. Reproduce locally with the same make target
4. Fix the issue
5. Push changes and wait for CI to re-run

**Specific Failure Types:**

**Test Failures:**
```bash
# Reproduce Go tests locally
cd kof-operator
make test

# Reproduce React tests locally
cd kof-operator/webapp/collector
npm install
npm test
```

**Lint Failures:**
```bash
# Go linting
cd kof-operator
make lint

# React linting
cd kof-operator/webapp/collector
npm run lint
```

**Chart Deployment Failures:**
```bash
# Full local deployment
make cli-install
make registry-deploy
make helm-push
make dev-operators-deploy
make dev-ms-deploy

# Check pod status
kubectl get pod -n kof -w

# Check logs
kubectl logs -n kof <pod-name>
```

**Download Support Bundle:**
- Navigate to failed workflow run
- Download support-bundle artifact from workflow artifacts
- Extract and review logs, resource states, and diagnostics

**Istio-Specific Failures:**
- Check `docs/workarounds.md` for known issues
- On Mac/arm64, CoreDNS patching may be required
- Test without Istio first to isolate issue

---

## Version Compatibility

### Supported Versions

**Runtime Requirements:**
- Kubernetes: 1.28+ (test against 1.28, 1.29, 1.30)
- Helm: 3.18.5+
- Istio: 1.20+ (optional, when enabled)

**Development Requirements:**
- Go: 1.24.0+ (must match `go.mod` go directive)
- Node: 18.18.2+ (check for .nvmrc if present)
- Docker: 17.03+
- kubectl: 1.11.3+
- yq: v4.44.2+
- kind: v0.27.0+ (for local development)

### Breaking Change Protocol

**Identifying Breaking Changes:**
- API schema changes (CRD field removal/rename)
- Default value changes affecting existing deployments
- Deprecated feature removal
- CLI interface changes
- Helm chart value structure changes
- Metric name or label changes
- Minimum version requirement bumps

**Breaking Change Checklist:**
1. ✓ Marked in commit with `!` or `BREAKING CHANGE:` footer
2. ✓ Documented in PR description with migration steps
3. ✓ Version updated following semver (major bump)
4. ✓ Upgrade notes added to `docs/release.md`
5. ✓ Migration scripts provided (if applicable)
6. ✓ Tested upgrade path from previous version
7. ✓ Announcement prepared for release notes

### API Deprecation Policy

**Deprecation Process:**
1. Deprecate for at least 2 minor versions before removal
2. Add deprecation warnings in:
   - Application logs (at WARN level)
   - CLI output (when deprecated flag used)
   - Documentation (clear "DEPRECATED" markers)
   - API responses (custom headers or response fields)
3. Provide migration path in documentation
4. Add removal date/version to deprecation notice
5. Keep deprecated features functional during deprecation period
6. Remove in major version bump with clear release notes

**Deprecation Notice Example:**
```
DEPRECATED: The field `spec.oldField` is deprecated and will be removed in v2.0.0.
Please use `spec.newField` instead. See migration guide: https://docs.k0rdent.io/migration/v2
```

---

## Performance Considerations

### Resource Usage

**Operator Performance:**
- Profile memory usage for operator changes (use pprof)
- Consider watch cache size for large clusters (>1000 nodes)
- Set appropriate resource limits based on load testing
- Monitor goroutine leaks (use runtime metrics)
- Use informer caching effectively (don't bypass cache unnecessarily)
- Batch API operations where possible (reduce API server load)

**Container Resources:**
- Set CPU/memory requests based on typical usage (80th percentile)
- Set CPU/memory limits with headroom for spikes (95th percentile)
- Document scaling characteristics in chart README
- Test with realistic load (use load testing tools)
- Monitor resource utilization in production

**Resource Guidelines:**
- Operator: requests 100m CPU / 128Mi memory, limits 1000m CPU / 512Mi memory
- Collectors: scale with cluster size, document scaling formula
- UI: requests 50m CPU / 64Mi memory, limits 500m CPU / 256Mi memory
- Adjust based on actual measurements

### Metrics Collection

**Cardinality Management:**
- Minimize metrics cardinality (avoid high-cardinality labels like IDs, UUIDs, IPs)
- Use aggregation for high-cardinality data
- Drop unnecessary labels at collection time
- Document expected cardinality in code comments
- Monitor total metric count and cardinality in VictoriaMetrics

**High-Cardinality Examples to Avoid:**
- ❌ Pod names as labels (use workload name instead)
- ❌ IP addresses as labels
- ❌ Timestamps as labels (use time range in queries)
- ❌ User IDs as labels (aggregate or use logs instead)
- ✓ Namespace, cluster, workload type (bounded cardinality)

**Sampling and Aggregation:**
- Sample high-frequency metrics (>1Hz) or aggregate before export
- Use histograms for latency metrics (not individual measurements)
- Configure appropriate scrape intervals (15s-60s typical)
- Consider storage costs for metric retention periods
- Test with realistic metric volumes (millions of time series)

### Multi-Cluster Scale

**Scalability Testing:**
- Test with multiple child clusters (simulate 10+ clusters)
- Measure network bandwidth for metrics aggregation
- Profile Promxy query performance under load
- Monitor VictoriaMetrics storage growth over time
- Test regional cluster failure scenarios

**Scaling Considerations:**
- Plan for 100+ child clusters per regional cluster
- Regional cluster resources scale with number of children
- Mothership resources scale with total cluster count
- Network bandwidth: ~1-10 Mbps per child cluster (metrics)
- Storage: ~1-10 GB per cluster per month (metrics retention)

**Optimization Techniques:**
- Use metric relabeling to reduce data volume
- Configure appropriate retention periods per metric importance
- Enable compression for cross-cluster communication
- Use dedicated network paths for observability traffic
- Implement backpressure and rate limiting

---

## Explicit Agent Guidance

### When Reviewing PRs

1. **Trust these instructions** for build, test, and validation standards
2. **Search the codebase** only when information here is incomplete, contradictory, or clearly outdated
3. **Always use Makefile targets** for build, test, and deploy steps unless a task is explicitly not covered
4. **Check all required CI validations** pass before approving
5. **Verify documentation updates** accompany code changes
6. **Flag security concerns** immediately with clear explanation
7. **Check for breaking changes** and ensure proper protocol followed
8. **Validate test coverage** is adequate for the changes
9. **Review resource implications** for Kubernetes deployments
10. **Consider multi-cluster impact** for all observability changes

### When Suggesting Code Changes

1. **Provide complete, working code** not pseudocode or partial snippets
2. **Follow existing patterns** in the codebase (check similar implementations)
3. **Include error handling** in all suggested code
4. **Add appropriate tests** alongside code suggestions
5. **Update documentation** when suggesting new features
6. **Consider backwards compatibility** in all suggestions
7. **Explain the reasoning** behind significant architectural choices
8. **Reference best practices** from these instructions when applicable

### When Running Commands

1. **Do not run commands requiring a cluster** unless environment is confirmed ready
2. **Use timeout_ms** for long-running commands (builds, tests, deployments)
3. **Check prerequisites** before running setup commands (Go version, Docker, etc.)
4. **Review command output** for errors or warnings
5. **Follow debugging steps** from these instructions on failures

### Red Flags to Watch For

- Secrets or credentials in code, logs, or documentation
- Unbounded loops or recursion
- Missing error handling
- Hardcoded values that should be configurable
- Deprecated Kubernetes API versions
- Missing resource limits in manifests
- High-cardinality metrics without justification
- Breaking changes without proper marking
- Tests that are flaky or environment-dependent
- Complex logic without comments or documentation

---

**If you encounter a failure or missing dependency, check the relevant documentation in `docs/` and the Makefile for workarounds and required steps. When in doubt, ask for clarification rather than making assumptions about project requirements or standards.**