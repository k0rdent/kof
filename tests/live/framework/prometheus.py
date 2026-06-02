"""Prometheus dashboard probe — resolve variables and execute panel queries.

Resolves Grafana template variables via live label_values queries against
Prometheus, then executes panel expressions through /api/ds/query.
"""
from __future__ import annotations

import re
import time
from dataclasses import dataclass
from typing import Any, Mapping
from urllib.parse import quote

from framework.grafana import GrafanaClient, GrafanaDashboard, GrafanaDatasource


PROMETHEUS_DATASOURCE_TYPES = {
    "prometheus",
    "victoriametrics-datasource",
    "victoriametrics-metrics-datasource",
}

VARIABLE_PATTERN = re.compile(
    r"\$\{(?P<braced>[A-Za-z_][A-Za-z0-9_]*)(?::(?P<braced_format>[^}]+))?\}"
    r"|\$(?P<plain>[A-Za-z_][A-Za-z0-9_]*)"
    r"|\[\[(?P<legacy>[A-Za-z_][A-Za-z0-9_]*)(?::(?P<legacy_format>[^\]]+))?\]\]"
)


# ---------------------------------------------------------------------------
# Data structures
# ---------------------------------------------------------------------------


@dataclass(frozen=True)
class TimeRange:
    """Query time range as epoch milliseconds."""

    from_ms: int
    to_ms: int
    interval_ms: int = 15_000
    max_data_points: int = 1000

    @classmethod
    def last_minutes(cls, minutes: int = 30) -> TimeRange:
        to_ms = int(time.time() * 1000)
        return cls(from_ms=to_ms - minutes * 60_000, to_ms=to_ms)

    @property
    def builtins(self) -> dict[str, str]:
        """Built-in Grafana variables derived from time range.

        NOTE: $__interval, $__rate_interval, $__range, $__interval_ms,
        $__range_s, $__range_ms are intentionally NOT included here.
        Grafana resolves these server-side via /api/ds/query using
        intervalMs and maxDataPoints from the request payload.
        Interpolating them client-side produces different values
        (e.g. probe computes $__rate_interval="1m" but Grafana uses "20s").
        """
        interval = _format_duration(self.interval_ms)
        return {
            "__from": str(self.from_ms),
            "__to": str(self.to_ms),
            # Grafana auto-generates $__auto_interval_X for interval variables;
            # these are NOT resolved server-side, must be handled client-side
            "__auto_interval_interval": interval,
            "__auto_interval_resolution": interval,
        }


@dataclass(frozen=True)
class VarValue:
    """Resolved template variable."""

    name: str
    values: tuple[str, ...]
    is_all: bool = False
    all_value: str | None = None

    def format(self, fmt: str | None = None) -> str:
        """Format for PromQL substitution.

        When is_all=True, matches Grafana UI behavior:
        - If all_value is explicitly set (from dashboard JSON allValue) → use it
        - If single value → escaped value without parens (Grafana sends bare value)
        - If multiple values → (val1|val2|...) regex alternation
        - If no values → () empty alternation
        """
        if self.is_all:
            if self.all_value is not None:
                return self.all_value
            if self.values:
                return _format_prometheus_regex_values(self.values)
            return "()"
        if not self.values:
            return ""
        if fmt == "regex":
            return _format_prometheus_regex_values(self.values)
        if len(self.values) == 1:
            return self.values[0]
        # Multi-value → regex alternation
        return _format_prometheus_regex_values(self.values)


@dataclass(frozen=True)
class QueryResult:
    """Result of a single panel query execution."""

    dashboard_title: str
    panel_title: str
    ref_id: str
    datasource_uid: str
    raw_expr: str
    expr: str
    response: dict[str, Any] | None
    error: str | None = None


def query_error_message(result: QueryResult) -> str | None:
    """Return the query-level error message, if Grafana reported one."""
    if result.error:
        return result.error
    if result.response is None:
        return "no response received"

    ref_data = _query_ref_data(result)
    if ref_data is None:
        return "unexpected response format"

    error = ref_data.get("error")
    return str(error) if error else None


def query_has_data(result: QueryResult) -> bool:
    """Return True when a Grafana dataframe contains at least one non-null value."""
    return query_data_summary(result)[1] > 0


def query_data_summary(result: QueryResult) -> tuple[int, int]:
    """Return (series_count, non_null_value_points) for a query response."""
    if query_error_message(result):
        return 0, 0

    ref_data = _query_ref_data(result)
    if ref_data is None:
        return 0, 0

    frames = ref_data.get("frames", [])
    if not isinstance(frames, list):
        return 0, 0

    total_series = 0
    total_points = 0
    for frame in frames:
        if not isinstance(frame, dict):
            continue
        total_series += 1
        schema = frame.get("schema", {})
        fields = schema.get("fields", []) if isinstance(schema, dict) else []
        values = frame.get("data", {}).get("values", [])
        if not isinstance(values, list):
            continue

        for index, value_array in enumerate(values):
            field = fields[index] if index < len(fields) and isinstance(fields[index], dict) else {}
            if field.get("type") == "time":
                continue
            if isinstance(value_array, list):
                total_points += sum(1 for value in value_array if value is not None)
            elif value_array is not None:
                total_points += 1

    return total_series, total_points


def _query_ref_data(result: QueryResult) -> dict[str, Any] | None:
    if not isinstance(result.response, dict):
        return None
    results = result.response.get("results", {})
    if not isinstance(results, dict):
        return None
    ref_data = results.get(result.ref_id)
    if ref_data is None and results:
        ref_data = next(iter(results.values()))
    return ref_data if isinstance(ref_data, dict) else None


