#!/usr/bin/env python3
"""
Full KOF/KCM support-bundle health check — runs all 12 analysis steps.

Usage:
    python3 analyze_bundle.py <bundle-dir> [<namespace> ...]

    <bundle-dir>    Path to a single timestamped bundle directory, e.g.
                    ~/Downloads/support-bundle-2026-04-10T05_51_50
    <namespace>     Additional namespaces to check for workloads (step 12).
                    Defaults: kof, kcm-system

Examples:
    python3 analyze_bundle.py ~/Downloads/bundle/support-bundle-2026-04-10T05_51_50
    python3 analyze_bundle.py ~/Downloads/bundle/support-bundle-2026-04-10T05_56_46 kof

Exit code: 0 = all healthy, 1 = one or more failures found.
"""
import sys
import os
import importlib.util

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))


def load_step(module_name):
    path = os.path.join(SCRIPT_DIR, f"{module_name}.py")
    spec = importlib.util.spec_from_file_location(module_name, path)
    mod  = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


def run_step(label, fn, *args):
    print(f"\n{'='*60}")
    print(f"  {label}")
    print('='*60)
    try:
        result = fn(*args)
        return result
    except Exception as exc:
        print(f"  [ERROR] {exc}")
        return False


def main():
    if len(sys.argv) < 2:
        print(__doc__)
        sys.exit(2)

    bundle_dir = os.path.expanduser(sys.argv[1])
    if not os.path.isdir(bundle_dir):
        print(f"ERROR: bundle directory not found: {bundle_dir}")
        sys.exit(2)

    extra_ns = sys.argv[2:] if len(sys.argv) > 2 else []

    # De-duplicate while preserving order
    seen_ns = set()
    deduped = []
    for n in ["kof", "kcm-system"] + extra_ns:
        if n not in seen_ns:
            seen_ns.add(n)
            deduped.append(n)

    print(f"Analysing bundle: {bundle_dir}")

    steps = [
        ("Step 1  — Management & Release",         "step1_management_release",   "check_management",        bundle_dir),
        ("Step 1b — Releases",                      "step1_management_release",   "check_releases",          bundle_dir),
        ("Step 2  — Templates",                     "step2_templates",            "check_templates",         bundle_dir),
        ("Step 3  — Credentials",                   "step3_credentials",          "check_credentials",       bundle_dir),
        ("Step 4  — ClusterDeployments",            "step4_clusterdeployments",   "check_clusterdeployments",bundle_dir),
        ("Step 5  — ServiceSets",                   "step5_servicesets",          "check_servicesets",       bundle_dir),
        ("Step 6  — MultiClusterServices",          "step6_multiclusterservices", "check_mcs",               bundle_dir),
        ("Step 7  — SveltosClusters",               "step7_sveltos_clusters",     "check_sveltos_clusters",  bundle_dir),
        ("Step 8  — Profiles",                      "step8_profiles",             "check_profiles",          bundle_dir),
        ("Step 9  — ClusterSummaries",              "step9_clustersummaries",     "check_clustersummaries",  bundle_dir),
        ("Step 10 — HelmReleases",                  "step10_helmreleases",        "check_helmreleases",      bundle_dir),
        ("Step 11 — PromxyServerGroups",            "step11_promxyservergroups",  "check_promxy",            bundle_dir),
    ]

    workload_ns = deduped

    all_ok = True
    loaded = {}

    for label, module_name, fn_name, *args in steps:
        if module_name not in loaded:
            loaded[module_name] = load_step(module_name)
        mod = loaded[module_name]
        fn  = getattr(mod, fn_name)
        result = run_step(label, fn, *args)
        if not result:
            all_ok = False

    # Step 12 — workloads
    step12 = load_step("step12_workloads")
    result = run_step(
        f"Step 12 — Workloads ({', '.join(workload_ns)})",
        step12.check_pods, bundle_dir, workload_ns
    )
    if not result:
        all_ok = False
    result = run_step(
        "Step 12 — Deployments",
        step12.check_deployments, bundle_dir, workload_ns
    )
    if not result:
        all_ok = False
    result = run_step(
        "Step 12 — StatefulSets",
        step12.check_statefulsets, bundle_dir, workload_ns
    )
    if not result:
        all_ok = False

    print(f"\n{'='*60}")
    if all_ok:
        print("  RESULT: All checks passed — no failures detected.")
    else:
        print("  RESULT: One or more failures detected. Review FAIL/WARN lines above.")
    print('='*60)

    sys.exit(0 if all_ok else 1)


if __name__ == "__main__":
    main()
