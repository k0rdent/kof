LOCALBIN ?= $(shell pwd)/bin
export LOCALBIN
$(LOCALBIN):
	mkdir -p $(LOCALBIN)


TEMPLATES_DIR := charts
PROVIDER_TEMPLATES_DIR := $(TEMPLATES_DIR)/provider
export PROVIDER_TEMPLATES_DIR
CHARTS_PACKAGE_DIR ?= $(LOCALBIN)/charts
EXTENSION_CHARTS_PACKAGE_DIR ?= $(LOCALBIN)/charts/extensions
$(EXTENSION_CHARTS_PACKAGE_DIR): | $(LOCALBIN)
	mkdir -p $(EXTENSION_CHARTS_PACKAGE_DIR)
$(CHARTS_PACKAGE_DIR): | $(LOCALBIN)
	rm -rf $(CHARTS_PACKAGE_DIR)
	mkdir -p $(CHARTS_PACKAGE_DIR)

REGISTRY_NAME ?= kcm-local-registry
REGISTRY_PORT ?= 5001
REGISTRY_REPO ?= oci://127.0.0.1:$(REGISTRY_PORT)/charts
REGISTRY_IS_OCI = $(shell echo $(REGISTRY_REPO) | grep -q oci && echo true || echo false)

TEMPLATE_FOLDERS = $(patsubst $(TEMPLATES_DIR)/%,%,$(wildcard $(TEMPLATES_DIR)/*))

USER_EMAIL=$(shell git config user.email)

CLOUD_CLUSTER_TEMPLATE ?= aws-standalone
MANAGED_CLUSTER_NAME = $(USER)-$(CLOUD_CLUSTER_TEMPLATE)-managed
STORAGE_DOMAIN = $(USER)-$(CLOUD_CLUSTER_TEMPLATE)-storage.$(KOF_DNS)
KOF_STORAGE_NAME = kof-storage
KOF_STORAGE_NG = kof

KIND_CLUSTER_NAME ?= kcm-dev


dev:
	mkdir -p dev

lint-chart-%:
	$(HELM) dependency update $(TEMPLATES_DIR)/$*
	$(HELM) lint --strict $(TEMPLATES_DIR)/$* --set global.lint=true

package-chart-%: lint-chart-%
	$(HELM) package --destination $(CHARTS_PACKAGE_DIR) $(TEMPLATES_DIR)/$*

.PHONY: helm-package
helm-package: $(CHARTS_PACKAGE_DIR) $(EXTENSION_CHARTS_PACKAGE_DIR)
	rm -rf $(CHARTS_PACKAGE_DIR)
	@make $(patsubst %,package-chart-%,$(TEMPLATE_FOLDERS))

.PHONY: helm-push
helm-push: helm-package
	@if [ ! $(REGISTRY_IS_OCI) ]; then \
	    repo_flag="--repo"; \
	fi; \
	for chart in $(CHARTS_PACKAGE_DIR)/*.tgz; do \
		base=$$(basename $$chart .tgz); \
		chart_version=$$(echo $$base | grep -o "v\{0,1\}[0-9]\+\.[0-9]\+\.[0-9].*"); \
		chart_name="$${base%-"$$chart_version"}"; \
		echo "Verifying if chart $$chart_name, version $$chart_version already exists in $(REGISTRY_REPO)"; \
		if $(REGISTRY_IS_OCI); then \
			chart_exists=$$($(HELM) pull $$repo_flag $(REGISTRY_REPO)/$$chart_name --version $$chart_version --destination /tmp 2>&1 | grep "not found" || true); \
		else \
			chart_exists=$$($(HELM) pull $$repo_flag $(REGISTRY_REPO) $$chart_name --version $$chart_version --destination /tmp 2>&1 | grep "not found" || true); \
		fi; \
		if [ -z "$$chart_exists" ]; then \
			echo "Chart $$chart_name version $$chart_version already exists in the repository."; \
		fi; \
		if $(REGISTRY_IS_OCI); then \
			echo "Pushing $$chart to $(REGISTRY_REPO)"; \
			$(HELM) push "$$chart" $(REGISTRY_REPO); \
		else \
			if [ ! $$REGISTRY_USERNAME ] && [ ! $$REGISTRY_PASSWORD ]; then \
				echo "REGISTRY_USERNAME and REGISTRY_PASSWORD must be populated to push the chart to an HTTPS repository"; \
				exit 1; \
			else \
				$(HELM) repo add kcm $(REGISTRY_REPO); \
				echo "Pushing $$chart to $(REGISTRY_REPO)"; \
				$(HELM) cm-push "$$chart" $(REGISTRY_REPO) --username $$REGISTRY_USERNAME --password $$REGISTRY_PASSWORD; \
			fi; \
		fi; \
	done

.PHONY: promxy-operator-docker-build
promxy-operator-docker-build: ## Build promxy-operator controller docker image
	cd promxy-operator && make docker-build
	$(KIND) load docker-image promxy-operator-controller --name $(KIND_CLUSTER_NAME)

.PHONY: dev-operators-deploy
dev-operators-deploy: dev ## Deploy kof-operators helm chart to the K8s cluster specified in ~/.kube/config
	cp -f $(TEMPLATES_DIR)/kof-operators/values.yaml dev/operators-values.yaml
	$(HELM) upgrade -i kof-operators ./charts/kof-operators --create-namespace -n kof -f dev/operators-values.yaml

.PHONY: dev-collectors-deploy
dev-collectors-deploy: dev ## Deploy kof-collector helm chart to the K8s cluster specified in ~/.kube/config
	cp -f $(TEMPLATES_DIR)/kof-collectors/values.yaml dev/collectors-values.yaml
	@$(YQ) eval -i '.kof.logs.endpoint = "http://$(KOF_STORAGE_NAME)-victoria-logs-single-server.$(KOF_STORAGE_NG):9428/insert/opentelemetry/v1/logs"' dev/collectors-values.yaml
	@$(YQ) eval -i '.kof.metrics.endpoint = "http://vminsert-cluster.$(KOF_STORAGE_NG):8480/insert/0/prometheus/api/v1/write"' dev/collectors-values.yaml
	@$(YQ) eval -i '.opencost.opencost.prometheus.external.url = "http://vmselect-cluster.$(KOF_STORAGE_NG):8481/select/0/prometheus"' dev/collectors-values.yaml
	$(HELM) upgrade -i kof-collectors ./charts/kof-collectors --create-namespace -n kof -f dev/collectors-values.yaml

.PHONY: dev-storage-deploy
dev-storage-deploy: dev ## Deploy kof-storage helm chart to the K8s cluster specified in ~/.kube/config
	cp -f $(TEMPLATES_DIR)/kof-storage/values.yaml dev/storage-values.yaml
	@$(YQ) eval -i '.grafana.enabled = false' dev/storage-values.yaml
	@$(YQ) eval -i '.victoria-metrics-operator.enabled = false' dev/storage-values.yaml
	@$(YQ) eval -i '.victoriametrics.enabled = false' dev/storage-values.yaml
	@$(YQ) eval -i '.promxy.enabled = true' dev/storage-values.yaml
	@$(YQ) eval -i '.global.storageClass = "standard"' dev/storage-values.yaml
	@$(YQ) eval -i '.["victoria-logs-single"].server.persistentVolume.storageClassName = "standard"' dev/storage-values.yaml
	$(HELM) upgrade -i $(KOF_STORAGE_NAME) ./charts/kof-storage --create-namespace -n $(KOF_STORAGE_NG) -f dev/storage-values.yaml

.PHONY: dev-ms-deploy-cloud
dev-ms-deploy-cloud: dev promxy-operator-docker-build ## Deploy Mothership helm chart to the K8s cluster specified in ~/.kube/config for a remote storage cluster
	cp -f $(TEMPLATES_DIR)/kof-mothership/values.yaml dev/mothership-values.yaml
	@$(YQ) eval -i '.kcm.installTemplates = true' dev/mothership-values.yaml
	@$(YQ) eval -i '.kcm.kof.clusterProfiles.kof-aws-dns-secrets = {"matchLabels": {"k0rdent.mirantis.com/kof-aws-dns-secrets": "true"}, "secrets": ["external-dns-aws-credentials"]}' dev/mothership-values.yaml
	@$(YQ) eval -i '.grafana.logSources = [{"name": "$(USER)-aws-storage", "url": "https://vmauth.$(STORAGE_DOMAIN)/vls", "type": "victoriametrics-logs-datasource", "auth": {"credentials_secret_name": "storage-vmuser-credentials", "username_key": "username", "password_key": "password"}}]' dev/mothership-values.yaml

	@$(YQ) eval -i '.promxy.operator.image.repository= "promxy-operator-controller"' dev/mothership-values.yaml
	@if [ "$(REGISTRY_REPO)" = "oci://127.0.0.1:$(REGISTRY_PORT)/charts" ]; then \
		$(YQ) eval -i '.kcm.kof.repo.url = "oci://$(REGISTRY_NAME):5000/charts"' dev/mothership-values.yaml; \
		$(YQ) eval -i '.kcm.kof.repo.insecure = true' dev/mothership-values.yaml; \
		$(YQ) eval -i '.kcm.kof.repo.type = "oci"' dev/mothership-values.yaml; \
	else \
		$(YQ) eval -i '.kcm.kof.repo.url = "$(REGISTRY_REPO)"' dev/mothership-values.yaml; \
	fi; \
	$(HELM) upgrade -i kof-mothership ./charts/kof-mothership -n kof --create-namespace -f dev/mothership-values.yaml

.PHONY: dev-storage-deploy-cloud
dev-storage-deploy-cloud: dev ## Deploy Regional Managed cluster using KCM
	cp -f demo/cluster/$(CLOUD_CLUSTER_TEMPLATE)-storage.yaml dev/$(CLOUD_CLUSTER_TEMPLATE)-storage.yaml
	@$(YQ) eval -i '.metadata.name = "$(USER)-$(CLOUD_CLUSTER_TEMPLATE)-storage"' dev/$(CLOUD_CLUSTER_TEMPLATE)-storage.yaml # set the same name for both documents in yaml
	@$(YQ) eval -i 'select(documentIndex == 1).spec.cluster_name = "$(USER)-$(CLOUD_CLUSTER_TEMPLATE)-storage"' dev/$(CLOUD_CLUSTER_TEMPLATE)-storage.yaml
	@$(YQ) 'select(documentIndex == 0).spec.serviceSpec.services[] | select(.name == "kof-storage") | .values' dev/$(CLOUD_CLUSTER_TEMPLATE)-storage.yaml > dev/kof-storage-values.yaml
	@$(YQ) eval -i '.["cert-manager"].email = "$(USER_EMAIL)"' dev/kof-storage-values.yaml
	@$(YQ) eval -i '.victoriametrics.vmauth.ingress.host = "vmauth.$(STORAGE_DOMAIN)"' dev/kof-storage-values.yaml
	@$(YQ) eval -i '.grafana.ingress.host = "grafana.$(STORAGE_DOMAIN)"' dev/kof-storage-values.yaml
	@$(YQ) eval -i '.["external-dns"].enabled = true' dev/kof-storage-values.yaml
	@$(YQ) eval -i '(select(documentIndex == 0).spec.serviceSpec.services[] | select(.name == "kof-storage")).values |= load_str("dev/kof-storage-values.yaml")' dev/$(CLOUD_CLUSTER_TEMPLATE)-storage.yaml
	@$(YQ) eval -i 'select(documentIndex == 1).spec.targets = ["vmauth.$(STORAGE_DOMAIN):443"]' dev/$(CLOUD_CLUSTER_TEMPLATE)-storage.yaml
	kubectl apply -f dev/$(CLOUD_CLUSTER_TEMPLATE)-storage.yaml

.PHONY: dev-managed-deploy-cloud
dev-managed-deploy-cloud: dev ## Deploy Regional Managed cluster using KCM
	cp -f demo/cluster/$(CLOUD_CLUSTER_TEMPLATE)-managed.yaml dev/$(CLOUD_CLUSTER_TEMPLATE)-managed.yaml
	@$(YQ) eval -i 'select(documentIndex == 0) | .metadata.name = "$(MANAGED_CLUSTER_NAME)"' dev/$(CLOUD_CLUSTER_TEMPLATE)-managed.yaml
	@$(YQ) '.spec.serviceSpec.services[] | select(.name == "kof-collectors") | .values' dev/$(CLOUD_CLUSTER_TEMPLATE)-managed.yaml > dev/kof-managed-values.yaml
	@$(YQ) eval -i '.global.clusterName = "$(MANAGED_CLUSTER_NAME)"' dev/kof-managed-values.yaml
	@$(YQ) eval -i '.opencost.opencost.exporter.defaultClusterId = "$(MANAGED_CLUSTER_NAME)"' dev/kof-managed-values.yaml
	@$(YQ) eval -i '.opencost.opencost.prometheus.external.url = "https://vmauth.$(STORAGE_DOMAIN)/vm/select/0/prometheus"' dev/kof-managed-values.yaml
	@$(YQ) eval -i '.kof.logs.endpoint = "https://vmauth.$(STORAGE_DOMAIN)/vls/insert/opentelemetry/v1/logs"' dev/kof-managed-values.yaml
	@$(YQ) eval -i '.kof.metrics.endpoint = "https://vmauth.$(STORAGE_DOMAIN)/vm/insert/0/prometheus/api/v1/write"' dev/kof-managed-values.yaml
	@$(YQ) eval -i '(.spec.serviceSpec.services[] | select(.name == "kof-collectors")).values |= load_str("dev/kof-managed-values.yaml")' dev/$(CLOUD_CLUSTER_TEMPLATE)-managed.yaml
	kubectl apply -f dev/$(CLOUD_CLUSTER_TEMPLATE)-managed.yaml

## Tool Binaries
KUBECTL ?= kubectl
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen-$(CONTROLLER_TOOLS_VERSION)
ENVTEST ?= $(LOCALBIN)/setup-envtest-$(ENVTEST_VERSION)
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint-$(GOLANGCI_LINT_VERSION)
HELM ?= $(LOCALBIN)/helm-$(HELM_VERSION)
export HELM
KIND ?= $(LOCALBIN)/kind-$(KIND_VERSION)
YQ ?= $(LOCALBIN)/yq-$(YQ_VERSION)
export YQ

## Tool Versions
HELM_VERSION ?= v3.15.1
YQ_VERSION ?= v4.44.2
KIND_VERSION ?= v0.23.0

.PHONY: yq
yq: $(YQ) ## Download yq locally if necessary.
$(YQ): | $(LOCALBIN)
	$(call go-install-tool,$(YQ),github.com/mikefarah/yq/v4,${YQ_VERSION})

.PHONY: kind
kind: $(KIND) ## Download kind locally if necessary.
$(KIND): | $(LOCALBIN)
	$(call go-install-tool,$(KIND),sigs.k8s.io/kind,${KIND_VERSION})

.PHONY: helm
helm: $(HELM) ## Download helm locally if necessary.
HELM_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3"
$(HELM): | $(LOCALBIN)
	rm -f $(LOCALBIN)/helm-*
	curl -s --fail $(HELM_INSTALL_SCRIPT) | USE_SUDO=false HELM_INSTALL_DIR=$(LOCALBIN) DESIRED_VERSION=$(HELM_VERSION) BINARY_NAME=helm-$(HELM_VERSION) PATH="$(LOCALBIN):$(PATH)" bash

.PHONY: cli-install
cli-install: yq helm kind ## Install the necessary CLI tools for deployment, development and testing.

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary (ideally with version)
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f $(1) ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
if [ ! -f $(1) ]; then mv -f "$$(echo "$(1)" | sed "s/-$(3)$$//")" $(1); fi ;\
}
endef
