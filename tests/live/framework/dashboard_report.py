from __future__ import annotations

from typing import Any

from framework.dashboard_policy import (
    DashboardPolicy,
    allowed_error_specs,
    component_dashboards,
    direct_allowed_errors,
    is_allowed_policy_error,
    min_ok_after_allowed_errors,
    optional_min_ok,
    required_min_ok,
)
from framework.dashboard_probe_session import ProbeResults
from framework.prometheus import QueryResult, query_error_message, query_has_data


def build_dashboard_health_report(
    policy: DashboardPolicy,
    probe: ProbeResults,
    detected_components: dict[str, bool],
) -> str:
    """Build a concise terminal report for CI output."""
    ok_count, total, no_data_count, error_count = _dashboard_counts(probe.results)
    lines = [
        "Dashboard Query Health Report",
        "-" * 29,
        (
            f"Summary: dashboards={len(probe.probed_dashboards)}, "
            f"queries={total}, OK={ok_count}, "
            f"no_data={no_data_count}, errors={error_count}, "
            f"warnings={len(probe.warnings)}"
        ),
    ]

    for title, specs in (
        ("Topology", policy.component_detectors),
        ("Optional components", policy.optional_dashboards),
    ):
        if state := _component_state_line(title, specs, detected_components):
            lines.append(state)

    required_widths = [62, 12, 9, 6, 6]
    optional_widths = [22, 7, 52, 13, 9, 6, 6]
    lines += [
        "",
        "Required dashboards",
        _table_line(
            ["Dashboard", "Expected", "OK/Total", "NoData", "Errors"],
            required_widths,
            right={1, 2, 3, 4},
        ),
        "-" * (sum(required_widths) + len(required_widths) - 1),
    ]

    for dashboard_title, dashboard_spec in policy.required_dashboards.items():
        results = probe.dashboard_results.get(dashboard_title, [])
        min_ok, override_component = required_min_ok(
            dashboard_spec,
            detected_components,
        )
        expected = _expected(
            min_ok_after_allowed_errors(
                results,
                min_ok,
                direct_allowed_errors(dashboard_spec),
            )[0],
            min_ok,
            override_component,
        )
        lines.append(_table_line(
            [
                dashboard_title,
                expected,
                _ok_total(results),
                _dashboard_counts(results)[2],
                _dashboard_counts(results)[3],
            ],
            required_widths,
            right={1, 2, 3, 4},
        ))

    lines += [
        "",
        "Optional dashboards",
        _table_line(
            [
                "Component", "State", "Dashboard", "Expected",
                "OK/Total", "NoData", "Errors",
            ],
            optional_widths,
            right={1, 3, 4, 5, 6},
        ),
        "-" * (sum(optional_widths) + len(optional_widths) - 1),
    ]

    for component_name, spec in policy.optional_dashboards.items():
        is_present = detected_components.get(component_name, False)
        state = "present" if is_present else "absent"
        when_present = spec.get("when_present", {})
        if not isinstance(when_present, dict):
            when_present = {}

        for dashboard_title, dashboard_spec in component_dashboards(spec):
            results = probe.dashboard_results.get(dashboard_title, [])
            if not is_present:
                expected = "no data ok"
            elif dashboard_spec.get("allow_no_data", when_present.get("allow_no_data", False)):
                expected = "data optional"
            else:
                min_ok, override_component = optional_min_ok(
                    dashboard_spec,
                    when_present,
                    detected_components,
                )
                expected = _expected(
                    min_ok_after_allowed_errors(
                        results,
                        min_ok,
                        allowed_error_specs(spec, dashboard_spec, is_present=True),
                    )[0],
                    min_ok,
                    override_component,
                )

            lines.append(_table_line(
                [
                    component_name,
                    state,
                    dashboard_title,
                    expected,
                    _ok_total(results),
                    _dashboard_counts(results)[2],
                    _dashboard_counts(results)[3],
                ],
                optional_widths,
                right={1, 3, 4, 5, 6},
            ))

    lines.extend(_error_report(policy, probe, detected_components))
    return "\n".join(lines)


def _dashboard_counts(results: list[QueryResult]) -> tuple[int, int, int, int]:
    ok_count = sum(1 for result in results if query_has_data(result))
    error_count = sum(1 for result in results if query_error_message(result))
    total = len(results)
    return ok_count, total, total - ok_count - error_count, error_count


def _ok_total(results: list[QueryResult]) -> str:
    ok_count, total, _, _ = _dashboard_counts(results)
    return f"{ok_count}/{total}"


def _component_state_line(
    title: str,
    specs: dict[str, Any],
    detected_components: dict[str, bool],
) -> str:
    if not specs:
        return ""

    present = sorted(name for name in specs if detected_components.get(name, False))
    absent = sorted(name for name in specs if not detected_components.get(name, False))
    return (
        f"{title}: present({len(present)}): {_join_names(present)}; "
        f"absent({len(absent)}): {_join_names(absent)}"
    )


def _join_names(names: list[str], max_items: int = 8) -> str:
    if not names:
        return "-"
    suffix = f", +{len(names) - max_items} more" if len(names) > max_items else ""
    return ", ".join(names[:max_items]) + suffix


def _expected(effective_min_ok: int, original_min_ok: int, component: str | None) -> str:
    value = f">={effective_min_ok}"
    if effective_min_ok != original_min_ok:
        value += f" of {original_min_ok}"
    if component:
        value += f" ({component})"
    return value


def _error_report(
    policy: DashboardPolicy,
    probe: ProbeResults,
    detected_components: dict[str, bool],
) -> list[str]:
    allowed: list[tuple[QueryResult, str]] = []
    unexpected: list[tuple[QueryResult, str]] = []

    for result in probe.results:
        error = query_error_message(result)
        if not error:
            continue
        if is_allowed_policy_error(result, error, policy, detected_components):
            allowed.append((result, error))
        else:
            unexpected.append((result, error))

    lines = ["", "Allowed errors"]
    if not allowed and not unexpected:
        return lines + ["  none"]

    if allowed:
        lines.append(f"  allowed: {len(allowed)}")
        for label, count in _group_errors_by_dashboard(allowed)[:8]:
            lines.append(f"    - {label}: {count}")

    if unexpected:
        lines.append(f"  unexpected: {len(unexpected)}")
        for result, error in unexpected[:8]:
            lines.append(
                f"    - {_clip(result.dashboard_title, 45)} / "
                f"{_clip(result.panel_title, 35)}: {_clip(error, 80)}"
            )

    return lines


def _group_errors_by_dashboard(errors: list[tuple[QueryResult, str]]) -> list[tuple[str, int]]:
    grouped: dict[str, int] = {}
    for result, _ in errors:
        grouped[result.dashboard_title] = grouped.get(result.dashboard_title, 0) + 1
    return sorted(grouped.items(), key=lambda item: item[0])


def _table_line(values: list[Any], widths: list[int], right: set[int]) -> str:
    cells = []
    for index, (value, width) in enumerate(zip(values, widths)):
        cell = _clip(str(value), width)
        cells.append(f"{cell:>{width}}" if index in right else f"{cell:<{width}}")
    return " ".join(cells)


def _clip(value: str, width: int) -> str:
    if len(value) <= width:
        return value
    if width <= 3:
        return value[:width]
    return value[:width - 3] + "..."
