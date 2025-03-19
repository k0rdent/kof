# GrafanaDatasource

This is a partial copy of https://github.com/grafana/grafana-operator/tree/master/api
required to use `grafanav1beta1.GrafanaDatasource` in `kof-operator`.

## Why

* https://github.com/grafana/grafana-operator/blob/master/api/go.mod says:
  `module github.com/grafana/grafana-operator/v5/api`

* `go get github.com/grafana/grafana-operator/v5/api` fails with:
  ```
  go: module github.com/grafana/grafana-operator/v5@upgrade found (v5.17.0),
  but does not contain package github.com/grafana/grafana-operator/v5/api
  ```

* `go get github.com/grafana/grafana-operator/v5` fails with:
  ```
  go: downloading github.com/grafana/grafana-operator/v5/api v0.0.0-00010101000000-000000000000
  go: github.com/grafana/grafana-operator/v5 imports
    github.com/grafana/grafana-operator/v5/api/v1beta1: github.com/grafana/grafana-operator/v5/api@v0.0.0-00010101000000-000000000000: invalid version: unknown revision 000000000000
  go: github.com/grafana/grafana-operator/v5 imports
    github.com/grafana/grafana-operator/v5/controllers imports
    github.com/grafana/grafana-operator/v5/api: github.com/grafana/grafana-operator/v5/api@v0.0.0-00010101000000-000000000000: invalid version: unknown revision 000000000000
  ```

* https://github.com/grafana/grafana-operator/blob/master/go.mod says:
  ```
  require github.com/grafana/grafana-operator/v5/api v0.0.0-00010101000000-000000000000
  replace github.com/grafana/grafana-operator/v5/api => ./api
  ```

* Looks like this module isn't published in a "ready‐to‐use" form for external projects.

## Workaround

This workaround is applied to kof repo already.

Repeat these steps if we need an updated version.

```bash
cd kof-operator
go mod edit -replace=github.com/grafana/grafana-operator/v5/api=./pkg/grafana-operator/api

mkdir -p pkg/grafana-operator && cd $_
git clone -b v5.17.0 --depth 1 https://github.com/grafana/grafana-operator.git
mkdir -p api/v1beta1
mv grafana-operator/api/v1beta1/{grafanadatasource_types.go,common.go,plugin_list.go,groupversion_info.go} api/v1beta1/
mv grafana-operator/{go.mod,go.sum} api/
rm -R grafana-operator

cd api/v1beta1
# Delete all except `package` and `type` from `plugin_list.go`:
awk '
  /^package / { print; next }
  /^type / { print; inBlock=1; next }
  inBlock && /^[\t}]/ { print; next }
  { inBlock=0 }
' plugin_list.go > tmp
mv tmp plugin_list.go

cd .. && pwd # api
go mod edit -dropreplace=github.com/openshift/api
go mod edit -replace=github.com/grafana/grafana-operator/v5/api=./
go mod tidy

cd ../../.. && pwd # kof-operator
go mod tidy
make test
```
