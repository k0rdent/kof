#!/usr/bin/env bash
set -euo pipefail

# This script can be executed with `make support-bundle` or directly.
#
# Optional env vars:
# * KUBECTL - default: `kubectl`
# * KUBECTL_CONTEXT - default: current, as in `kubectl config current-context`
# * SUPPORT_BUNDLE_CLI - default: `kubectl-support_bundle`
# * SUPPORT_BUNDLE_OUTPUT - default: `support-bundle-{timestamp}.tar.gz`

sb="${SUPPORT_BUNDLE_CLI:-kubectl-support_bundle}"
command -v "$sb" || {
  echo "$sb is not found."
  echo 'Please run `make support-bundle` or install this plugin first:'
  echo 'https://docs.replicated.com/vendor/support-bundle-generating#prerequisite-install-the-support-bundle-plugin'
  exit 1
}

k="${KUBECTL:-kubectl}"
ctx="${KUBECTL_CONTEXT:-}"  # Empty string means the current context.

dev_dir=$(dirname "$0")/../dev
mkdir -p "$dev_dir"
config_file="$dev_dir/support-bundle.yaml"

cat >"$config_file" <<EOF
apiVersion: troubleshoot.sh/v1beta2
kind: SupportBundle
metadata:
  name: support-bundle
spec:
  collectors:
EOF

"$k" --context "$ctx" get ns \
  -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' \
  | while read -r ns; do cat >>"$config_file" <<EOF
    - logs:
        namespace: "$ns"
        name: "logs/$ns"
    - secret:
        namespace: "$ns"
        selector:
          - ""
        includeAllData: true
EOF
done

"$sb" --context "$ctx" -o "${SUPPORT_BUNDLE_OUTPUT:-}" "$config_file"

echo '
To analyze this support bundle:
* Run: scripts/support-bundle-analyzer.py support-bundle-{timestamp}.tar.gz
* Install `sbctl`: https://github.com/replicatedhq/sbctl/blob/main/README.md
* Run: sbctl shell -s support-bundle-{timestamp}.tar.gz
* Use: kubectl, k9s, etc.
'
