from __future__ import annotations

import difflib
from dataclasses import dataclass
from pathlib import Path
from typing import Any

import yaml

from framework.prometheus import QueryResult, query_error_message


@dataclass(frozen=True)
class DashboardPolicy:
    """Loaded dashboard query health policy."""

    probe_config: dict[str, Any]
    component_detectors: dict[str, str]
    required_dashboards: dict[str, dict[str, Any]]
    optional_dashboards: dict[str, dict[str, Any]]


def load_dashboard_policy(policy_path: Path) -> DashboardPolicy:
    """Load policy configuration from YAML."""
    raw = _load_raw_policy(policy_path)
    return DashboardPolicy(
        probe_config=raw.get("probe", {}),
        component_detectors=raw.get("component_detectors", {}),
        required_dashboards=raw.get("required_dashboards", {}),
        optional_dashboards=raw.get("optional_dashboards", {}),
    )


def required_dashboard_params(policy_path: Path) -> list[tuple[str, dict[str, Any]]]:
    """Return required dashboard policy entries for pytest parametrization."""
    required = _load_raw_policy(policy_path).get("required_dashboards", {})
    return [
        (str(title), spec if isinstance(spec, dict) else {})
        for title, spec in required.items()
    ]


def optional_component_params(policy_path: Path) -> list[tuple[str, dict[str, Any]]]:
    """Return optional component policy entries for pytest parametrization."""
    optional = _load_raw_policy(policy_path).get("optional_dashboards", {})
    return [
        (str(name), spec if isinstance(spec, dict) else {})
        for name, spec in optional.items()
    ]


def policy_dashboard_titles(policy: DashboardPolicy) -> list[str]:
    """Return all dashboard titles referenced by query health policy."""
    titles: list[str] = list(policy.required_dashboards)
    for spec in policy.optional_dashboards.values():
        titles.extend(title for title, _ in component_dashboards(spec))
    return sorted(dict.fromkeys(titles))


def format_unknown_policy_title(title: str, reference_titles: set[str]) -> str:
    close_matches = difflib.get_close_matches(
        title,
        sorted(reference_titles),
        n=3,
        cutoff=0.6,
    )
    if not close_matches:
        return f"  - {title}"
    return (
        f"  - {title}\n"
        + "\n".join(f"      maybe: {match}" for match in close_matches)
    )


def component_dashboards(spec: dict[str, Any]) -> list[tuple[str, dict[str, Any]]]:
    """Return dashboard policy entries as (title, per_dashboard_spec)."""
    raw = spec.get("dashboards", [])
    if isinstance(raw, dict):
        return [
            (str(title), cfg if isinstance(cfg, dict) else {})
            for title, cfg in raw.items()
        ]
    if not isinstance(raw, list):
        return []

    dashboards: list[tuple[str, dict[str, Any]]] = []
    for item in raw:
        if isinstance(item, str):
            dashboards.append((item, {}))
        elif isinstance(item, dict):
            for title, cfg in item.items():
                dashboards.append((str(title), cfg if isinstance(cfg, dict) else {}))
    return dashboards


def dashboard_probe_spec(policy: DashboardPolicy, title: str) -> dict[str, Any]:
    """Return probe settings for a required or optional dashboard."""
    if title in policy.required_dashboards:
        return policy.required_dashboards[title]

    for component_spec in policy.optional_dashboards.values():
        when_present = _dict_or_empty(component_spec.get("when_present", {}))
        for dashboard_title, dashboard_spec in component_dashboards(component_spec):
            if dashboard_title == title:
                return {**when_present, **dashboard_spec}
    return {}


def required_min_ok(
    dashboard_spec: dict[str, Any],
    detected_components: dict[str, bool],
) -> tuple[int, str | None]:
    """Return required dashboard min_ok_queries with component overrides."""
    return _resolve_min_ok(
        dashboard_spec,
        detected_components,
        default=dashboard_spec["min_ok_queries"],
    )


def optional_min_ok(
    dashboard_spec: dict[str, Any],
    component_when_present: dict[str, Any],
    detected_components: dict[str, bool],
) -> tuple[int, str | None]:
    """Return optional dashboard min_ok_queries with component overrides."""
    return _resolve_min_ok(
        dashboard_spec,
        detected_components,
        default=dashboard_spec.get(
            "min_ok_queries",
            component_when_present.get("min_ok_queries", 1),
        ),
        inherited=component_when_present,
    )


