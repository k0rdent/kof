LOCALBIN ?= $(shell pwd)/bin
export LOCALBIN
$(LOCALBIN):
	mkdir -p $(LOCALBIN)


TEMPLATES_DIR := charts
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
CLOUD_CLUSTER_REGION ?= us-east-2
CHILD_CLUSTER_NAME = $(USER)-$(CLOUD_CLUSTER_TEMPLATE)-child
REGIONAL_CLUSTER_NAME = $(USER)-$(CLOUD_CLUSTER_TEMPLATE)-regional
REGIONAL_DOMAIN = $(REGIONAL_CLUSTER_NAME).$(KOF_DNS)
KOF_STORAGE_NAME = kof-storage
KOF_STORAGE_NG = kof

KIND_CLUSTER_NAME ?= kcm-dev

define set_local_registry
	$(eval $@_VALUES = $(1))
	@if [ "$(REGISTRY_REPO)" = "oci://127.0.0.1:$(REGISTRY_PORT)/charts" ]; then \
		$(YQ) eval -i '.kcm.kof.repo.url = "oci://$(REGISTRY_NAME):5000/charts"' ${$@_VALUES}; \
		$(YQ) eval -i '.kcm.kof.repo.insecure = true' ${$@_VALUES}; \
		$(YQ) eval -i '.kcm.kof.repo.type = "oci"' ${$@_VALUES}; \
	else \
		$(YQ) eval -i '.kcm.kof.repo.url = "$(REGISTRY_REPO)"' ${$@_VALUES}; \
	fi;
endef

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

.PHONY: kof-operator-docker-build
kof-operator-docker-build: ## Build kof-operator controller docker image
	cd kof-operator && make docker-build
	$(KIND) load docker-image kof-operator-controller --name $(KIND_CLUSTER_NAME)

.PHONY: dev-operators-deploy
dev-operators-deploy: dev ## Deploy kof-operators helm chart to the K8s cluster specified in ~/.kube/config
	cp -f $(TEMPLATES_DIR)/kof-operators/values.yaml dev/operators-values.yaml
	$(HELM) upgrade -i --wait kof-operators ./charts/kof-operators --create-namespace -n kof -f dev/operators-values.yaml

.PHONY: dev-collectors-deploy
dev-collectors-deploy: dev ## Deploy kof-collector helm chart to the K8s cluster specified in ~/.kube/config
	cp -f $(TEMPLATES_DIR)/kof-collectors/values.yaml dev/collectors-values.yaml
	@$(YQ) eval -i '.kof.logs.endpoint = "http://$(KOF_STORAGE_NAME)-victoria-logs-single-server.$(KOF_STORAGE_NG):9428/insert/opentelemetry/v1/logs"' dev/collectors-values.yaml
	@$(YQ) eval -i '.kof.metrics.endpoint = "http://vminsert-cluster.$(KOF_STORAGE_NG):8480/insert/0/prometheus/api/v1/write"' dev/collectors-values.yaml
	@$(YQ) eval -i '.opencost.opencost.prometheus.external.url = "http://vmselect-cluster.$(KOF_STORAGE_NG):8481/select/0/prometheus"' dev/collectors-values.yaml
	$(HELM) upgrade -i --wait kof-collectors ./charts/kof-collectors --create-namespace -n kof -f dev/collectors-values.yaml

.PHONY: dev-istio-deploy
dev-istio-deploy: dev ## Deploy kof-istio helm chart to the K8s cluster specified in ~/.kube/config
	cp -f $(TEMPLATES_DIR)/kof-istio/values.yaml dev/istio-values.yaml
	@$(call set_local_registry, "dev/istio-values.yaml")
	$(HELM) upgrade -i --wait kof-istio ./charts/kof-istio --create-namespace -n istio-system -f dev/istio-values.yaml

.PHONY: dev-storage-deploy
dev-storage-deploy: dev ## Deploy kof-storage helm chart to the K8s cluster specified in ~/.kube/config
	cp -f $(TEMPLATES_DIR)/kof-storage/values.yaml dev/storage-values.yaml
	@$(YQ) eval -i '.grafana.enabled = false' dev/storage-values.yaml
	@$(YQ) eval -i '.grafana.security.create_secret = false' dev/storage-values.yaml
	@$(YQ) eval -i '.victoria-metrics-operator.enabled = false' dev/storage-values.yaml
	@$(YQ) eval -i '.victoriametrics.enabled = false' dev/storage-values.yaml
	@$(YQ) eval -i '.promxy.enabled = true' dev/storage-values.yaml
	@$(YQ) eval -i '.global.storageClass = "standard"' dev/storage-values.yaml
	@$(YQ) eval -i '.["victoria-logs-single"].server.persistentVolume.storageClassName = "standard"' dev/storage-values.yaml
	$(HELM) upgrade -i --wait $(KOF_STORAGE_NAME) ./charts/kof-storage --create-namespace -n $(KOF_STORAGE_NG) -f dev/storage-values.yaml

