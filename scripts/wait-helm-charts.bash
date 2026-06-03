#!/usr/bin/env bash

set -euo pipefail

HELM="${1:-helm}"
YQ="${2:-yq}"
CONTEXT="$3"
DEPLOYED_CHARTS="$4"
UPGRADED_CHARTS="${5:-}"
# Optional: explicit expected app_version for UPGRADED_CHARTS.
# When provided, used instead of reading appVersion from local charts/*/Chart.yaml.
# Useful in CI where charts are installed from a remote registry and the local
# Chart.yaml version may differ from the installed one.
EXPECTED_VERSION="${6:-}"

for attempt in $(seq 1 20); do
  deployed="true"
  for name in $DEPLOYED_CHARTS $UPGRADED_CHARTS; do
    helm_status=$($HELM list --kube-context "$CONTEXT" -A --deployed --pending -o yaml | $YQ ".[] | select(.name == \"$name\") | .status")
    if [ ! "$helm_status" = "deployed" ]; then
      echo "$attempt: Waiting for the $name helm chart status to be deployed. Current: [$helm_status]"
      sleep 30
      deployed="false"
      break
    fi

    echo $UPGRADED_CHARTS | tr " " "\n" | grep -qx "$name" || continue
    if [ -n "$EXPECTED_VERSION" ]; then
      expected_version="$EXPECTED_VERSION"
    else
      expected_version=$($YQ .appVersion "charts/$name/Chart.yaml")
    fi
    actual_version=$(
      $HELM list --kube-context "$CONTEXT" -A -o yaml \
      | $YQ ".[] | select(.name == \"$name\") | .app_version"
    )
    if [[ "$expected_version" != "$actual_version" ]]; then
      echo "$attempt: Waiting for the $name helm chart to be upgraded to $expected_version. Current: $actual_version"
      sleep 30
      deployed="false"
      break
    fi
  done
  if [ "$deployed" = "true" ]; then break; fi
done

if [ "$deployed" = "false" ]; then
  $HELM list --kube-context "$CONTEXT" -A
  echo "Timout waiting for helm charts deployment"
  exit 1
fi
