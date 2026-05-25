from __future__ import annotations

from collections import Counter

from framework.config import LiveTestConfig
from framework.grafana import GrafanaClient
from framework.reference import DashboardReference
from framework.waiting import wait_for


def test_grafana_dashboard_titles_match_reference(
    live_config: LiveTestConfig,
    grafana_client: GrafanaClient,
    dashboard_reference: DashboardReference,
) -> None:
    """Verify Grafana exposes exactly the dashboards defined in the static reference.

    This test polls Grafana until all expected dashboards appear (or timeout).
    Dashboards are matched by title using multiset comparison to detect
    both missing and unexpected entries.
    """
    expected_titles = Counter(dashboard_reference.titles)
    last_actual_titles: Counter[str] = Counter()
    last_missing: list[str] = []
    last_unexpected: list[str] = []

    def check() -> None:
        nonlocal last_actual_titles, last_missing, last_unexpected

        try:
            actual_dashboards = grafana_client.list_dashboards()
        except RuntimeError as exc:
            raise AssertionError(
                f"Failed to fetch Grafana dashboard inventory: {exc}"
            ) from exc

        actual_titles = Counter(dashboard.title for dashboard in actual_dashboards)
        last_actual_titles = actual_titles

        missing = sorted((expected_titles - actual_titles).elements())
        unexpected = sorted((actual_titles - expected_titles).elements())
        last_missing = missing
        last_unexpected = unexpected

        if missing or (unexpected and not live_config.allow_extra_dashboards):
            raise AssertionError(
                _format_diff(
                    expected_count=sum(expected_titles.values()),
                    actual_count=sum(actual_titles.values()),
                    missing=missing,
                    unexpected=unexpected,
                    allow_extra=live_config.allow_extra_dashboards,
                )
            )

    wait_for(
        description="Grafana dashboard inventory to match static reference",
        check=check,
        timeout_seconds=live_config.timeout_seconds,
        poll_interval_seconds=live_config.poll_interval_seconds,
    )

    if live_config.print_diagnostics:
        print(
            _format_diagnostics(
                request=grafana_client.dashboard_search_request(),
                reference_path=str(live_config.reference_dashboards_path),
                expected=expected_titles,
                actual=last_actual_titles,
                missing=last_missing,
                unexpected=last_unexpected,
                allow_extra=live_config.allow_extra_dashboards,
            )
        )


def _format_diff(
    expected_count: int,
    actual_count: int,
    missing: list[str],
    unexpected: list[str],
    allow_extra: bool,
) -> str:
    """Format a human-readable diff of expected vs actual dashboards."""
    lines = [
        f"Dashboard inventory mismatch: "
        f"expected {expected_count}, got {actual_count} from Grafana",
    ]
    if missing:
        lines.append(f"\nMissing ({len(missing)} dashboards not found in Grafana):")
        lines.extend(f"  - {title}" for title in missing)
    if unexpected and not allow_extra:
        lines.append(
            f"\nUnexpected ({len(unexpected)} dashboards not in reference — "
            f"set GRAFANA_ALLOW_EXTRA_DASHBOARDS=true to ignore):"
        )
        lines.extend(f"  - {title}" for title in unexpected)
    return "\n".join(lines)


def _format_diagnostics(
    request: str,
    reference_path: str,
    expected: Counter[str],
    actual: Counter[str],
    missing: list[str],
    unexpected: list[str],
    allow_extra: bool,
) -> str:
    matched_count = sum((expected & actual).values())
    lines = [
        "",
        "Grafana dashboard inventory diagnostics:",
        f"  request: {request}",
        f"  reference: {reference_path}",
        f"  expected dashboards: {sum(expected.values())}",
        f"  live dashboards: {sum(actual.values())}",
        f"  matched dashboards: {matched_count}",
        f"  missing dashboards: {len(missing)}",
        f"  unexpected dashboards: {len(unexpected)}"
        + (" (ignored)" if unexpected and allow_extra else ""),
    ]

    if missing:
        lines.append("  missing titles:")
        lines.extend(f"    - {title}" for title in missing)

    if unexpected:
        lines.append("  unexpected titles:")
        lines.extend(f"    - {title}" for title in unexpected)

    lines.append("  live titles:")
    lines.extend(f"    - {title}" for title in sorted(actual.elements()))
    return "\n".join(lines)