.PHONY: dev-ms-deploy
dev-ms-deploy: dev kof-operator-docker-build ## Deploy `kof-mothership` helm chart to the management cluster
	cp -f $(TEMPLATES_DIR)/kof-mothership/values.yaml dev/mothership-values.yaml
	@$(YQ) eval -i '.kcm.installTemplates = true' dev/mothership-values.yaml
	@$(YQ) eval -i '.kcm.kof.clusterProfiles.kof-aws-dns-secrets = {"matchLabels": {"k0rdent.mirantis.com/kof-aws-dns-secrets": "true"}, "secrets": ["external-dns-aws-credentials"]}' dev/mothership-values.yaml

	@$(YQ) eval -i '.kcm.kof.operator.image.repository = "kof-operator-controller"' dev/mothership-values.yaml
	@$(call set_local_registry, "dev/mothership-values.yaml")
	$(HELM) upgrade -i --wait --create-namespace -n kof kof-mothership ./charts/kof-mothership -f dev/mothership-values.yaml
	kubectl rollout restart -n kof deployment/kof-mothership-kof-operator

.PHONY: dev-regional-deploy-cloud
dev-regional-deploy-cloud: dev ## Deploy regional cluster using k0rdent
	cp -f demo/cluster/$(CLOUD_CLUSTER_TEMPLATE)-regional.yaml dev/$(CLOUD_CLUSTER_TEMPLATE)-regional.yaml
	@$(YQ) eval -i '.metadata.name = "$(REGIONAL_CLUSTER_NAME)"' dev/$(CLOUD_CLUSTER_TEMPLATE)-regional.yaml # set the same name for all documents in yaml
	@$(YQ) eval -i 'select(documentIndex == 0).spec.config.region = "$(CLOUD_CLUSTER_REGION)"' dev/$(CLOUD_CLUSTER_TEMPLATE)-regional.yaml
	@$(YQ) eval -i 'select(documentIndex == 1).spec.cluster_name = "$(REGIONAL_CLUSTER_NAME)"' dev/$(CLOUD_CLUSTER_TEMPLATE)-regional.yaml
	@$(YQ) eval -i 'select(documentIndex == 0).metadata.labels["k0rdent.mirantis.com/kof-regional-domain"] = "$(REGIONAL_DOMAIN)"' dev/$(CLOUD_CLUSTER_TEMPLATE)-regional.yaml
	@$(YQ) 'select(documentIndex == 0).spec.serviceSpec.services[] | select(.name == "kof-storage") | .values' dev/$(CLOUD_CLUSTER_TEMPLATE)-regional.yaml > dev/kof-storage-values.yaml
	@$(YQ) eval -i '.["cert-manager"].email = "$(USER_EMAIL)"' dev/kof-storage-values.yaml
	@$(YQ) eval -i '.["external-dns"] = {"enabled": true, "env": [{"name": "AWS_SHARED_CREDENTIALS_FILE", "value": "/etc/aws/credentials/external-dns-aws-credentials"}, {"name": "AWS_DEFAULT_REGION", "value": "$(CLOUD_CLUSTER_REGION)"}]}' dev/kof-storage-values.yaml
	@$(YQ) eval -i '(select(documentIndex == 0).spec.serviceSpec.services[] | select(.name == "kof-storage")).values |= load_str("dev/kof-storage-values.yaml")' dev/$(CLOUD_CLUSTER_TEMPLATE)-regional.yaml
	@$(YQ) eval -i 'select(documentIndex == 1).spec.targets = ["vmauth.$(REGIONAL_DOMAIN):443"]' dev/$(CLOUD_CLUSTER_TEMPLATE)-regional.yaml
	@$(YQ) eval -i 'select(documentIndex == 2).spec.datasource.name =  "$(REGIONAL_CLUSTER_NAME)"' dev/$(CLOUD_CLUSTER_TEMPLATE)-regional.yaml
	@$(YQ) eval -i 'select(documentIndex == 2).spec.datasource.url = "https://vmauth.$(REGIONAL_DOMAIN)/vls"' dev/$(CLOUD_CLUSTER_TEMPLATE)-regional.yaml
	kubectl apply -f dev/$(CLOUD_CLUSTER_TEMPLATE)-regional.yaml

.PHONY: dev-child-deploy-cloud
dev-child-deploy-cloud: dev ## Deploy child cluster using k0rdent
	cp -f demo/cluster/$(CLOUD_CLUSTER_TEMPLATE)-child.yaml dev/$(CLOUD_CLUSTER_TEMPLATE)-child.yaml
	@$(YQ) eval -i 'select(documentIndex == 0).metadata.name = "$(CHILD_CLUSTER_NAME)"' dev/$(CLOUD_CLUSTER_TEMPLATE)-child.yaml
	@$(YQ) eval -i 'select(documentIndex == 0).spec.config.region = "$(CLOUD_CLUSTER_REGION)"' dev/$(CLOUD_CLUSTER_TEMPLATE)-child.yaml
	@# Optional, auto-detected by region:
	@# $(YQ) eval -i 'select(documentIndex == 0).metadata.labels["k0rdent.mirantis.com/kof-regional-cluster-name"] = "$(REGIONAL_CLUSTER_NAME)"' dev/$(CLOUD_CLUSTER_TEMPLATE)-child.yaml
	kubectl apply -f dev/$(CLOUD_CLUSTER_TEMPLATE)-child.yaml

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
