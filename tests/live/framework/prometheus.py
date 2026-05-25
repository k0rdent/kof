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
        """Built-in Grafana variables derived from time range."""
        range_ms = max(self.to_ms - self.from_ms, 1)
        return {
            "__from": str(self.from_ms),
            "__to": str(self.to_ms),
            "__interval": _format_duration(self.interval_ms),
            "__interval_ms": str(self.interval_ms),
            "__rate_interval": _format_duration(max(self.interval_ms, 60_000)),
            "__range": _format_duration(range_ms),
            "__range_s": str(max(range_ms // 1000, 1)),
            "__range_ms": str(range_ms),
        }


@dataclass(frozen=True)
class VarValue:
    """Resolved template variable."""

    name: str
    values: tuple[str, ...]
    is_all: bool = False
    all_value: str = ".*"

    def format(self, fmt: str | None = None) -> str:
        """Format for PromQL substitution."""
        if self.is_all:
            return self.all_value
        if not self.values:
            return ""
        if len(self.values) == 1:
            return self.values[0]
        # Multi-value → regex alternation
        return "(" + "|".join(re.escape(v) for v in self.values) + ")"


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

    results: list[QueryResult] = []
    for panel_title, target, panel_ds in _iter_targets(model):
        if target.get("hide") is True:
            continue
        raw_expr = target.get("expr")
        if not isinstance(raw_expr, str) or not raw_expr.strip():
            continue

        # Resolve datasource
        ds = _resolve_ds_ref(
            target.get("datasource") or panel_ds, variables, ds_list, preferred_datasource,
        )
        if ds is None or not _is_prometheus_type(ds.type):
            continue

        # Interpolate variables
        expr = interpolate(raw_expr, variables, tr.builtins)
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
                val = _current_value(var_def) or "5m"
                resolved = VarValue(name=name, values=(str(val),))
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

    # Kahn's algorithm
    in_degree = {n: 0 for n in var_defs}
    for name, dep_set in deps.items():
        for dep in dep_set:
            if dep in in_degree:
                in_degree[name] += 1  # name depends on dep

    # Reverse map: who depends on me
    dependents: dict[str, list[str]] = {n: [] for n in var_defs}
    for name, dep_set in deps.items():
        for dep in dep_set:
            if dep in dependents:
                dependents[dep].append(name)

    queue = [n for n in var_defs if in_degree[n] == 0]
    # Preserve original order for items with same in-degree
    original_order = {name: i for i, name in enumerate(var_defs)}
    queue.sort(key=lambda n: original_order.get(n, 0))

    result: list[str] = []
    while queue:
        node = queue.pop(0)
        result.append(node)
        for dependent in sorted(dependents[node], key=lambda n: original_order.get(n, 0)):
            in_degree[dependent] -= 1
            if in_degree[dependent] == 0:
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
        return VarValue(name=name, values=(), is_all=True,
                        all_value=str(var_def.get("allValue") or ".*"))
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
    """
    if _is_all(var_def):
        return VarValue(name=name, values=(), is_all=True,
                        all_value=str(var_def.get("allValue") or ".*"))

    query = var_def.get("query") or var_def.get("definition") or ""
    if isinstance(query, dict):
        query = query.get("query", "")
    if not query:
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
                return None  # Cannot resolve, fallback to current.value
            qr_expr = qr_expr_clean
        available = _exec_query_result(grafana, ds_uid, qr_expr, time_range)
        available = _apply_var_regex(available, var_def)
        if not available:
            raise RuntimeError(f"query_result returned empty for {query[:80]}")
        return _pick_value(var_def, name, available)

    # Parse label_values(metric{filter}, label) pattern
    label_query = _parse_label_values(query)
    if label_query is None:
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
        raise RuntimeError(f"label_values returned empty for {query}")

    return _pick_value(var_def, name, available)


def _pick_value(var_def: dict[str, Any], name: str, available: list[str]) -> VarValue:
    """Pick best value: use current if valid, otherwise first available."""
    current = _current_values(var_def)
    if current and all(v in available for v in current):
        return VarValue(name=name, values=current)
    return VarValue(name=name, values=(available[0],))


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




def _fallback_current(var_def: dict[str, Any], name: str) -> VarValue | None:
    """Last resort: use current.value from dashboard JSON."""
    if _is_all(var_def):
        return VarValue(name=name, values=(), is_all=True,
                        all_value=str(var_def.get("allValue") or ".*"))
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


def _iter_targets(model: dict[str, Any]):
    """Yield (panel_title, target_dict, panel_datasource) from all panels."""
    def _walk(panels):
        if not isinstance(panels, list):
            return
        for panel in panels:
            if not isinstance(panel, dict):
                continue
            title = str(panel.get("title") or "(untitled)")
            ds = panel.get("datasource")
            for target in panel.get("targets", []):
                if isinstance(target, dict):
                    yield title, target, ds
            # Recurse into row panels
            yield from _walk(panel.get("panels"))

    yield from _walk(model.get("panels"))


def is_prometheus_dashboard(model: dict[str, Any]) -> bool:
    """Check if dashboard has at least one target with a Prometheus-like datasource."""
    for _, target, panel_ds in _iter_targets(model):
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


def _format_duration(ms: int) -> str:
    """Format milliseconds as Prometheus duration (1h, 5m, 30s)."""
    ms = max(int(ms), 1)
    for unit_ms, suffix in [(3_600_000, "h"), (60_000, "m"), (1000, "s")]:
        if ms >= unit_ms and ms % unit_ms == 0:
            return f"{ms // unit_ms}{suffix}"
    return f"{ms}ms"