# ---------------------------------------------------------------------------
# Core: probe a dashboard
# ---------------------------------------------------------------------------


def probe_dashboard(
    grafana: GrafanaClient,
    dashboard: GrafanaDashboard,
    *,
    datasources: list[GrafanaDatasource] | None = None,
    variable_overrides: Mapping[str, str | list[str]] | None = None,
    preferred_datasource: str | None = None,
    time_range: TimeRange | None = None,
    max_queries: int | None = None,
) -> tuple[list[QueryResult], list[str]]:
    """Probe all Prometheus queries in a dashboard.

    Returns (results, warnings).
    """
    tr = time_range or TimeRange.last_minutes()
    ds_list = datasources or grafana.list_datasources()
    model = grafana.get_dashboard_json(dashboard.uid)

    # Resolve variables
    variables, warnings = resolve_variables(
        model, grafana, ds_list,
        overrides=variable_overrides or {},
        preferred_ds=preferred_datasource,
        time_range=tr,
    )
    adhoc_filters = _extract_adhoc_filters(model)

    results: list[QueryResult] = []
    for panel_title, target, panel_ds, scoped_vars in _iter_targets(model, variables):
        if target.get("hide") is True:
            continue
        raw_expr = target.get("expr")
        if not isinstance(raw_expr, str) or not raw_expr.strip():
            continue

        target_vars = {**variables, **scoped_vars}

        # Resolve datasource
        ds = _resolve_ds_ref(
            target.get("datasource") or panel_ds, target_vars, ds_list, preferred_datasource,
        )
        if ds is None or not _is_prometheus_type(ds.type):
            continue

        # Interpolate variables
        expr = interpolate(raw_expr, target_vars, tr.builtins)
        expr = _apply_adhoc_filters(expr, adhoc_filters)
        if _has_unresolved(expr):
            warnings.append(f"{panel_title}: unresolved vars in {expr[:80]}")
            continue

        # Execute
        payload = _build_payload(target, ds, expr, tr)
        try:
            response = grafana.query_datasource(payload)
            error = None
        except RuntimeError as exc:
            response = None
            error = str(exc)

        results.append(QueryResult(
            dashboard_title=dashboard.title,
            panel_title=panel_title,
            ref_id=str(target.get("refId", "A")),
            datasource_uid=ds.uid,
            raw_expr=raw_expr,
            expr=expr,
            response=response,
            error=error,
        ))

        if max_queries and len(results) >= max_queries:
            break

    return results, warnings


# ---------------------------------------------------------------------------
# Variable resolution
# ---------------------------------------------------------------------------


def resolve_variables(
    model: dict[str, Any],
    grafana: GrafanaClient,
    datasources: list[GrafanaDatasource],
    *,
    overrides: Mapping[str, str | list[str]],
    preferred_ds: str | None,
    time_range: TimeRange,
) -> tuple[dict[str, VarValue], list[str]]:
    """Resolve all dashboard template variables with dependency-aware ordering.

    Uses topological sort to handle chained dependencies (e.g. node depends on
    cluster which depends on job). Supports query, adhoc, datasource, custom,
    constant, textbox, and interval variable types.

    Returns (values, warnings).
    """
    variables: dict[str, VarValue] = {}
    warnings: list[str] = []
    var_list = model.get("templating", {}).get("list", [])
    if not isinstance(var_list, list):
        return variables, warnings

    # Build name→def map
    var_defs: dict[str, dict[str, Any]] = {}
    for var_def in var_list:
        if not isinstance(var_def, dict):
            continue
        name = var_def.get("name")
        if isinstance(name, str) and name:
            var_defs[name] = var_def

    # Topological sort based on $var references in query strings
    resolution_order = _topological_sort(var_defs)

    for name in resolution_order:
        var_def = var_defs[name]

        # Manual override
        if name in overrides:
            raw = overrides[name]
            vals = tuple(raw) if isinstance(raw, list) else (raw,)
            variables[name] = VarValue(name=name, values=vals)
            continue

        var_type = str(var_def.get("type", ""))
        resolved = None

        try:
            if var_type == "datasource":
                resolved = _resolve_ds_var(var_def, datasources, preferred_ds)
            elif var_type in ("constant", "textbox"):
                val = _current_value(var_def) or var_def.get("query") or ""
                resolved = VarValue(name=name, values=(str(val),))
            elif var_type == "interval":
                val = str(_current_value(var_def) or "5m")
                # Grafana stores "$__auto_interval_<name>" as current.value
                # when auto-interval is enabled. Resolve to computed interval.
                if val.startswith("$"):
                    val = _format_duration(_auto_interval_ms(time_range, var_def))
                resolved = VarValue(name=name, values=(val,))
            elif var_type == "custom":
                resolved = _resolve_custom_var(var_def, name)
            elif var_type == "adhoc":
                # Adhoc filters: resolve to empty (= no user-selected filters)
                resolved = VarValue(name=name, values=("",), is_all=True, all_value="")
            elif var_type == "query":
                resolved = _resolve_query_var(
                    var_def, name, variables, grafana, datasources,
                    preferred_ds, time_range,
                )
        except RuntimeError as exc:
            warnings.append(f"{name}: {exc}")

        # Fallback: use current.value from dashboard JSON
        if resolved is None:
            resolved = _fallback_current(var_def, name)
        if resolved is None:
            warnings.append(f"{name}: could not resolve (type={var_type})")
            continue

        variables[name] = resolved

    return variables, warnings


