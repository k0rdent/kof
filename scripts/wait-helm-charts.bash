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
  done
  if [ "$deployed" = "true" ]; then break; fi
done
if [ "$deployed" = "false" ]; then echo "Timout waiting for helm charts deployment"; exit 1; fi
