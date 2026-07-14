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

import logging
import re
from pathlib import Path
from typing import Any

import pytest

from framework.config import LiveTestConfig
from framework.dashboard_policy import (
    DashboardPolicy,
    allowed_error_specs,
    component_dashboards,
    direct_allowed_errors,
    error_matches_any,
    format_unknown_policy_title,
    is_allowed_policy_error,
    load_dashboard_policy,
    min_ok_after_allowed_errors,
    optional_component_params,
    optional_min_ok,
    policy_dashboard_titles,
    required_dashboard_params,
    required_min_ok,
)
from framework.dashboard_probe_session import ProbeResults, run_dashboard_probe_session
from framework.dashboard_report import build_dashboard_health_report
from framework.grafana import GrafanaClient
from framework.kubernetes import KubectlClient
from framework.prometheus import QueryResult, query_error_message, query_has_data
from framework.reference import DashboardReference

logger = logging.getLogger(__name__)

POLICY_FILE = Path(__file__).parent / "dashboard_query_policy.yaml"
_REPORT_STATE: dict[str, Any] = {}
_MAX_EMPTY_PANELS_SHOWN = 5

_REQUIRED_DASHBOARD_PARAMS = [
    pytest.param(title, spec, id=title)
    for title, spec in required_dashboard_params(POLICY_FILE)
]
_OPTIONAL_COMPONENT_PARAMS = [
    pytest.param(name, spec, id=name)
    for name, spec in optional_component_params(POLICY_FILE)
]


@pytest.fixture(scope="session")
def dashboard_policy() -> DashboardPolicy:
    """Load the dashboard query policy from YAML."""
    if not POLICY_FILE.exists():
        pytest.fail(f"Policy file not found: {POLICY_FILE}")

    policy = load_dashboard_policy(POLICY_FILE)
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

    report = build_dashboard_health_report(policy, probe, detected)
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
    kubectl_client: KubectlClient,
    live_config: LiveTestConfig,
) -> ProbeResults:
    """Run full probe across all Prometheus dashboards once per session."""
    probe, detected = run_dashboard_probe_session(
        grafana_client,
        dashboard_policy,
        kubectl_client,
        fast_retry=live_config.fast_retry,
    )
    _REPORT_STATE["probe_results"] = probe
    _REPORT_STATE["detected_components"] = detected
    return probe


@pytest.fixture(scope="session")
def detected_components(
    probe_results: ProbeResults,
) -> dict[str, bool]:
    """Return component state detected after the initial dashboard probe."""
    return dict(_REPORT_STATE.get("detected_components", {}))


class TestPolicySanity:
    """Static policy checks that do not require a live Grafana query run."""

    def test_policy_dashboard_titles_exist_in_reference(
        self,
        dashboard_policy: DashboardPolicy,
        dashboard_reference: DashboardReference,
    ) -> None:
        """Every dashboard title in the query policy must exist in static reference."""
        reference_titles = set(dashboard_reference.titles)
        unknown_titles = [
            title for title in policy_dashboard_titles(dashboard_policy)
            if title not in reference_titles
        ]

        assert not unknown_titles, (
            "Dashboard query policy references titles that are not present "
            "in tests/reference/dashboards.yaml:\n"
            + "\n".join(
                format_unknown_policy_title(title, reference_titles)
                for title in unknown_titles
            )
        )