def _topological_sort(var_defs: dict[str, dict[str, Any]]) -> list[str]:
    """Sort variables so dependencies are resolved first.

    Extracts $var references from query strings and orders resolution
    accordingly. Falls back to original order for cycles or missing deps.
    """
    all_names = set(var_defs.keys())

    # Build adjacency: name → set of names it depends on
    deps: dict[str, set[str]] = {}
    for name, var_def in var_defs.items():
        query = var_def.get("query") or var_def.get("definition") or ""
        if isinstance(query, dict):
            query = query.get("query", "")
        query = str(query)
        # Find all $var references in the query
        refs = set()
        for m in VARIABLE_PATTERN.finditer(query):
            ref = m.group("braced") or m.group("plain") or m.group("legacy") or ""
            if ref and ref in all_names and ref != name:
                refs.add(ref)
        deps[name] = refs

    # Kahn's algorithm — unsatisfied_deps tracks how many dependencies
    # each variable still needs resolved before it can be processed.
    unsatisfied_deps = {n: 0 for n in var_defs}
    for name, dep_set in deps.items():
        for dep in dep_set:
            if dep in unsatisfied_deps:
                unsatisfied_deps[name] += 1

    # Reverse map: who depends on me
    dependents: dict[str, list[str]] = {n: [] for n in var_defs}
    for name, dep_set in deps.items():
        for dep in dep_set:
            if dep in dependents:
                dependents[dep].append(name)

    queue = [n for n in var_defs if unsatisfied_deps[n] == 0]
    # Preserve original order for items with same dependency count
    original_order = {name: i for i, name in enumerate(var_defs)}
    queue.sort(key=lambda n: original_order.get(n, 0))

    result: list[str] = []
    while queue:
        node = queue.pop(0)
        result.append(node)
        for dependent in sorted(dependents[node], key=lambda n: original_order.get(n, 0)):
            unsatisfied_deps[dependent] -= 1
            if unsatisfied_deps[dependent] == 0:
                queue.append(dependent)

    # Append any remaining (cycles) in original order
    remaining = [n for n in var_defs if n not in set(result)]
    remaining.sort(key=lambda n: original_order.get(n, 0))
    result.extend(remaining)

    return result


def _resolve_ds_var(
    var_def: dict[str, Any],
    datasources: list[GrafanaDatasource],
    preferred: str | None,
) -> VarValue:
    """Resolve datasource variable to UID."""
    name = str(var_def["name"])
    ds_type = str(var_def.get("query") or "prometheus")
    # Try preferred first, then current value, then first matching type
    for candidate in [preferred, _current_value(var_def), _current_text(var_def)]:
        if candidate:
            for ds in datasources:
                if ds.uid == candidate or ds.name == candidate:
                    return VarValue(name=name, values=(ds.uid,))
    # Fallback: first matching type
    for ds in datasources:
        if _is_prometheus_type(ds.type) if _is_prometheus_type(ds_type) else ds.type == ds_type:
            return VarValue(name=name, values=(ds.uid,))
    raise RuntimeError(f"no datasource found for type {ds_type}")


def _resolve_custom_var(var_def: dict[str, Any], name: str) -> VarValue | None:
    """Resolve custom variable from current selection or options."""
    if _is_all(var_def):
        explicit_all = var_def.get("allValue") or None
        if explicit_all:
            return VarValue(name=name, values=(), is_all=True,
                            all_value=str(explicit_all))
        # No explicit allValue → collect all option values for (v1|v2|...) format
        options = var_def.get("options", [])
        all_values = []
        if isinstance(options, list):
            for opt in options:
                if isinstance(opt, dict):
                    val = str(opt.get("value", ""))
                    if val and val != "$__all":
                        all_values.append(val)
        return VarValue(name=name, values=tuple(all_values), is_all=True,
                        all_value=None)
    values = _current_values(var_def)
    if values:
        return VarValue(name=name, values=values)
    # Try first selected option
    options = var_def.get("options", [])
    if isinstance(options, list):
        for opt in options:
            if isinstance(opt, dict) and opt.get("selected"):
                return VarValue(name=name, values=(str(opt.get("value", "")),))
    return None


