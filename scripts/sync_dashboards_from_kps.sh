#!/bin/bash -eu

WORKDIR=`git rev-parse --show-toplevel`

rm -rf .build
mkdir -p .build/kps-dashboards
mkdir -p .build/kps-upstream
pushd .build
pushd kps-upstream
git clone https://github.com/prometheus-community/helm-charts.git kps-helm-charts
pushd kps-helm-charts
pushd charts/kube-prometheus-stack/
helm dependency build
python3 hack/sync_grafana_dashboards.py
helm template -n kof-storage --release-name kps-dashboards --set grafana.sidecar.dashboards.multicluster.global.enabled=true . | yq -r 'select(.kind == "ConfigMap" and .metadata.labels.grafana_dashboard == "1" ) | .data | to_entries[]' | python3 ${WORKDIR}/scripts/deserialize_dashboards.py ${WORKDIR}/.build/kps-dashboards
popd
popd
popd
pushd kps-dashboards
for i in *.json; do yq -P -oy '.title |= "KPS / " + .' ${i} > kps-$(basename $i .json).yaml; done
popd
popd
cp .build/kps-dashboards/*.yaml charts/kof-storage/files/dashboards/
sed -i -e '1 i\\{\{\`' charts/kof-storage/files/dashboards/kps-*.yaml
sed -i -e '$a \`\}\}' charts/kof-storage/files/dashboards/kps-*.yaml
pushd charts/kof-mothership/files/dashboards
for i in ../../../../charts/kof-storage/files/dashboards/*.yaml; do if ! [ -f $(basename ${i}) ]; then ln -s ${i} ./; fi; done
popd

