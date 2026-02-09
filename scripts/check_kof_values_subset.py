import argparse
import yaml
import json
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Dict, Iterable, List, Sequence, Tuple, Union

Scalar = Union[str, int, float, bool, None]

KOF_ROOT = Path(__file__).resolve().parents[1]
CHARTS_ROOT = KOF_ROOT / "charts"
AGGREGATOR_VALUES_PATH = CHARTS_ROOT / "kof" / "values.yaml"

@dataclass(frozen=True)
class Mismatch:
    component: str
    subset_path: str
    aggregator_value: Any
    component_value: Any
    component_values_file: str


def load_yaml_file(yaml_path: Path) -> Any:
    return yaml.safe_load(yaml_path.read_text(encoding="utf-8"))


def is_scalar(value: Any) -> bool:
    return value is None or isinstance(value, (str, int, float, bool))


def iter_scalar_leaves(node: Any, current_path: List[Union[str, int]]) -> Iterable[Tuple[List[Union[str, int]], Scalar]]:
    """
    Yield (path_segments, scalar_value) for every scalar leaf under node.

    Supports:
      - dict keys (strings)
      - list indices (ints)
    """
    if is_scalar(node):
        yield (current_path, node)
        return

    if isinstance(node, dict):
        for key, value in node.items():
            # YAML keys should be strings for our case; keep robust anyway
            key_str = str(key)
            yield from iter_scalar_leaves(value, current_path + [key_str])
        return

    if isinstance(node, list):
        for index, value in enumerate(node):
            yield from iter_scalar_leaves(value, current_path + [index])
        return

    # Unknown type (e.g., custom object) – ignore, but this should not happen with YAML safe_load
    return


def format_path(path_segments: Sequence[Union[str, int]]) -> str:
    """
    Convert ["grafana","enabled"] -> grafana.enabled
    Convert ["clients",0,"id"] -> clients[0].id
    """
    parts: List[str] = []
    for segment in path_segments:
        if isinstance(segment, int):
            if not parts:
                parts.append(f"[{segment}]")
            else:
                parts[-1] = parts[-1] + f"[{segment}]"
        else:
            parts.append(segment)
    return ".".join(parts)


def get_by_path(root: Any, path_segments: Sequence[Union[str, int]]) -> Tuple[bool, Any]:
    """
    Return (found, value) for a nested path in dict/list.
    """
    current = root
    for segment in path_segments:
        if isinstance(segment, int):
            if not isinstance(current, list):
                return (False, None)
            if segment < 0 or segment >= len(current):
                return (False, None)
            current = current[segment]
        else:
            if not isinstance(current, dict):
                return (False, None)
            if segment not in current:
                return (False, None)
            current = current[segment]
    return (True, current)


def to_compact_json(value: Any) -> str:
    # For stable mismatch output only
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False)


def read_global_components(aggregator_values: Dict[str, Any]) -> List[str]:
    global_section = aggregator_values.get("global") if isinstance(aggregator_values, dict) else None
    if not isinstance(global_section, dict):
        return []
    components = global_section.get("components")
    if not isinstance(components, list):
        return []
    result: List[str] = []
    for item in components:
        if isinstance(item, str) and item.strip():
            result.append(item.strip())
    return result


def run_check() -> int:
    aggregator_values = load_yaml_file(AGGREGATOR_VALUES_PATH)

    components = read_global_components(aggregator_values)

    mismatches: List[Mismatch] = []
    checked_leaves_count = 0
    skipped_components: List[str] = []

    for component_name in components:
        component_values_path = CHARTS_ROOT / component_name / "values.yaml"
        component_values = load_yaml_file(component_values_path)

        # Subset lives at: .<component>.values in aggregator
        component_section = aggregator_values.get(component_name)
        if not isinstance(component_section, dict) or "values" not in component_section:
            # Subset might be intentionally absent; skip but report.
            skipped_components.append(component_name)
            continue

        subset_values = component_section.get("values")
        if subset_values is None:
            skipped_components.append(component_name)
            continue

        for path_segments, subset_scalar_value in iter_scalar_leaves(subset_values, current_path=[]):
            checked_leaves_count += 1

            found, component_scalar_value = get_by_path(component_values, path_segments)

            if not found:
                mismatches.append(
                    Mismatch(
                        component=component_name,
                        subset_path=format_path(path_segments),
                        aggregator_value=subset_scalar_value,
                        component_value="__MISSING__",
                        component_values_file=str(component_values_path),
                    )
                )
                continue

            # Compare scalar values exactly (safe + predictable).
            # If component value is non-scalar, that's a mismatch by design.
            if component_scalar_value != subset_scalar_value:
                mismatches.append(
                    Mismatch(
                        component=component_name,
                        subset_path=format_path(path_segments),
                        aggregator_value=subset_scalar_value,
                        component_value=component_scalar_value,
                        component_values_file=str(component_values_path),
                    )
                )

    # Output
    print("KOF values subset consistency check")
    print(f"Aggregator: {AGGREGATOR_VALUES_PATH}")
    print(f"Components (from global.components): {len(components)}")
    print(f"Checked subset scalar leaves: {checked_leaves_count}")
    if skipped_components:
        print(f"Skipped components (no .<component>.values in aggregator): {', '.join(skipped_components)}")

    if not mismatches:
        print("PASSED: all subset values match component values.yaml")
        return 0

    print(f"\nFAILED: found {len(mismatches)} mismatch(es)\n")
    for mismatch in mismatches:
        print(f"- component: {mismatch.component}")
        print(f"  path:      {mismatch.subset_path}")
        print(f"  file:      {mismatch.component_values_file}")
        print(f"  aggregator: {to_compact_json(mismatch.aggregator_value)}")
        print(f"  component : {to_compact_json(mismatch.component_value)}")
        print()
    return 1


def main():
    run_check()



if __name__ == "__main__":
    raise SystemExit(main())
