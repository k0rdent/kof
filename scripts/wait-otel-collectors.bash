#!/usr/bin/env bash
set -euo pipefail

ns="${NAMESPACE}"
timeout="${OTEL_WAIT_TIMEOUT}"
kctx="${KUBECTL_CONTEXT:-}"
kubectl_cmd="${KUBECTL}"

if [ -n "$kctx" ]; then
  kubectl_cmd="${KUBECTL} --context=$kctx"
fi

istio_injection="$($kubectl_cmd get ns "$ns" -o jsonpath="{.metadata.labels.istio-injection}" 2>/dev/null || true)"

wait_one() {
  c="$1"
  want="$2"

  echo "Wait create: $ns/$c${kctx:+ (context $kctx)}"
  $kubectl_cmd -n "$ns" wait --for=create "opentelemetrycollector/$c" --timeout="$timeout"

  echo "Wait ready: $ns/$c statusReplicas=$want${kctx:+ (context $kctx)}"
  $kubectl_cmd -n "$ns" wait --for="jsonpath={.status.scale.statusReplicas}=$want" "opentelemetrycollector/$c" --timeout="$timeout" || {
    echo "TIMEOUT waiting for $ns/$c to reach statusReplicas=$want${kctx:+ (context $kctx)}"
    selector="$($kubectl_cmd -n "$ns" get "opentelemetrycollector/$c" -o jsonpath="{.status.scale.selector}" 2>/dev/null || true)"
    if [ -n "$selector" ]; then
      echo "Pods for $ns/$c:"
      $kubectl_cmd -n "$ns" get pods -l "$selector" -o wide || true
      $kubectl_cmd -n "$ns" get pods -l "$selector" -o jsonpath="{range .items[*]}{.metadata.name}{\"\n\"}{range .status.containerStatuses[*]}{\"  - \"}{.name}{\": ready=\"}{.ready}{\", restarts=\"}{.restartCount}{\", waiting=\"}{.state.waiting.reason}{\"\n\"}{end}{end}" || true
    fi
    exit 1
  }

  selector="$($kubectl_cmd -n "$ns" get "opentelemetrycollector/$c" -o jsonpath="{.status.scale.selector}")"

  echo "Wait pod health: $ns/$c${kctx:+ (context $kctx)}"
  $kubectl_cmd -n "$ns" wait --for=condition=Ready pod -l "$selector" --timeout="$timeout" || {
    echo "TIMEOUT waiting for healthy pods for $ns/$c${kctx:+ (context $kctx)}"
    $kubectl_cmd -n "$ns" get pods -l "$selector" -o wide || true
    $kubectl_cmd -n "$ns" get pods -l "$selector" -o jsonpath="{range .items[*]}{.metadata.name}{\"\n\"}{range .status.containerStatuses[*]}{\"  - \"}{.name}{\": ready=\"}{.ready}{\", restarts=\"}{.restartCount}{\", waiting=\"}{.state.waiting.reason}{\"\n\"}{end}{end}" || true
    exit 1
  }
}

wait_one kof-collectors-cluster-stats 1/1

if [ "$istio_injection" != "enabled" ]; then
  wait_one kof-collectors-controller-k0s-daemon 1/1
fi

wait_one kof-collectors-ta-daemon 2/2
wait_one kof-collectors-daemon 2/2