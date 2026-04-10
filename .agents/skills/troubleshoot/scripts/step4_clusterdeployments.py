#!/usr/bin/env python3
"""
Step 4 — ClusterDeployment health check.

Usage:
    python3 step4_clusterdeployments.py <bundle-dir>

Exits 0 if all ClusterDeployments healthy, 1 if any failures.
"""
import sys
import os

sys.path.insert(0, os.path.dirname(__file__))
from lib import load_yaml_list, cr_path, flag


def check_clusterdeployments(bundle_dir):
    path = cr_path(bundle_dir, "clusterdeployments.k0rdent.mirantis.com", "kcm-system.yaml")
    items = load_yaml_list(path)
    if not items:
        print(f"  [WARN] No ClusterDeployment objects found at {path}")
        return True

    ok = True
    for r in items:
        name = (r.get("metadata") or {}).get("name", "?")
        ns   = (r.get("metadata") or {}).get("namespace", "?")
        status = r.get("status") or {}
        conditions = status.get("conditions") or []
        services   = status.get("services") or []

        failing_conds = [c for c in conditions if c.get("status") != "True"]
        failing_svcs  = [s for s in services if s.get("state") != "Deployed"]
        healthy = not failing_conds and not failing_svcs
        f = flag(healthy)
        if not healthy:
            ok = False
        print(f"  [{f}] {ns}/{name}")
        for c in failing_conds:
            print(f"    [FAIL] cond {c.get('type')}: {c.get('status')} "
                  f"({c.get('reason', '')}) {str(c.get('message', ''))[:160]}")
        for s in failing_svcs:
            print(f"    [FAIL] service {s.get('name', '?')} "
                  f"state={s.get('state', '?')} {str(s.get('message', ''))[:120]}")
    return ok


def main():
    if len(sys.argv) < 2:
        print(f"Usage: {sys.argv[0]} <bundle-dir>")
        sys.exit(2)
    bundle_dir = sys.argv[1]

    print("=== Step 4: ClusterDeployments ===")
    ok = check_clusterdeployments(bundle_dir)
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
