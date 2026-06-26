from __future__ import annotations

import logging
import time

from framework.dashboard_policy import DashboardPolicy
from framework.grafana import GrafanaClient
from framework.kubernetes import KubectlClient, KubectlError

logger = logging.getLogger(__name__)


def detect_components(
    grafana_client: GrafanaClient,
    kubectl_client: KubectlClient,
    policy: DashboardPolicy,
) -> dict[str, bool]:
    """Detect topology and optional components."""
    detected: dict[str, bool] = {}
    datasources = grafana_client.list_datasources()
    rules = {
        **policy.component_detectors,
        **{
            name: str(spec.get("detect", ""))
            for name, spec in policy.optional_dashboards.items()
        },
    }
    for component_name, detect_rule in rules.items():
        detected[component_name] = detect_component(
            grafana_client,
            str(detect_rule),
            datasources,
            kubectl_client,
        )
        logger.debug(
            "Component %s: %s (detect: %s)",
            component_name,
            "PRESENT" if detected[component_name] else "absent",
            detect_rule,
        )

    return detected


def detect_component(
    grafana_client: GrafanaClient,
    detect_rule: str,
    datasources: list,
    kubectl_client: KubectlClient | None = None,
) -> bool:
    """Detect if a component is present based on a detection rule.

    Rules:
      - "namespace:NAME" checks if namespace labels are scraped
      - "metric:METRIC_NAME{labels}" checks if metric exists
      - "kubernetes:container:NAMESPACE/CONTAINER" checks live pod specs
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
        query = f'count(kube_namespace_labels{{namespace="{namespace}"}})'
        return _run_detection_query(grafana_client, query, datasources)

    if detect_rule.startswith("metric:"):
        metric_expr = detect_rule[len("metric:"):]
        query = (
            f"count({metric_expr})"
            if "{" in metric_expr
            else f'count({{__name__="{metric_expr}"}})'
        )
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
            (
                r"jsonpath={range .items[*]}{.metadata.name}{'\t'}"
                r"{range .spec.containers[*]}{.name}{','}{end}{'\n'}{end}"
            ),
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
    for datasource in datasources:
        if datasource.type in (
            "prometheus",
            "victoriametrics-datasource",
            "victoriametrics-metrics-datasource",
        ):
            prom_ds = datasource
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
