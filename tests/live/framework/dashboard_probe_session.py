from __future__ import annotations

import logging
import time
from collections.abc import Callable
from dataclasses import dataclass, field
from typing import Any

from framework.component_detection import detect_components
from framework.dashboard_policy import (
    DashboardPolicy,
    component_dashboards,
    dashboard_probe_spec,
    optional_min_ok,
)
from framework.grafana import GrafanaClient, GrafanaDashboard
from framework.kubernetes import KubectlClient
from framework.prometheus import (
    QueryResult,
    TimeRange,
    is_prometheus_dashboard,
    probe_dashboard,
    query_error_message,
    query_has_data,
)
from framework.victoria_logs import (
    is_victoria_logs_dashboard,
    probe_victoria_logs_dashboard,
)

logger = logging.getLogger(__name__)
DashboardProbe = Callable[..., tuple[list[QueryResult], list[str]]]

# When fast_retry is enabled, every retry window (per-dashboard and
# optional-present) is capped at this many seconds instead of the value
# configured in dashboard_query_policy.yaml. Intended for fast local
# iteration, not CI.
_FAST_RETRY_CAP_SECONDS = 20


@dataclass
class ProbeResults:
    """Aggregated results from probing all dashboards."""

    results: list[QueryResult] = field(default_factory=list)
    warnings: list[str] = field(default_factory=list)
    dashboard_results: dict[str, list[QueryResult]] = field(default_factory=dict)
    probed_dashboards: list[str] = field(default_factory=list)
    fetch_errors: list[str] = field(default_factory=list)


def run_dashboard_probe_session(
    grafana_client: GrafanaClient,
    policy: DashboardPolicy,
    kubectl_client: KubectlClient,
    *,
    fast_retry: bool = False,
) -> tuple[ProbeResults, dict[str, bool]]:
    """Run one shared probe pass across all supported dashboards."""
    minutes = int(policy.probe_config.get("time_range_minutes", 120))
    max_queries_raw = policy.probe_config.get("max_queries_per_dashboard", 0)
    max_queries = int(max_queries_raw or 0) or None
    time_range = TimeRange.last_minutes(minutes)

    all_dashboards = grafana_client.list_dashboards()
    aggregated = ProbeResults()
    supported_dashboards: list[tuple[GrafanaDashboard, DashboardProbe]] = []
    dashboards_by_title: dict[str, GrafanaDashboard] = {}
    probes_by_title: dict[str, DashboardProbe] = {}
    dashboard_warnings: dict[str, list[str]] = {}

    for dashboard in all_dashboards:
        try:
            model = grafana_client.get_dashboard_json(dashboard.uid)
            if is_prometheus_dashboard(model):
                probe = probe_dashboard
            elif is_victoria_logs_dashboard(model):
                probe = probe_victoria_logs_dashboard
            else:
                continue
            supported_dashboards.append((dashboard, probe))
            dashboards_by_title[dashboard.title] = dashboard
            probes_by_title[dashboard.title] = probe
        except RuntimeError as exc:
            aggregated.fetch_errors.append(
                f"{dashboard.title} ({dashboard.uid}): {exc}"
            )

    logger.debug("Probing %d supported dashboards...", len(supported_dashboards))

    for index, (dashboard, dashboard_probe) in enumerate(supported_dashboards, 1):
        logger.debug(
            "  [%d/%d] %s", index, len(supported_dashboards), dashboard.title,
        )
        results, warnings = _probe_dashboard_with_retry(
            grafana_client,
            dashboard,
            dashboard_probe_spec(policy, dashboard.title),
            probe=dashboard_probe,
            minutes=minutes,
            initial_time_range=time_range,
            max_queries=max_queries,
            fast_retry=fast_retry,
        )
        aggregated.probed_dashboards.append(dashboard.title)
        aggregated.dashboard_results[dashboard.title] = results
        dashboard_warnings[dashboard.title] = warnings

    detected = detect_components(grafana_client, kubectl_client, policy)
    _retry_present_optional_dashboards(
        grafana_client,
        policy,
        detected,
        dashboards_by_title,
        probes_by_title,
        aggregated.dashboard_results,
        dashboard_warnings,
        minutes=minutes,
        max_queries=max_queries,
        fast_retry=fast_retry,
    )

    aggregated.results = [
        result
        for title in aggregated.probed_dashboards
        for result in aggregated.dashboard_results.get(title, [])
    ]
    aggregated.warnings = [
        warning
        for title in aggregated.probed_dashboards
        for warning in dashboard_warnings.get(title, [])
    ]

    ok_count = sum(1 for result in aggregated.results if query_has_data(result))
    err_count = sum(1 for result in aggregated.results if query_error_message(result))
    no_data_count = len(aggregated.results) - ok_count - err_count
    logger.debug(
        "Probe complete: %d total, %d OK, %d no_data, %d errors, %d warnings",
        len(aggregated.results), ok_count, no_data_count, err_count,
        len(aggregated.warnings),
    )
    return aggregated, detected


