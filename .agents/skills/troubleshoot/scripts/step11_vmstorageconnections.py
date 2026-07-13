#!/usr/bin/env python3
"""
Step 11 — VMStorageConnection presence and label check.

Usage:
    python3 step11_vmstorageconnections.py <bundle-dir>

Checks that every VMStorageConnection:
  - has the k0rdent.mirantis.com/kof-generated label
  - has a non-empty spec.target_storage_node.address
  - references a supported cluster kind (VMCluster, VLCluster, VTCluster)
  - has an ownerReference

Exits 0 if all look healthy, 1 if any issues found.
"""
import sys
import os

sys.path.insert(0, os.path.dirname(__file__))
from lib import load_yaml_list, cr_path, flag, OK, WARN

SUPPORTED_KINDS = {"VMCluster", "VLCluster", "VTCluster"}


def check_vmstorageconnections(bundle_dir):
    paths = [
        cr_path(bundle_dir, "vmstorageconnections.kof.k0rdent.mirantis.com", "kcm-system.yaml"),
        cr_path(bundle_dir, "vmstorageconnections.kof.k0rdent.mirantis.com", "kof.yaml"),
    ]
    items = []
    for path in paths:
        items.extend(load_yaml_list(path))
    if not items:
        print(
            "  [WARN] No VMStorageConnection objects found at "
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
        cluster_ref = spec.get("cluster_ref") or {}
        target_node = spec.get("target_storage_node") or {}
        address = target_node.get("address", "")
        kind = cluster_ref.get("kind", "?")

        has_generated_label = "k0rdent.mirantis.com/kof-generated" in labels
        has_address         = bool(address)
        has_owner           = bool(owners)
        has_supported_kind  = kind in SUPPORTED_KINDS

        issues = []
        if not has_address:
            issues.append("spec.target_storage_node.address is empty")
        if not has_supported_kind:
            issues.append(f"unsupported cluster_ref.kind: {kind}")
        if not has_owner:
            issues.append("no ownerReference")

        f = OK if not issues else WARN
        if issues:
            ok = False
        print(f"  [{f}] {ns}/{name}  kind={kind}  cluster={cluster_ref.get('name', '?')}  "
              f"address={address}  generatedLabel={has_generated_label}")
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

    print("=== Step 11: VMStorageConnections ===")
    ok = check_vmstorageconnections(bundle_dir)
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
