# Development

## kcm

[Apply kcm dev docs](https://github.com/k0rdent/kcm/blob/main/docs/dev.md) or just run:

```bash
git clone https://github.com/k0rdent/kcm.git
cd kcm

# Downgrade Sveltos to avoid `server gave HTTP response to HTTPS client` for `kcm-local-registry`:
yq -i '.dependencies[0].version = "0.45.0"' templates/provider/projectsveltos/Chart.yaml

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
make helm-push
make dev-operators-deploy
```

## Local deployment

Quick option without regional/child clusters.

```bash
make dev-ms-deploy-cloud
make dev-storage-deploy
make dev-collectors-deploy
```

Apply [Grafana](https://docs.k0rdent.io/head/admin-kof/#grafana) section.

## Deployment to AWS

This is a full-featured option.

`export AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`
as in [kcm dev docs](https://github.com/k0rdent/kcm/blob/main/docs/dev.md#aws-provider-setup)
and run:

```bash
cd ../kcm
make dev-creds-apply
cd ../kof
```

Apply [DNS auto-config](https://docs.k0rdent.io/head/admin-kof/#dns-auto-config) and run:

```bash
export KOF_DNS=kof.example.com
```

Deploy `kof-mothership` chart to local management cluster:

```bash
make dev-ms-deploy-cloud
kubectl get pod -n kof
```

Wait for all pods to show that they're `Running`.

Deploy regional and child clusters to AWS:

```bash
make dev-regional-deploy-cloud
make dev-child-deploy-cloud
```

To verify, run:

```bash
REGIONAL_CLUSTER_NAME=$USER-aws-standalone-regional
CHILD_CLUSTER_NAME=$USER-aws-standalone-child

clusterctl describe cluster --show-conditions all -n kcm-system $REGIONAL_CLUSTER_NAME
clusterctl describe cluster --show-conditions all -n kcm-system $CHILD_CLUSTER_NAME
```

...and apply these sections:
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

cd kcm && make dev-destroy
```
