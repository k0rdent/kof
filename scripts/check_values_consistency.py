"""
Check that a subset of important defaults in `charts/kof/values.yaml` (main values)
matches corresponding `values.yaml` inside component charts.

Usage:
  scripts/check_values_consistency.py [--main MAIN] [--charts CHARTS] [--verbose]

By default, MAIN=charts/kof/values.yaml and CHARTS=charts

Special-case behavior for `kof-child` and `kof-regional`:
  If a value path under their `.values` is not present in their own chart values,
  the script will try to locate the value in a related sub-chart named after the
  first path segment (e.g. `collectors` -> `charts/kof-collectors/values.yaml`).

Exit code: 0 on success, 2 on fatal error, 3 if mismatches/errors found.
"""
from __future__ import annotations

import argparse
import sys
from pathlib import Path
from typing import Any, List, Optional, Tuple

try:
    import yaml
except ImportError:
    yaml = None


def load_yaml(path: Path) -> Any:
    if yaml is None:
        raise RuntimeError("pyyaml is required: pip install pyyaml")
    return yaml.safe_load(path.read_text(encoding="utf-8"))


def flatten_leaves(d: Any, prefix: List[str] = None) -> List[Tuple[List[str], Any]]:
    """Return list of (path_list, value) for all non-dict leaves in d."""
    if prefix is None:
        prefix = []
    flattened_items: List[Tuple[List[str], Any]] = []
    if isinstance(d, dict):
        for key, value in d.items():
            flattened_items.extend(flatten_leaves(value, prefix + [str(key)]))
    else:
        flattened_items.append((prefix, d))
    return flattened_items


def get_by_path(data: Any, path: List[str]) -> Tuple[bool, Any]:
    current_node = data
    for segment in path:
        if not isinstance(current_node, dict) or segment not in current_node:
            return False, None
        current_node = current_node[segment]
    return True, current_node


def comp_candidates(charts_dir: Path, component_key: str) -> List[Path]:
    candidate = charts_dir / component_key / "values.yaml"
    return [candidate] if candidate.exists() else []


def find_fallback_chart(charts_dir: Path, first_path_segment: str) -> Optional[Path]:
    prefixed_candidate = charts_dir / f"kof-{first_path_segment}" / "values.yaml"
    if prefixed_candidate.exists():
        return prefixed_candidate
    return None


def main() -> int:
    argument_parser = argparse.ArgumentParser()
    argument_parser.add_argument("--main", default="charts/kof/values.yaml", help="main kof values file to check")
    argument_parser.add_argument("--charts", default="charts", help="charts directory")
    argument_parser.add_argument("--verbose", action="store_true")
    args = argument_parser.parse_args()

    script_path = Path(__file__).resolve()
    repo_root = script_path.parents[1]

    main_values_path = repo_root / args.main
    charts_root_dir = repo_root / args.charts

    if not main_values_path.exists():
        print(f"ERROR: main values file not found: {main_values_path}")
        return 2
    if not charts_root_dir.exists():
        print(f"ERROR: charts dir not found: {charts_root_dir}")
        return 2

    if yaml is None:
        print("ERROR: pyyaml not installed. Please run: pip install pyyaml")
        return 2

    try:
        main_values_yaml = load_yaml(main_values_path) or {}
    except Exception as exception:
        print(f"ERROR: failed to parse {main_values_path}: {exception}")
        return 2

    global_block = main_values_yaml.get("global", {}) if isinstance(main_values_yaml, dict) else {}
    managed_components = global_block.get("components", []) if isinstance(global_block, dict) else []

    if not isinstance(managed_components, list):
        print("ERROR: global.components must be a list")
        return 2
    errors: List[str] = []

    # iterate only components from global.components
    for component_key in managed_components:
        component_block = main_values_yaml.get(component_key) if isinstance(main_values_yaml, dict) else None
        if not isinstance(component_block, dict):
            continue
        # Skip syncing for components explicitly disabled in main (UX: pre-configured use-case).
        # Exception: do NOT skip kof-child/kof-regional here, they rely on fallback validation logic.
        if component_key in {"kof-storage", "kof-collectors"}:
            enabled_flag = component_block.get("enabled", None)
            if enabled_flag is False:
                if args.verbose:
                    print(f"Skipping disabled component {component_key} (enabled: false)")
                continue
        if "values" not in component_block:
            continue

        component_values_block = component_block.get("values") or {}
        if not isinstance(component_values_block, dict):
            continue

        value_leaves = flatten_leaves(component_values_block)
        if args.verbose:
            print(f"Checking component {component_key}: {len(value_leaves)} value(s)")

        # load candidate component charts
        component_values_files = comp_candidates(charts_root_dir, component_key)
        component_chart_values_yaml = None
        component_chart_values_path = None
        if component_values_files:
            component_chart_values_path = component_values_files[0]
            try:
                component_chart_values_yaml = load_yaml(component_chart_values_path) or {}
            except Exception as exception:
                errors.append(f"Failed to load {component_chart_values_path}: {exception}")
                continue

        for value_path_segments, main_value in value_leaves:
            # attempt to read from component chart as-is
            value_found = False
            component_value = None

            resolved_component_file: Optional[Path] = component_chart_values_path
            used_fallback = False
            fallback_first_segment: Optional[str] = None

            if component_chart_values_yaml is not None:
                path_exists, found_value = get_by_path(component_chart_values_yaml, value_path_segments)
                if path_exists:
                    value_found = True
                    component_value = found_value
                    resolved_component_file = component_chart_values_path

            # special-case fallback for kof-child / kof-regional
            if not value_found and component_key in {"kof-child", "kof-regional"} and value_path_segments:
                first_path_segment = value_path_segments[0]
                fallback_values_path = find_fallback_chart(charts_root_dir, first_path_segment)
                if fallback_values_path:
                    try:
                        fallback_values_yaml = load_yaml(fallback_values_path) or {}
                        fallback_path_exists, fallback_value = (
                            get_by_path(fallback_values_yaml, value_path_segments[1:])
                            if len(value_path_segments) > 1
                            else get_by_path(fallback_values_yaml, [first_path_segment])
                        )
                        if fallback_path_exists:
                            value_found = True
                            component_value = fallback_value
                            resolved_component_file = fallback_values_path
                            used_fallback = True
                            fallback_first_segment = first_path_segment
                    except Exception as exception:
                        if args.verbose:
                            errors.append(
                                f"Fallback error for {component_key}.values.{'.'.join(value_path_segments)} "
                                f"in {fallback_values_path}: {exception}"
                            )

            source_path = resolved_component_file or component_chart_values_path

            # Ignore missing enable/disable flags in component charts.
            # These are commonly defined only in the parent chart values.
            if not value_found and value_path_segments and value_path_segments[-1] == "enabled":
                continue

            if not value_found:
                errors.append(
                    f"MISSING: main -> {component_key}.values.{'.'.join(value_path_segments)} exists but not found "
                    f"in component charts (checked {source_path or 'none'})"
                )
                continue

            main_compare_value = main_value
            main_compare_path = f"{component_key}.values.{'.'.join(value_path_segments)}"

            # compare
            if main_compare_value != component_value:
                errors.append(
                    f"DIFFER: {main_compare_path} -> "
                    f"main={main_compare_value!r} vs chart={component_value!r} "
                    f"(component file: {source_path})"
                )

    if errors:
        print("Values consistency check FAILED:\n")
        for error in errors:
            print(" - ", error)
        print(f"\nTotal issues: {len(errors)}")
        return 3

    print("Values consistency check passed: all checked values match component charts.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
