"""Dashboard query health tests — policy-based validation.

Validates that Grafana dashboard queries return expected data based on the
declarative dashboard_query_policy.yaml file.

Tests:
  1. test_policy_dashboard_titles_exist_in_reference — policy titles are valid
  2. test_no_unresolved_variables — no literal $var after resolution
  3. test_no_unexpected_errors — all errors must be explicitly allowed
  4. test_required_dashboard_health — core dashboards meet min OK thresholds
  5. test_optional_component — data is required only when component is present
"""
from __future__ import annotations

import difflib
import logging
import re
import time
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

import pytest
import yaml

from framework.grafana import GrafanaClient, GrafanaDashboard
from framework.kubernetes import KubectlClient, KubectlError
from framework.prometheus import (
    QueryResult,
    TimeRange,
    is_prometheus_dashboard,
    probe_dashboard,
    query_error_message,
    query_has_data,
)
from framework.reference import DashboardReference

logger = logging.getLogger(__name__)

POLICY_FILE = Path(__file__).parent / "dashboard_query_policy.yaml"
_REPORT_STATE: dict[str, Any] = {}


# ---------------------------------------------------------------------------
# Policy data structures
# ---------------------------------------------------------------------------


@dataclass
class ProbeResults:
    """Aggregated results from probing all dashboards."""

    results: list[QueryResult] = field(default_factory=list)
    warnings: list[str] = field(default_factory=list)
    dashboard_results: dict[str, list[QueryResult]] = field(default_factory=dict)
    probed_dashboards: list[str] = field(default_factory=list)
    fetch_errors: list[str] = field(default_factory=list)


@dataclass
class DashboardPolicy:
    """Loaded policy configuration."""

    probe_config: dict[str, Any]
    component_detectors: dict[str, str]
    required_dashboards: dict[str, dict[str, Any]]
    optional_dashboards: dict[str, dict[str, Any]]
    known_errors: list[dict[str, str]]


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


@pytest.fixture(scope="session")
def dashboard_policy() -> DashboardPolicy:
    """Load the dashboard query policy from YAML."""
    if not POLICY_FILE.exists():
        pytest.fail(f"Policy file not found: {POLICY_FILE}")

    with open(POLICY_FILE) as f:
        raw = yaml.safe_load(f)

    policy = DashboardPolicy(
        probe_config=raw.get("probe", {}),
        component_detectors=raw.get("component_detectors", {}),
        required_dashboards=raw.get("required_dashboards", {}),
        optional_dashboards=raw.get("optional_dashboards", {}),
        known_errors=raw.get("known_errors", []),
    )
    _REPORT_STATE["policy"] = policy
    return policy


@pytest.fixture(scope="session", autouse=True)
def dashboard_query_health_report(request: pytest.FixtureRequest):
    """Print a compact report after dashboard query health tests finish."""
    yield

    policy = _REPORT_STATE.get("policy")
    probe = _REPORT_STATE.get("probe_results")
    detected = _REPORT_STATE.get("detected_components", {})
    if not isinstance(policy, DashboardPolicy) or not isinstance(probe, ProbeResults):
        return

    report = _build_dashboard_health_report(policy, probe, detected)
    terminal = request.config.pluginmanager.get_plugin("terminalreporter")
    if terminal is None:
        print(report)
        return

    terminal.write("\n")
    terminal.write(report)
    terminal.write("\n")