def _resolve_query_var(
    var_def: dict[str, Any],
    name: str,
    resolved: dict[str, VarValue],
    grafana: GrafanaClient,
    datasources: list[GrafanaDatasource],
    preferred_ds: str | None,
    time_range: TimeRange,
) -> VarValue | None:
    """Resolve query variable via live Prometheus queries.

    Supports:
    - label_values(metric{filter}, label)
    - query_result(expr) — extracts label values from instant query
    - Retry without unresolved $var selectors in filters

    Matches Grafana UI behavior:
    - includeAll=True with no saved selection → resolve as "All"
    - Empty results + includeAll → resolve as "All"
    - Empty results + no includeAll → resolve as "" (empty dropdown)
    """
    # Grafana defaults to "All" for provisioned dashboards with includeAll=True
    use_all = _should_use_all(var_def)
    explicit_all_value = var_def.get("allValue") or None
    if use_all and explicit_all_value:
        # Dashboard has explicit allValue (e.g. ".*", ".+") — use as-is, skip query
        return VarValue(name=name, values=(), is_all=True,
                        all_value=str(explicit_all_value))

    query = var_def.get("query") or var_def.get("definition") or ""
    if isinstance(query, dict):
        query = query.get("query", "")
    if not query:
        if use_all:
            return VarValue(name=name, values=(), is_all=True, all_value=None)
        return None

    # Interpolate already-resolved vars into the query
    query = interpolate(str(query), resolved, time_range.builtins)

    # Find datasource for query
    ds_uid = _find_ds_uid(var_def.get("datasource"), datasources, preferred_ds)
    if not ds_uid:
        raise RuntimeError("no datasource for label_values query")

    # Try query_result(expr) pattern first
    qr_expr = _parse_query_result(query)
    if qr_expr is not None:
        if _has_unresolved(qr_expr):
            # Strip unresolved vars from selectors and retry
            qr_expr_clean = _strip_unresolved_selectors(qr_expr)
            if _has_unresolved(qr_expr_clean):
                if use_all:
                    return VarValue(name=name, values=(), is_all=True, all_value=None)
                return None  # Cannot resolve, fallback to current.value
            qr_expr = qr_expr_clean
        available = _exec_query_result(grafana, ds_uid, qr_expr, time_range)
        available = _apply_var_regex(available, var_def)
        if not available:
            return _empty_var_value(var_def, name, query, "query_result")
        if use_all:
            return VarValue(name=name, values=tuple(available), is_all=True, all_value=None)
        return _pick_value(var_def, name, available)

    # Parse label_values(metric{filter}, label) pattern
    label_query = _parse_label_values(query)
    if label_query is None:
        if use_all:
            return VarValue(name=name, values=(), is_all=True, all_value=None)
        return None  # Unsupported query format → will fallback to current.value

    match_expr, label = label_query

    # If match_expr still has unresolved vars, try stripping those selectors
    if match_expr and _has_unresolved(match_expr):
        match_expr = _strip_unresolved_selectors(match_expr)
        if _has_unresolved(match_expr):
            # Last resort: query just the label without metric filter
            match_expr = None

    # Execute label_values
    available = _exec_label_values(grafana, ds_uid, match_expr, label)
    available = _apply_var_regex(available, var_def)
    if not available:
        return _empty_var_value(var_def, name, query, "label_values")

    if use_all:
        return VarValue(name=name, values=tuple(available), is_all=True, all_value=None)
    return _pick_value(var_def, name, available)


def _pick_value(var_def: dict[str, Any], name: str, available: list[str]) -> VarValue:
    """Pick best value: use current if valid, otherwise smartest available.

    For namespace-like variables, skips known-empty namespaces (default,
    kube-public, kube-node-lease) and prefers namespaces likely to have
    workloads (kube-system, kcm-system, kof).
    """
    current = _current_values(var_def)
    if current and all(v in available for v in current):
        # Validate current is not a known-empty namespace
        if not _is_namespace_var(name) or not _is_boring_namespace(current[0]):
            return VarValue(name=name, values=current)
    return VarValue(name=name, values=(_pick_best_from_available(name, available),))


# Namespaces unlikely to have user workloads — skip when picking defaults
_BORING_NAMESPACES = frozenset({
    "default", "kube-public", "kube-node-lease", "monitoring",
})

# Preferred namespaces — likely to have running workloads, in priority order
_PREFERRED_NAMESPACES = ("kube-system", "kcm-system", "kof")


def _is_namespace_var(name: str) -> bool:
    """Check if variable name looks like a namespace selector."""
    return "namespace" in name.lower()


def _is_boring_namespace(value: str) -> bool:
    """Check if a namespace is known to be empty/boring."""
    return value in _BORING_NAMESPACES


def _pick_best_from_available(name: str, available: list[str]) -> str:
    """Pick the best value from available list.

    For namespace variables: prefer well-known active namespaces,
    skip known-empty ones. For other variables: first available.
    """
    if not _is_namespace_var(name):
        return available[0]

    # Try preferred namespaces first
    for preferred in _PREFERRED_NAMESPACES:
        if preferred in available:
            return preferred

    # Filter out boring namespaces
    interesting = [v for v in available if v not in _BORING_NAMESPACES]
    if interesting:
        return interesting[0]

    # All are boring — return first available anyway
    return available[0]


def _empty_var_value(
    var_def: dict[str, Any], name: str, query: str, source: str,
) -> VarValue:
    """Handle empty results from variable query — match Grafana UI behavior.

    Grafana behavior when query returns no results:
    - If includeAll=True → "All" option still available → use allValue
    - If includeAll=False → empty dropdown → variable is "" (empty string)

    In both cases, Grafana does NOT error — panels just show "No data".
    """
    if var_def.get("includeAll"):
        explicit_all = var_def.get("allValue") or None
        # explicit allValue → use it; otherwise → () empty alternation (matches UI)
        return VarValue(name=name, values=(), is_all=True,
                        all_value=str(explicit_all) if explicit_all else None)
    # No includeAll: Grafana shows empty dropdown, variable = ""
    return VarValue(name=name, values=("",))


def _exec_label_values(
    grafana: GrafanaClient,
    ds_uid: str,
    match_expr: str | None,
    label: str,
) -> list[str]:
    """Execute label_values query against Prometheus."""
    if match_expr:
        data = grafana.datasource_proxy_get(ds_uid, "/api/v1/series", query={"match[]": [match_expr]})
        series = data.get("data") if isinstance(data, dict) else None
        if not isinstance(series, list):
            return []
        return sorted({str(s[label]) for s in series if isinstance(s, dict) and s.get(label)})
    else:
        data = grafana.datasource_proxy_get(ds_uid, f"/api/v1/label/{quote(label, safe='')}/values")
        values_list = data.get("data") if isinstance(data, dict) else None
        if not isinstance(values_list, list):
            return []
        return sorted(str(v) for v in values_list if v)