class TestProbeIntegrity:
    """Hard-fail checks — any failure indicates a bug in probe or dashboards."""

    def test_no_dashboard_fetch_errors(self, probe_results: ProbeResults) -> None:
        """Every dashboard returned by search should be fetchable."""
        assert not probe_results.fetch_errors, (
            f"Failed to fetch {len(probe_results.fetch_errors)} dashboards:\n"
            + "\n".join(f"  {item}" for item in probe_results.fetch_errors[:20])
        )

    def test_no_unresolved_variables(self, probe_results: ProbeResults) -> None:
        """No executed query should contain literal $var after resolution."""
        builtin_re = re.compile(
            r"\$__[A-Za-z_][A-Za-z0-9_]*"
            r"|\$\{__[^}]+\}"
            r"|\[\[__[^\]]+\]\]"
        )

        unresolved = []
        for result in probe_results.results:
            cleaned = builtin_re.sub("", result.expr)
            if re.search(r"\$[A-Za-z_]|\$\{[^}]+\}|\[\[[^\]]+\]\]", cleaned):
                unresolved.append(
                    f"  {result.dashboard_title} / {result.panel_title}: "
                    f"{result.expr[:100]}"
                )

        var_warnings = [
            warning for warning in probe_results.warnings
            if "unresolved" in warning.lower()
        ]
        issues = unresolved + [f"  (warning) {warning}" for warning in var_warnings]

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
        """All query errors must be tracked by scoped policy allowances."""
        unexpected = []
        for result in probe_results.results:
            error = _error_message(result)
            if not error:
                continue
            if not is_allowed_policy_error(
                result,
                error,
                dashboard_policy,
                detected_components,
            ):
                unexpected.append(
                    f"  {result.dashboard_title} / {result.panel_title}: "
                    f"{error[:300]}"
                )

        assert not unexpected, (
            f"Found {len(unexpected)} unexpected query errors:\n"
            + "\n".join(unexpected[:20])
            + "\n\nAdd a tightly scoped allowed_errors entry in policy YAML "
            "if these are expected."
        )

    def test_no_resolver_warnings(self, probe_results: ProbeResults) -> None:
        """Variable resolution should not produce warnings."""
        other_warnings = [
            warning for warning in probe_results.warnings
            if "unresolved" not in warning.lower()
        ]

        assert not other_warnings, (
            f"Found {len(other_warnings)} resolver warnings:\n"
            + "\n".join(f"  {warning}" for warning in other_warnings[:20])
        )


class TestRequiredDashboards:
    """Required dashboards must meet minimum OK query thresholds."""

    @pytest.mark.parametrize(
        "dashboard_title,dashboard_spec",
        _REQUIRED_DASHBOARD_PARAMS,
    )
    def test_required_dashboard_health(
        self,
        dashboard_title: str,
        dashboard_spec: dict[str, Any],
        probe_results: ProbeResults,
        detected_components: dict[str, bool],
    ) -> None:
        """Dashboard must have at least min_ok_queries returning data."""
        results = probe_results.dashboard_results.get(dashboard_title)
        if results is None:
            pytest.fail(
                f"Dashboard '{dashboard_title}' was not probed. "
                f"Check that it exists in Grafana and uses a supported datasource."
            )

        ok_count = _ok_count(results)
        total = len(results)
        expected_queries = dashboard_spec.get("expected_queries")
        if expected_queries is not None:
            assert total == int(expected_queries), (
                f"Dashboard '{dashboard_title}' executed {total} queries, "
                f"expected {expected_queries}. A panel query may have been skipped "
                "because its datasource or variables could not be resolved."
            )
        min_ok, override_component = required_min_ok(
            dashboard_spec,
            detected_components,
        )
        allowed_errors = direct_allowed_errors(dashboard_spec)
        effective_min_ok, allowed_error_count = min_ok_after_allowed_errors(
            results,
            min_ok,
            allowed_errors,
        )

        expectation = _threshold_expectation(
            effective_min_ok,
            min_ok,
            override_component,
            allowed_error_count,
        )
        assert ok_count >= effective_min_ok, (
            f"Dashboard '{dashboard_title}': {ok_count}/{total} queries have data, "
            f"{expectation}.\n"
            + _format_empty_panels(results)
        )