@pytest.fixture(scope="session")
def probe_results(
    grafana_client: GrafanaClient,
    dashboard_policy: DashboardPolicy,
) -> ProbeResults:
    """Run full probe across all Prometheus dashboards (session-scoped).

    Executes once, results are shared by all test functions.
    """
    minutes = dashboard_policy.probe_config.get("time_range_minutes", 120)
    max_queries_raw = dashboard_policy.probe_config.get("max_queries_per_dashboard", 0)
    max_queries = int(max_queries_raw or 0) or None
    time_range = TimeRange.last_minutes(minutes)

    all_dashboards = grafana_client.list_dashboards()
    aggregated = ProbeResults()
    prometheus_dashboards = []
    for d in all_dashboards:
        try:
            model = grafana_client.get_dashboard_json(d.uid)
            if is_prometheus_dashboard(model):
                prometheus_dashboards.append(d)
        except RuntimeError as exc:
            aggregated.fetch_errors.append(f"{d.title} ({d.uid}): {exc}")

    logger.debug("Probing %d Prometheus dashboards...", len(prometheus_dashboards))

    for i, dashboard in enumerate(prometheus_dashboards, 1):
        logger.debug(
            "  [%d/%d] %s", i, len(prometheus_dashboards), dashboard.title,
        )
        results, warnings = _probe_dashboard_with_retry(
            grafana_client,
            dashboard,
            _dashboard_probe_spec(dashboard_policy, dashboard.title),
            minutes=minutes,
            initial_time_range=time_range,
            max_queries=max_queries,
        )
        aggregated.results.extend(results)
        aggregated.warnings.extend(warnings)
        aggregated.probed_dashboards.append(dashboard.title)
        aggregated.dashboard_results.setdefault(dashboard.title, []).extend(results)

    ok_count = sum(1 for r in aggregated.results if _has_data(r))
    err_count = sum(1 for r in aggregated.results if _error_message(r))
    no_data_count = len(aggregated.results) - ok_count - err_count
    logger.debug(
        "Probe complete: %d total, %d OK, %d no_data, %d errors, %d warnings",
        len(aggregated.results), ok_count, no_data_count, err_count,
        len(aggregated.warnings),
    )
    _REPORT_STATE["probe_results"] = aggregated
    return aggregated


def _probe_dashboard_with_retry(
    grafana_client: GrafanaClient,
    dashboard: GrafanaDashboard,
    dashboard_spec: dict[str, Any],
    *,
    minutes: int,
    initial_time_range: TimeRange,
    max_queries: int | None,
) -> tuple[list[QueryResult], list[str]]:
    """Probe a dashboard, retrying only when its configured threshold is missed."""
    results, warnings = probe_dashboard(
        grafana_client,
        dashboard,
        variable_overrides=dashboard_spec.get("variable_overrides"),
        time_range=initial_time_range,
        max_queries=max_queries,
    )
    threshold = int(dashboard_spec.get("min_ok_queries", 0))
    retry_seconds = int(dashboard_spec.get("retry_seconds", 0))
    if retry_seconds <= 0 or _ok_count(results) >= threshold:
        return results, warnings

    interval = float(dashboard_spec.get("retry_interval_seconds", 20))
    deadline = time.monotonic() + retry_seconds
    best_results, best_warnings = results, warnings

    while time.monotonic() < deadline:
        time.sleep(min(interval, max(0, deadline - time.monotonic())))
        retried_results, retried_warnings = probe_dashboard(
            grafana_client,
            dashboard,
            variable_overrides=dashboard_spec.get("variable_overrides"),
            time_range=TimeRange.last_minutes(minutes),
            max_queries=max_queries,
        )
        newly_ok = _newly_ok_panels(results, retried_results)
        logger.debug(
            "Retrying %s: %d -> %d OK; newly OK: %s",
            dashboard.title,
            _ok_count(results),
            _ok_count(retried_results),
            ", ".join(newly_ok) if newly_ok else "none",
        )
        results, warnings = retried_results, retried_warnings

        if _ok_count(results) > _ok_count(best_results):
            best_results, best_warnings = results, warnings
        if _ok_count(results) >= threshold:
            return results, warnings

    return best_results, best_warnings


@pytest.fixture(scope="session")
def detected_components(
    grafana_client: GrafanaClient,
    kubectl_client: KubectlClient,
) -> dict[str, bool]:
    """Detect which optional components are present by querying metrics."""
    policy_path = POLICY_FILE
    if not policy_path.exists():
        return {}

    with open(policy_path) as f:
        raw = yaml.safe_load(f)

    component_detectors = raw.get("component_detectors", {})
    optional = raw.get("optional_dashboards", {})
    detected: dict[str, bool] = {}
    datasources = grafana_client.list_datasources()

    for component_name, detect_rule in component_detectors.items():
        detected[component_name] = _detect_component(
            grafana_client, str(detect_rule), datasources, kubectl_client,
        )
        logger.debug(
            "Component %s: %s (detect: %s)",
            component_name,
            "PRESENT" if detected[component_name] else "absent",
            detect_rule,
        )

    for component_name, spec in optional.items():
        detect_rule = spec.get("detect", "")
        detected[component_name] = _detect_component(
            grafana_client, detect_rule, datasources, kubectl_client,
        )
        logger.debug(
            "Component %s: %s (detect: %s)",
            component_name,
            "PRESENT" if detected[component_name] else "absent",
            detect_rule,
        )

    _REPORT_STATE["detected_components"] = detected
    return detected


