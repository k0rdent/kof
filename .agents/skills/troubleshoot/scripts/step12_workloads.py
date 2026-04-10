#!/usr/bin/env python3
"""
Step 12 — Pod, Deployment, StatefulSet health check.

Usage:
    python3 step12_workloads.py <bundle-dir> [<namespace> ...]

Default namespaces checked: kof, kcm-system
Exits 0 if all workloads healthy, 1 if any failures found.

Note: pods with phase=Succeeded are treated as completed Jobs and skipped.
"""
import sys
import os

sys.path.insert(0, os.path.dirname(__file__))
from lib import load_json_list, core_path, flag

DEFAULT_NAMESPACES = ["kof", "kcm-system"]


def check_pods(bundle_dir, namespaces):
    ok = True
    for ns in namespaces:
        path = core_path(bundle_dir, "pods", ns)
        items = load_json_list(path)
        if not items:
            continue
        for p in items:
            name  = (p.get("metadata") or {}).get("name", "?")
            phase = (p.get("status") or {}).get("phase", "?")

            # Skip completed Jobs
            if phase == "Succeeded":
                continue

            if phase != "Running":
                ok = False
                print(f"  [FAIL] pod {ns}/{name}  phase={phase}")
                cs = (p.get("status") or {}).get("containerStatuses") or []
                for c in cs:
                    state   = c.get("state") or {}
                    waiting = state.get("waiting") or {}
                    term    = state.get("terminated") or {}
                    if waiting:
                        print(f"    container={c['name']} waiting={waiting.get('reason')} "
                              f"msg={str(waiting.get('message', ''))[:120]}")
                    elif term and term.get("exitCode", 0) != 0:
                        print(f"    container={c['name']} terminated "
                              f"reason={term.get('reason')} exitCode={term.get('exitCode')}")
                continue

            # Running — check Ready condition
            conds      = (p.get("status") or {}).get("conditions") or []
            ready_cond = next((c for c in conds if c.get("type") == "Ready"), None)
            if ready_cond and ready_cond.get("status") != "True":
                ok = False
                print(f"  [WARN] pod {ns}/{name}  phase=Running but Ready=False")
                cs = (p.get("status") or {}).get("containerStatuses") or []
                for c in cs:
                    state   = c.get("state") or {}
                    waiting = state.get("waiting") or {}
                    if waiting:
                        print(f"    container={c['name']} waiting={waiting.get('reason')} "
                              f"msg={str(waiting.get('message', ''))[:120]}")
    return ok


def check_deployments(bundle_dir, namespaces):
    ok = True
    for ns in namespaces:
        path = core_path(bundle_dir, "deployments", ns)
        items = load_json_list(path)
        if not items:
            continue
        for dep in items:
            name    = (dep.get("metadata") or {}).get("name", "?")
            desired = (dep.get("spec") or {}).get("replicas") or 1
            status  = dep.get("status") or {}
            avail   = status.get("availableReplicas") or 0
            if avail < desired:
                ok = False
                print(f"  [FAIL] deployment {ns}/{name}  desired={desired} available={avail}")
                conds = status.get("conditions") or []
                for c in conds:
                    if c.get("status") != "True":
                        print(f"    [{c.get('type')}] {c.get('status')} "
                              f"({c.get('reason', '')}) {str(c.get('message', ''))[:130]}")
    return ok


def check_statefulsets(bundle_dir, namespaces):
    ok = True
    for ns in namespaces:
        path = core_path(bundle_dir, "statefulsets", ns)
        items = load_json_list(path)
        if not items:
            continue
        for sts in items:
            name    = (sts.get("metadata") or {}).get("name", "?")
            desired = (sts.get("spec") or {}).get("replicas") or 1
            status  = sts.get("status") or {}
            ready   = status.get("readyReplicas") or 0
            if ready < desired:
                ok = False
                print(f"  [FAIL] statefulset {ns}/{name}  desired={desired} ready={ready}")
    return ok


def main():
    if len(sys.argv) < 2:
        print(f"Usage: {sys.argv[0]} <bundle-dir> [<namespace> ...]")
        sys.exit(2)
    bundle_dir = sys.argv[1]
    namespaces = sys.argv[2:] if len(sys.argv) > 2 else DEFAULT_NAMESPACES

    print(f"=== Step 12: Workloads (namespaces: {', '.join(namespaces)}) ===")
    ok1 = check_pods(bundle_dir, namespaces)
    ok2 = check_deployments(bundle_dir, namespaces)
    ok3 = check_statefulsets(bundle_dir, namespaces)
    if ok1 and ok2 and ok3:
        print("  All workloads healthy.")
    sys.exit(0 if (ok1 and ok2 and ok3) else 1)


if __name__ == "__main__":
    main()
