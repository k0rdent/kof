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

## Explicit Agent Guidance

- **Trust these instructions** for build, test, and validation. Only search the codebase if information here is incomplete or in error.
- **Always use the Makefile targets** for build, test, and deploy steps unless a task is not covered.
- **Do not run commands that require a running cluster unless the environment is ready.**
- **For PRs:** Ensure all CI checks pass, including lint, test, and Helm chart validation.
- **For Go code:** Use the provided `Makefile` and `golangci-lint` config.
- **For React/TypeScript:** Use `npm run build`, `npm run lint`, and `npm run test` in `kof-operator/webapp/collector`.
- **For Helm:** Use `make helm-package`, `make helm-push`, and `make dev-*-deploy` targets.
- **For upgrades:** Follow the steps in `docs/dev.md` and `docs/release.md`.

---

**If you encounter a failure or missing dependency, check the relevant documentation in `docs/` and the Makefile for workarounds and required steps.**
