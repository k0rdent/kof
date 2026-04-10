#!/usr/bin/env python3
"""
Step 7 — SveltosCluster connectivity check.

Usage:
    python3 step7_sveltos_clusters.py <bundle-dir>

Checks kcm-system.yaml and mgmt.yaml (if present).
Exits 0 if all healthy, 1 if any failures.
"""
import sys
import os

sys.path.insert(0, os.path.dirname(__file__))
from lib import load_yaml_list, cr_path, flag, OK, WARN, FAIL


def check_sveltos_clusters(bundle_dir):
    ok = True
    found_any = False

    for ns_yaml in ["kcm-system.yaml", "mgmt.yaml"]:
        path = cr_path(bundle_dir, "sveltosclusters.lib.projectsveltos.io", ns_yaml)
        items = load_yaml_list(path)
        if not items:
            continue
        found_any = True
        for r in items:
            name   = (r.get("metadata") or {}).get("name", "?")
            ns     = (r.get("metadata") or {}).get("namespace", "?")
            status = r.get("status") or {}
            conn   = status.get("connectionStatus", "?")
            ready  = status.get("ready")
            fails  = status.get("connectionFailures", 0) or 0

            healthy = (conn == "Healthy" and ready is True)
            warn    = (healthy and fails > 0)
            if warn:
                f = WARN
            elif healthy:
                f = OK
            else:
                f = FAIL
                ok = False
            print(f"  [{f}] {ns}/{name}  connectionStatus={conn} ready={ready} "
                  f"connectionFailures={fails}")

    if not found_any:
        print(f"  [WARN] No SveltosCluster objects found in bundle")
    return ok


def main():
    if len(sys.argv) < 2:
        print(f"Usage: {sys.argv[0]} <bundle-dir>")
        sys.exit(2)
    bundle_dir = sys.argv[1]

    print("=== Step 7: SveltosClusters ===")
    ok = check_sveltos_clusters(bundle_dir)
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
