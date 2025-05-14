#!/bin/bash -eu

WORKDIR=$(git rev-parse --show-toplevel)

SED_IS_GNU=$(sed --version | grep GNU)
rc=$?
if [ ${rc} -ne 0 ]
then
  echo "this script requires GNU sed"
  exit 1
fi

rm -rf .build
mkdir -p .build/kps-dashboards
mkdir -p .build/kps-recording
mkdir -p .build/kps-alerts
mkdir -p .build/kps-upstream
pushd .build
pushd kps-upstream
git clone https://github.com/prometheus-community/helm-charts.git kps-helm-charts --depth=1
pushd kps-helm-charts
pushd charts/kube-prometheus-stack/
KPS_UPSTREAM_DIR=$(pwd)
helm dependency build
python3 hack/sync_grafana_dashboards.py
helm template -n kof-storage --release-name kps-dashboards --set grafana.sidecar.dashboards.multicluster.global.enabled=true . | yq -r 'select(.kind == "ConfigMap" and .metadata.labels.grafana_dashboard == "1" ) | .data | to_entries[]' | python3 ${WORKDIR}/scripts/deserialize_dashboards.py ${WORKDIR}/.build/kps-dashboards
popd
popd
popd
pushd kps-dashboards
for i in *.json; do yq -P -oy '.title |= "KPS / " + .' ${i} >kps-"$(basename $i .json)".yaml; done
popd
pushd kps-recording
ln -s ${KPS_UPSTREAM_DIR}/charts ./charts
helm template --release-name 'kps' -n kof-storage ${KPS_UPSTREAM_DIR} | yq 'select(.kind == "PrometheusRule")' | yq eval 'del(.. | select(has("alert"))) | select(.spec.groups[].rules | length > 0)' | yq -s '.metadata.name'
rm charts
popd
pushd kps-alerts
ln -s ${KPS_UPSTREAM_DIR}/charts ./charts
helm template --release-name 'kps' -n kof ${KPS_UPSTREAM_DIR} | yq 'select(.kind == "PrometheusRule")' | yq eval 'del(.. | select(has("record"))) | select(.spec.groups[].rules | length > 0)' | yq -s '.metadata.name'
rm charts
popd
popd
cp .build/kps-dashboards/*.yaml  charts/kof-storage/files/dashboards/
cp .build/kps-recording/kps-* charts/kof-storage/files/rules/
cp .build/kps-alerts/kps-* charts/kof-mothership/files/rules/
for tdir in charts/kof-storage/files/dashboards charts/kof-storage/files/rules charts/kof-mothership/files/rules; do
  mkdir -p ${tdir}
  rm -f ${tdir}/null.yml
  (find ${tdir} -name '*.yml' && ls ${tdir}/*.yml | sed -e 'p;s[\.yml[.yaml[' | xargs -n2 mv) || :
  (find ${tdir} -name '*.rules' && ls ${tdir}/*.rules | sed -e 'p;s/$/.yaml/' | xargs -n2 mv) || :
  sed -i -e '1 i\\{\{\`' "${tdir}"/kps-*
  sed -i -e '$a \`\}\}' "${tdir}"/kps-*
done
pushd charts/kof-mothership/files/dashboards
for i in ../../../../charts/kof-storage/files/dashboards/*.yaml; do if ! [ -f $(basename ${i}) ]; then ln -s ${i} ./; fi; done
popd
