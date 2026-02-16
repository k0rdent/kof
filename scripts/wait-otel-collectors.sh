#!/usr/bin/env bash
set -euo pipefail

lines="$(kubectl get opentelemetrycollector -A -o jsonpath='{range .items[*]}{.metadata.namespace}{" "}{.metadata.name}{"\n"}{end}')"
[[ -z "$lines" ]] && { echo "ERROR: no OpenTelemetryCollector resources found" >&2; exit 1; }

while IFS= read -r line; do
  [[ -z "$line" ]] && continue
  ns="${line%% *}"
  name="${line#* }"

  sel="$(kubectl -n "$ns" get opentelemetrycollector "$name" -o jsonpath='{.status.scale.selector}')"
  [[ -z "$sel" ]] && { echo "ERROR: empty selector for $ns/$name" >&2; exit 1; }

  kubectl -n "$ns" wait --for=condition=Ready pod -l "$sel" --timeout=600s
done <<< "$lines"
