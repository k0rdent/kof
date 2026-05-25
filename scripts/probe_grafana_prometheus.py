#!/usr/bin/env python3
"""Probe live Grafana Prometheus dashboard queries via /api/ds/query.

Usage:
    python scripts/probe_grafana_prometheus.py --dashboard "Node Exporter / Nodes"
    python scripts/probe_grafana_prometheus.py --all --max-queries 3
    python scripts/probe_grafana_prometheus.py --var cluster=kind-kcm-dev
"""
from __future__ import annotations

import argparse
import json
import re
import sys
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

REPO_ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(REPO_ROOT / "tests" / "live"))

from framework.config import LiveTestConfig  # noqa: E402
from framework.grafana import GrafanaClient, GrafanaDashboard  # noqa: E402
from framework.kubernetes import KubectlClient  # noqa: E402
from framework.prometheus import (  # noqa: E402
    QueryResult,
    TimeRange,
    is_prometheus_dashboard,
    probe_dashboard,
)
from framework.runtime import ensure_grafana_port_forward, resolve_grafana_auth  # noqa: E402


# ---------------------------------------------------------------------------
# Categorized result
# ---------------------------------------------------------------------------


@dataclass
class CategorizedQuery:
    """A query result enriched with category and parsed info."""

    dashboard_title: str
    dashboard_uid: str
    panel_title: str
    ref_id: str
    expr: str
    category: str  # "ok", "no_data", "error"
    detail: str  # series/points summary or error message
    grafana_url: str = ""


@dataclass
class ProbeReport:
    ok: list[CategorizedQuery] = field(default_factory=list)
    no_data: list[CategorizedQuery] = field(default_factory=list)
    errors: list[CategorizedQuery] = field(default_factory=list)

    @property
    def total(self) -> int:
        return len(self.ok) + len(self.no_data) + len(self.errors)


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------


def main() -> int:
    args = parse_args()
    var_overrides = parse_var_overrides(args.var)

    config = LiveTestConfig.from_environment()
    kubectl = KubectlClient(config.kubectl, config.kubectl_context)
    grafana_url = config.grafana_url

    port_forward = None
    try:
        if not args.no_port_forward:
            port_forward = ensure_grafana_port_forward(config)

        auth = resolve_grafana_auth(config, kubectl)
        grafana = GrafanaClient(grafana_url, auth, timeout_seconds=args.timeout)

        dashboards = select_dashboards(grafana, args.dashboard, args.all, args.max_dashboards)
        print(f"Selected dashboards: {len(dashboards)}")

        time_range = TimeRange.last_minutes(args.minutes)
        max_queries = None if args.max_queries == 0 else args.max_queries

        report = ProbeReport()

        for dashboard in dashboards:
            results, warnings = probe_dashboard(
                grafana, dashboard,
                variable_overrides=var_overrides,
                preferred_datasource=args.datasource,
                time_range=time_range,
                max_queries=max_queries,
            )
            # Categorize warnings as errors (unresolved vars etc.)
            for w in warnings:
                report.errors.append(CategorizedQuery(
                    dashboard_title=dashboard.title,
                    dashboard_uid=dashboard.uid,
                    panel_title="(variable resolution)",
                    ref_id="-",
                    expr="",
                    category="error",
                    detail=w,
                    grafana_url=f"{grafana_url}/d/{dashboard.uid}",
                ))

            # Categorize query results
            for r in results:
                cq = _categorize(r, dashboard.uid, grafana_url)
                if cq.category == "ok":
                    report.ok.append(cq)
                elif cq.category == "no_data":
                    report.no_data.append(cq)
                else:
                    report.errors.append(cq)

        print_report(report)
    finally:
        if port_forward is not None:
            port_forward.stop()

    return 0


# ---------------------------------------------------------------------------
# Categorization
# ---------------------------------------------------------------------------


def _categorize(r: QueryResult, dashboard_uid: str, grafana_url: str) -> CategorizedQuery:
    """Categorize a query result into ok/no_data/error."""
    base = dict(
        dashboard_title=r.dashboard_title,
        dashboard_uid=dashboard_uid,
        panel_title=r.panel_title,
        ref_id=r.ref_id,
        expr=re.sub(r"[ \t]*\n[ \t]*", " ", r.expr).strip(),
        grafana_url=f"{grafana_url}/d/{dashboard_uid}",
    )

    if r.error:
        return CategorizedQuery(**base, category="error", detail=r.error[:200])

    if not r.response:
        return CategorizedQuery(**base, category="error", detail="no response received")

    # Parse response
    results = r.response.get("results", {})
    ref_data = results.get(r.ref_id) or next(iter(results.values()), {})
    if not isinstance(ref_data, dict):
        return CategorizedQuery(**base, category="error", detail="unexpected response format")

    error = ref_data.get("error")
    if error:
        return CategorizedQuery(**base, category="error", detail=str(error)[:200])

    frames = ref_data.get("frames", [])
    total_series = len(frames)
    total_points = 0
    for frame in frames:
        if not isinstance(frame, dict):
            continue
        values = frame.get("data", {}).get("values", [])
        if values and isinstance(values[0], list):
            total_points += len(values[0])

    if total_points == 0:
        return CategorizedQuery(**base, category="no_data", detail=f"{total_series} series, 0 points")

    return CategorizedQuery(**base, category="ok", detail=f"{total_series} series, {total_points} points")


