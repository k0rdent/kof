"""
Shared helpers for KOF/KCM support-bundle analysis scripts.

Usage in other scripts:
    import sys, os
    sys.path.insert(0, os.path.dirname(__file__))
    from lib import load_yaml_list, load_json_list, bundle_path, FAIL, OK, WARN
"""
import json
import os

try:
    import yaml
    _YAML_OK = True
except ImportError:
    _YAML_OK = False


# ── Colour / label helpers ────────────────────────────────────────────────────

def flag(ok, warn=False):
    if ok:
        return "OK  "
    if warn:
        return "WARN"
    return "FAIL"


OK   = "OK  "
WARN = "WARN"
FAIL = "FAIL"


# ── File loaders ──────────────────────────────────────────────────────────────

def load_yaml_list(path):
    """
    Load a YAML file that is either:
      - a YAML list (root is a list)            → [item, ...]
      - a multi-document stream (--- separated) → [doc, ...]
      - a single dict                           → [dict]

    Returns a list of dicts, skipping None / non-dict entries.
    """
    if not _YAML_OK:
        raise RuntimeError("pyyaml is not installed; run: pip3 install pyyaml --break-system-packages")
    if not os.path.exists(path):
        return []

    items = []
    try:
        with open(path) as f:
            for doc in yaml.safe_load_all(f):
                if isinstance(doc, list):
                    items.extend(d for d in doc if isinstance(d, dict))
                elif isinstance(doc, dict):
                    items.append(doc)
    except yaml.YAMLError as exc:
        print(f"  [WARN] YAML parse error in {path}: {exc}")
    return items


def load_json_list(path):
    """
    Load a JSON file that is a Kubernetes List object {"kind":"List","items":[...]}.
    Returns the items list (may be empty).  Returns [] if the file is missing.
    """
    if not os.path.exists(path):
        return []
    with open(path) as f:
        data = json.load(f)
    if isinstance(data, list):
        return data
    if isinstance(data, dict):
        return data.get("items") or []
    return []


# ── Bundle path helpers ───────────────────────────────────────────────────────

def bundle_path(bundle_dir, *parts):
    """Join bundle_dir with the given path parts, returning the full path."""
    return os.path.join(bundle_dir, *parts)


def cr_path(bundle_dir, group_file, namespace=None):
    """
    Return the path to a custom-resource YAML file inside a bundle.

    Examples:
        cr_path(bd, "managements.k0rdent.mirantis.com.yaml")
          → <bd>/cluster-resources/custom-resources/managements.k0rdent.mirantis.com.yaml

        cr_path(bd, "clusterdeployments.k0rdent.mirantis.com", "kcm-system.yaml")
          → <bd>/cluster-resources/custom-resources/clusterdeployments.k0rdent.mirantis.com/kcm-system.yaml
    """
    base = os.path.join(bundle_dir, "cluster-resources", "custom-resources")
    if namespace:
        return os.path.join(base, group_file, namespace)
    return os.path.join(base, group_file)


def core_path(bundle_dir, kind, namespace):
    """
    Return the path to a core-resource JSON file.

    Example:
        core_path(bd, "pods", "kof")
          → <bd>/cluster-resources/pods/kof.json
    """
    return os.path.join(bundle_dir, "cluster-resources", kind, f"{namespace}.json")


# ── Condition helpers ─────────────────────────────────────────────────────────

def conditions_failing(obj):
    """Return list of condition dicts where status != 'True'."""
    conds = (obj.get("status") or {}).get("conditions") or []
    return [c for c in conds if c.get("status") != "True"]


def get_condition(obj, ctype):
    """Return the first condition of the given type, or None."""
    conds = (obj.get("status") or {}).get("conditions") or []
    return next((c for c in conds if c.get("type") == ctype), None)


def name_ns(obj):
    meta = obj.get("metadata") or {}
    return meta.get("name", "?"), meta.get("namespace", "")
