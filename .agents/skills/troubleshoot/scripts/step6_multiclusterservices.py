#!/usr/bin/env python3
"""
Step 6 — MultiClusterService health check.

Usage:
    python3 step6_multiclusterservices.py <bundle-dir>

Exits 0 if all MCSes healthy, 1 if any failures.
"""
import sys
import os

sys.path.insert(0, os.path.dirname(__file__))
from lib import load_yaml_list, cr_path, flag


def check_mcs(bundle_dir):
    path = cr_path(bundle_dir, "multiclusterservices.k0rdent.mirantis.com.yaml")
    items = load_yaml_list(path)
    if not items:
        print(f"  [WARN] No MultiClusterService objects found at {path}")
        return True

    ok = True
    for r in items:
        name = (r.get("metadata") or {}).get("name", "?")
        status = r.get("status") or {}
        conditions = status.get("conditions") or []
        depends_on = (r.get("spec") or {}).get("dependsOn") or []

        failing = [c for c in conditions if c.get("status") != "True"]
        healthy = not failing
        f = flag(healthy)
        if not healthy:
            ok = False
        dep_str = f"  dependsOn={depends_on}" if depends_on else ""
        print(f"  [{f}] {name}{dep_str}")
        for c in failing:
            print(f"    [FAIL] {c.get('type')}: {c.get('status')} "
                  f"({c.get('reason', '')}) {str(c.get('message', ''))[:200]}")
    return ok


def main():
    if len(sys.argv) < 2:
        print(f"Usage: {sys.argv[0]} <bundle-dir>")
        sys.exit(2)
    bundle_dir = sys.argv[1]

    print("=== Step 6: MultiClusterServices ===")
    ok = check_mcs(bundle_dir)
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
