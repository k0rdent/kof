LOCALBIN ?= $(shell pwd)/bin
export LOCALBIN
$(LOCALBIN):
	mkdir -p $(LOCALBIN)


HOSTOS := $(shell go env GOHOSTOS)
HOSTARCH := $(shell go env GOHOSTARCH)

TEMPLATES_DIR := charts
CHARTS_PACKAGE_DIR ?= $(LOCALBIN)/charts
EXTENSION_CHARTS_PACKAGE_DIR ?= $(LOCALBIN)/charts/extensions
$(EXTENSION_CHARTS_PACKAGE_DIR): | $(LOCALBIN)
	mkdir -p $(EXTENSION_CHARTS_PACKAGE_DIR)
$(CHARTS_PACKAGE_DIR): | $(LOCALBIN)
	rm -rf $(CHARTS_PACKAGE_DIR)
	mkdir -p $(CHARTS_PACKAGE_DIR)

KCM_NAMESPACE ?= kcm-system
CONTAINER_TOOL ?= docker
KIND_NETWORK ?= kind
REGISTRY_NAME ?= kof
REGISTRY_PORT ?= 8080
REGISTRY_REPO ?= http://127.0.0.1:$(REGISTRY_PORT)
REGISTRY_IS_OCI = $(shell echo $(REGISTRY_REPO) | grep -q oci && echo true || echo false)
REGISTRY_PLAIN_HTTP ?= false

