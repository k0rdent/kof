#!/usr/bin/env python3
"""
Step 8 — Profile matching check.

Usage:
    python3 step8_profiles.py <bundle-dir>

Warns when matchingClusters is empty (no cluster matched the selector).
Exits 0 if all Profiles have matches, 1 otherwise.
"""
import sys
import os

sys.path.insert(0, os.path.dirname(__file__))
from lib import load_yaml_list, cr_path, flag, OK, WARN


def check_profiles(bundle_dir):
    path = cr_path(bundle_dir, "profiles.config.projectsveltos.io", "kcm-system.yaml")
    items = load_yaml_list(path)
    if not items:
        print(f"  [WARN] No Profile objects found at {path}")
        return True

    ok = True
    for r in items:
        name = (r.get("metadata") or {}).get("name", "?")
        ns   = (r.get("metadata") or {}).get("namespace", "?")
        status = r.get("status") or {}
        mc = status.get("matchingClusters") or []

        has_match = bool(mc)
        f = OK if has_match else WARN
        if not has_match:
            ok = False
        cluster_names = [
            f"{c.get('namespace', '')}/{c.get('name', '?')}"
            for c in (mc if isinstance(mc, list) else [])
        ]
        match_str = ", ".join(cluster_names) if cluster_names else "NONE"
        print(f"  [{f}] {ns}/{name}  matchingClusters=[{match_str}]")
    return ok


def main():
    if len(sys.argv) < 2:
        print(f"Usage: {sys.argv[0]} <bundle-dir>")
        sys.exit(2)
    bundle_dir = sys.argv[1]

    print("=== Step 8: Profiles ===")
    ok = check_profiles(bundle_dir)
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