def _retry_present_optional_dashboards(
    grafana_client: GrafanaClient,
    policy: DashboardPolicy,
    detected: dict[str, bool],
    dashboards: dict[str, GrafanaDashboard],
    probes: dict[str, DashboardProbe],
    results_by_dashboard: dict[str, list[QueryResult]],
    warnings_by_dashboard: dict[str, list[str]],
    *,
    minutes: int,
    max_queries: int | None,
    fast_retry: bool = False,
) -> None:
    """Retry present optional dashboards below threshold within one deadline."""
    retry_seconds = int(policy.probe_config.get("optional_present_retry_seconds", 0))
    if fast_retry:
        retry_seconds = min(retry_seconds, _FAST_RETRY_CAP_SECONDS)
    if retry_seconds <= 0:
        return

    pending: dict[str, tuple[GrafanaDashboard, dict[str, Any], int]] = {}
    for component_name, component_spec in policy.optional_dashboards.items():
        if not detected.get(component_name, False):
            continue
        when_present = _dict_or_empty(component_spec.get("when_present", {}))
        for title, dashboard_spec in component_dashboards(component_spec):
            merged_spec = {**when_present, **dashboard_spec}
            if merged_spec.get("allow_no_data", False):
                continue
            dashboard = dashboards.get(title)
            results = results_by_dashboard.get(title)
            if dashboard is None or results is None:
                continue
            threshold, _ = optional_min_ok(dashboard_spec, when_present, detected)
            if _ok_count(results) < threshold:
                pending[title] = (dashboard, merged_spec, threshold)

    interval = float(
        policy.probe_config.get("optional_present_retry_interval_seconds", 20),
    )
    deadline = time.monotonic() + retry_seconds
    while pending and time.monotonic() < deadline:
        for title, (dashboard, dashboard_spec, threshold) in list(pending.items()):
            dashboard_probe = probes.get(title, probe_dashboard)
            results, warnings = dashboard_probe(
                grafana_client,
                dashboard,
                variable_overrides=dashboard_spec.get("variable_overrides"),
                variable_preferences=dashboard_spec.get("variable_preferences"),
                time_range=TimeRange.last_minutes(minutes),
                max_queries=max_queries,
            )
            if (_ok_count(results), len(results)) > (
                _ok_count(results_by_dashboard[title]),
                len(results_by_dashboard[title]),
            ):
                results_by_dashboard[title] = results
                warnings_by_dashboard[title] = warnings
            if _ok_count(results) >= threshold:
                pending.pop(title)
        if pending:
            time.sleep(min(interval, max(0, deadline - time.monotonic())))


def _probe_dashboard_with_retry(
    grafana_client: GrafanaClient,
    dashboard: GrafanaDashboard,
    dashboard_spec: dict[str, Any],
    *,
    probe: DashboardProbe = probe_dashboard,
    minutes: int,
    initial_time_range: TimeRange,
    max_queries: int | None,
    fast_retry: bool = False,
) -> tuple[list[QueryResult], list[str]]:
    """Probe a dashboard, retrying only when its configured threshold is missed."""
    results, warnings = probe(
        grafana_client,
        dashboard,
        variable_overrides=dashboard_spec.get("variable_overrides"),
        variable_preferences=dashboard_spec.get("variable_preferences"),
        time_range=initial_time_range,
        max_queries=max_queries,
    )
    threshold = int(dashboard_spec.get("min_ok_queries", 0))
    retry_seconds = int(dashboard_spec.get("retry_seconds", 0))
    if fast_retry:
        retry_seconds = min(retry_seconds, _FAST_RETRY_CAP_SECONDS)
    if retry_seconds <= 0 or _ok_count(results) >= threshold:
        return results, warnings

    interval = float(dashboard_spec.get("retry_interval_seconds", 20))
    deadline = time.monotonic() + retry_seconds
    best_results, best_warnings = results, warnings

    while time.monotonic() < deadline:
        time.sleep(min(interval, max(0, deadline - time.monotonic())))
        retried_results, retried_warnings = probe(
            grafana_client,
            dashboard,
            variable_overrides=dashboard_spec.get("variable_overrides"),
            variable_preferences=dashboard_spec.get("variable_preferences"),
            time_range=TimeRange.last_minutes(minutes),
            max_queries=max_queries,
        )
        logger.debug(
            "Retrying %s: %d -> %d OK",
            dashboard.title,
            _ok_count(results),
            _ok_count(retried_results),
        )
        results, warnings = retried_results, retried_warnings

        if _ok_count(results) > _ok_count(best_results):
            best_results, best_warnings = results, warnings
        if _ok_count(results) >= threshold:
            return results, warnings

    return best_results, best_warnings


def _ok_count(results: list[QueryResult]) -> int:
    return sum(1 for result in results if query_has_data(result))


def _dict_or_empty(value: Any) -> dict[str, Any]:
    return value if isinstance(value, dict) else {}