def min_ok_after_allowed_errors(
    results: list[QueryResult],
    min_ok: int,
    allowed_errors: list[dict[str, Any]],
) -> tuple[int, int]:
    """Adjust min_ok when allowed errors reduce the executable query pool."""
    allowed_count = sum(
        1 for result in results
        if (error := query_error_message(result))
        and error_matches_any(result, error, allowed_errors)
    )
    effective_total = len(results) - allowed_count
    return min(max(0, min_ok - allowed_count), effective_total), allowed_count


def is_allowed_policy_error(
    result: QueryResult,
    error: str,
    policy: DashboardPolicy,
    detected_components: dict[str, bool],
) -> bool:
    """Return True if dashboard policy explicitly allows this query error."""
    required_spec = policy.required_dashboards.get(result.dashboard_title, {})
    if isinstance(required_spec, dict):
        allowed = direct_allowed_errors(required_spec)
        if error_matches_any(result, error, allowed):
            return True

    for component_name, spec in policy.optional_dashboards.items():
        is_present = detected_components.get(component_name, False)
        for dashboard_title, dashboard_spec in component_dashboards(spec):
            if dashboard_title != result.dashboard_title:
                continue
            allowed = allowed_error_specs(spec, dashboard_spec, is_present)
            if error_matches_any(result, error, allowed):
                return True
    return False


def allowed_error_specs(
    component_spec: dict[str, Any],
    dashboard_spec: dict[str, Any],
    is_present: bool,
) -> list[dict[str, Any]]:
    """Return error allowances for the active optional component state."""
    state_key = "when_present" if is_present else "when_absent"
    state = _dict_or_empty(component_spec.get(state_key, {}))

    allowed: list[dict[str, Any]] = []
    for source in (state, dashboard_spec):
        allowed.extend(direct_allowed_errors(source))
    return allowed


def direct_allowed_errors(spec: dict[str, Any]) -> list[dict[str, Any]]:
    """Return directly declared allowed_errors entries from a policy spec."""
    raw = spec.get("allowed_errors", [])
    if not isinstance(raw, list):
        return []
    return [item for item in raw if isinstance(item, dict)]


def error_matches_any(
    result: QueryResult,
    error: str,
    entries: list[dict[str, Any]],
) -> bool:
    return any(error_matches_entry(result, error, entry) for entry in entries)


def error_matches_entry(
    result: QueryResult,
    error: str,
    entry: dict[str, Any],
) -> bool:
    has_matcher = any(
        entry.get(key)
        for key in ("dashboard", "panel", "message_contains", "expr_contains")
    )
    if not has_matcher:
        return False

    dashboard = entry.get("dashboard")
    if dashboard and dashboard != result.dashboard_title:
        return False

    panel = entry.get("panel")
    if panel and panel != result.panel_title:
        return False

    message_contains = entry.get("message_contains")
    if message_contains and str(message_contains) not in error:
        return False

    expr_contains = entry.get("expr_contains")
    if expr_contains and str(expr_contains) not in result.expr:
        return False

    return True


def _load_raw_policy(policy_path: Path) -> dict[str, Any]:
    if not policy_path.exists():
        return {}
    with open(policy_path) as f:
        raw = yaml.safe_load(f) or {}
    return raw if isinstance(raw, dict) else {}


def _resolve_min_ok(
    dashboard_spec: dict[str, Any],
    detected_components: dict[str, bool],
    *,
    default: Any,
    inherited: dict[str, Any] | None = None,
) -> tuple[int, str | None]:
    base = int(default)
    overrides: dict[str, Any] = {}

    inherited_overrides = (inherited or {}).get("min_ok_queries_by_component", {})
    if isinstance(inherited_overrides, dict):
        overrides.update(inherited_overrides)

    dashboard_overrides = dashboard_spec.get("min_ok_queries_by_component", {})
    if isinstance(dashboard_overrides, dict):
        overrides.update(dashboard_overrides)

    for component_name, min_ok in overrides.items():
        if detected_components.get(str(component_name), False):
            return int(min_ok), str(component_name)

    return base, None


def _dict_or_empty(value: Any) -> dict[str, Any]:
    return value if isinstance(value, dict) else {}
