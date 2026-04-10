#!/usr/bin/env python3
"""
Step 10 — HelmRelease health check.

Usage:
    python3 step10_helmreleases.py <bundle-dir>

Checks kof.yaml and kcm-system.yaml namespaces.
Exits 0 if all HelmReleases ready, 1 if any failures.
"""
import sys
import os

sys.path.insert(0, os.path.dirname(__file__))
from lib import load_yaml_list, cr_path, flag


def check_helmreleases(bundle_dir):
    ok = True
    found_any = False

    for ns_yaml in ["kof.yaml", "kcm-system.yaml"]:
        path = cr_path(bundle_dir, "helmreleases.helm.toolkit.fluxcd.io", ns_yaml)
        items = load_yaml_list(path)
        if not items:
            continue
        found_any = True
        for r in items:
            name = (r.get("metadata") or {}).get("name", "?")
            ns   = (r.get("metadata") or {}).get("namespace", "?")
            status = r.get("status") or {}
            conditions = status.get("conditions") or []
            history    = status.get("history") or []

            ready_cond = next((c for c in conditions if c.get("type") == "Ready"), None)
            ready_ok   = ready_cond is not None and ready_cond.get("status") == "True"
            history_ok = (not history) or history[0].get("status") == "deployed"
            healthy    = ready_ok and history_ok

            f = flag(healthy)
            if not healthy:
                ok = False
            reason = ready_cond.get("reason", "?") if ready_cond else "?"
            print(f"  [{f}] {ns}/{name}  reason={reason}")
            if not ready_ok and ready_cond:
                print(f"    [FAIL] {str(ready_cond.get('message', ''))[:200]}")
            if not history_ok and history:
                print(f"    [FAIL] history[0].status={history[0].get('status')}")

    if not found_any:
        print(f"  [WARN] No HelmRelease objects found in bundle")
    return ok


def main():
    if len(sys.argv) < 2:
        print(f"Usage: {sys.argv[0]} <bundle-dir>")
        sys.exit(2)
    bundle_dir = sys.argv[1]

    print("=== Step 10: HelmReleases ===")
    ok = check_helmreleases(bundle_dir)
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
