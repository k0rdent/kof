#!/usr/bin/env bash

set -euo pipefail

HELM="${1:-helm}"
YQ="${2:-yq}"
CONTEXT="$3"
CHARTS="$4"

for attempt in $(seq 1 20); do
  deployed="true"
  for name in $CHARTS ; do
    helm_status=$($HELM list --kube-context "$CONTEXT" -A --deployed --pending -o yaml | $YQ ".[] | select(.name == \"$name\") | .status")
    if [ ! "$helm_status" = "deployed" ]; then
      echo "$attempt: Waiting for the $name helm chart status to be deployed. Current: [$helm_status]"
      sleep 20
      deployed="false"
      break
    fi

    chart_file="charts/$name/Chart.yaml"
    test -f "$chart_file" || continue
    expected_version=$($YQ .appVersion "$chart_file")
    actual_version=$(
      $HELM list --kube-context "$CONTEXT" -A -o yaml \
      | $YQ ".[] | select(.name == \"$name\") | .app_version"
    )
    if [[ "$expected_version" != "$actual_version" ]]; then
      echo "Upgrade of \"$name\" chart failed:" \
        "expected_version=$expected_version," \
        "actual_version=$actual_version"
      exit 1
    fi
  done
  if [ "$deployed" = "true" ]; then break; fi
done
if [ "$deployed" = "false" ]; then echo "Timout waiting for helm charts deployment"; exit 1; fi
