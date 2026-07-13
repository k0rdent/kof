"""VictoriaLogs dashboard probe using the same requests as Grafana UI."""
from __future__ import annotations

from datetime import datetime, timedelta
from typing import Any, Mapping

from framework.grafana import GrafanaClient, GrafanaDashboard, GrafanaDatasource
from framework.prometheus import (
    QueryResult,
    TimeRange,
    VarValue,
    _apply_var_regex,
    _current_values,
    _iter_targets,
    _resolve_ds_ref,
    _resolve_query_var,
    _should_use_all,
    execute_dashboard_queries,
    interpolate,
    resolve_variables,
)


VICTORIA_LOGS_DATASOURCE_TYPE = "victoriametrics-logs-datasource"
FIELD_VALUES_RESOURCE = "select/logsql/field_values"


def is_victoria_logs_dashboard(model: dict[str, Any]) -> bool:
    """Return True when a dashboard contains VictoriaLogs panel queries."""
    for _, target, panel_ds, _ in _iter_targets(model):
        if not isinstance(target.get("expr"), str):
            continue
        for datasource in (target.get("datasource"), panel_ds):
            if (
                isinstance(datasource, dict)
                and datasource.get("type") == VICTORIA_LOGS_DATASOURCE_TYPE
            ):
                return True
    return False


def probe_victoria_logs_dashboard(
    grafana: GrafanaClient,
    dashboard: GrafanaDashboard,
    *,
    datasources: list[GrafanaDatasource] | None = None,
    variable_overrides: Mapping[str, str | list[str]] | None = None,
    variable_preferences: Mapping[str, Mapping[str, str]] | None = None,
    preferred_datasource: str | None = None,
    time_range: TimeRange | None = None,
    max_queries: int | None = None,
) -> tuple[list[QueryResult], list[str]]:
    """Resolve variables and execute all VictoriaLogs panel queries."""
    tr = time_range or TimeRange.last_minutes()
    ds_list = datasources or grafana.list_datasources()
    model = grafana.get_dashboard_json(dashboard.uid)
    logs_ds = _resolve_ds_ref(
        {"type": VICTORIA_LOGS_DATASOURCE_TYPE},
        {},
        ds_list,
        preferred_datasource,
    )
    if logs_ds is not None and logs_ds.type != VICTORIA_LOGS_DATASOURCE_TYPE:
        logs_ds = next(
            (ds for ds in ds_list if ds.type == VICTORIA_LOGS_DATASOURCE_TYPE),
            None,
        )
    if logs_ds is None:
        return [], [f"{dashboard.title}: no VictoriaLogs datasource found"]

    variables, warnings = resolve_variables(
        model,
        grafana,
        ds_list,
        overrides=variable_overrides or {},
        preferences=variable_preferences or {},
        preferred_ds=None,
        time_range=tr,
        query_resolver=_resolve_field_value_variable,
    )

    return execute_dashboard_queries(
        grafana,
        dashboard,
        model,
        variables,
        warnings,
        datasources=ds_list,
        preferred_datasource=logs_ds.uid,
        time_range=tr,
        max_queries=max_queries,
        datasource_matches=lambda ds_type: ds_type == VICTORIA_LOGS_DATASOURCE_TYPE,
        build_payload=_build_logs_payload,
    )


def _resolve_field_value_variable(
    var_def: dict[str, Any],
    name: str,
    resolved: dict[str, VarValue],
    grafana: GrafanaClient,
    datasources: list[GrafanaDatasource],
    preference: Mapping[str, str] | None,
    preferred_datasource: str | None,
    time_range: TimeRange,
) -> VarValue | None:
    query_model = var_def.get("query")
    if not isinstance(query_model, dict) or query_model.get("type") != "fieldValue":
        return _resolve_query_var(
            var_def, name, resolved, grafana, datasources,
            preference, preferred_datasource, time_range,
        )

    field = str(query_model.get("field") or "")
    if not field:
        raise RuntimeError("fieldValue variable has no field")

    datasource = _resolve_ds_ref(
        var_def.get("datasource"),
        resolved,
        datasources,
        preferred_datasource,
    )
    if datasource is None or datasource.type != VICTORIA_LOGS_DATASOURCE_TYPE:
        raise RuntimeError("no VictoriaLogs datasource for fieldValue query")

    query = interpolate(str(query_model.get("query") or "*"), resolved, {})
    start_ms, end_ms = _local_day_range_ms()
    response = grafana.datasource_resource_post(
        datasource.uid,
        FIELD_VALUES_RESOURCE,
        {
            "query": query,
            "field": field,
            "start": str(start_ms),
            "end": str(end_ms),
        },
    )
    values = _field_values(response)
    values = _apply_var_regex(values, var_def)

    if _should_use_all(var_def):
        all_value = var_def.get("allValue")
        return VarValue(
            name=name,
            values=tuple(values),
            is_all=True,
            all_value=str(all_value) if all_value is not None else None,
        )

    current = _current_values(var_def)
    if current and all(value in values for value in current):
        return VarValue(name=name, values=current)
    return VarValue(name=name, values=(next((value for value in values if value), ""),))


def _field_values(response: Any) -> list[str]:
    if not isinstance(response, dict):
        return []
    values = response.get("values", [])
    if not isinstance(values, list):
        return []
    return [
        str(item.get("value") or "")
        for item in values
        if isinstance(item, dict)
    ]


def _local_day_range_ms() -> tuple[int, int]:
    """Match Grafana UI variable queries, which use the local calendar day."""
    now = datetime.now().astimezone()
    start = now.replace(hour=0, minute=0, second=0, microsecond=0)
    end = start + timedelta(days=1)
    return int(start.timestamp() * 1000), int(end.timestamp() * 1000) - 1


def _build_logs_payload(
    target: dict[str, Any],
    datasource: GrafanaDatasource,
    expr: str,
    time_range: TimeRange,
) -> dict[str, Any]:
    query = {
        **target,
        "datasource": {"type": datasource.type, "uid": datasource.uid},
        "expr": expr,
        "extraFilters": target.get("extraFilters", ""),
        "maxLines": target.get("maxLines", 1000),
        "intervalMs": time_range.interval_ms,
        "maxDataPoints": time_range.max_data_points,
    }
    return {
        "queries": [query],
        "from": str(time_range.from_ms),
        "to": str(time_range.to_ms),
    }