# ---------------------------------------------------------------------------
# Test: Policy sanity
# ---------------------------------------------------------------------------


class TestPolicySanity:
    """Static policy checks that do not require a live Grafana query run."""

    def test_policy_dashboard_titles_exist_in_reference(
        self,
        dashboard_policy: DashboardPolicy,
        dashboard_reference: DashboardReference,
    ) -> None:
        """Every dashboard title in the query policy must exist in static reference."""
        reference_titles = set(dashboard_reference.titles)
        policy_titles = _policy_dashboard_titles(dashboard_policy)
        unknown_titles = [
            title for title in policy_titles
            if title not in reference_titles
        ]

        assert not unknown_titles, (
            "Dashboard query policy references titles that are not present "
            "in tests/reference/dashboards.yaml:\n"
            + "\n".join(
                _format_unknown_policy_title(title, reference_titles)
                for title in unknown_titles
            )
        )


# ---------------------------------------------------------------------------
# Test: No unresolved variables
# ---------------------------------------------------------------------------


class TestProbeIntegrity:
    """Hard-fail checks — any failure indicates a bug in probe or dashboards."""

    def test_no_dashboard_fetch_errors(self, probe_results: ProbeResults) -> None:
        """Every dashboard returned by search should be fetchable."""
        assert not probe_results.fetch_errors, (
            f"Failed to fetch {len(probe_results.fetch_errors)} dashboards:\n"
            + "\n".join(f"  {item}" for item in probe_results.fetch_errors[:20])
        )

    def test_no_unresolved_variables(self, probe_results: ProbeResults) -> None:
        """No executed query should contain literal $var after resolution.

        Unresolved variables indicate broken variable resolution or missing
        dependencies between template variables.

        Note: Grafana built-in variables ($__rate_interval, $__interval,
        $__range, $__auto, etc.) are excluded — they are resolved by Grafana
        at query execution time, not during template variable resolution.
        """
        # Pattern for Grafana built-in variables (start with __)
        _BUILTIN_RE = re.compile(
            r"\$__[A-Za-z_][A-Za-z0-9_]*"
            r"|\$\{__[^}]+\}"
            r"|\[\[__[^\]]+\]\]"
        )

        unresolved = []
        for r in probe_results.results:
            # Remove built-in variables before checking for unresolved ones
            cleaned = _BUILTIN_RE.sub("", r.expr)
            if re.search(r"\$[A-Za-z_]|\$\{[^}]+\}|\[\[[^\]]+\]\]", cleaned):
                unresolved.append(
                    f"  {r.dashboard_title} / {r.panel_title}: {r.expr[:100]}"
                )

        # Also check warnings from variable resolution phase
        var_warnings = [
            w for w in probe_results.warnings
            if "unresolved" in w.lower()
        ]

        issues = unresolved + [f"  (warning) {w}" for w in var_warnings]
        assert not issues, (
            f"Found {len(issues)} queries with unresolved variables:\n"
            + "\n".join(issues[:20])
        )

    def test_no_unexpected_errors(
        self,
        probe_results: ProbeResults,
        dashboard_policy: DashboardPolicy,
        detected_components: dict[str, bool],
    ) -> None:
        """All query errors must be tracked by global or component policy.

        Unexpected errors indicate broken PromQL, connectivity issues,
        or datasource misconfiguration.
        """
        known = dashboard_policy.known_errors
        errors = [
            (r, error)
            for r in probe_results.results
            if (error := _error_message(r))
        ]

        unexpected = []
        for r, error in errors:
            if not (
                _is_known_error(r, known, error)
                or _is_allowed_policy_error(
                    r, error, dashboard_policy, detected_components,
                )
            ):
                unexpected.append(
                    f"  {r.dashboard_title} / {r.panel_title}: {error[:100]}"
                )

        assert not unexpected, (
            f"Found {len(unexpected)} unexpected query errors:\n"
            + "\n".join(unexpected[:20])
            + "\n\nAdd a tightly scoped allowed_errors entry in policy YAML "
            "if these are expected."
        )

    def test_no_resolver_warnings(self, probe_results: ProbeResults) -> None:
        """Variable resolution should not produce warnings.

        Warnings indicate issues like missing datasources, failed queries,
        or circular variable dependencies.
        """
        # Filter out "unresolved vars" warnings (covered by separate test)
        other_warnings = [
            w for w in probe_results.warnings
            if "unresolved" not in w.lower()
        ]

        assert not other_warnings, (
            f"Found {len(other_warnings)} resolver warnings:\n"
            + "\n".join(f"  {w}" for w in other_warnings[:20])
        )


