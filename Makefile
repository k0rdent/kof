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
KCM_REPO_PATH ?= "../kcm"
CONTAINER_TOOL ?= docker
KIND_NETWORK ?= kind
SQUID_NAME ?= squid-proxy
SQUID_PORT ?= 3128
REGISTRY_NAME ?= kcm-local-registry
REGISTRY_PORT ?= 5001
REGISTRY_REPO ?= oci://127.0.0.1:$(REGISTRY_PORT)/charts
REGISTRY_IS_OCI = $(shell echo $(REGISTRY_REPO) | grep -q oci && echo true || echo false)
REGISTRY_PLAIN_HTTP ?= true

TEMPLATE_FOLDERS = $(patsubst $(TEMPLATES_DIR)/%,%,$(wildcard $(TEMPLATES_DIR)/*))

USER_EMAIL=$(shell git config user.email)

CLOUD_CLUSTER_TEMPLATE ?= aws-standalone
CLOUD_CLUSTER_REGION ?= us-east-2
CHILD_CLUSTER_NAME = $(USER)-$(CLOUD_CLUSTER_TEMPLATE)-child
REGIONAL_CLUSTER_NAME = $(USER)-$(CLOUD_CLUSTER_TEMPLATE)-regional
REGIONAL_DOMAIN = $(REGIONAL_CLUSTER_NAME).$(KOF_DNS)

KIND_CLUSTER_NAME ?= kcm-dev
KOF_VERSION=$(shell $(YQ) .version $(TEMPLATES_DIR)/kof/Chart.yaml)

define set_local_registry
	$(eval $@_VALUES = $(1))
	$(YQ) eval -i '.global.helmRepo.kofManaged.url = "oci://$(REGISTRY_NAME):5000/charts"' ${$@_VALUES}
	$(YQ) eval -i '.global.helmRepo.kofManaged.insecure = $(REGISTRY_PLAIN_HTTP)' ${$@_VALUES}
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

define run_kind_deploy
	if [ "$(2)" = "1" ]; then \
		HTTPS_PROXY=http://$(SQUID_NAME):$(SQUID_PORT) \
		HTTP_PROXY=http://$(SQUID_NAME):$(SQUID_PORT) \
		http_proxy=http://$(SQUID_NAME):$(SQUID_PORT) \
		https_proxy=http://$(SQUID_NAME):$(SQUID_PORT) \
		make _kind_deploy KIND_CONFIG_PATH=$(1); \
	else \
		make _kind_deploy KIND_CONFIG_PATH=$(1); \
	fi; \
	make csr-approver-deploy
endef

.PHONY: csr-approver-deploy
csr-approver-deploy: ## Deploy kubelet-csr-approver to auto-approve kubelet serving certificate CSRs
	$(HELM) repo add kubelet-csr-approver https://postfinance.github.io/kubelet-csr-approver
	$(HELM) upgrade --install kubelet-csr-approver kubelet-csr-approver/kubelet-csr-approver -n kube-system --set replicas=1

dev:
	mkdir -p dev
lint-chart-%:
	$(HELM) dependency update $(TEMPLATES_DIR)/$*
	$(HELM) lint --strict $(TEMPLATES_DIR)/$* --set global.lint=true

package-chart-%: lint-chart-%
	$(HELM) package --destination $(CHARTS_PACKAGE_DIR) $(TEMPLATES_DIR)/$*

.PHONY: kcm-dev-apply
kcm-dev-apply: dev cli-install kind-deploy
	$(YQ) eval -i '.resources.limits.memory = "512Mi"' $(KCM_REPO_PATH)/config/dev/kcm_values.yaml
	make -C $(KCM_REPO_PATH) dev-apply
	$(KUBECTL) wait --for create mgmt/kcm --timeout=1m
	$(KUBECTL) wait --for=condition=Ready mgmt/kcm --timeout=10m
	$(KUBECTL) wait --for condition=available deployment/kcm-controller-manager --timeout=1m -n $(KCM_NAMESPACE)

.PHONY: kcm-dev-upgrade
kcm-dev-upgrade: dev cli-install
	$(YQ) eval -i '.resources.limits.memory = "512Mi"' $(KCM_REPO_PATH)/config/dev/kcm_values.yaml
	make -C $(KCM_REPO_PATH) dev-upgrade
	$(KUBECTL) wait --for=condition=Ready mgmt/kcm --timeout=10m
	$(KUBECTL) wait --for condition=available deployment/kcm-controller-manager --timeout=1m -n $(KCM_NAMESPACE)

.PHONY: kind-deploy
kind-deploy:
	@cp -f "$(or $(KIND_CONFIG_PATH),config/kind-local.yaml)" dev/kind-local.yaml; \
	if [ -f dev/docker/config.json ]; then \
		$(YQ) eval -i '.nodes[0].extraMounts += {"containerPath": "/var/lib/kubelet/config.json", "hostPath": "$(PWD)/dev/docker/config.json"}' dev/kind-local.yaml; \
	fi; \
	if [ -f "dev/$(SQUID_NAME).crt" ]; then \
		$(YQ) eval -i '.nodes[0].extraMounts += {"containerPath": "/usr/local/share/ca-certificates/squid.crt", "hostPath": "$(PWD)/dev/$(SQUID_NAME).crt"}' dev/kind-local.yaml; \
	fi; \
	if [ -f dev/$(REGISTRY_NAME).crt ]; then \
		$(YQ) eval -i '.nodes[0].extraMounts += {"containerPath": "/usr/local/share/ca-certificates/$(REGISTRY_NAME).crt", "hostPath": "$(PWD)/dev/$(REGISTRY_NAME).crt"}' dev/kind-local.yaml; \
	fi; \
	USE_PROXY=0; \
	if [ "$$($(CONTAINER_TOOL) ps -aq -f name=$(SQUID_NAME))" ]; then \
		USE_PROXY=1; \
	fi; \
	$(call run_kind_deploy,"dev/kind-local.yaml",$$USE_PROXY)
	$(CONTAINER_TOOL) exec $(KIND_CLUSTER_NAME)-control-plane update-ca-certificates

.PHONY: _kind_deploy
_kind_deploy:
	@if ! $(KIND) get clusters | grep -q "^$(KIND_CLUSTER_NAME)$$"; then \
		$(KIND) create cluster -n $(KIND_CLUSTER_NAME) --wait 1m \
			--config "$(KIND_CONFIG_PATH)" ; \
	fi

.PHONY: registry-deploy
registry-deploy:
		@if [ ! "$$($(CONTAINER_TOOL) ps -aq -f name=$(REGISTRY_NAME))" ]; then \
			echo "Starting new local registry container $(REGISTRY_NAME)"; \
			$(CONTAINER_TOOL) run -d --restart=always -p "127.0.0.1:$(REGISTRY_PORT):5000" --network bridge --name "$(REGISTRY_NAME)" registry:2; \
		fi; \
		if [ "$$($(CONTAINER_TOOL) inspect -f='{{json .NetworkSettings.Networks.$(KIND_NETWORK)}}' $(REGISTRY_NAME))" = 'null' ]; then \
			$(CONTAINER_TOOL) network connect $(KIND_NETWORK) $(REGISTRY_NAME); \
		fi

.PHONY: squid-deploy
squid-deploy: dev
	@if [ ! -f "dev/$(SQUID_NAME).crt" ]; then \
		openssl req -x509 -newkey rsa:4096 -keyout "dev/$(SQUID_NAME).key" -out "dev/$(SQUID_NAME).crt" -sha256 -days 3650 -nodes -subj "/CN=$(SQUID_NAME)" -addext "subjectAltName=DNS:$(SQUID_NAME)"; \
	fi; \
	if [ ! "$$($(CONTAINER_TOOL) ps -aq -f name=$(SQUID_NAME))" ]; then \
		echo "Starting new local squid container $(SQUID_NAME)"; \
		$(CONTAINER_TOOL) run -d --restart=always -p "127.0.0.1:$(SQUID_PORT):3128" --network bridge \
			--name "$(SQUID_NAME)" \
			-e TZ=UTC \
			-v $$PWD/config/squid.conf:/etc/squid/squid.conf \
			-v $$PWD/dev/$(SQUID_NAME).crt:/var/lib/squid/ssl/squid.crt \
			-v $$PWD/dev/$(SQUID_NAME).key:/var/lib/squid/ssl/squid.key \
			ecat/squid-openssl:latest ; \
	fi; \
	if [ "$$($(CONTAINER_TOOL) inspect -f='{{json .NetworkSettings.Networks.$(KIND_NETWORK)}}' $(SQUID_NAME))" = 'null' ]; then \
		$(CONTAINER_TOOL) network connect $(KIND_NETWORK) $(SQUID_NAME); \
	fi

.PHONY: set-charts-version
set-charts-version: ## Set KOF charts version, e.g. `make set-charts-version V=1.2.3`
	@echo "Updating KOF charts version from $(KOF_VERSION) to $(V)"; \
	for file in $(TEMPLATES_DIR)/*/Chart.yaml; do \
		echo "$$file"; \
		$(YQ) -i '.version = "$(V)"' "$$file"; \
		$(YQ) -i '.appVersion = "$(V)"' "$$file"; \
		$(YQ) -i '(.dependencies[] | select(.name == "kof-dashboards") | .version) = "$(V)"' "$$file"; \
	done
	$(YQ) -i '.opentelemetry-kube-stack.collectors.daemon.image.tag = "v$(V)"' $(TEMPLATES_DIR)/kof-collectors/values.yaml
	make helm-push

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
		echo "Pushing $$chart to $(REGISTRY_REPO)"; \
		$(HELM) push "$$chart" $(REGISTRY_REPO) $${plain_http_flag}; \
	done

.PHONY: kof-operator-docker-build
kof-operator-docker-build: ## Build kof-operator controller docker image
	cd kof-operator && make docker-build
	@$(CONTAINER_TOOL) tag kof-operator-controller kof-operator-controller:v$(KOF_VERSION); \
	$(KIND) load docker-image kof-operator-controller:v$(KOF_VERSION) --name $(KIND_CLUSTER_NAME); \
	$(CONTAINER_TOOL) tag kof-opentelemetry-collector-contrib ghcr.io/k0rdent/kof/kof-opentelemetry-collector-contrib:v$(KOF_VERSION); \
	$(KIND) load docker-image ghcr.io/k0rdent/kof/kof-opentelemetry-collector-contrib:v$(KOF_VERSION) --name $(KIND_CLUSTER_NAME); \
	$(CONTAINER_TOOL) tag kof-acl-server kof-acl-server:v$(KOF_VERSION); \
	$(KIND) load docker-image kof-acl-server:v$(KOF_VERSION) --name $(KIND_CLUSTER_NAME)

.PHONY: dev-adopted-rm
dev-adopted-rm: dev kind envsubst ## Create adopted cluster deployment
	@if $(KIND) get clusters | grep -q "^$(KIND_CLUSTER_NAME)$$"; then \
		if [ -n "$(KIND_CONFIG_PATH)" ]; then \
			$(KIND) delete cluster -n $(KIND_CLUSTER_NAME) --config "$(KIND_CONFIG_PATH)"; \
		else \
			$(KIND) delete cluster -n $(KIND_CLUSTER_NAME); \
		fi \
	fi; \
	$(KUBECTL) delete clusterdeployment --ignore-not-found=true $(KIND_CLUSTER_NAME) -n $(KCM_NAMESPACE) || true

.PHONY: dev-adopted-deploy
dev-adopted-deploy: dev kind envsubst ## Create adopted cluster deployment
	make kind-deploy KIND_CONFIG_PATH="$(or $(KIND_CONFIG_PATH),config/kind-local.yaml)"
	$(KUBECTL) config use kind-kcm-dev
	NAMESPACE=$(KCM_NAMESPACE) \
	KUBECONFIG_DATA=$$($(KIND) get kubeconfig --internal -n $(KIND_CLUSTER_NAME) | base64 -w 0) \
	KIND_CLUSTER_NAME=$(KIND_CLUSTER_NAME) \
	$(ENVSUBST) -no-unset -i demo/creds/adopted-credentials.yaml \
	| $(KUBECTL) apply -f -
	@if [ -n "$(KCM_REGION_NAME)" ]; then \
		echo "Checking if region $(KCM_REGION_NAME) exists..."; \
		if $(KUBECTL) get region $(KCM_REGION_NAME) -n $(KCM_NAMESPACE) >/dev/null 2>&1; then \
			$(KUBECTL) patch credential child-adopted-cred \
				-n $(KCM_NAMESPACE) \
				--type=merge \
				-p "{\"spec\": {\"region\": \"$(KCM_REGION_NAME)\"}}"; \
		fi; \
	fi
	@$(KIND) load docker-image ghcr.io/k0rdent/kof/kof-opentelemetry-collector-contrib:v$(KOF_VERSION) --name $(KIND_CLUSTER_NAME)

.PHONY: dev-deploy
dev-deploy: dev ## Deploy KOF umbrella chart with local development configuration. Optional: HELM_CHART_NAME to deploy a specific subchart
	@if [ -z "$(HELM_CHART_NAME)" ] || [ "$(HELM_CHART_NAME)" = "kof-mothership" ]; then \
		echo "Building kof-operator docker image..."; \
		$(MAKE) kof-operator-docker-build; \
	fi
	cp -f $(TEMPLATES_DIR)/kof/values-local.yaml dev/values-local.yaml
	@if $(KUBECTL) get svctmpl -A | grep -q 'cert-manager'; then \
		echo "⚠️ ServiceTemplate cert-manager found"; \
		$(YQ) eval -i '.kof-mothership.values.cert-manager-service-template.enabled = false' dev/values-local.yaml; \
	fi
	@[ -f dev/dex.env ] && { \
		source dev/dex.env; \
		$(YQ) eval -i '.kof-mothership.values.dex.enabled = true' dev/values-local.yaml; \
		$(YQ) eval -i ".kof-mothership.values.dex.config.connectors[0].config.clientID = \"$${GOOGLE_CLIENT_ID}\"" dev/values-local.yaml; \
		$(YQ) eval -i ".kof-mothership.values.dex.config.connectors[0].config.clientSecret = \"$${GOOGLE_CLIENT_SECRET}\"" dev/values-local.yaml; \
		host_ip=$$(${CONTAINER_TOOL} inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "${KIND_CLUSTER_NAME}-control-plane"); \
		bash ./scripts/generate-dex-secret.bash; \
		bash ./scripts/patch-coredns.bash $(KUBECTL) "dex.example.com" "$$host_ip"; \
		$(KUBECTL) rollout restart -n kof deployment/kof-mothership-dex; \
	} || true
	@if [ "$(DISABLE_KOF_COLLECTORS)" = "true" ]; then \
		echo "⚠️ Disabling kof-collectors"; \
		$(YQ) eval -i '.kof-collectors.enabled = false' dev/values-local.yaml; \
	fi
	@if [ "$(DISABLE_KOF_STORAGE)" = "true" ]; then \
		echo "⚠️ Disabling kof-storage"; \
		$(YQ) eval -i '.kof-storage.enabled = false' dev/values-local.yaml; \
	fi
	@$(call set_local_registry, "dev/values-local.yaml")
	@if [ -n "$(HELM_CHART_NAME)" ]; then \
		echo "Deploying specific chart: $(HELM_CHART_NAME)"; \
		$(YQ) eval '.$(HELM_CHART_NAME).values' dev/values-local.yaml > dev/$(HELM_CHART_NAME)-values.yaml; \
		$(KUBECTL) patch helmrelease/$(HELM_CHART_NAME) -n kof --type='json' -p '[{"op": "replace", "path": "/spec/suspend", "value":true}]'; \
		$(HELM_UPGRADE) --take-ownership -n kof --create-namespace $(HELM_CHART_NAME) ./charts/$(HELM_CHART_NAME) -f dev/$(HELM_CHART_NAME)-values.yaml --set kcm.installTemplates=false; \
	else \
		$(HELM_UPGRADE) --take-ownership -n kof --create-namespace kof ./charts/kof -f dev/values-local.yaml; \
	fi
	@if [ -z "$(HELM_CHART_NAME)" ]; then \
		if [ "$(SKIP_WAIT)" != "true" ]; then \
			echo "Wait for helmreleases readiness ..."; \
			$(KUBECTL) wait --for=condition=Ready helmreleases --all -n kof --timeout=10m; \
		else \
			echo "⚠️ Skipping wait for helmreleases"; \
		fi; \
	fi
	@if [ -z "$(HELM_CHART_NAME)" ] || [ "$(HELM_CHART_NAME)" = "kof-mothership" ]; then \
		echo "Restarting kof-operator to pick up new image..."; \
		$(KUBECTL) rollout restart -n kof deployment/kof-mothership-kof-operator || true; \
	fi

.PHONY: dev-kcm-region-deploy-cloud
dev-kcm-region-deploy-cloud: dev ## Deploy kcm region cluster using k0rdent
	cp -f demo/cluster/$(CLOUD_CLUSTER_TEMPLATE)-kcm-region.yaml dev/$(CLOUD_CLUSTER_TEMPLATE)-kcm-region.yaml
	@$(YQ) eval -i '.metadata.name = "$(KCM_REGION_NAME)"' dev/$(CLOUD_CLUSTER_TEMPLATE)-kcm-region.yaml
	@$(YQ) eval -i '.metadata.labels["k0rdent.mirantis.com/kof-cluster-name"] = "$(KCM_REGION_NAME)"' dev/$(CLOUD_CLUSTER_TEMPLATE)-kcm-region.yaml
	@$(call set_region, "dev/$(CLOUD_CLUSTER_TEMPLATE)-kcm-region.yaml")
	$(KUBECTL) apply -f dev/$(CLOUD_CLUSTER_TEMPLATE)-kcm-region.yaml

.PHONY: dev-kcm-region-deploy-adopted
dev-kcm-region-deploy-adopted: dev ## Deploy adopted kcm region cluster using k0rdent
	cp -f demo/cluster/adopted-cluster-kcm-region.yaml dev/adopted-cluster-kcm-region.yaml
	@$(YQ) eval -i '.metadata.name = "$(KCM_REGION_NAME)"' dev/adopted-cluster-kcm-region.yaml
	@$(YQ) eval -i '.spec.config.clusterAnnotations["k0rdent.mirantis.com/kof-regional-domain"] = "$(KCM_REGION_NAME)"' dev/adopted-cluster-kcm-region.yaml
	@$(YQ) eval -i '.spec.config.clusterAnnotations["k0rdent.mirantis.com/kof-cert-email"] = "$(USER_EMAIL)"' dev/adopted-cluster-kcm-region.yaml
	@$(YQ) eval -i '.metadata.namespace = "$(KCM_NAMESPACE)"' dev/adopted-cluster-kcm-region.yaml
	$(KUBECTL) apply -f dev/adopted-cluster-kcm-region.yaml
	cp -f demo/kcm-region/region.yaml dev/region.yaml
	@$(YQ) eval -i '.metadata.name = "$(KCM_REGION_NAME)"' dev/region.yaml
	@$(YQ) eval -i '.spec.kubeConfig.name = "$(KCM_REGION_NAME)-kubeconf"' dev/region.yaml
	$(KUBECTL) apply -f dev/region.yaml
	./scripts/wait-helm-charts.bash $(HELM) $(YQ) kind-regional-adopted "kcm-regional cert-manager ingress-nginx" "kof-operators kof-storage kof-collectors"

.PHONY: dev-istio-kcm-region-deploy-adopted
dev-istio-kcm-region-deploy-adopted: dev ## Deploy adopted kcm region cluster using k0rdent
	cp -f demo/cluster/adopted-cluster-istio-kcm-region.yaml dev/adopted-istio-cluster-kcm-region.yaml
	@$(YQ) eval -i '.metadata.name = "$(KCM_REGION_NAME)"' dev/adopted-istio-cluster-kcm-region.yaml
	@$(YQ) eval -i '.metadata.namespace = "$(KCM_NAMESPACE)"' dev/adopted-istio-cluster-kcm-region.yaml
	@$(YQ) eval -i '.metadata.labels["k0rdent.mirantis.com/istio-mesh"] = "$(ISTIO_MESH)"' dev/adopted-istio-cluster-kcm-region.yaml;
	$(KUBECTL) apply -f dev/adopted-istio-cluster-kcm-region.yaml
	cp -f demo/kcm-region/region.yaml dev/region.yaml
	@$(YQ) eval -i '.metadata.name = "$(KCM_REGION_NAME)"' dev/region.yaml
	@$(YQ) eval -i '.spec.kubeConfig.name = "$(KCM_REGION_NAME)-kubeconf"' dev/region.yaml
	$(KUBECTL) apply -f dev/region.yaml
	./scripts/wait-helm-charts.bash $(HELM) $(YQ) kind-regional-adopted "kcm-regional k0rdent-istio istio-gateway cert-manager" "kof-operators kof-storage kof-collectors"

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
	@$(YQ) eval -i '.metadata.namespace = "$(KCM_NAMESPACE)"' dev/adopted-cluster-regional.yaml
	$(KUBECTL) apply -f dev/adopted-cluster-regional.yaml
	./scripts/wait-helm-charts.bash $(HELM) $(YQ) kind-regional-adopted "cert-manager ingress-nginx" "kof-operators kof-storage kof-collectors"

.PHONY: dev-istio-regional-deploy-adopted
dev-istio-regional-deploy-adopted: dev ## Deploy regional adopted cluster with istio using k0rdent
	cp -f demo/cluster/adopted-cluster-istio-regional.yaml dev/adopted-cluster-istio-regional.yaml
	@$(YQ) eval -i '.spec.config.clusterAnnotations["k0rdent.mirantis.com/kof-storage-values"] = "{\"victoria-logs-cluster\":{\"vlinsert\":{\"replicaCount\":1},\"vlselect\":{\"replicaCount\":1},\"vlstorage\":{\"replicaCount\":1}},\"victoriametrics\":{\"vmcluster\":{\"spec\":{\"replicationFactor\":1,\"vminsert\":{\"replicaCount\":1},\"vmselect\":{\"replicaCount\":1},\"vmstorage\":{\"replicaCount\":1}}}}}"' dev/adopted-cluster-istio-regional.yaml
	$(KUBECTL) apply -f dev/adopted-cluster-istio-regional.yaml
	./scripts/wait-helm-charts.bash $(HELM) $(YQ) kind-regional-adopted "cert-manager k0rdent-istio istio-gateway" "kof-operators kof-storage kof-collectors"

.PHONY: dev-child-deploy-adopted
dev-child-deploy-adopted: dev ## Deploy regional adopted cluster using k0rdent
	cp -f demo/cluster/adopted-cluster-child.yaml dev/adopted-cluster-child.yaml
	$(KUBECTL) apply -f dev/adopted-cluster-child.yaml
	./scripts/wait-helm-charts.bash $(HELM) $(YQ) kind-child-adopted "cert-manager" "kof-operators kof-collectors"

.PHONY: dev-istio-child-deploy-adopted
dev-istio-child-deploy-adopted: dev ## Deploy regional adopted cluster using k0rdent
	cp -f demo/cluster/adopted-cluster-istio-child.yaml dev/adopted-cluster-istio-child.yaml
	@if [ -n "$(ISTIO_MESH)" ]; then \
		$(YQ) eval -i '.metadata.labels["k0rdent.mirantis.com/istio-mesh"] = "$(ISTIO_MESH)"' dev/adopted-cluster-istio-child.yaml; \
	fi
	$(KUBECTL) apply -f dev/adopted-cluster-istio-child.yaml
	./scripts/wait-helm-charts.bash $(HELM) $(YQ) kind-child-adopted "cert-manager" "kof-operators kof-collectors"

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
YQ_VERSION ?= v4.45.1
KIND_VERSION ?= v0.29.0
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

.PHONY: cli-install
cli-install: yq helm kind ## Install the necessary CLI tools for deployment, development and testing.

.PHONY: support-bundle
support-bundle: SUPPORT_BUNDLE_OUTPUT=$(CURDIR)/support-bundle-$(shell date +"%Y-%m-%dT%H_%M_%S")
support-bundle: envsubst support-bundle-cli
	@if [ -n "$(KUBECTL_CONTEXT)" ]; then \
		NAMESPACE=$(NAMESPACE) $(ENVSUBST) -no-unset -i config/support-bundle.yaml | $(SUPPORT_BUNDLE_CLI) -o $(SUPPORT_BUNDLE_OUTPUT) --context $(KUBECTL_CONTEXT) --debug - ; \
	else \
		NAMESPACE=$(NAMESPACE) $(ENVSUBST) -no-unset -i config/support-bundle.yaml | $(SUPPORT_BUNDLE_CLI) -o $(SUPPORT_BUNDLE_OUTPUT) --debug - ; \
	fi
	@archive=""; \
	if [ -f "$(SUPPORT_BUNDLE_OUTPUT).tar.gz" ]; then archive="$(SUPPORT_BUNDLE_OUTPUT).tar.gz"; \
	elif [ -f "$(SUPPORT_BUNDLE_OUTPUT)" ]; then archive="$(SUPPORT_BUNDLE_OUTPUT)"; \
	else archive=$$(ls -t support-bundle-*.tar.gz 2>/dev/null | head -n 1); fi; \
	if [ -z "$$archive" ]; then echo "ERROR: support bundle archive not found" >&2; exit 2; fi; \
	echo "Analyzing support bundle at: $$archive"; \
	python3 scripts/support-bundle-analyzer.py "$$archive" --details --output auto

.PHONY: wait-otel-collectors
wait-otel-collectors:
	@set -euo pipefail; \
	ns="kof"; timeout="5m"; \
	wait_one() { \
		c="$$1"; want="$$2"; \
		echo "Wait create: $$ns/$$c"; \
		kubectl -n "$$ns" wait --for=create "opentelemetrycollector/$$c" --timeout="$$timeout"; \
		echo "Wait ready:  $$ns/$$c statusReplicas=$$want"; \
		kubectl -n "$$ns" wait --for=jsonpath='{.status.scale.statusReplicas}'="$$want" "opentelemetrycollector/$$c" --timeout="$$timeout"; \
	}; \
	wait_one kof-collectors-cluster-stats 1/1; \
	wait_one kof-collectors-controller-k0s-daemon 1/1; \
	wait_one kof-collectors-ta-daemon 1/1; \
	wait_one kof-collectors-daemon 2/2

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
