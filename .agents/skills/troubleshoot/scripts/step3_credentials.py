#!/usr/bin/env python3
"""
Step 3 — Credential health check.

Usage:
    python3 step3_credentials.py <bundle-dir>

Exits 0 if all credentials ready, 1 if any failures.
"""
import sys
import os

sys.path.insert(0, os.path.dirname(__file__))
from lib import load_yaml_list, cr_path, flag


def check_credentials(bundle_dir):
    path = cr_path(bundle_dir, "credentials.k0rdent.mirantis.com", "kcm-system.yaml")
    items = load_yaml_list(path)
    if not items:
        print(f"  [WARN] No Credential objects found at {path}")
        return True

    ok = True
    for r in items:
        name = (r.get("metadata") or {}).get("name", "?")
        ns   = (r.get("metadata") or {}).get("namespace", "?")
        status = r.get("status") or {}
        ready = status.get("ready")
        conditions = status.get("conditions") or []
        iref = (r.get("spec") or {}).get("identityRef") or {}

        healthy = ready is True
        f = flag(healthy)
        if not healthy:
            ok = False
        print(f"  [{f}] {ns}/{name}  ready={ready}")
        for c in conditions:
            cf = flag(c.get("status") == "True")
            if c.get("status") != "True":
                ok = False
            print(f"    [{cf}] {c.get('type')}: {c.get('status')} "
                  f"({c.get('reason', '')}) {str(c.get('message', ''))[:160]}")
        print(f"    identityRef: kind={iref.get('kind', '?')} "
              f"name={iref.get('name', '?')} namespace={iref.get('namespace', '?')}")
    return ok


def main():
    if len(sys.argv) < 2:
        print(f"Usage: {sys.argv[0]} <bundle-dir>")
        sys.exit(2)
    bundle_dir = sys.argv[1]

    print("=== Step 3: Credentials ===")
    ok = check_credentials(bundle_dir)
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