# ---------------------------------------------------------------------------
# Test: Required dashboards
# ---------------------------------------------------------------------------


def _required_dashboard_params(policy_path: Path) -> list[tuple[str, dict]]:
    """Load required dashboard parameters for pytest parametrize.

    Returns list of (dashboard_title, dashboard_policy) tuples.
    """
    if not policy_path.exists():
        return []
    with open(policy_path) as f:
        raw = yaml.safe_load(f)
    required = raw.get("required_dashboards", {})
    return [(title, spec) for title, spec in required.items()]


class TestRequiredDashboards:
    """Required dashboards must meet minimum OK query thresholds."""

    @pytest.mark.parametrize(
        "dashboard_title,dashboard_spec",
        _required_dashboard_params(POLICY_FILE),
        ids=[t for t, _ in _required_dashboard_params(POLICY_FILE)],
    )
    def test_required_dashboard_health(
        self,
        dashboard_title: str,
        dashboard_spec: dict,
        probe_results: ProbeResults,
        detected_components: dict[str, bool],
    ) -> None:
        """Dashboard must have at least min_ok_queries returning data."""
        results = probe_results.dashboard_results.get(dashboard_title)

        if results is None:
            pytest.fail(
                f"Dashboard '{dashboard_title}' was not probed. "
                f"Check that it exists in Grafana and is a Prometheus dashboard."
            )

        ok_count = sum(1 for r in results if _has_data(r))
        total = len(results)
        min_ok, override_component = _required_min_ok(
            dashboard_spec,
            detected_components,
        )

        errors = [(r, error) for r in results if (error := _error_message(r))]
        allowed_errors = _direct_allowed_errors(dashboard_spec)
        allowed_error_count = sum(
            1 for r, error in errors
            if _error_matches_any(r, error, allowed_errors)
        )
        effective_total = total - allowed_error_count
        effective_min_ok = min(
            max(0, min_ok - allowed_error_count),
            effective_total,
        )
        expectation = f"expected at least {effective_min_ok}"
        if override_component:
            expectation += f" ({override_component} override: {min_ok})"
        if effective_min_ok < min_ok:
            expectation += (
                f", adjusted from {min_ok} due to "
                f"{allowed_error_count} allowed errors"
            )

        assert ok_count >= effective_min_ok, (
            f"Dashboard '{dashboard_title}': {ok_count}/{total} queries have data, "
            f"{expectation}.\n"
            + _format_empty_panels(results)
        )


# ---------------------------------------------------------------------------
# Test: Optional dashboards
# ---------------------------------------------------------------------------


def _optional_component_params(policy_path: Path) -> list[tuple[str, dict]]:
    """Load optional component parameters for pytest parametrize."""
    if not policy_path.exists():
        return []
    with open(policy_path) as f:
        raw = yaml.safe_load(f)
    optional = raw.get("optional_dashboards", {})
    return [(name, spec) for name, spec in optional.items()]


