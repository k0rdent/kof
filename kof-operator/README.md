# kof-operator

[Promxy](https://github.com/jacksontj/promxy) configuration is a list of serverGroup pointed to provisioned storage clusters.

This operator dynamically builds and reloads the promxy configuration as `PromxyServerGroup` custom resources are deployed along with storage cluster.

## Description

This is not a generic kof-operator, but rather an automation workaround for KOF as the promxy-config [template](internal/controller/template/secret.tmpl) is limited to KOF implementation so far.

## Getting Started

### Prerequisites
- go version v1.24.0+
- node version v18.18.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### Build and Deploy on the cluster for Development

From the repo root makefile

```sh
make dev-ms-deploy
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete PromxyServerGroup <name> -n <namespace>
```

**Delete the Mothership helm chart from the cluster:**


```sh
helm del kof-mothership -n kof

```

## Project Distribution

Github docker repo releases: ghcr.io/k0rdent/kof/kof-operator-controller:<tag>


## Contributing

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

