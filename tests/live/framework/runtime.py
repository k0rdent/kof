from __future__ import annotations

import socket

from framework.config import LiveTestConfig
from framework.grafana import GrafanaAuth
from framework.kubernetes import KubectlClient, KubectlError
from framework.port_forward import PortForwardProcess


def port_is_open(port: int) -> bool:
    """Return True when localhost:port accepts TCP connections."""
    try:
        with socket.create_connection(("127.0.0.1", port), timeout=1):
            return True
    except (ConnectionRefusedError, OSError):
        return False


def resolve_grafana_auth(
    config: LiveTestConfig,
    kubectl: KubectlClient,
) -> GrafanaAuth:
    """Resolve Grafana credentials from env vars or the cluster secret."""
    if config.grafana_api_token:
        return GrafanaAuth.bearer(config.grafana_api_token)

    if config.grafana_username and config.grafana_password:
        return GrafanaAuth.basic(config.grafana_username, config.grafana_password)

    secret_name = config.grafana_credentials_secret
    try:
        username = kubectl.get_secret_value(
            config.namespace,
            secret_name,
            "GF_SECURITY_ADMIN_USER",
        )
        password = kubectl.get_secret_value(
            config.namespace,
            secret_name,
            "GF_SECURITY_ADMIN_PASSWORD",
        )
    except KubectlError as exc:
        raise RuntimeError(
            f"Cannot obtain Grafana credentials from secret "
            f"'{config.namespace}/{secret_name}': {exc}"
        ) from exc

    return GrafanaAuth.basic(username, password)


def ensure_grafana_port_forward(
    config: LiveTestConfig,
) -> PortForwardProcess | None:
    """Start Grafana port-forward unless localhost already has a listener.

    Used by standalone scripts (e.g. probe_grafana_prometheus.py).
    Test fixtures in conftest.py have their own implementation with
    pytest-specific error handling and generator-based cleanup.
    """
    if port_is_open(config.grafana_port):
        return None

    return PortForwardProcess.start(
        kubectl=config.kubectl,
        context=config.kubectl_context,
        namespace=config.namespace,
        service=config.grafana_service,
        local_port=config.grafana_port,
    )