class TestOptionalDashboards:
    """Optional dashboards — require data only when component is present."""

    @pytest.mark.parametrize(
        "component_name,spec",
        _optional_component_params(POLICY_FILE),
        ids=[name for name, _ in _optional_component_params(POLICY_FILE)],
    )
    def test_optional_component(
        self,
        component_name: str,
        spec: dict,
        probe_results: ProbeResults,
        detected_components: dict[str, bool],
    ) -> None:
        """Validate optional dashboard based on component presence."""
        is_present = detected_components.get(component_name, False)
        dashboards = _component_dashboards(spec)

        if not is_present:
            # Component absent: queries may legitimately return no data, but they
            # still must be executable when the dashboard is provisioned.
            when_absent = spec.get("when_absent", {})
            if not isinstance(when_absent, dict):
                when_absent = {}
            require_no_errors = when_absent.get("require_no_errors", True)
            checked_dashboards = 0

            for db_title, _ in dashboards:
                results = probe_results.dashboard_results.get(db_title, [])
                if not results:
                    continue
                checked_dashboards += 1
                errors = [
                    (r, error)
                    for r in results
                    if (error := _error_message(r))
                ]
                if require_no_errors and errors:
                    pytest.fail(
                        f"Component '{component_name}' absent but dashboard "
                        f"'{db_title}' has {len(errors)} errors:\n"
                        + "\n".join(f"  {error[:80]}" for _, error in errors[:5])
                    )
            logger.debug(
                "Component %s absent; accepted %d dashboard(s) without data requirement",
                component_name,
                checked_dashboards,
            )
            return

        # Component present — assert minimum queries
        when_present = spec.get("when_present", {})

        for db_title, dashboard_spec in dashboards:
            results = probe_results.dashboard_results.get(db_title)
            if results is None:
                pytest.fail(
                    f"Component '{component_name}' is present but dashboard "
                    f"'{db_title}' was not probed. Check provisioning and datasource type."
                )

            errors = [(r, error) for r in results if (error := _error_message(r))]
            allowed_errors = _allowed_error_specs(spec, dashboard_spec, is_present=True)
            disallowed_errors = [
                (r, error)
                for r, error in errors
                if not _error_matches_any(r, error, allowed_errors)
            ]
            require_no_errors = dashboard_spec.get(
                "require_no_errors",
                when_present.get("require_no_errors", True),
            )
            if require_no_errors and disallowed_errors:
                pytest.fail(
                    f"Component '{component_name}' is present but dashboard "
                    f"'{db_title}' has {len(disallowed_errors)} query errors:\n"
                    + "\n".join(
                        f"  - [{r.panel_title}] {error[:120]}"
                        for r, error in disallowed_errors[:10]
                    )
                )
            if errors and len(disallowed_errors) < len(errors):
                logger.debug(
                    "Component %s dashboard %s accepted %d scoped query error(s)",
                    component_name,
                    db_title,
                    len(errors) - len(disallowed_errors),
                )

            if dashboard_spec.get("allow_no_data", when_present.get("allow_no_data", False)):
                continue

            min_ok, override_component = _optional_min_ok(
                dashboard_spec,
                when_present,
                detected_components,
            )
            ok_count = sum(1 for r in results if _has_data(r))
            total = len(results)

            # Adjust min_ok when known errors reduce the testable query pool.
            # If all queries fail with known errors, we can't demand data from
            # queries that are not executable due to infrastructure issues.
            known_error_count = len(errors) - len(disallowed_errors)
            effective_total = total - known_error_count
            effective_min_ok = min(
                max(0, min_ok - known_error_count),
                effective_total,
            )

            assert ok_count >= effective_min_ok, (
                f"Component '{component_name}' is present but dashboard "
                f"'{db_title}' has only {ok_count}/{total} OK queries "
                f"(expected >= {effective_min_ok}"
                f"{f', {override_component} override: {min_ok}' if override_component else ''}"
                f"{f', adjusted from {min_ok} due to {known_error_count} known errors' if effective_min_ok < min_ok else ''}"
                f").\n"
                + _format_empty_panels(results)
            )


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _has_data(r: QueryResult) -> bool:
    """Check if a query result contains actual data points."""
    return query_has_data(r)


def _ok_count(results: list[QueryResult]) -> int:
    return sum(1 for result in results if _has_data(result))


def _newly_ok_panels(
    previous: list[QueryResult],
    current: list[QueryResult],
) -> list[str]:
    """Return panel names whose matching query changed from no-data to OK."""
    previous_no_data = {
        (result.panel_title, result.ref_id, result.raw_expr)
        for result in previous
        if not _has_data(result) and not _error_message(result)
    }
    return list(dict.fromkeys(
        result.panel_title
        for result in current
        if _has_data(result)
        and (result.panel_title, result.ref_id, result.raw_expr) in previous_no_data
    ))


def _error_message(r: QueryResult) -> str | None:
    return query_error_message(r)


_MAX_EMPTY_PANELS_SHOWN = 5


def _format_empty_panels(results: list[QueryResult]) -> str:
    """Format a compact summary of empty panels for assertion messages."""
    empty = [
        r for r in results
        if not _has_data(r) and not _error_message(r)
    ]
    if not empty:
        return "Empty panels: (none)"
    lines = [f"  - [{r.panel_title}] {r.expr[:80]}" for r in empty[:_MAX_EMPTY_PANELS_SHOWN]]
    if len(empty) > _MAX_EMPTY_PANELS_SHOWN:
        lines.append(f"  ... and {len(empty) - _MAX_EMPTY_PANELS_SHOWN} more")
    return f"Empty panels ({len(empty)}):\n" + "\n".join(lines)