TEMPLATE_FOLDERS = $(patsubst $(TEMPLATES_DIR)/%,%,$(wildcard $(TEMPLATES_DIR)/*))

USER_EMAIL=$(shell git config user.email)

CLOUD_CLUSTER_TEMPLATE ?= aws-standalone
CLOUD_CLUSTER_REGION ?= us-east-2
CHILD_CLUSTER_NAME = $(USER)-$(CLOUD_CLUSTER_TEMPLATE)-child
REGIONAL_CLUSTER_NAME = $(USER)-$(CLOUD_CLUSTER_TEMPLATE)-regional
REGIONAL_DOMAIN = $(REGIONAL_CLUSTER_NAME).$(KOF_DNS)

KIND_CLUSTER_NAME ?= kcm-dev

define set_local_registry
	$(eval $@_VALUES = $(1))
	$(YQ) eval -i '.kcm.kof.repo.spec.url = "http://$(REGISTRY_NAME):8080"' ${$@_VALUES}
	$(YQ) eval -i '.kcm.kof.repo.spec.type = "default"' ${$@_VALUES}
endef

define set_region
	$(eval $@_VALUES = $(1))
	if [[ "$(CLOUD_CLUSTER_TEMPLATE)" == aws-* ]]; \
	then \
		$(YQ) -i '.spec.config.region = "$(CLOUD_CLUSTER_REGION)"' ${$@_VALUES}; \
	elif [[ "$(CLOUD_CLUSTER_TEMPLATE)" == azure-* ]]; \
	then \
		$(YQ) -i '.spec.config.location = "$(CLOUD_CLUSTER_REGION)"' ${$@_VALUES}; \
		$(YQ) -i '.spec.config.subscriptionID = "'"$$AZURE_SUBSCRIPTION_ID"'"' ${$@_VALUES}; \
	fi
endef

dev:
	mkdir -p dev
lint-chart-%:
	$(HELM) dependency update $(TEMPLATES_DIR)/$*
	$(HELM) lint --strict $(TEMPLATES_DIR)/$* --set global.lint=true

package-chart-%: lint-chart-%
	$(HELM) package --destination $(CHARTS_PACKAGE_DIR) $(TEMPLATES_DIR)/$*


.PHONY: registry-deploy
registry-deploy:
	@if [ ! "$$($(CONTAINER_TOOL) ps -aq -f name=$(REGISTRY_NAME))" ]; then \
		echo "Starting new local registry container $(REGISTRY_NAME)"; \
		$(CONTAINER_TOOL) run -d --restart=always -p "127.0.0.1:$(REGISTRY_PORT):8080" --network bridge \
			--name "$(REGISTRY_NAME)" \
			-e STORAGE=local \
			-e STORAGE_LOCAL_ROOTDIR=/var/tmp \
			ghcr.io/helm/chartmuseum:v0.16.2 ;\
	fi; \
	if [ "$$($(CONTAINER_TOOL) inspect -f='{{json .NetworkSettings.Networks.$(KIND_NETWORK)}}' $(REGISTRY_NAME))" = 'null' ]; then \
		$(CONTAINER_TOOL) network connect $(KIND_NETWORK) $(REGISTRY_NAME); \
	fi

.PHONY: helm-package
helm-package: $(CHARTS_PACKAGE_DIR) $(EXTENSION_CHARTS_PACKAGE_DIR)
	rm -rf $(CHARTS_PACKAGE_DIR)
	@make $(patsubst %,package-chart-%,$(TEMPLATE_FOLDERS))

.PHONY: helm-push
helm-push: helm-package
	@if [ ! $(REGISTRY_IS_OCI) ]; then \
	    repo_flag="--repo"; \
	fi; \
	if [ $(REGISTRY_PLAIN_HTTP) = "true" ]; \
	then plain_http_flag="--plain-http"; \
	else plain_http_flag=""; \
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
			$(HELM) push $${plain_http_flag} "$$chart" $(REGISTRY_REPO); \
		else \
			$(HELM) repo add kcm $(REGISTRY_REPO); \
			echo "Pushing $$chart to $(REGISTRY_REPO)"; \
			$(HELM) cm-push -f "$$chart" $(REGISTRY_REPO) --insecure; \
		fi; \
	done

.PHONY: kof-operator-docker-build
kof-operator-docker-build: ## Build kof-operator controller docker image
	cd kof-operator && make docker-build
	@kof_version=v$$($(YQ) .version $(TEMPLATES_DIR)/kof-mothership/Chart.yaml); \
	$(CONTAINER_TOOL) tag kof-operator-controller kof-operator-controller:$$kof_version; \
	$(KIND) load docker-image kof-operator-controller:$$kof_version --name $(KIND_CLUSTER_NAME)

.PHONY: dev-operators-deploy
dev-operators-deploy: dev ## Deploy kof-operators helm chart to the K8s cluster specified in ~/.kube/config
	$(HELM_UPGRADE) --create-namespace -n kof kof-operators ./charts/kof-operators

.PHONY: dev-collectors-deploy
dev-collectors-deploy: dev ## Deploy kof-collector helm chart to the K8s cluster specified in ~/.kube/config
	$(HELM_UPGRADE) -n kof kof-collectors ./charts/kof-collectors -f demo/collectors-values.yaml

.PHONY: dev-istio-deploy
dev-istio-deploy: dev ## Deploy kof-istio helm chart to the K8s cluster specified in ~/.kube/config
	cp -f $(TEMPLATES_DIR)/kof-istio/values.yaml dev/istio-values.yaml
	@$(call set_local_registry, "dev/istio-values.yaml")
	$(HELM_UPGRADE) --create-namespace -n istio-system kof-istio ./charts/kof-istio -f dev/istio-values.yaml

.PHONY: dev-adopted-rm
dev-adopted-rm: dev kind envsubst ## Create adopted cluster deployment
	@if $(KIND) get clusters | grep -q "^$(KIND_CLUSTER_NAME)$$"; then \
		if [ -n "$(KIND_CONFIG_PATH)" ]; then \
			$(KIND) delete cluster -n $(KIND_CLUSTER_NAME) --config "$(KIND_CONFIG_PATH)"; \
		else \
			$(KIND) delete cluster -n $(KIND_CLUSTER_NAME); \
		fi \
	fi; \
	$(KUBECTL) delete clusterdeployment --ignore-not-found=true $(KIND_CLUSTER_NAME) -n kcm-system || true

.PHONY: dev-adopted-deploy
dev-adopted-deploy: dev kind envsubst ## Create adopted cluster deployment
	@if ! $(KIND) get clusters | grep -q "^$(KIND_CLUSTER_NAME)$$"; then \
		if [ -n "$(KIND_CONFIG_PATH)" ]; then \
			$(KIND) create cluster -n $(KIND_CLUSTER_NAME) --config "$(KIND_CONFIG_PATH)" --wait 1m; \
		else \
			$(KIND) create cluster -n $(KIND_CLUSTER_NAME) --wait 1m; \
		fi \
	fi
	$(KUBECTL) config use kind-kcm-dev
	NAMESPACE=$(KCM_NAMESPACE) \
	KUBECONFIG_DATA=$$($(KIND) get kubeconfig --internal -n $(KIND_CLUSTER_NAME) | base64 -w 0) \
	KIND_CLUSTER_NAME=$(KIND_CLUSTER_NAME) \
	$(ENVSUBST) -no-unset -i demo/creds/adopted-credentials.yaml \
	| $(KUBECTL) apply -f -

.PHONY: dev-storage-deploy
dev-storage-deploy: dev ## Deploy kof-storage helm chart to the K8s cluster specified in ~/.kube/config
	cp -f $(TEMPLATES_DIR)/kof-storage/values.yaml dev/storage-values.yaml
	@$(YQ) eval -i '.grafana.enabled = false' dev/storage-values.yaml
	@$(YQ) eval -i '.grafana.security.create_secret = false' dev/storage-values.yaml
	@$(YQ) eval -i '.victoria-metrics-operator.enabled = false' dev/storage-values.yaml
	@$(YQ) eval -i '.victoriametrics.enabled = false' dev/storage-values.yaml
	@$(YQ) eval -i '.promxy.enabled = true' dev/storage-values.yaml
	@touch dev/vmrules.yaml
	$(HELM_UPGRADE) -n kof kof-storage ./charts/kof-storage -f dev/storage-values.yaml -f dev/vmrules.yaml

.PHONY: dev-ms-deploy
dev-ms-deploy: dev kof-operator-docker-build ## Deploy `kof-mothership` helm chart to the management cluster
	cp -f $(TEMPLATES_DIR)/kof-mothership/values.yaml dev/mothership-values.yaml
	@$(YQ) eval -i '.kcm.installTemplates = true' dev/mothership-values.yaml
	@$(YQ) eval -i '.kcm.kof.clusterProfiles.kof-aws-dns-secrets = {"matchLabels": {"k0rdent.mirantis.com/kof-aws-dns-secrets": "true"}, "secrets": ["external-dns-aws-credentials"]}' dev/mothership-values.yaml
	@$(YQ) eval -i '.kcm.kof.operator.image.registry = "docker.io/library"' dev/mothership-values.yaml # See `load docker-image`
	@$(YQ) eval -i '.kcm.kof.operator.image.repository = "kof-operator-controller"' dev/mothership-values.yaml
	@[ -f dev/dex.env ] && { \
		source dev/dex.env; \
		$(YQ) eval -i '.dex.enabled = true' dev/mothership-values.yaml; \
		$(YQ) eval -i ".dex.config.connectors[0].config.clientID = \"$${GOOGLE_CLIENT_ID}\"" dev/mothership-values.yaml; \
		$(YQ) eval -i ".dex.config.connectors[0].config.clientSecret = \"$${GOOGLE_CLIENT_SECRET}\"" dev/mothership-values.yaml; \
		host_ip=$$(${CONTAINER_TOOL} inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "${KIND_CLUSTER_NAME}-control-plane"); \
		bash ./scripts/generate-dex-secret.bash; \
		bash ./scripts/patch-coredns.bash $(KUBECTL) "dex.example.com" "$$host_ip"; \
		$(KUBECTL) rollout restart -n kof deployment/kof-mothership-dex; \
	} || true
	@$(call set_local_registry, "dev/mothership-values.yaml")
	$(KUBECTL) apply -f ./kof-operator/config/crd/bases/k0rdent.mirantis.com_servicetemplates.yaml
	$(KUBECTL) apply -f ./kof-operator/config/crd/bases/k0rdent.mirantis.com_multiclusterservices.yaml
	$(KUBECTL) apply -f ./kof-operator/config/crd/bases/k0rdent.mirantis.com_clusterdeployments.yaml
	$(KUBECTL) delete deployment kof-mothership-promxy -n kof --ignore-not-found=true
	$(HELM_UPGRADE) --take-ownership -n kof kof-mothership ./charts/kof-mothership -f dev/mothership-values.yaml
	$(KUBECTL) rollout restart -n kof deployment/kof-mothership-kof-operator
	@svctmpls='cert-manager-v1-16-4|ingress-nginx-4-12-1|kof-collectors-1-4-0|kof-operators-1-4-0|kof-storage-1-4-0'; \
	for attempt in $$(seq 1 10); do \
		if [ $$($(KUBECTL) get svctmpl -A | grep -E "$$svctmpls" | grep -c true) -eq 5 ]; then break; fi; \
		echo "|Waiting for the next service templates to become VALID:|$$svctmpls|Found:" | tr "|" "\n"; \
		$(KUBECTL) get svctmpl -A | grep -E "$$svctmpls"; \
		sleep 5; \
	done
	$(HELM_UPGRADE) -n kof kof-regional ./charts/kof-regional
	$(HELM_UPGRADE) -n kof kof-child ./charts/kof-child
	@# Workaround for `no cached repo found` in ClusterSummary for non-OCI repos only,
	@# like local `kof` HelmRepo created in kof-mothership after ClusterProfile in kof-istio:
	@if $(KUBECTL) get deploy -n projectsveltos addon-controller; then \
		$(KUBECTL) rollout restart -n projectsveltos deploy/addon-controller; \
	fi

.PHONY: dev-regional-deploy-cloud
dev-regional-deploy-cloud: dev ## Deploy regional cluster using k0rdent
	cp -f demo/cluster/$(CLOUD_CLUSTER_TEMPLATE)-regional.yaml dev/$(CLOUD_CLUSTER_TEMPLATE)-regional.yaml
	@$(YQ) eval -i '.metadata.name = "$(REGIONAL_CLUSTER_NAME)"' dev/$(CLOUD_CLUSTER_TEMPLATE)-regional.yaml
	@$(YQ) eval -i '.spec.config.clusterAnnotations["k0rdent.mirantis.com/kof-regional-domain"] = "$(REGIONAL_DOMAIN)"' dev/$(CLOUD_CLUSTER_TEMPLATE)-regional.yaml
	@$(YQ) eval -i '.spec.config.clusterAnnotations["k0rdent.mirantis.com/kof-cert-email"] = "$(USER_EMAIL)"' dev/$(CLOUD_CLUSTER_TEMPLATE)-regional.yaml
	@$(call set_region, "dev/$(CLOUD_CLUSTER_TEMPLATE)-regional.yaml")
	$(KUBECTL) apply -f dev/$(CLOUD_CLUSTER_TEMPLATE)-regional.yaml

.PHONY: dev-regional-deploy-adopted
dev-regional-deploy-adopted: dev ## Deploy regional adopted cluster using k0rdent
	cp -f demo/cluster/adopted-cluster-regional.yaml dev/adopted-cluster-regional.yaml
	@$(YQ) eval -i '.spec.config.clusterAnnotations["k0rdent.mirantis.com/kof-regional-domain"] = "adopted-cluster-regional"' dev/adopted-cluster-regional.yaml
	@$(YQ) eval -i '.spec.config.clusterAnnotations["k0rdent.mirantis.com/kof-cert-email"] = "$(USER_EMAIL)"' dev/adopted-cluster-regional.yaml
	$(KUBECTL) apply -f dev/adopted-cluster-regional.yaml
	./scripts/wait-helm-charts.bash $(HELM) $(YQ) kind-regional-adopted "cert-manager ingress-nginx kof-operators kof-storage kof-collectors"

.PHONY: dev-istio-regional-deploy-adopted
dev-istio-regional-deploy-adopted: dev ## Deploy regional adopted cluster with istio using k0rdent
	cp -f demo/cluster/adopted-cluster-istio-regional.yaml dev/adopted-cluster-istio-regional.yaml
	@$(YQ) eval -i '.spec.config.clusterAnnotations["k0rdent.mirantis.com/kof-storage-values"] = "{\"victoria-logs-cluster\":{\"vlinsert\":{\"replicaCount\":1},\"vlselect\":{\"replicaCount\":1},\"vlstorage\":{\"replicaCount\":1}},\"victoriametrics\":{\"vmcluster\":{\"spec\":{\"replicationFactor\":1,\"vminsert\":{\"replicaCount\":1},\"vmselect\":{\"replicaCount\":1},\"vmstorage\":{\"replicaCount\":1}}}}}"' dev/adopted-cluster-istio-regional.yaml
	$(KUBECTL) apply -f dev/adopted-cluster-istio-regional.yaml
	./scripts/wait-helm-charts.bash $(HELM) $(YQ) kind-regional-adopted "cert-manager kof-istio kof-istio-gateway kof-operators kof-storage kof-collectors"

.PHONY: dev-child-deploy-adopted
dev-child-deploy-adopted: dev ## Deploy regional adopted cluster using k0rdent
	cp -f demo/cluster/adopted-cluster-child.yaml dev/adopted-cluster-child.yaml
	$(KUBECTL) apply -f dev/adopted-cluster-child.yaml
	./scripts/wait-helm-charts.bash $(HELM) $(YQ) kind-child-adopted "cert-manager kof-operators kof-collectors"

.PHONY: dev-istio-child-deploy-adopted
dev-istio-child-deploy-adopted: dev ## Deploy regional adopted cluster using k0rdent
	cp -f demo/cluster/adopted-cluster-istio-child.yaml dev/adopted-cluster-istio-child.yaml
	$(KUBECTL) apply -f dev/adopted-cluster-istio-child.yaml
	./scripts/wait-helm-charts.bash $(HELM) $(YQ) kind-child-adopted "cert-manager kof-istio kof-operators kof-collectors"

.PHONY: dev-child-deploy-cloud
dev-child-deploy-cloud: dev ## Deploy child cluster using k0rdent
	cp -f demo/cluster/$(CLOUD_CLUSTER_TEMPLATE)-child.yaml dev/$(CLOUD_CLUSTER_TEMPLATE)-child.yaml
	@$(YQ) eval -i '.metadata.name = "$(CHILD_CLUSTER_NAME)"' dev/$(CLOUD_CLUSTER_TEMPLATE)-child.yaml
	@# Optional, auto-detected by region:
	@# $(YQ) eval -i '.metadata.labels["k0rdent.mirantis.com/kof-regional-cluster-name"] = "$(REGIONAL_CLUSTER_NAME)"' dev/$(CLOUD_CLUSTER_TEMPLATE)-child.yaml
	@$(call set_region, "dev/$(CLOUD_CLUSTER_TEMPLATE)-child.yaml")
	$(KUBECTL) apply -f dev/$(CLOUD_CLUSTER_TEMPLATE)-child.yaml

.PHONY: dev-promxy-port-forward
dev-promxy-port-forward: dev cli-install
	$(KUBECTL) port-forward -n kof deploy/kof-mothership-promxy 8082:8082 &

.PHONY: dev-coredns
dev-coredns: dev cli-install## Configure child and mothership coredns cluster for connectivity with kind-regional-adopted cluster
	@for attempt in $$(seq 1 10); do \
		if ! kubectl --context kind-regional-adopted get ingress vmauth-cluster -n kof ; then \
			sleep 10; \
			continue; \
		fi; \
		IFS=';'; for record in $$($(KUBECTL) --context kind-regional-adopted get ingress -n kof -o jsonpath='{range .items[*]}{.spec.rules[0].host} {.status.loadBalancer.ingress[0].ip}{";"}{end}'); do \
			host_name=$$(echo $$record | cut -d ' ' -f1); \
			host_ip=$$(echo $$record | cut -d ' ' -f2); \
			if [ -z "$$host_ip" ]; then \
				echo "$$host_name IP address is not ready yet"; \
				sleep 5; \
				continue 2; \
			fi; \
			./scripts/patch-coredns.bash "$(KUBECTL) --context kind-child-adopted" $$host_name $$host_ip; \
			./scripts/patch-coredns.bash "$(KUBECTL)" $$host_name $$host_ip; \
		done; \
		echo "🔄 Restarting CoreDNS pods..."; \
		$(KUBECTL) --context kind-child-adopted -n kube-system rollout restart deploy/coredns; \
		$(KUBECTL) -n kube-system rollout restart deploy/coredns; \
		exit 0; \
	done; \
	echo "Timeout waiting ingress IP address provisioning"; \
	exit 1

## Tool Binaries
KUBECTL ?= kubectl
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen-$(CONTROLLER_TOOLS_VERSION)
ENVTEST ?= $(LOCALBIN)/setup-envtest-$(ENVTEST_VERSION)
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint-$(GOLANGCI_LINT_VERSION)
HELM ?= $(LOCALBIN)/helm-$(HELM_VERSION)
HELM_UPGRADE = $(HELM) upgrade -i --reset-values --wait
export HELM HELM_UPGRADE
KIND ?= $(LOCALBIN)/kind-$(KIND_VERSION)
YQ ?= $(LOCALBIN)/yq-$(YQ_VERSION)
ENVSUBST ?= $(LOCALBIN)/envsubst-$(ENVSUBST_VERSION)
SUPPORT_BUNDLE_CLI ?= $(LOCALBIN)/support-bundle-$(SUPPORT_BUNDLE_CLI_VERSION)

export YQ

## Tool Versions
HELM_VERSION ?= v3.18.5
YQ_VERSION ?= v4.44.2
KIND_VERSION ?= v0.27.0
ENVSUBST_VERSION ?= v1.4.2
SUPPORT_BUNDLE_CLI_VERSION ?= v0.117.0

.PHONY: yq
yq: $(YQ) ## Download yq locally if necessary.
$(YQ): | $(LOCALBIN)
	$(call go-install-tool,$(YQ),github.com/mikefarah/yq/v4,${YQ_VERSION})

.PHONY: kind
kind: $(KIND) ## Download kind locally if necessary.
$(KIND): | $(LOCALBIN)
	$(call go-install-tool,$(KIND),sigs.k8s.io/kind,${KIND_VERSION})

.PHONY: envsubst
envsubst: $(ENVSUBST)
$(ENVSUBST): | $(LOCALBIN)
	$(call go-install-tool,$(ENVSUBST),github.com/a8m/envsubst/cmd/envsubst,${ENVSUBST_VERSION})

.PHONY: helm
helm: $(HELM) ## Download helm locally if necessary.
HELM_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3"
$(HELM): | $(LOCALBIN)
	rm -f $(LOCALBIN)/helm-*
	curl -s --fail $(HELM_INSTALL_SCRIPT) | USE_SUDO=false HELM_INSTALL_DIR=$(LOCALBIN) DESIRED_VERSION=$(HELM_VERSION) BINARY_NAME=helm-$(HELM_VERSION) PATH="$(LOCALBIN):$(PATH)" bash

.PHONY: support-bundle-cli
support-bundle-cli: $(SUPPORT_BUNDLE_CLI) ## Download support-bundle locally if necessary.
$(SUPPORT_BUNDLE_CLI): | $(LOCALBIN)
	curl -sL --fail https://github.com/replicatedhq/troubleshoot/releases/download/$(SUPPORT_BUNDLE_CLI_VERSION)/support-bundle_$(HOSTOS)_$(HOSTARCH).tar.gz | tar -xz -C $(LOCALBIN) && \
	mv $(LOCALBIN)/support-bundle $(SUPPORT_BUNDLE_CLI) && \
	chmod +x $(SUPPORT_BUNDLE_CLI)

.PHONY: helm-plugin
helm-plugin:
	@if ! $(HELM) plugin list | grep -q "cm-push"; then \
		$(HELM) plugin install https://github.com/chartmuseum/helm-push; \
	fi

.PHONY: cli-install
cli-install: yq helm kind helm-plugin ## Install the necessary CLI tools for deployment, development and testing.

.PHONY: support-bundle
support-bundle: SUPPORT_BUNDLE_OUTPUT=$(CURDIR)/support-bundle-$(shell date +"%Y-%m-%dT%H_%M_%S")
support-bundle: envsubst support-bundle-cli
	@if [ -n "$(KUBECTL_CONTEXT)" ]; then \
		NAMESPACE=$(NAMESPACE) $(ENVSUBST) -no-unset -i config/support-bundle.yaml | $(SUPPORT_BUNDLE_CLI) -o $(SUPPORT_BUNDLE_OUTPUT) --context $(KUBECTL_CONTEXT) --debug - ; \
	else \
	    NAMESPACE=$(NAMESPACE) $(ENVSUBST) -no-unset -i config/support-bundle.yaml | $(SUPPORT_BUNDLE_CLI) -o $(SUPPORT_BUNDLE_OUTPUT) --debug - ; \
	fi

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
