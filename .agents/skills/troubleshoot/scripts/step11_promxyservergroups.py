#!/usr/bin/env python3
"""
Step 11 — PromxyServerGroup presence and label check.

Usage:
    python3 step11_promxyservergroups.py <bundle-dir>

Checks that every PromxyServerGroup:
  - has the k0rdent.mirantis.com/secret-name label
  - has at least one target in spec.targets
  - has an ownerReference

Exits 0 if all look healthy, 1 if any issues found.
"""
import sys
import os

sys.path.insert(0, os.path.dirname(__file__))
from lib import load_yaml_list, cr_path, flag, OK, WARN


def check_promxy(bundle_dir):
    paths = [
        cr_path(bundle_dir, "promxyservergroups.kof.k0rdent.mirantis.com", "kcm-system.yaml"),
        cr_path(bundle_dir, "promxyservergroups.kof.k0rdent.mirantis.com", "kof.yaml"),
    ]
    items = []
    for path in paths:
        items.extend(load_yaml_list(path))
    if not items:
        print(
            "  [WARN] No PromxyServerGroup objects found at "
            + " or ".join(paths)
        )
        # Not necessarily a failure — only mothership bundles have these
        return True

    ok = True
    for r in items:
        name   = (r.get("metadata") or {}).get("name", "?")
        ns     = (r.get("metadata") or {}).get("namespace", "?")
        labels = (r.get("metadata") or {}).get("labels") or {}
        owners = (r.get("metadata") or {}).get("ownerReferences") or []
        spec   = r.get("spec") or {}
        targets     = spec.get("targets") or []
        cluster_name = spec.get("cluster_name", "?")

        has_secret_label = "k0rdent.mirantis.com/secret-name" in labels
        has_targets      = bool(targets)
        has_owner        = bool(owners)

        issues = []
        if not has_secret_label:
            issues.append("missing label k0rdent.mirantis.com/secret-name")
        if not has_targets:
            issues.append("spec.targets is empty")
        if not has_owner:
            issues.append("no ownerReference")

        f = OK if not issues else WARN
        if issues:
            ok = False
        print(f"  [{f}] {ns}/{name}  cluster={cluster_name}  "
              f"targets={len(targets)}  secretLabel={has_secret_label}")
        for owner in owners:
            print(f"    owner: {owner.get('kind')}/{owner.get('name')}")
        for issue in issues:
            print(f"    [WARN] {issue}")

    return ok


def main():
    if len(sys.argv) < 2:
        print(f"Usage: {sys.argv[0]} <bundle-dir>")
        sys.exit(2)
    bundle_dir = sys.argv[1]

    print("=== Step 11: PromxyServerGroups ===")
    ok = check_promxy(bundle_dir)
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