def _build_dashboard_health_report(
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
        min_ok, override_component = _required_min_ok(
            dashboard_spec,
            detected_components,
        )
        expected = _expected(
            _effective_report_min_ok(
                results,
                min_ok,
                _direct_allowed_errors(dashboard_spec),
            ),
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

        for dashboard_title, dashboard_spec in _component_dashboards(spec):
            results = probe.dashboard_results.get(dashboard_title, [])
            if not is_present:
                expected = "no data ok"
            elif dashboard_spec.get("allow_no_data", when_present.get("allow_no_data", False)):
                expected = "data optional"
            else:
                min_ok, override_component = _optional_min_ok(
                    dashboard_spec,
                    when_present,
                    detected_components,
                )
                expected = _expected(
                    _effective_report_min_ok(
                        results,
                        min_ok,
                        _allowed_error_specs(spec, dashboard_spec, is_present=True),
                    ),
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
    ok_count = sum(1 for r in results if _has_data(r))
    error_count = sum(1 for r in results if _error_message(r))
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


def _effective_report_min_ok(
    results: list[QueryResult],
    min_ok: int,
    allowed_errors: list[dict],
) -> int:
    allowed_count = sum(
        1 for r in results
        if (error := _error_message(r))
        and _error_matches_any(r, error, allowed_errors)
    )
    return min(max(0, min_ok - allowed_count), len(results) - allowed_count)


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
        error = _error_message(result)
        if not error:
            continue
        if (
            _is_known_error(result, policy.known_errors, error)
            or _is_allowed_policy_error(result, error, policy, detected_components)
        ):
            allowed.append((result, error))
        else:
            unexpected.append((result, error))

    lines = ["", "Known/allowed errors"]
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


def _is_known_error(r: QueryResult, known_errors: list[dict], error: str) -> bool:
    """Check if a query error matches a known_errors entry."""
    return _error_matches_any(r, error, known_errors)


def _is_allowed_policy_error(
    r: QueryResult,
    error: str,
    policy: DashboardPolicy,
    detected_components: dict[str, bool],
) -> bool:
    """Return True if dashboard policy explicitly allows this query error."""
    required_spec = policy.required_dashboards.get(r.dashboard_title, {})
    if isinstance(required_spec, dict):
        allowed = _direct_allowed_errors(required_spec)
        if _error_matches_any(r, error, allowed):
            return True

    for component_name, spec in policy.optional_dashboards.items():
        is_present = detected_components.get(component_name, False)
        for dashboard_title, dashboard_spec in _component_dashboards(spec):
            if dashboard_title != r.dashboard_title:
                continue
            allowed = _allowed_error_specs(spec, dashboard_spec, is_present)
            if _error_matches_any(r, error, allowed):
                return True
    return False


def _allowed_error_specs(
    component_spec: dict,
    dashboard_spec: dict,
    is_present: bool,
) -> list[dict]:
    """Return error allowances for the active component state."""
    state_key = "when_present" if is_present else "when_absent"
    state = component_spec.get(state_key, {})
    if not isinstance(state, dict):
        state = {}

    allowed: list[dict] = []
    for source in (state, dashboard_spec):
        allowed.extend(_direct_allowed_errors(source))
    return allowed


def _direct_allowed_errors(spec: dict) -> list[dict]:
    """Return directly declared allowed_errors entries from a policy spec."""
    raw = spec.get("allowed_errors", [])
    if not isinstance(raw, list):
        return []
    return [item for item in raw if isinstance(item, dict)]


def _error_matches_any(r: QueryResult, error: str, entries: list[dict]) -> bool:
    for entry in entries:
        if _error_matches_entry(r, error, entry):
            return True
    return False


def _error_matches_entry(r: QueryResult, error: str, entry: dict) -> bool:
    has_matcher = any(
        entry.get(key)
        for key in ("dashboard", "panel", "message_contains", "expr_contains")
    )
    if not has_matcher:
        return False

    dashboard = entry.get("dashboard")
    if dashboard and dashboard != r.dashboard_title:
        return False

    panel = entry.get("panel")
    if panel and panel != r.panel_title:
        return False

    message_contains = entry.get("message_contains")
    if message_contains and str(message_contains) not in error:
        return False

    expr_contains = entry.get("expr_contains")
    if expr_contains and str(expr_contains) not in r.expr:
        return False

    return True


def _component_dashboards(spec: dict) -> list[tuple[str, dict]]:
    """Return dashboard policy entries as (title, per_dashboard_spec)."""
    raw = spec.get("dashboards", [])
    if isinstance(raw, dict):
        return [
            (str(title), cfg if isinstance(cfg, dict) else {})
            for title, cfg in raw.items()
        ]
    if not isinstance(raw, list):
        return []

    dashboards: list[tuple[str, dict]] = []
    for item in raw:
        if isinstance(item, str):
            dashboards.append((item, {}))
        elif isinstance(item, dict):
            for title, cfg in item.items():
                dashboards.append((str(title), cfg if isinstance(cfg, dict) else {}))
    return dashboards


def _dashboard_probe_spec(policy: DashboardPolicy, title: str) -> dict[str, Any]:
    """Return probe settings for a required or optional dashboard."""
    if title in policy.required_dashboards:
        return policy.required_dashboards[title]

    for component_spec in policy.optional_dashboards.values():
        when_present = component_spec.get("when_present", {})
        if not isinstance(when_present, dict):
            when_present = {}
        for dashboard_title, dashboard_spec in _component_dashboards(component_spec):
            if dashboard_title == title:
                return {**when_present, **dashboard_spec}
    return {}


def _required_min_ok(
    dashboard_spec: dict,
    detected_components: dict[str, bool],
) -> tuple[int, str | None]:
    """Return min_ok_queries, applying component-specific overrides."""
    base = int(dashboard_spec["min_ok_queries"])
    overrides = dashboard_spec.get("min_ok_queries_by_component", {})
    if not isinstance(overrides, dict):
        return base, None

    for component_name, min_ok in overrides.items():
        if detected_components.get(str(component_name), False):
            return int(min_ok), str(component_name)

    return base, None


def _optional_min_ok(
    dashboard_spec: dict,
    component_when_present: dict,
    detected_components: dict[str, bool],
) -> tuple[int, str | None]:
    """Return optional dashboard min_ok with dashboard-level overrides."""
    base = int(dashboard_spec.get(
        "min_ok_queries",
        component_when_present.get("min_ok_queries", 1),
    ))
    overrides: dict[str, Any] = {}

    component_overrides = component_when_present.get("min_ok_queries_by_component", {})
    if isinstance(component_overrides, dict):
        overrides.update(component_overrides)

    dashboard_overrides = dashboard_spec.get("min_ok_queries_by_component", {})
    if isinstance(dashboard_overrides, dict):
        overrides.update(dashboard_overrides)

    for component_name, min_ok in overrides.items():
        if detected_components.get(str(component_name), False):
            return int(min_ok), str(component_name)

    return base, None


def _policy_dashboard_titles(policy: DashboardPolicy) -> list[str]:
    """Return all dashboard titles referenced by query health policy."""
    titles: list[str] = list(policy.required_dashboards)
    for spec in policy.optional_dashboards.values():
        titles.extend(title for title, _ in _component_dashboards(spec))
    return sorted(dict.fromkeys(titles))


def _format_unknown_policy_title(title: str, reference_titles: set[str]) -> str:
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


def _detect_component(
    grafana_client: GrafanaClient,
    detect_rule: str,
    datasources: list,
    kubectl_client: KubectlClient | None = None,
) -> bool:
    """Detect if a component is present based on a detection rule.

    Rules:
      - "namespace:NAME" — check if namespace has pods via metric
      - "metric:METRIC_NAME{labels}" — check if metric exists
      - "kubernetes:container:NAMESPACE/CONTAINER" — check live pod specs
      - "kubernetes:container-in-pod-prefix:NAMESPACE/PREFIX/CONTAINER"
    """
    if not detect_rule:
        return False

    if detect_rule.startswith("kubernetes:container:"):
        if kubectl_client is None:
            logger.debug("Kubernetes detect rule requires kubectl: %s", detect_rule)
            return False
        return _detect_kubernetes_container(
            kubectl_client,
            detect_rule[len("kubernetes:container:"):],
        )

    if detect_rule.startswith("kubernetes:container-in-pod-prefix:"):
        if kubectl_client is None:
            logger.debug("Kubernetes detect rule requires kubectl: %s", detect_rule)
            return False
        return _detect_kubernetes_container_in_pod_prefix(
            kubectl_client,
            detect_rule[len("kubernetes:container-in-pod-prefix:"):],
        )

    if detect_rule.startswith("namespace:"):
        namespace = detect_rule[len("namespace:"):]
        # Check if the namespace exists (has labels scraped by kube-state-metrics)
        query = f'count(kube_namespace_labels{{namespace="{namespace}"}})'
        return _run_detection_query(grafana_client, query, datasources)

    if detect_rule.startswith("metric:"):
        metric_expr = detect_rule[len("metric:"):]
        # Wrap in count() to check existence
        if "{" in metric_expr:
            query = f"count({metric_expr})"
        else:
            query = f'count({{__name__="{metric_expr}"}})'
        return _run_detection_query(grafana_client, query, datasources)

    logger.warning("Unknown detect rule format: %s", detect_rule)
    return False


def _detect_kubernetes_container(
    kubectl_client: KubectlClient,
    value: str,
) -> bool:
    """Return True if live pods in namespace include the given container."""
    try:
        namespace, container = value.split("/", 1)
    except ValueError:
        logger.warning("Invalid kubernetes container detect rule: %s", value)
        return False

    namespace = namespace.strip()
    container = container.strip()
    if not namespace or not container:
        logger.warning("Invalid kubernetes container detect rule: %s", value)
        return False

    try:
        output = kubectl_client.run(
            "-n", namespace,
            "get", "pods",
            "-o", r"jsonpath={range .items[*].spec.containers[*]}{.name}{'\n'}{end}",
        )
    except KubectlError as exc:
        logger.debug("Kubernetes container detection failed for %s: %s", value, exc)
        return False

    return any(line.strip() == container for line in output.splitlines())


def _detect_kubernetes_container_in_pod_prefix(
    kubectl_client: KubectlClient,
    value: str,
) -> bool:
    """Return True if a pod with prefix includes the given container."""
    try:
        namespace, pod_prefix, container = value.split("/", 2)
    except ValueError:
        logger.warning("Invalid kubernetes pod-prefix detect rule: %s", value)
        return False

    namespace = namespace.strip()
    pod_prefix = pod_prefix.strip()
    container = container.strip()
    if not namespace or not pod_prefix or not container:
        logger.warning("Invalid kubernetes pod-prefix detect rule: %s", value)
        return False

    try:
        output = kubectl_client.run(
            "-n", namespace,
            "get", "pods",
            "-o",
            r"jsonpath={range .items[*]}{.metadata.name}{'\t'}{range .spec.containers[*]}{.name}{','}{end}{'\n'}{end}",
        )
    except KubectlError as exc:
        logger.debug("Kubernetes pod-prefix detection failed for %s: %s", value, exc)
        return False

    for line in output.splitlines():
        pod_name, _, containers = line.partition("\t")
        if pod_name.startswith(pod_prefix) and container in containers.split(","):
            return True
    return False


def _run_detection_query(
    grafana_client: GrafanaClient,
    query: str,
    datasources: list,
) -> bool:
    """Execute a detection query and return True if it returns any data."""
    try:
        value = _run_scalar_query(grafana_client, query, datasources)
        return value is not None and value > 0
    except Exception as exc:
        logger.debug("Detection query failed for %s: %s", query, exc)
        return False


def _run_scalar_query(
    grafana_client: GrafanaClient,
    query: str,
    datasources: list,
) -> float | None:
    """Execute a minimal instant query and return its first numeric value."""
    prom_ds = None
    for ds in datasources:
        if ds.type in ("prometheus", "victoriametrics-datasource",
                       "victoriametrics-metrics-datasource"):
            prom_ds = ds
            break

    if prom_ds is None:
        return None

    now_ms = int(time.time() * 1000)
    payload = {
        "queries": [{
            "refId": "detect",
            "datasource": {"uid": prom_ds.uid, "type": prom_ds.type},
            "expr": query,
            "instant": True,
            "intervalMs": 60000,
            "maxDataPoints": 1,
        }],
        "from": str(now_ms - 300_000),
        "to": str(now_ms),
    }

    response = grafana_client.query_datasource(payload)
    results = response.get("results", {})
    detect_data = results.get("detect", {})
    frames = detect_data.get("frames", [])

    for frame in frames:
        data_section = frame.get("data", {})
        values = data_section.get("values", [])
        if len(values) < 2 or not values[1]:
            continue
        value = values[1][0]
        if value is not None:
            return float(value)

    return None
