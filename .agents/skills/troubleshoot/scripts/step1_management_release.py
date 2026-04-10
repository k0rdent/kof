#!/usr/bin/env python3
"""
Step 1 — Management and Release health check.

Usage:
    python3 step1_management_release.py <bundle-dir>

Exits 0 if all healthy, 1 if any failures found.
"""
import sys
import os

sys.path.insert(0, os.path.dirname(__file__))
from lib import load_yaml_list, cr_path, flag, FAIL, OK

def check_management(bundle_dir):
    path = cr_path(bundle_dir, "managements.k0rdent.mirantis.com.yaml")
    items = load_yaml_list(path)
    if not items:
        print(f"  [WARN] No Management objects found at {path}")
        return True

    ok = True
    for r in items:
        name = (r.get("metadata") or {}).get("name", "?")
        status = r.get("status") or {}
        conditions = status.get("conditions") or []
        components = status.get("components") or {}

        # components may be a dict (name→obj) or a list
        if isinstance(components, dict):
            comp_list = [{"name": k, **v} for k, v in components.items()]
        else:
            comp_list = components

        print(f"  Management: {name}")
        for c in conditions:
            healthy = c.get("status") == "True"
            f = flag(healthy)
            if not healthy:
                ok = False
            print(f"    [{f}] {c.get('type')}: {c.get('status')} "
                  f"({c.get('reason', '')}) {str(c.get('message', ''))[:160]}")

        for comp in comp_list:
            success = comp.get("success", False)
            f = flag(success)
            if not success:
                ok = False
            print(f"    [{f}] component={comp.get('name', '?')} "
                  f"template={comp.get('template', '?')} success={success}")
    return ok


def check_releases(bundle_dir):
    path = cr_path(bundle_dir, "releases.k0rdent.mirantis.com.yaml")
    items = load_yaml_list(path)
    if not items:
        print(f"  [WARN] No Release objects found at {path}")
        return True

    ok = True
    for r in items:
        name = (r.get("metadata") or {}).get("name", "?")
        status = r.get("status") or {}
        ready = status.get("ready")
        conditions = status.get("conditions") or []

        healthy = ready is True
        f = flag(healthy)
        if not healthy:
            ok = False
        print(f"  [{f}] Release: {name}  ready={ready}")
        for c in conditions:
            cf = flag(c.get("status") == "True")
            if c.get("status") != "True":
                ok = False
            print(f"    [{cf}] {c.get('type')}: {c.get('status')} "
                  f"({c.get('reason', '')}) {str(c.get('message', ''))[:160]}")
    return ok


def main():
    if len(sys.argv) < 2:
        print(f"Usage: {sys.argv[0]} <bundle-dir>")
        sys.exit(2)
    bundle_dir = sys.argv[1]

    print("=== Step 1: Management ===")
    ok1 = check_management(bundle_dir)
    print()
    print("=== Step 1: Releases ===")
    ok2 = check_releases(bundle_dir)

    sys.exit(0 if (ok1 and ok2) else 1)


if __name__ == "__main__":
    main()
