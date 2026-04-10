#!/usr/bin/env python3
"""
Step 2 — ProviderTemplate, ClusterTemplate, ServiceTemplate validity check.

Usage:
    python3 step2_templates.py <bundle-dir>

Prints only invalid templates. Exits 0 if all valid, 1 if any invalid.
"""
import sys
import os

sys.path.insert(0, os.path.dirname(__file__))
from lib import load_yaml_list, cr_path, flag

TEMPLATE_FILES = [
    ("ProviderTemplate",  "providertemplates.k0rdent.mirantis.com.yaml",           None),
    ("ClusterTemplate",   "clustertemplates.k0rdent.mirantis.com",                  "kcm-system.yaml"),
    ("ServiceTemplate",   "servicetemplates.k0rdent.mirantis.com",                  "kcm-system.yaml"),
]


def check_templates(bundle_dir):
    any_invalid = False
    for kind, group_file, ns in TEMPLATE_FILES:
        path = cr_path(bundle_dir, group_file, ns)
        items = load_yaml_list(path)
        if not items:
            print(f"  [WARN] No {kind} objects found at {path}")
            continue
        invalid = []
        for r in items:
            name = (r.get("metadata") or {}).get("name", "?")
            status = r.get("status") or {}
            valid = status.get("valid")
            if valid is not True:
                invalid.append((name, valid, status.get("validationError", "")))
        if invalid:
            any_invalid = True
            for name, valid, err in invalid:
                print(f"  [FAIL] {kind}/{name}  valid={valid}")
                if err:
                    print(f"         {str(err)[:200]}")
        else:
            total = len(items)
            print(f"  [OK  ] {kind}: all {total} valid")
    return not any_invalid


def main():
    if len(sys.argv) < 2:
        print(f"Usage: {sys.argv[0]} <bundle-dir>")
        sys.exit(2)
    bundle_dir = sys.argv[1]

    print("=== Step 2: Templates ===")
    ok = check_templates(bundle_dir)
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
