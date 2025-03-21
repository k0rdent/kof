# Development

## kcm

[Apply kcm dev docs](https://github.com/k0rdent/kcm/blob/main/docs/dev.md) or just run:

```bash
git clone https://github.com/k0rdent/kcm.git
cd kcm

make cli-install
make dev-apply
```

## kof

Fork https://github.com/k0rdent/kof to `https://github.com/YOUR_USERNAME/kof` and run:

```bash
cd ..
git clone git@github.com:YOUR_USERNAME/kof.git
cd kof

make cli-install
make registry-deploy
make helm-push
```

To use Istio servicemesh:

```bash
kubectl create namespace kof
kubectl label namespace kof istio-injection=enabled
```

```bash
make dev-operators-deploy
```

* Deploy `kof-mothership` chart to local management cluster:
```bash
make dev-ms-deploy
```

* If it fails with `the template is not valid` and no more details,
  ensure all templates became `VALID`:
  ```bash
  kubectl get clustertmpl -A
  kubectl get svctmpl -A
  ```
  and then retry.


To use Istio servicemesh install helm chart and re-start all pods in kof namespace
```bash
make dev-istio-deploy
kubectl delete pod --all -n kof
```

* Wait for all pods to show that they're `Running`:
```bash
kubectl get pod -n kof
```

## Local deployment

Quick option without regional/child clusters.


```bash
make dev-storage-deploy
make dev-collectors-deploy
```

Apply [Grafana](https://docs.k0rdent.io/head/admin-kof/#grafana) section.

## Deployment to AWS

This is a full-featured option.

* `export AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`
  as in [kcm dev docs](https://github.com/k0rdent/kcm/blob/main/docs/dev.md#aws-provider-setup)
  and run:
  ```bash
  cd ../kcm
  make dev-creds-apply
  cd ../kof
  ```

### Without Istio servicemesh

* Apply [DNS auto-config](https://docs.k0rdent.io/head/admin-kof/#dns-auto-config) and run:
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
kubectl apply -f demo/aws-standalone-istio-regional.yaml
kubectl apply -f demo/aws-standalone-istio-child.yaml
```

* To verify, run:
  ```bash
  REGIONAL_CLUSTER_NAME=$USER-aws-standalone-regional
  CHILD_CLUSTER_NAME=$USER-aws-standalone-child

  clusterctl describe cluster --show-conditions all -n kcm-system $REGIONAL_CLUSTER_NAME
  clusterctl describe cluster --show-conditions all -n kcm-system $CHILD_CLUSTER_NAME
  ```
  wait for all `READY` to become `True` and then apply:
  * [Verification](https://docs.k0rdent.io/head/admin-kof/#verification)
  * [Sveltos](https://docs.k0rdent.io/head/admin-kof/#sveltos)
  * [Grafana](https://docs.k0rdent.io/head/admin-kof/#grafana)

## Uninstall

```bash
kubectl delete --wait --cascade=foreground -f dev/aws-standalone-child.yaml && \
kubectl delete --wait --cascade=foreground -f dev/aws-standalone-regional.yaml && \
helm uninstall --wait --cascade foreground -n kof kof-mothership && \
helm uninstall --wait --cascade foreground -n kof kof-operators && \
kubectl delete namespace kof --wait --cascade=foreground

cd ../kcm && make dev-destroy
```

## Adopted local cluster

* For quick dev/test iterations, update the related `demo/cluster/` file to use:
  ```
    credential: adopted-cluster-cred
    template: adopted-cluster-0-1-0
  ```

* Run this to create the `adopted-cluster-cred`
  and to verify the version of the `template`:
  ```bash
  cd ../kcm
  kind create cluster -n adopted
  kubectl config use kind-kcm-dev
  KUBECONFIG_DATA=$(kind get kubeconfig --internal -n adopted | base64) make dev-adopted-creds
  kubectl get clustertemplate -n kcm-system | grep adopted
  ```

* Use `kubectl --context=kind-adopted` to inspect the cluster.
