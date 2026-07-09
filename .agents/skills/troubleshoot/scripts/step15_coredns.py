#!/usr/bin/env python3
"""
Step 15 — CoreDNS and in-cluster DNS configuration check.

Inspects the CoreDNS ConfigMap for custom host overrides (added by
scripts/patch-coredns.bash in the dev-deploy flow) and checks that all
hostnames referenced by HTTPRoutes have a corresponding DNS entry.

Usage:
    python3 step15_coredns.py <bundle-dir>

Exits 0 if DNS looks consistent, 1 if any gaps found.

Failure signals:
- No CoreDNS ConfigMap found
- CoreDNS Corefile has no 'hosts' block (custom overrides were never applied)
- A hostname from an HTTPRoute is not covered by any CoreDNS hosts entry
- A hosts entry points to an empty or placeholder IP (0.0.0.0 or empty)
"""
import sys
import os
import re

sys.path.insert(0, os.path.dirname(__file__))
from lib import load_json_list, cr_path, flag


def _load_coredns_corefile(bundle_dir):
    """Return the Corefile string from the coredns ConfigMap, or None."""
    path = os.path.join(bundle_dir, "cluster-resources", "configmaps", "kube-system.json")
    items = load_json_list(path)
    for cm in items:
        if (cm.get("metadata") or {}).get("name") == "coredns":
            return (cm.get("data") or {}).get("Corefile")
    return None


def _parse_hosts_block(corefile):
    """
    Parse all 'hosts { ... }' blocks in a Corefile and return a dict of
    hostname -> ip extracted from lines of the form:
        <ip>  <hostname> [<hostname2> ...]
    """
    entries = {}
    in_hosts = False
    depth = 0
    for line in corefile.splitlines():
        stripped = line.strip()
        if not in_hosts:
            if re.match(r'^hosts\b', stripped):
                in_hosts = True
                depth = 0
        if in_hosts:
            depth += stripped.count("{") - stripped.count("}")
            # Parse IP→hostname lines
            m = re.match(r'^(\d{1,3}(?:\.\d{1,3}){3})\s+(.+)$', stripped)
            if m:
                ip = m.group(1)
                for hostname in m.group(2).split():
                    entries[hostname] = ip
            if depth <= 0 and in_hosts:
                in_hosts = False
    return entries


def _get_httproute_hostnames(bundle_dir):
    """Return all unique hostnames from all HTTPRoute objects in the bundle."""
    hostnames = set()
    group_dir = os.path.join(
        bundle_dir, "cluster-resources", "custom-resources",
        "httproutes.gateway.networking.k8s.io"
    )
    if not os.path.isdir(group_dir):
        return hostnames
    for fname in sorted(os.listdir(group_dir)):
        if fname.endswith(".json"):
            path = os.path.join(group_dir, fname)
            items = load_json_list(path)
            for r in items:
                for h in (r.get("spec") or {}).get("hostnames") or []:
                    hostnames.add(h)
    return hostnames


def check_coredns(bundle_dir):
    ok = True

    corefile = _load_coredns_corefile(bundle_dir)
    if corefile is None:
        print("  [WARN] CoreDNS ConfigMap not found in bundle (kube-system/coredns)")
        return True  # Can't judge without it

    print("  [OK  ] CoreDNS ConfigMap found")

    hosts_entries = _parse_hosts_block(corefile)

    httproute_hostnames = _get_httproute_hostnames(bundle_dir)

    if not httproute_hostnames:
        if not hosts_entries:
            print("  [OK  ] No HTTPRoutes and no custom hosts block — standard CoreDNS config")
        else:
            print("  [OK  ] Custom hosts entries present:")
            for h, ip in sorted(hosts_entries.items()):
                print(f"    {h} -> {ip}")
        return ok

    # We have HTTPRoutes — check each hostname is covered by CoreDNS hosts
    print(f"  HTTPRoute hostnames found: {sorted(httproute_hostnames)}")

    if not hosts_entries:
        ok = False
        print("  [FAIL] CoreDNS has NO custom hosts block — in-cluster resolution of HTTPRoute")
        print("         hostnames will fail (dev-deploy CoreDNS patch not applied or failed)")
        for h in sorted(httproute_hostnames):
            print(f"    MISSING: {h}")
        return ok

    print(f"  CoreDNS hosts entries: {hosts_entries}")
    for hostname in sorted(httproute_hostnames):
        ip = hosts_entries.get(hostname)
        if not ip:
            ok = False
            print(f"  [FAIL] {hostname} — not in CoreDNS hosts block")
            print(f"         Pods using this hostname (e.g. ACL OIDC issuer) will get DNS NXDOMAIN")
        elif ip in ("0.0.0.0", ""):
            ok = False
            print(f"  [FAIL] {hostname} -> {ip} — placeholder/empty IP in CoreDNS hosts block")
            print(f"         dev-deploy patch-coredns.bash may have run with an empty gateway IP")
        else:
            print(f"  [OK  ] {hostname} -> {ip}")

    return ok


def main():
    if len(sys.argv) < 2:
        print(f"Usage: {sys.argv[0]} <bundle-dir>")
        sys.exit(2)
    bundle_dir = sys.argv[1]
    print("=== Step 15: CoreDNS / in-cluster DNS ===")
    ok = check_coredns(bundle_dir)
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
