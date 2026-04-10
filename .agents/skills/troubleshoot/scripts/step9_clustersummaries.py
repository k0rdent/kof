#!/usr/bin/env python3
"""
Step 9 — ClusterSummary feature and HelmRelease status check.

Usage:
    python3 step9_clustersummaries.py <bundle-dir>

Exits 0 if all features Provisioned and all HelmReleases Managing, 1 otherwise.
"""
import sys
import os

sys.path.insert(0, os.path.dirname(__file__))
from lib import load_yaml_list, cr_path, flag


def check_clustersummaries(bundle_dir):
    path = cr_path(bundle_dir, "clustersummaries.config.projectsveltos.io", "kcm-system.yaml")
    items = load_yaml_list(path)
    if not items:
        print(f"  [WARN] No ClusterSummary objects found at {path}")
        return True

    ok = True
    for r in items:
        name = (r.get("metadata") or {}).get("name", "?")
        ns   = (r.get("metadata") or {}).get("namespace", "?")
        status = r.get("status") or {}
        feat_summaries = status.get("featureSummaries") or []
        helm_summaries = status.get("helmReleaseSummaries") or []

        feat_failures = [f for f in feat_summaries if f.get("status") != "Provisioned"]
        helm_failures = [h for h in helm_summaries if h.get("status") != "Managing"]

        healthy = not feat_failures and not helm_failures
        f = flag(healthy)
        if not healthy:
            ok = False
        print(f"  [{f}] {ns}/{name}")
        for ff in feat_failures:
            msg = str(ff.get("failureMessage", ""))[:200]
            print(f"    [FAIL] featureID={ff.get('featureID')} status={ff.get('status')}")
            if msg:
                print(f"           {msg}")
        for hf in helm_failures:
            msg = str(hf.get("failureMessage", ""))[:200]
            print(f"    [FAIL] helm {hf.get('releaseNamespace')}/{hf.get('releaseName')} "
                  f"status={hf.get('status')}")
            if msg:
                print(f"           {msg}")
    return ok


def main():
    if len(sys.argv) < 2:
        print(f"Usage: {sys.argv[0]} <bundle-dir>")
        sys.exit(2)
    bundle_dir = sys.argv[1]

    print("=== Step 9: ClusterSummaries ===")
    ok = check_clustersummaries(bundle_dir)
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
