#!/usr/bin/env python3
"""
Step 14 — Gateway API resource health.

Checks Gateway, GatewayClass, and HTTPRoute resources in the bundle.
Relevant for deployments using Envoy Gateway (kof-mothership regionless mode,
kof-storage, etc.).

Usage:
    python3 step14_gateway.py <bundle-dir>

Exits 0 if all Gateway resources are healthy, 1 if any failures.

Failure signals:
- No Gateway objects found when HTTPRoutes exist (Gateway not yet created /
  kof-mothership install still in progress)
- Gateway condition Programmed != True
- Gateway has no address assigned (LoadBalancer IP not provisioned)
- HTTPRoute has empty/failing parent conditions
- GatewayClass condition Accepted != True
"""
import sys
import os
import json

sys.path.insert(0, os.path.dirname(__file__))
from lib import load_json_list, load_yaml_list, cr_path, flag, WARN, FAIL, OK


def _load_cr_json(bundle_dir, filename):
    """Load a cluster-scoped or namespaced CR JSON file, returning items list."""
    path = os.path.join(
        bundle_dir, "cluster-resources", "custom-resources", filename
    )
    return load_json_list(path)


def _load_namespaced_json(bundle_dir, group_dir, namespace):
    path = cr_path(bundle_dir, group_dir, f"{namespace}.json")
    return load_json_list(path)


def check_gateway_classes(bundle_dir):
    ok = True
    # GatewayClass is cluster-scoped
    path = os.path.join(
        bundle_dir, "cluster-resources", "custom-resources",
        "gatewayclasses.gateway.networking.k8s.io.json"
    )
    items = load_json_list(path)
    if not items:
        print(f"  [WARN] No GatewayClass objects found (CRD may not be installed or not collected)")
        return True  # not a hard failure on its own

    for gc in items:
        name = (gc.get("metadata") or {}).get("name", "?")
        conditions = (gc.get("status") or {}).get("conditions") or []
        accepted = next((c for c in conditions if c.get("type") == "Accepted"), None)
        if accepted and accepted.get("status") == "True":
            print(f"  [OK  ] GatewayClass/{name}  Accepted=True")
        else:
            ok = False
            msg = (accepted or {}).get("message", "no Accepted condition")
            print(f"  [FAIL] GatewayClass/{name}  Accepted != True: {msg}")
    return ok


def check_gateways(bundle_dir):
    ok = True
    found_any = False

    # Gateways are namespaced; check kof namespace primarily
    gw_group = "gateways.gateway.networking.k8s.io"
    for ns in ["kof", "kcm-system", "envoy-gateway-system", "istio-system"]:
        items = _load_namespaced_json(bundle_dir, gw_group, ns)
        if not items:
            continue
        found_any = True
        for gw in items:
            name = (gw.get("metadata") or {}).get("name", "?")
            status = gw.get("status") or {}
            conditions = status.get("conditions") or []
            addresses = status.get("addresses") or []

            programmed = next((c for c in conditions if c.get("type") == "Programmed"), None)
            prog_ok = programmed and programmed.get("status") == "True"

            ip = addresses[0].get("value", "") if addresses else ""
            addr_ok = bool(ip)

            healthy = prog_ok and addr_ok
            if not healthy:
                ok = False
            f = flag(healthy)
            print(f"  [{f}] Gateway/{ns}/{name}  Programmed={prog_ok}  address={ip or '(none)'}")
            if not prog_ok and programmed:
                print(f"    reason={programmed.get('reason')} msg={str(programmed.get('message',''))[:150]}")
            elif not prog_ok:
                print(f"    no Programmed condition found")
            if not addr_ok:
                print(f"    [WARN] no address assigned — LoadBalancer IP not yet provisioned")

    if not found_any:
        # Check if HTTPRoutes exist referencing a gateway — if so, the missing Gateway is a problem
        httproutes = _get_all_httproutes(bundle_dir)
        if httproutes:
            print(f"  [FAIL] No Gateway objects found, but {len(httproutes)} HTTPRoute(s) exist referencing a gateway")
            print(f"         This means kof-mothership Helm install has not yet created the Gateway resource")
            print(f"         (install may still be in progress or blocked by a failing workload)")
            ok = False
        else:
            print(f"  [WARN] No Gateway objects found in bundle (gateway.enabled=false or not yet created)")
    return ok


def _get_all_httproutes(bundle_dir):
    """Return all HTTPRoute items across all collected namespaces."""
    items = []
    group_dir = os.path.join(
        bundle_dir, "cluster-resources", "custom-resources",
        "httproutes.gateway.networking.k8s.io"
    )
    if not os.path.isdir(group_dir):
        return items
    for fname in sorted(os.listdir(group_dir)):
        if fname.endswith(".json"):
            path = os.path.join(group_dir, fname)
            items.extend(load_json_list(path))
    return items


def check_httproutes(bundle_dir):
    ok = True
    items = _get_all_httproutes(bundle_dir)

    if not items:
        print(f"  [WARN] No HTTPRoute objects found")
        return True

    for r in items:
        name = (r.get("metadata") or {}).get("name", "?")
        ns   = (r.get("metadata") or {}).get("namespace", "?")
        spec = r.get("spec") or {}
        hostnames = spec.get("hostnames") or []
        parents   = (r.get("status") or {}).get("parents") or []

        # An HTTPRoute with no parent status means the Gateway hasn't accepted it yet
        if not parents:
            ok = False
            print(f"  [FAIL] HTTPRoute/{ns}/{name}  hostnames={hostnames}  no parent status (Gateway not ready)")
            continue

        route_ok = True
        for p in parents:
            p_ref = p.get("parentRef") or {}
            p_name = p_ref.get("name", "?")
            p_ns   = p_ref.get("namespace", ns)
            conditions = p.get("conditions") or []
            accepted = next((c for c in conditions if c.get("type") == "Accepted"), None)
            resolved = next((c for c in conditions if c.get("type") == "ResolvedRefs"), None)
            a_ok = accepted and accepted.get("status") == "True"
            r_ok = resolved is None or resolved.get("status") == "True"
            if not (a_ok and r_ok):
                route_ok = False
                ok = False
                print(f"  [FAIL] HTTPRoute/{ns}/{name}  parent={p_ns}/{p_name}")
                if not a_ok:
                    print(f"    Accepted={accepted}")
                if not r_ok:
                    print(f"    ResolvedRefs={resolved}")
            else:
                print(f"  [OK  ] HTTPRoute/{ns}/{name}  hostnames={hostnames}  parent={p_ns}/{p_name}")
    return ok


def check_gateway(bundle_dir):
    ok1 = check_gateway_classes(bundle_dir)
    ok2 = check_gateways(bundle_dir)
    ok3 = check_httproutes(bundle_dir)
    return ok1 and ok2 and ok3


def main():
    if len(sys.argv) < 2:
        print(f"Usage: {sys.argv[0]} <bundle-dir>")
        sys.exit(2)
    bundle_dir = sys.argv[1]
    print("=== Step 14: Gateway API resources ===")
    ok = check_gateway(bundle_dir)
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
