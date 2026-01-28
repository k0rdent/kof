#!/usr/bin/env bash

set -euo pipefail

NAMESPACE_LIST="kof"
CRD_LIST="grafana victoria opentelemetry"
HELM_CHARTS_LIST="kof-collectors kof-storage kof-child kof-regional kof-mothership kof-operators"

# Uninstall Helm charts
for chart in $HELM_CHARTS_LIST; do
  echo "Uninstalling Helm chart: $chart"
  helm uninstall $chart -n kof --wait --timeout 1m0s >/dev/null || true
done

# Remove finalizers from namespaced resources
for ns in $NAMESPACE_LIST; do
  echo "Removing finalizers from resources in namespace: $ns"
  for resource in $(kubectl api-resources --verbs=list --namespaced -o name); do
    if [[ "$resource" == "events" || "$resource" == "events.events.k8s.io" ]]; then
      continue
    fi
    for item in $(kubectl get $resource -n $ns -o name 2>/dev/null); do
      echo "  Deleting $item finalizers"
      kubectl patch $item -n $ns -p '{"metadata":{"finalizers":null}}' --type=merge >/dev/null || true
    done
  done
done

# Remove finalizers from CRDs
for pattern in $CRD_LIST; do
  crds=$(kubectl get crd -o name 2>/dev/null | grep "$pattern" || true)
  if [[ -z "$crds" ]]; then
    echo "No CRDs found matching pattern '$pattern'"
    continue
  fi
  for crd in $crds; do
    echo "Deleting $crd finalizers"
    kubectl patch "$crd" -p '{"metadata":{"finalizers":null}}' --type=merge >/dev/null || true
  done
done

# Delete namespaces
for ns in $NAMESPACE_LIST; do
  echo "Deleting namespace: $ns"
  kubectl delete ns $ns --wait --timeout=1m0s --cascade=foreground || true
  kubectl delete ns $ns --grace-period=0 --force >/dev/null || true
done
