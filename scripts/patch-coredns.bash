#!/usr/bin/env bash

set -euo pipefail

CLUSTER_NAME="${1:-kcm-dev}"
CONTAINER_TOOL="${2:-docker}"
DEX_HOSTNAME="dex.example.com"

echo "🔍 Getting kind control-plane container IP..."
CONTROL_PLANE_IP=$(${CONTAINER_TOOL} inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "${CLUSTER_NAME}-control-plane")
echo "➡️ Found IP: ${CONTROL_PLANE_IP}"

echo "📦 Fetching existing Corefile..."
COREFILE=$(kubectl get configmap coredns -n kube-system -o jsonpath='{.data.Corefile}')

echo "🔍 Checking if host entry already exists..."
if echo "$COREFILE" | grep -q "$CONTROL_PLANE_IP.*$DEX_HOSTNAME\|$DEX_HOSTNAME.*$CONTROL_PLANE_IP"; then
    echo "✅ Host entry already exists, skipping update"
    exit 0
fi

echo "🛠️ Injecting hosts plugin block into Corefile..."
PATCHED_COREFILE=$(echo "$COREFILE" | awk -v ip="$CONTROL_PLANE_IP" -v host="$DEX_HOSTNAME" '
  BEGIN { inserted = 0 }
  {
    print
    if ($0 ~ /^[[:space:]]*loadbalance[[:space:]]*$/ && inserted == 0) {
      print "    hosts {"
      print "        " ip " " host
      print "        fallthrough"
      print "    }"
      inserted = 1
    }
  }
')

echo "💾 Replacing ConfigMap with updated Corefile..."
kubectl create configmap coredns \
  --from-literal=Corefile="$PATCHED_COREFILE" \
  -n kube-system \
  --dry-run=client -o yaml | kubectl apply -f -

echo "🔄 Restarting CoreDNS pods..."
kubectl -n kube-system rollout restart deploy -l k8s-app=kube-dns

echo "✅ CoreDNS updated. '${DEX_HOSTNAME}' should now resolve to ${CONTROL_PLANE_IP} inside the cluster."