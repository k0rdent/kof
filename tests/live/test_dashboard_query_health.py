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

from framework.grafana import GrafanaClient
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

    return DashboardPolicy(
        probe_config=raw.get("probe", {}),
        required_dashboards=raw.get("required_dashboards", {}),
        optional_dashboards=raw.get("optional_dashboards", {}),
        known_errors=raw.get("known_errors", []),
    )


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

    logger.info("Probing %d Prometheus dashboards...", len(prometheus_dashboards))

    for i, dashboard in enumerate(prometheus_dashboards, 1):
        logger.info(
            "  [%d/%d] %s", i, len(prometheus_dashboards), dashboard.title,
        )
        results, warnings = probe_dashboard(
            grafana_client,
            dashboard,
            time_range=time_range,
            max_queries=max_queries,
        )
        aggregated.results.extend(results)
        aggregated.warnings.extend(warnings)
        aggregated.probed_dashboards.append(dashboard.title)
        aggregated.dashboard_results.setdefault(dashboard.title, []).extend(results)

    ok_count = sum(1 for r in aggregated.results if _has_data(r))
    err_count = sum(1 for r in aggregated.results if _error_message(r))
    no_data_count = len(aggregated.results) - ok_count - err_count
    logger.info(
        "Probe complete: %d total, %d OK, %d no_data, %d errors, %d warnings",
        len(aggregated.results), ok_count, no_data_count, err_count,
        len(aggregated.warnings),
    )
    return aggregated


@pytest.fixture(scope="session")
def detected_components(grafana_client: GrafanaClient) -> dict[str, bool]:
    """Detect which optional components are present by querying metrics."""
    policy_path = POLICY_FILE
    if not policy_path.exists():
        return {}

    with open(policy_path) as f:
        raw = yaml.safe_load(f)

    optional = raw.get("optional_dashboards", {})
    detected: dict[str, bool] = {}
    datasources = grafana_client.list_datasources()

    for component_name, spec in optional.items():
        detect_rule = spec.get("detect", "")
        detected[component_name] = _detect_component(
            grafana_client, detect_rule, datasources,
        )
        logger.info(
            "Component %s: %s (detect: %s)",
            component_name,
            "PRESENT" if detected[component_name] else "absent",
            detect_rule,
        )

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


def _required_dashboard_params(policy_path: Path) -> list[tuple[str, int]]:
    """Load required dashboard parameters for pytest parametrize.

    Returns list of (dashboard_title, min_ok_queries) tuples.
    """
    if not policy_path.exists():
        return []
    with open(policy_path) as f:
        raw = yaml.safe_load(f)
    required = raw.get("required_dashboards", {})
    return [(title, spec["min_ok_queries"]) for title, spec in required.items()]


class TestRequiredDashboards:
    """Required dashboards must meet minimum OK query thresholds."""

    @pytest.mark.parametrize(
        "dashboard_title,min_ok",
        _required_dashboard_params(POLICY_FILE),
        ids=[t for t, _ in _required_dashboard_params(POLICY_FILE)],
    )
    def test_required_dashboard_health(
        self,
        dashboard_title: str,
        min_ok: int,
        probe_results: ProbeResults,
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

        assert ok_count >= min_ok, (
            f"Dashboard '{dashboard_title}': {ok_count}/{total} queries have data, "
            f"expected at least {min_ok}.\n"
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
            logger.info(
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
                logger.info(
                    "Component %s dashboard %s accepted %d scoped query error(s)",
                    component_name,
                    db_title,
                    len(errors) - len(disallowed_errors),
                )

            if dashboard_spec.get("allow_no_data", when_present.get("allow_no_data", False)):
                continue

            min_ok = int(dashboard_spec.get(
                "min_ok_queries",
                when_present.get("min_ok_queries", 1),
            ))
            ok_count = sum(1 for r in results if _has_data(r))
            total = len(results)

            assert ok_count >= min_ok, (
                f"Component '{component_name}' is present but dashboard "
                f"'{db_title}' has only {ok_count}/{total} OK queries "
                f"(expected >= {min_ok}).\n"
                + _format_empty_panels(results)
            )


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _has_data(r: QueryResult) -> bool:
    """Check if a query result contains actual data points."""
    return query_has_data(r)


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


def _is_known_error(r: QueryResult, known_errors: list[dict], error: str) -> bool:
    """Check if a query error matches a known_errors entry."""
    return _error_matches_any(r, error, known_errors)


def _is_allowed_policy_error(
    r: QueryResult,
    error: str,
    policy: DashboardPolicy,
    detected_components: dict[str, bool],
) -> bool:
    """Return True if an optional component policy explicitly allows this error."""
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
        raw = source.get("allowed_errors", [])
        if isinstance(raw, list):
            allowed.extend(item for item in raw if isinstance(item, dict))
    return allowed


def _error_matches_any(r: QueryResult, error: str, entries: list[dict]) -> bool:
    for entry in entries:
        if _error_matches_entry(r, error, entry):
            return True
    return False


def _error_matches_entry(r: QueryResult, error: str, entry: dict) -> bool:
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
) -> bool:
    """Detect if a component is present based on a detection rule.

    Rules:
      - "namespace:NAME" — check if namespace has pods via metric
      - "metric:METRIC_NAME{labels}" — check if metric exists
    """
    if not detect_rule:
        return False

    if detect_rule.startswith("namespace:"):
        namespace = detect_rule[len("namespace:"):]
        # Check if any kube_pod_info exists for this namespace
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


def _run_detection_query(
    grafana_client: GrafanaClient,
    query: str,
    datasources: list,
) -> bool:
    """Execute a detection query and return True if it returns any data."""
    try:
        prom_ds = None
        for ds in datasources:
            if ds.type in ("prometheus", "victoriametrics-datasource",
                           "victoriametrics-metrics-datasource"):
                prom_ds = ds
                break

        if prom_ds is None:
            return False

        # Build a minimal query payload
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
            if len(values) >= 2 and values[1]:
                # Has a result value — component exists
                val = values[1][0] if values[1] else None
                if val is not None and float(val) > 0:
                    return True

        return False
    except Exception as exc:
        logger.debug("Detection query failed for %s: %s", query, exc)
        return False
