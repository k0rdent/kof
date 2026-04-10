#!/usr/bin/env python3
"""
Step 5 — ServiceSet health check.

Usage:
    python3 step5_servicesets.py <bundle-dir>

Exits 0 if all ServiceSets healthy, 1 if any failures.
"""
import sys
import os

sys.path.insert(0, os.path.dirname(__file__))
from lib import load_yaml_list, cr_path, flag


def check_servicesets(bundle_dir):
    path = cr_path(bundle_dir, "servicesets.k0rdent.mirantis.com", "kcm-system.yaml")
    items = load_yaml_list(path)
    if not items:
        print(f"  [WARN] No ServiceSet objects found at {path}")
        return True

    ok = True
    for r in items:
        name = (r.get("metadata") or {}).get("name", "?")
        ns   = (r.get("metadata") or {}).get("namespace", "?")
        status = r.get("status") or {}
        deployed     = status.get("deployed")
        prov_ready   = (status.get("provider") or {}).get("ready")
        conditions   = status.get("conditions") or []

        failing_conds = [c for c in conditions if c.get("status") != "True"]
        healthy = (deployed is True and prov_ready is True and not failing_conds)
        f = flag(healthy)
        if not healthy:
            ok = False
        print(f"  [{f}] {ns}/{name}  deployed={deployed} provider.ready={prov_ready}")
        for c in failing_conds:
            print(f"    [FAIL] {c.get('type')}: {c.get('status')} "
                  f"({c.get('reason', '')}) {str(c.get('message', ''))[:160]}")
    return ok


def main():
    if len(sys.argv) < 2:
        print(f"Usage: {sys.argv[0]} <bundle-dir>")
        sys.exit(2)
    bundle_dir = sys.argv[1]

    print("=== Step 5: ServiceSets ===")
    ok = check_servicesets(bundle_dir)
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