# ---------------------------------------------------------------------------
# Report printing
# ---------------------------------------------------------------------------


def print_report(report: ProbeReport) -> None:
    """Print categorized report: OK, NO DATA, ERRORS."""
    print(f"\n{'='*70}")
    print(f"PROBE REPORT")
    print(f"{'='*70}")
    print(f"  Total queries:  {report.total}")
    print(f"  OK (has data):  {len(report.ok)}")
    print(f"  No data:        {len(report.no_data)}")
    print(f"  Errors:         {len(report.errors)}")
    print(f"{'='*70}")

    # --- OK ---
    if report.ok:
        print(f"\n{'─'*70}")
        print(f"OK — {len(report.ok)} queries returned data")
        print(f"{'─'*70}")
        _print_by_dashboard(report.ok, show_expr=False)

    # --- NO DATA ---
    if report.no_data:
        print(f"\n{'─'*70}")
        print(f"NO DATA — {len(report.no_data)} queries returned empty results")
        print(f"{'─'*70}")
        _print_by_dashboard(report.no_data, show_expr=True)

    # --- ERRORS ---
    if report.errors:
        print(f"\n{'─'*70}")
        print(f"ERRORS — {len(report.errors)} queries failed")
        print(f"{'─'*70}")
        _print_by_dashboard(report.errors, show_expr=True)


def _print_by_dashboard(items: list[CategorizedQuery], show_expr: bool) -> None:
    """Group and print items by dashboard."""
    # Group by dashboard
    by_dashboard: dict[str, list[CategorizedQuery]] = {}
    for item in items:
        key = item.dashboard_title
        by_dashboard.setdefault(key, []).append(item)

    for dash_title, queries in by_dashboard.items():
        uid = queries[0].dashboard_uid
        url = queries[0].grafana_url
        print(f"\n  {dash_title} ({len(queries)} queries)")
        print(f"  {url}")
        for q in queries:
            line = f"    [{q.panel_title}] {q.detail}"
            print(line)
            if show_expr and q.expr:
                print(f"      expr: {q.expr[:150]}")


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------


def parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description="Probe Grafana Prometheus dashboard queries.")
    p.add_argument("--dashboard", help="Dashboard title or UID.")
    p.add_argument("--all", action="store_true", help="Probe all dashboards.")
    p.add_argument("--max-dashboards", type=int, default=1, help="Max dashboards (0=all).")
    p.add_argument("--max-queries", type=int, default=3, help="Max queries per dashboard (0=all).")
    p.add_argument("--var", action="append", default=[], metavar="NAME=VALUE")
    p.add_argument("--datasource", help="Preferred datasource name/UID.")
    p.add_argument("--minutes", type=int, default=30, help="Query last N minutes.")
    p.add_argument("--timeout", type=int, default=30, help="HTTP timeout seconds.")
    p.add_argument("--no-port-forward", action="store_true")
    p.add_argument("--full-response", action="store_true", help="Print full JSON responses.")
    return p.parse_args()


def parse_var_overrides(raw: list[str]) -> dict[str, str | list[str]]:
    overrides: dict[str, str | list[str]] = {}
    for item in raw:
        if "=" not in item:
            raise SystemExit(f"--var must be NAME=VALUE, got {item!r}")
        name, value = item.split("=", 1)
        overrides[name.strip()] = value.split(",") if "," in value else value
    return overrides


def select_dashboards(
    grafana: GrafanaClient,
    selector: str | None,
    all_flag: bool,
    max_count: int,
) -> list[GrafanaDashboard]:
    dashboards = grafana.list_dashboards()
    if selector:
        matches = [d for d in dashboards if d.uid == selector or d.title == selector]
        if not matches:
            raise SystemExit(f"Dashboard not found: {selector}")
        return matches

    # Filter to prometheus-only dashboards
    prometheus_dashboards = []
    skipped = 0
    for d in dashboards:
        try:
            model = grafana.get_dashboard_json(d.uid)
            if is_prometheus_dashboard(model):
                prometheus_dashboards.append(d)
            else:
                skipped += 1
        except RuntimeError:
            skipped += 1

    print(f"Prometheus dashboards: {len(prometheus_dashboards)}, skipped: {skipped}")

    if all_flag or max_count == 0:
        return prometheus_dashboards
    return prometheus_dashboards[:max_count]


if __name__ == "__main__":
    raise SystemExit(main())
