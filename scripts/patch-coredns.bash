#!/usr/bin/env bash

set -euo pipefail

KUBECTL="${1:-kubectl}"
HOST_NAME="$2"
HOST_IP="$3"

echo "📦 Fetching existing Corefile..."
COREFILE=$($KUBECTL get configmap coredns -n kube-system -o jsonpath='{.data.Corefile}')

echo "🔍 Checking if host entry already exists..."
if echo "$COREFILE" | grep -q "$HOST_IP $HOST_NAME"; then
    echo "✅ Host entry already exists, skipping update"
    exit 0
fi

HOSTS=0

echo "🔍 Checking if hosts section exists..."
if echo "$COREFILE" | grep -q "hosts {"; then
  HOSTS=1
fi

echo "🛠️ Injecting hosts plugin block into Corefile..."
PATCHED_COREFILE=$(echo "$COREFILE" | awk -v ip="$HOST_IP" -v host="$HOST_NAME" -v hosts="$HOSTS" '
  BEGIN { inserted = 0 }
  {
    print
    if ($0 ~ /^[[:space:]]*loadbalance[[:space:]]*$/ && inserted == 0 && hosts == 0) {
      print "    hosts {"
      print "        " ip " " host
      print "        fallthrough"
      print "    }"
      inserted = 1
    }
    if ($0 ~ /^[[:space:]]*hosts[[:space:]]\{$/ && inserted == 0 && hosts == 1) {
      print "        " ip " " host
      inserted = 1
    }
  }
')

echo "💾 Replacing ConfigMap with updated Corefile..."
$KUBECTL create configmap coredns \
  --from-literal=Corefile="$PATCHED_COREFILE" \
  -n kube-system \
  --dry-run=client -o yaml | $KUBECTL apply -f -

echo "🔄 Restarting CoreDNS pods..."
$KUBECTL -n kube-system rollout restart deploy/coredns

echo "✅ CoreDNS updated. '${HOST_NAME}' should now resolve to ${HOST_IP} inside the cluster."
