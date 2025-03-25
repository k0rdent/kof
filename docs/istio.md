# Istio servicemesh

To secure connectivity between clusters it is possible to install [Istio servicemesh](https://istio.io) to all KOF clusters using [kof-istio](../charts/kof-istio/) helm chart.

## Multicluster architecture

The reference architecture for Istio deployment is [Multi-Primary on different networks](https://istio.io/latest/docs/setup/install/multicluster/multi-primary_multi-network)

Components deployed:

* Self-signed Istio Root CA on Mothership cluster. [Reference architecture](https://istio.io/latest/docs/tasks/security/cert-management/plugin-ca-cert/)
* Intermediate CA generated for each Istio cluster (labeled with `k0rdent.mirantis.com/istio-role: child`) by kof-operator
* Remote secret created for each Istio Regional cluster (labeled with `k0rdent.mirantis.com/kof-cluster-role: regional`) by kof-operator
* Istio Gateway is installed in Istio Regional cluster for endpoint connectivity protected by [mTLS](https://istio.io/latest/docs/tasks/security/authentication/authn-policy/#enable-mutual-tls-per-workload)

## Istio observability

Istio clusters have [observability](https://istio.io/latest/docs/concepts/observability/) enabled with metrics and traces