def _exec_query_result(
    grafana: GrafanaClient,
    ds_uid: str,
    expr: str,
    time_range: TimeRange,
) -> list[str]:
    """Execute query_result() — run instant query, extract label values.

    For 'sum(...) by (label)' queries, extracts all label values from results.
    Returns raw values before regex filtering.
    """
    # Use instant query via /api/v1/query
    data = grafana.datasource_proxy_get(
        ds_uid, "/api/v1/query",
        query={"query": re.sub(r"[ \t]*\n[ \t]*", " ", expr).strip(), "time": str(time_range.to_ms // 1000)},
    )
    result = data.get("data", {}).get("result", []) if isinstance(data, dict) else []
    if not isinstance(result, list):
        return []

    # Extract ALL label values from each result item
    values: set[str] = set()
    for item in result:
        metric = item.get("metric", {}) if isinstance(item, dict) else {}
        if not isinstance(metric, dict):
            continue
        # Collect all non-__name__ label values
        for k, v in metric.items():
            if k != "__name__" and v:
                values.add(str(v))

    return sorted(values)


def _apply_var_regex(values: list[str], var_def: dict[str, Any]) -> list[str]:
    """Apply variable's regex field to filter/transform values.

    Grafana supports:
    - Simple filter: /regex/ — keep only matching values
    - Capture group: /(.*)/ — extract captured group as value
    """
    regex_str = var_def.get("regex")
    if not regex_str or not isinstance(regex_str, str):
        return values
    regex_str = regex_str.strip()
    if not regex_str:
        return values

    # Strip surrounding slashes if present (Grafana convention)
    if regex_str.startswith("/") and regex_str.endswith("/"):
        regex_str = regex_str[1:-1]
    if not regex_str:
        return values

    try:
        pattern = re.compile(regex_str)
    except re.error:
        return values  # Invalid regex, skip filtering

    result: list[str] = []
    for v in values:
        m = pattern.search(v)
        if m:
            # If regex has capture groups, use first group as value
            if m.lastindex and m.lastindex >= 1:
                captured = m.group(1)
                if captured:
                    result.append(captured)
            else:
                result.append(v)

    return sorted(set(result))


def _parse_query_result(query: str) -> str | None:
    """Parse query_result(expr) → expr."""
    m = re.match(r"^\s*query_result\s*\((.+)\)\s*$", query, re.DOTALL)
    if not m:
        return None
    return m.group(1).strip()


def _strip_unresolved_selectors(expr: str) -> str:
    """Remove label selectors that still contain $var from a PromQL expression.

    Example: metric{job="$job", cluster="$cluster", instance="foo"}
    → metric{instance="foo"}
    """
    def _clean_selectors(m: re.Match[str]) -> str:
        inside = m.group(1)
        # Split selectors by comma, keep only those without $var
        parts = []
        for part in _split_selectors(inside):
            stripped = part.strip()
            if stripped and not VARIABLE_PATTERN.search(stripped):
                parts.append(stripped)
        if parts:
            return "{" + ", ".join(parts) + "}"
        return ""

    return re.sub(r"\{([^}]*)\}", _clean_selectors, expr)


def _split_selectors(s: str) -> list[str]:
    """Split comma-separated label selectors, respecting quotes."""
    parts: list[str] = []
    current: list[str] = []
    in_quote = False
    quote_char = ""
    for ch in s:
        if ch in ('"', "'") and not in_quote:
            in_quote = True
            quote_char = ch
            current.append(ch)
        elif ch == quote_char and in_quote:
            in_quote = False
            current.append(ch)
        elif ch == "," and not in_quote:
            parts.append("".join(current))
            current = []
        else:
            current.append(ch)
    if current:
        parts.append("".join(current))
    return parts


def _format_prometheus_regex_values(values: tuple[str, ...]) -> str:
    """Format values as a Grafana-like Prometheus regex alternation.

    Grafana interpolates multi/all Prometheus variables into regex matchers and
    escapes regex metacharacters for a PromQL double-quoted string. That means a
    literal dot must become two backslashes plus dot in the final PromQL
    source, not one backslash plus dot.
    Python's re.escape() is not suitable here because it over-escapes harmless
    characters like '-' and under-escapes backslashes for PromQL string syntax.
    """
    escaped = [_escape_prometheus_regex_value(value) for value in values]
    if len(escaped) == 1:
        return escaped[0]
    return "(" + "|".join(escaped) + ")"


def _escape_prometheus_regex_value(value: str) -> str:
    """Escape a literal value for use inside a Prometheus regex string.

    PromQL regex matchers use double-quoted Go regex strings. To match a
    literal regex metacharacter (e.g. '.'), the final regex engine needs
    '\\.' — but since it's inside a double-quoted PromQL string, each
    backslash must itself be escaped as '\\\\'. So a literal '.' in the
    source value becomes '\\\\.' in the interpolated PromQL expression.
    """
    # Hyphen is intentionally not escaped: Grafana leaves it as-is outside
    # character classes, and escaping it creates noisy diffs from UI payloads.
    regex_special = set(r"\.^$*+?()|{}[]")
    out: list[str] = []
    for ch in value:
        if ch in regex_special:
            out.append("\\\\")
        out.append(ch)
    return "".join(out)




def _fallback_current(var_def: dict[str, Any], name: str) -> VarValue | None:
    """Last resort: use current.value from dashboard JSON."""
    if _is_all(var_def):
        explicit_all = var_def.get("allValue") or None
        return VarValue(name=name, values=(), is_all=True,
                        all_value=str(explicit_all) if explicit_all else None)
    values = _current_values(var_def)
    if values:
        return VarValue(name=name, values=values)
    return None


# ---------------------------------------------------------------------------
# Interpolation
# ---------------------------------------------------------------------------


def interpolate(
    text: str,
    variables: Mapping[str, VarValue],
    builtins: Mapping[str, str],
) -> str:
    """Replace $var, ${var:fmt}, [[var]] with resolved values."""
    def _replace(match: re.Match[str]) -> str:
        name = match.group("braced") or match.group("plain") or match.group("legacy") or ""
        fmt = match.group("braced_format") or match.group("legacy_format")
        if name in variables:
            return variables[name].format(fmt)
        if name in builtins:
            return builtins[name]
        # Grafana auto-generates $__auto_interval_<varname> for interval variables;
        # NOT resolved server-side — must interpolate client-side
        if name.startswith("__auto_interval_"):
            interval_var = name[len("__auto_interval_"):]
            if interval_var in variables:
                return variables[interval_var].format(fmt)
            # Fallback to explicit auto_interval entries or default 15s
            return builtins.get(name, "15s")
        return match.group(0)

    return VARIABLE_PATTERN.sub(_replace, text)


def _has_unresolved(expr: str) -> bool:
    """Check if expression still has unresolved non-builtin variables."""
    for m in VARIABLE_PATTERN.finditer(expr):
        name = m.group("braced") or m.group("plain") or m.group("legacy")
        if name and not name.startswith("__"):
            return True
    return False


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _iter_targets(
    model: dict[str, Any],
    variables: Mapping[str, VarValue] | None = None,
):
    """Yield panel targets, expanding Grafana repeat panels when possible."""
    def _walk(panels, inherited_scope: dict[str, VarValue]):
        if not isinstance(panels, list):
            return
        for panel in panels:
            if not isinstance(panel, dict):
                continue
            for scoped_vars in _repeat_scopes(panel, variables, inherited_scope):
                title = str(panel.get("title") or "(untitled)")
                if scoped_vars:
                    title = interpolate(title, scoped_vars, {})
                ds = panel.get("datasource")
                for target in panel.get("targets", []):
                    if isinstance(target, dict):
                        yield title, target, ds, scoped_vars
                # Recurse into row panels
                yield from _walk(panel.get("panels"), scoped_vars)

    yield from _walk(model.get("panels"), {})


def _repeat_scopes(
    panel: dict[str, Any],
    variables: Mapping[str, VarValue] | None,
    inherited_scope: dict[str, VarValue],
) -> list[dict[str, VarValue]]:
    repeat_var = panel.get("repeat")
    if not repeat_var or not isinstance(repeat_var, str) or variables is None:
        return [inherited_scope]

    value = inherited_scope.get(repeat_var) or variables.get(repeat_var)
    if value is None or not value.values:
        return [inherited_scope]

    return [
        {**inherited_scope, repeat_var: VarValue(name=repeat_var, values=(item,))}
        for item in value.values
        if item
    ] or [inherited_scope]


def _extract_adhoc_filters(model: dict[str, Any]) -> tuple[str, ...]:
    templating = model.get("templating", {}).get("list", [])
    if not isinstance(templating, list):
        return ()

    filters: list[str] = []
    for var_def in templating:
        if not isinstance(var_def, dict) or var_def.get("type") != "adhoc":
            continue
        raw_filters = var_def.get("filters", [])
        if not isinstance(raw_filters, list):
            continue
        for item in raw_filters:
            if not isinstance(item, dict):
                continue
            key = str(item.get("key") or "").strip()
            operator = str(item.get("operator") or "=").strip()
            value = str(item.get("value") or "")
            if key and operator in {"=", "!=", "=~", "!~"}:
                filters.append(f'{key}{operator}"{_escape_promql_string(value)}"')
    return tuple(filters)


def _apply_adhoc_filters(expr: str, filters: tuple[str, ...]) -> str:
    if not filters:
        return expr

    out: list[str] = []
    i = 0
    while i < len(expr):
        ch = expr[i]
        if ch in {'"', "'"}:
            end = _consume_quoted(expr, i)
            out.append(expr[i:end])
            i = end
            continue
        if ch == "{":
            end = _consume_selector(expr, i)
            out.append(_merge_adhoc_filters(expr[i:end], filters))
            i = end
            continue
        if _is_ident_start(ch) and not (i > 0 and _is_ident_char(expr[i - 1])):
            end = i + 1
            while end < len(expr) and _is_ident_char(expr[end]):
                end += 1
            ident = expr[i:end]
            next_char, next_index = _next_non_space(expr, end)
            if ident in _PROMQL_LABEL_LIST_MODIFIERS and next_char == "(":
                close = _consume_balanced_parens(expr, next_index)
                out.append(expr[i:close])
                i = close
                continue
            if _should_add_adhoc_selector(expr, i, end, ident, next_char):
                out.append(ident)
                out.append("{" + ",".join(filters) + "}")
            else:
                out.append(ident)
            i = end
            continue
        out.append(ch)
        i += 1
    return "".join(out)


_PROMQL_KEYWORDS = {
    "and", "or", "unless", "bool", "offset", "by", "without", "on",
    "ignoring", "group_left", "group_right", "nan", "inf",
    # Aggregation operators — needed so that `<op> by (...) (expr)` syntax
    # is not mistaken for a metric name (next_char would be 'b', not '(').
    "sum", "min", "max", "avg", "group", "stddev", "stdvar", "count",
    "count_values", "bottomk", "topk", "quantile", "limitk", "limit_ratio",
    # Functions that look like metric names
    "abs", "absent", "absent_over_time", "ceil", "changes", "clamp",
    "clamp_max", "clamp_min", "day_of_month", "day_of_week", "day_of_year",
    "days_in_month", "delta", "deriv", "exp", "floor", "histogram_quantile",
    "holt_winters", "hour", "idelta", "increase", "irate", "label_join",
    "label_replace", "ln", "log2", "log10", "minute", "month", "predict_linear",
    "rate", "resets", "round", "scalar", "sgn", "sort", "sort_desc", "sqrt",
    "time", "timestamp", "vector", "year",
    "avg_over_time", "min_over_time", "max_over_time", "sum_over_time",
    "count_over_time", "quantile_over_time", "stddev_over_time",
    "stdvar_over_time", "last_over_time", "present_over_time",
}
_PROMQL_LABEL_LIST_MODIFIERS = {
    "by", "without", "on", "ignoring", "group_left", "group_right",
}


def _should_add_adhoc_selector(
    expr: str,
    start: int,
    end: int,
    ident: str,
    next_char: str | None,
) -> bool:
    if ident in _PROMQL_KEYWORDS:
        return False
    if next_char in {"(", "{"}:
        return False
    if start > 0 and expr[start - 1].isdigit():
        return False
    return True


def _merge_adhoc_filters(selector: str, filters: tuple[str, ...]) -> str:
    body = selector[1:-1].strip()
    additions = [
        item for item in filters
        if not _selector_has_filter(body, item.split("=", 1)[0].split("!", 1)[0].strip())
    ]
    if not additions:
        return selector
    if body:
        return "{" + body + "," + ",".join(additions) + "}"
    return "{" + ",".join(additions) + "}"


def _selector_has_filter(selector_body: str, key: str) -> bool:
    return bool(re.search(rf'(^|,)\s*{re.escape(key)}\s*(=|!=|=~|!~)', selector_body))


def _escape_promql_string(value: str) -> str:
    return value.replace("\\", "\\\\").replace('"', '\\"')


def _is_ident_start(ch: str) -> bool:
    return ch.isalpha() or ch in "_:"


def _is_ident_char(ch: str) -> bool:
    return ch.isalnum() or ch in "_:"


def _next_non_space(text: str, index: int) -> tuple[str | None, int]:
    while index < len(text) and text[index].isspace():
        index += 1
    if index >= len(text):
        return None, index
    return text[index], index


def _consume_quoted(text: str, start: int) -> int:
    quote = text[start]
    i = start + 1
    while i < len(text):
        if text[i] == "\\":
            i += 2
            continue
        if text[i] == quote:
            return i + 1
        i += 1
    return len(text)


def _consume_selector(text: str, start: int) -> int:
    i = start + 1
    while i < len(text):
        if text[i] in {'"', "'"}:
            i = _consume_quoted(text, i)
            continue
        if text[i] == "}":
            return i + 1
        i += 1
    return len(text)


def _consume_balanced_parens(text: str, start: int) -> int:
    depth = 0
    i = start
    while i < len(text):
        if text[i] in {'"', "'"}:
            i = _consume_quoted(text, i)
            continue
        if text[i] == "(":
            depth += 1
        elif text[i] == ")":
            depth -= 1
            if depth == 0:
                return i + 1
        i += 1
    return len(text)


def is_prometheus_dashboard(model: dict[str, Any]) -> bool:
    """Check if dashboard has at least one target with a Prometheus-like datasource."""
    for _, target, panel_ds, _ in _iter_targets(model):
        if not isinstance(target.get("expr"), str):
            continue
        ds_ref = target.get("datasource") or panel_ds
        ds_type = None
        if isinstance(ds_ref, dict):
            ds_type = ds_ref.get("type")
        if ds_type and _is_prometheus_type(str(ds_type)):
            return True
        # No explicit type but has expr → likely prometheus
        if ds_type is None and target.get("expr"):
            return True
    return False


def _build_payload(
    target: dict[str, Any],
    ds: GrafanaDatasource,
    expr: str,
    tr: TimeRange,
) -> dict[str, Any]:
    """Build /api/ds/query request payload.

    Respects target's instant/range flags — instant queries return 1 value,
    range queries return time series over the full time range.
    """
    is_instant = bool(target.get("instant", False))
    is_range = bool(target.get("range", not is_instant))

    return {
        "queries": [{
            "refId": str(target.get("refId", "A")),
            "datasource": {"type": ds.type, "uid": ds.uid},
            "expr": re.sub(r"[ \t]*\n[ \t]*", " ", expr).strip(),
            "intervalMs": tr.interval_ms,
            "maxDataPoints": 1 if is_instant else tr.max_data_points,
            "instant": is_instant,
            "range": is_range,
            "format": target.get("format", "time_series"),
        }],
        "from": str(tr.from_ms),
        "to": str(tr.to_ms),
    }


def _resolve_ds_ref(
    ds_ref: Any,
    variables: dict[str, VarValue],
    datasources: list[GrafanaDatasource],
    preferred: str | None,
) -> GrafanaDatasource | None:
    """Resolve a panel's datasource reference to a concrete datasource."""
    uid_or_name: str | None = None
    ds_type: str | None = None

    if isinstance(ds_ref, dict):
        uid_or_name = str(ds_ref.get("uid") or ds_ref.get("name") or "")
        ds_type = str(ds_ref.get("type") or "")
    elif isinstance(ds_ref, str):
        uid_or_name = ds_ref

    # Resolve variable references in uid
    if uid_or_name and "$" in uid_or_name:
        uid_or_name = interpolate(uid_or_name, variables, {})

    if uid_or_name and uid_or_name not in {"", "default", "-- Grafana --"}:
        for ds in datasources:
            if ds.uid == uid_or_name or ds.name == uid_or_name:
                return ds

    # Fallback: preferred or first prometheus
    return _find_ds(datasources, preferred, ds_type or "prometheus")


def _find_ds_uid(
    ds_ref: Any,
    datasources: list[GrafanaDatasource],
    preferred: str | None,
) -> str | None:
    """Find datasource UID for a variable's datasource field."""
    ds = _resolve_ds_ref(ds_ref, {}, datasources, preferred)
    return ds.uid if ds else None


def _find_ds(
    datasources: list[GrafanaDatasource],
    preferred: str | None,
    ds_type: str,
) -> GrafanaDatasource | None:
    """Find datasource by preference or type."""
    if preferred:
        for ds in datasources:
            if ds.uid == preferred or ds.name == preferred:
                return ds
    for ds in datasources:
        if _is_prometheus_type(ds_type) and _is_prometheus_type(ds.type):
            return ds
        if ds.type == ds_type:
            return ds
    return None


def _is_prometheus_type(t: str) -> bool:
    return t.lower() in PROMETHEUS_DATASOURCE_TYPES or "prometheus" in t.lower()


def _parse_label_values(query: str) -> tuple[str | None, str] | None:
    """Parse label_values(metric{filter}, label) → (match_expr, label)."""
    m = re.match(r"^\s*label_values\s*\((.+)\)\s*$", query)
    if not m:
        return None
    body = m.group(1)
    # Split on last comma not inside braces
    depth = 0
    last_comma = -1
    for i, ch in enumerate(body):
        if ch in "({[":
            depth += 1
        elif ch in ")}]":
            depth -= 1
        elif ch == "," and depth == 0:
            last_comma = i
    if last_comma == -1:
        return None, body.strip().strip("\"'")
    return body[:last_comma].strip(), body[last_comma + 1:].strip().strip("\"'")


def _current_value(var_def: dict[str, Any]) -> Any:
    current = var_def.get("current")
    return current.get("value") if isinstance(current, dict) else None


def _current_text(var_def: dict[str, Any]) -> Any:
    current = var_def.get("current")
    return current.get("text") if isinstance(current, dict) else None


def _current_values(var_def: dict[str, Any]) -> tuple[str, ...]:
    value = _current_value(var_def)
    if value is None or value == "" or value == "$__all":
        return ()
    if isinstance(value, list):
        return tuple(str(v) for v in value if v not in (None, "", "$__all"))
    return (str(value),)


def _is_all(var_def: dict[str, Any]) -> bool:
    value = _current_value(var_def)
    if value == "$__all":
        return True
    return isinstance(value, list) and "$__all" in {str(v) for v in value}


def _should_use_all(var_def: dict[str, Any]) -> bool:
    """Check if variable should default to 'All' on initial load.

    Grafana behavior: when includeAll=True AND no specific value is saved
    (provisioned dashboard with current=None), the default selection is "All".
    """
    if _is_all(var_def):
        return True
    if var_def.get("includeAll") and _current_value(var_def) in (None, "", "$__all"):
        return True
    return False


def _auto_interval_ms(time_range: TimeRange, var_def: dict[str, Any]) -> int:
    range_ms = max(time_range.to_ms - time_range.from_ms, 1)
    auto_count = int(var_def.get("auto_count") or 30)
    raw_interval = range_ms / max(auto_count, 1)
    rounded = _round_interval_ms(raw_interval)
    auto_min = _parse_duration_ms(str(var_def.get("auto_min") or "")) or 0
    return max(rounded, auto_min)


def _round_interval_ms(interval_ms: float) -> int:
    """Round an interval similarly to Grafana's dashboard auto interval."""
    thresholds = [
        (10, 1),
        (15, 10),
        (35, 20),
        (75, 50),
        (150, 100),
        (350, 200),
        (750, 500),
        (1_500, 1_000),
        (3_500, 2_000),
        (7_500, 5_000),
        (12_500, 10_000),
        (17_500, 15_000),
        (25_000, 20_000),
        (45_000, 30_000),
        (90_000, 60_000),
        (210_000, 120_000),
        (450_000, 300_000),
        (750_000, 600_000),
        (1_050_000, 900_000),
        (1_500_000, 1_200_000),
        (2_700_000, 1_800_000),
        (5_400_000, 3_600_000),
        (9_000_000, 7_200_000),
        (16_200_000, 10_800_000),
        (32_400_000, 21_600_000),
        (86_400_000, 43_200_000),
        (604_800_000, 86_400_000),
        (1_814_400_000, 604_800_000),
        (3_628_800_000, 2_592_000_000),
    ]
    for threshold, rounded in thresholds:
        if interval_ms < threshold:
            return rounded
    return 31_536_000_000


def _parse_duration_ms(value: str) -> int | None:
    match = re.fullmatch(r"\s*(\d+)\s*(ms|s|m|h|d|w|M|y)\s*", value)
    if not match:
        return None
    amount = int(match.group(1))
    unit_ms = {
        "ms": 1,
        "s": 1_000,
        "m": 60_000,
        "h": 3_600_000,
        "d": 86_400_000,
        "w": 604_800_000,
        "M": 2_592_000_000,
        "y": 31_536_000_000,
    }
    return amount * unit_ms[match.group(2)]


def _format_duration(ms: int) -> str:
    """Format milliseconds as Prometheus duration (1h, 5m, 30s)."""
    ms = max(int(ms), 1)
    for unit_ms, suffix in [(3_600_000, "h"), (60_000, "m"), (1000, "s")]:
        if ms >= unit_ms and ms % unit_ms == 0:
            return f"{ms // unit_ms}{suffix}"
    return f"{ms}ms"