class TestOptionalDashboards:
    """Optional dashboards — require data only when component is present."""

    @pytest.mark.parametrize(
        "component_name,spec",
        _OPTIONAL_COMPONENT_PARAMS,
    )
    def test_optional_component(
        self,
        component_name: str,
        spec: dict[str, Any],
        probe_results: ProbeResults,
        detected_components: dict[str, bool],
    ) -> None:
        """Validate optional dashboard based on component presence."""
        is_present = detected_components.get(component_name, False)
        dashboards = component_dashboards(spec)

        if not is_present:
            self._assert_absent_component_dashboards(
                component_name,
                spec,
                dashboards,
                probe_results,
            )
            return

        when_present = spec.get("when_present", {})
        if not isinstance(when_present, dict):
            when_present = {}

        for dashboard_title, dashboard_spec in dashboards:
            results = probe_results.dashboard_results.get(dashboard_title)
            if results is None:
                pytest.fail(
                    f"Component '{component_name}' is present but dashboard "
                    f"'{dashboard_title}' was not probed. Check provisioning "
                    "and datasource type."
                )

            disallowed_errors, allowed_error_count = _disallowed_errors(
                results,
                allowed_error_specs(spec, dashboard_spec, is_present=True),
            )
            require_no_errors = dashboard_spec.get(
                "require_no_errors",
                when_present.get("require_no_errors", True),
            )
            if require_no_errors and disallowed_errors:
                pytest.fail(
                    f"Component '{component_name}' is present but dashboard "
                    f"'{dashboard_title}' has {len(disallowed_errors)} query errors:\n"
                    + "\n".join(
                        f"  - [{result.panel_title}] {error[:300]}"
                        for result, error in disallowed_errors[:10]
                    )
                )
            if allowed_error_count:
                logger.debug(
                    "Component %s dashboard %s accepted %d scoped query error(s)",
                    component_name,
                    dashboard_title,
                    allowed_error_count,
                )

            if dashboard_spec.get("allow_no_data", when_present.get("allow_no_data", False)):
                continue

            min_ok, override_component = optional_min_ok(
                dashboard_spec,
                when_present,
                detected_components,
            )
            effective_min_ok, _ = min_ok_after_allowed_errors(
                results,
                min_ok,
                allowed_error_specs(spec, dashboard_spec, is_present=True),
            )
            ok_count = _ok_count(results)
            total = len(results)
            expectation = _threshold_expectation(
                effective_min_ok,
                min_ok,
                override_component,
                allowed_error_count,
            )

            assert ok_count >= effective_min_ok, (
                f"Component '{component_name}' is present but dashboard "
                f"'{dashboard_title}' has only {ok_count}/{total} OK queries "
                f"({expectation}).\n"
                + _format_empty_panels(results)
            )

    def _assert_absent_component_dashboards(
        self,
        component_name: str,
        spec: dict[str, Any],
        dashboards: list[tuple[str, dict[str, Any]]],
        probe_results: ProbeResults,
    ) -> None:
        when_absent = spec.get("when_absent", {})
        if not isinstance(when_absent, dict):
            when_absent = {}
        require_no_errors = when_absent.get("require_no_errors", True)
        checked_dashboards = 0

        for dashboard_title, _ in dashboards:
            results = probe_results.dashboard_results.get(dashboard_title, [])
            if not results:
                continue
            checked_dashboards += 1
            errors = [
                (result, error)
                for result in results
                if (error := _error_message(result))
            ]
            if require_no_errors and errors:
                pytest.fail(
                    f"Component '{component_name}' absent but dashboard "
                    f"'{dashboard_title}' has {len(errors)} errors:\n"
                    + "\n".join(f"  {error[:300]}" for _, error in errors[:5])
                )

        logger.debug(
            "Component %s absent; accepted %d dashboard(s) without data requirement",
            component_name,
            checked_dashboards,
        )


def _has_data(result: QueryResult) -> bool:
    return query_has_data(result)


def _ok_count(results: list[QueryResult]) -> int:
    return sum(1 for result in results if _has_data(result))


def _error_message(result: QueryResult) -> str | None:
    return query_error_message(result)


def _disallowed_errors(
    results: list[QueryResult],
    allowed_errors: list[dict[str, Any]],
) -> tuple[list[tuple[QueryResult, str]], int]:
    disallowed = []
    allowed_count = 0
    for result in results:
        error = _error_message(result)
        if not error:
            continue
        if error_matches_any(result, error, allowed_errors):
            allowed_count += 1
        else:
            disallowed.append((result, error))
    return disallowed, allowed_count


def _threshold_expectation(
    effective_min_ok: int,
    original_min_ok: int,
    override_component: str | None,
    allowed_error_count: int,
) -> str:
    expectation = f"expected at least {effective_min_ok}"
    if override_component:
        expectation += f" ({override_component} override: {original_min_ok})"
    if effective_min_ok < original_min_ok:
        expectation += (
            f", adjusted from {original_min_ok} due to "
            f"{allowed_error_count} allowed errors"
        )
    return expectation


def _format_empty_panels(results: list[QueryResult]) -> str:
    """Format a compact summary of empty panels for assertion messages."""
    empty = [
        result for result in results
        if not _has_data(result) and not _error_message(result)
    ]
    if not empty:
        return "Empty panels: (none)"
    lines = [
        f"  - [{result.panel_title}] {result.expr[:80]}"
        for result in empty[:_MAX_EMPTY_PANELS_SHOWN]
    ]
    if len(empty) > _MAX_EMPTY_PANELS_SHOWN:
        lines.append(f"  ... and {len(empty) - _MAX_EMPTY_PANELS_SHOWN} more")
    return f"Empty panels ({len(empty)}):\n" + "\n".join(lines)
