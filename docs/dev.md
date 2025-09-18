# Development

## kcm

* [Apply kcm dev docs](https://github.com/k0rdent/kcm/blob/main/docs/dev.md)
  or just run:
  ```bash
  git clone https://github.com/k0rdent/kcm.git
  cd kcm
  make cli-install
  make dev-apply
  ```

## kof

* Fork https://github.com/k0rdent/kof to `https://github.com/YOUR_USERNAME/kof`
* Run:
  ```bash
  cd ..
  git clone git@github.com:YOUR_USERNAME/kof.git
  cd kof

  make cli-install
  make registry-deploy
  make helm-push
  ```

* To use [Istio servicemesh](./istio.md):
  ```bash
  kubectl create namespace kof
  kubectl label namespace kof istio-injection=enabled
  make dev-istio-deploy
  ```

* Deploy CRDs required for `kof-mothership`:
  ```bash
  make dev-operators-deploy
  ```

* Deploy `kof-mothership` chart to local management cluster:
  ```bash
  make dev-ms-deploy
  ```

* Wait for all pods to became `Running`:
  ```bash
  kubectl get pod -n kof
  ```

## Upgrade instructions

* If you're upgrading from KOF version less than `1.1.0`, please run after upgrade:
  ```shell
  kubectl apply --server-side --force-conflicts \
  -f https://github.com/grafana/grafana-operator/releases/download/v5.18.0/crds.yaml
  ```
  And run the same for each regional cluster:
  ```shell
  kubectl get secret -n kcm-system $REGIONAL_CLUSTER_NAME-kubeconfig \
    -o=jsonpath={.data.value} | base64 -d > regional-kubeconfig

  KUBECONFIG=regional-kubeconfig kubectl apply --server-side --force-conflicts \
  -f https://github.com/grafana/grafana-operator/releases/download/v5.18.0/crds.yaml
  ```
  This is required by [grafana-operator release notes](https://github.com/grafana/grafana-operator/releases/tag/v5.18.0).

## Local deployment

Quick option without regional/child clusters.

* Run:
  ```bash
  make dev-storage-deploy
  make dev-collectors-deploy
  ```

* Apply [Grafana](https://docs.k0rdent.io/next/admin/kof/kof-using/#access-to-grafana) section.

## Deployment to AWS

This is a full-featured option.

* `export AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`
  as documented in the [kcm dev docs for AWS](https://github.com/k0rdent/kcm/blob/main/docs/dev.md#aws-provider-setup)
  and run:
  ```bash
  cd ../kcm
  make dev-creds-apply
  cd ../kof
  ```

### Without Istio servicemesh

* Apply [DNS auto-config](https://docs.k0rdent.io/next/admin/kof/kof-install/#dns-auto-config) and run:
  ```bash
  export KOF_DNS=kof.example.com
  ```

* Deploy regional and child clusters to AWS:
  ```bash
  make dev-regional-deploy-cloud
  make dev-child-deploy-cloud
  ```

### With Istio servicemesh

* Change the cluster name and apply the istio clusterdeployments from demo

  ```bash
  kubectl apply -f demo/cluster/aws-standalone-istio-regional.yaml
  kubectl apply -f demo/cluster/aws-standalone-istio-child.yaml
  ```

### Verification

* To verify, run:
  ```bash
  REGIONAL_CLUSTER_NAME=$USER-aws-standalone-regional
  CHILD_CLUSTER_NAME=$USER-aws-standalone-child

  clusterctl describe cluster --show-conditions all -n kcm-system $REGIONAL_CLUSTER_NAME
  clusterctl describe cluster --show-conditions all -n kcm-system $CHILD_CLUSTER_NAME
  ```
  wait for all `READY` to become `True` and then apply:
  * [Verification](https://docs.k0rdent.io/next/admin/kof/kof-verification/)
  * [Grafana](https://docs.k0rdent.io/next/admin/kof/kof-using/#access-to-grafana)
  * [Jaeger](https://docs.k0rdent.io/next/admin/kof/kof-using/#access-to-jaeger)

### Uninstall

* `cd dev` and apply [Uninstallation](https://docs.k0rdent.io/next/admin/kof/kof-maintainence/#uninstallation).
* `cd ../kcm && make dev-destroy`

## Deployment to Azure

* Ensure your kcm repo has https://github.com/k0rdent/kcm/pull/1334 applied.

* Export all `AZURE_` variables as documented in the [kcm dev docs for Azure](https://github.com/k0rdent/kcm/blob/main/docs/dev.md#azure-provider-setup)
  and run:
  ```bash
  cd ../kcm
  make dev-azure-creds
  cd ../kof
  ```

* Deploy regional and child clusters to Azure:
  ```bash
  export CLOUD_CLUSTER_TEMPLATE=azure-standalone
  export CLOUD_CLUSTER_REGION=westus3
  make dev-regional-deploy-cloud
  make dev-child-deploy-cloud
  ```

* [Verification](#verification) and [Uninstall](#uninstall) are similar,
  just replace `aws` with `azure`.

* Please apply the [Verification](#verification) now,
  and then the [Manual DNS config](https://docs.k0rdent.io/next/admin/kof/kof-verification/#manual-dns-config),
  because we keep the Azure version of [DNS auto-config](https://docs.k0rdent.io/next/admin/kof/kof-install/#dns-auto-config)
  as an optional customization for now.

## Adopted local kind cluster

This method does not help when you need a real cluster, but may help with other cases.

* Create a regional adopted kind cluster for quick dev/test iterations:
  ```bash
  make dev-adopted-deploy KIND_CLUSTER_NAME=regional-adopted
  ```

* Run kind cloud provider to support external IP allocation for ingress services
  ```bash
  docker run --rm --network kind -v /var/run/docker.sock:/var/run/docker.sock registry.k8s.io/cloud-provider-kind/cloud-controller-manager:v0.7.0
  ```

* Create regional adopted cluster deployment either with or without Istio:
  ```bash
  make dev-istio-regional-deploy-adopted
  # or
  make dev-regional-deploy-adopted
  ```
* Inspect the regional adopted cluster:
  ```bash
  kubectl --context=kind-regional-adopted get pod -A
  ```

* Either create child adopted cluster without Istio:
  ```bash
  make dev-adopted-deploy KIND_CLUSTER_NAME=child-adopted
  make dev-coredns
  make dev-child-deploy-adopted
  kubectl --context=kind-child-adopted get pod -A
  ```

* ...or with Istio:
  ```bash
  make dev-adopted-deploy KIND_CLUSTER_NAME=child-adopted
  make dev-istio-child-deploy-adopted
  kubectl --context=kind-child-adopted get pod -A
  ```

## See also

* [Options to collect data from DEV management cluster](collect-from-management.md).
* Helm docs: apply the steps in [.pre-commit-config.yaml](../.pre-commit-config.yaml) file.
