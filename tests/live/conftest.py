from __future__ import annotations

import logging

import pytest

from framework.config import LiveTestConfig
from framework.grafana import GrafanaAuth, GrafanaClient
from framework.kubernetes import KubectlClient
from framework.port_forward import PortForwardError, PortForwardProcess
from framework.reference import DashboardReference
from framework.runtime import port_is_open, resolve_grafana_auth

logger = logging.getLogger(__name__)


# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------


@pytest.fixture(scope="session")
def live_config() -> LiveTestConfig:
    """Load test configuration from environment variables."""
    return LiveTestConfig.from_environment()


# ---------------------------------------------------------------------------
# Kubernetes
# ---------------------------------------------------------------------------


@pytest.fixture(scope="session")
def kubectl_client(live_config: LiveTestConfig) -> KubectlClient:
    """Provide a configured kubectl client."""
    return KubectlClient(
        live_config.kubectl,
        live_config.kubectl_context,
    )


# ---------------------------------------------------------------------------
# Port-forward lifecycle
# ---------------------------------------------------------------------------

@pytest.fixture(scope="session")
def grafana_port_forward(
    live_config: LiveTestConfig,
    kubectl_client: KubectlClient,
) -> PortForwardProcess | None:
    """Ensure a port-forward to Grafana is running for the test session.

    If the port is already open (user started port-forward manually),
    this fixture is a no-op and returns None.

    Otherwise, starts a managed port-forward that is cleaned up after tests.
    """
    if port_is_open(live_config.grafana_port):
        logger.info(
            "port %d already open — assuming external port-forward",
            live_config.grafana_port,
        )
        yield None
        return

    logger.info("no listener on port %d — starting port-forward", live_config.grafana_port)
    try:
        pf = PortForwardProcess.start(
            kubectl=live_config.kubectl,
            context=live_config.kubectl_context,
            namespace=live_config.namespace,
            service=live_config.grafana_service,
            local_port=live_config.grafana_port,
        )
    except PortForwardError as exc:
        pytest.fail(
            f"Failed to start port-forward to Grafana: {exc}\n\n"
            f"Either start it manually before running tests:\n"
            f"  kubectl port-forward -n {live_config.namespace} "
            f"{live_config.grafana_service} {live_config.grafana_port}:{live_config.grafana_port}\n\n"
            f"Or ensure the cluster is accessible and Grafana is running."
        )
    yield pf
    pf.stop()


# ---------------------------------------------------------------------------
# Grafana authentication
# ---------------------------------------------------------------------------


@pytest.fixture(scope="session")
def grafana_auth(
    live_config: LiveTestConfig,
    kubectl_client: KubectlClient,
) -> GrafanaAuth:
    """Resolve Grafana credentials with clear error messages.

    Priority:
    1. GRAFANA_API_TOKEN env var (bearer token)
    2. GRAFANA_USER + GRAFANA_PASSWORD env vars (basic auth)
    3. Auto-discovery from Kubernetes Secret
    """
    try:
        return resolve_grafana_auth(live_config, kubectl_client)
    except RuntimeError as exc:
        pytest.fail(
            f"Cannot obtain Grafana credentials.\n\n"
            f"{exc}\n\n"
            f"Options to fix:\n"
            f"  1. Set GRAFANA_USER and GRAFANA_PASSWORD env vars\n"
            f"  2. Set GRAFANA_API_TOKEN env var\n"
            f"  3. Ensure the secret exists in the cluster"
        )


# ---------------------------------------------------------------------------
# Grafana client
# ---------------------------------------------------------------------------


@pytest.fixture(scope="session")
def grafana_client(
    live_config: LiveTestConfig,
    grafana_port_forward: PortForwardProcess | None,
    grafana_auth: GrafanaAuth,
) -> GrafanaClient:
    """Provide an authenticated Grafana client.

    Depends on grafana_port_forward to ensure connectivity exists.
    Verifies auth works before returning — fails fast with actionable message.
    """
    client = GrafanaClient(live_config.grafana_url, grafana_auth)

    # Validate connectivity and auth upfront.
    try:
        client.list_dashboards()
    except RuntimeError as exc:
        pytest.fail(
            f"Cannot query Grafana dashboard API: {exc}\n\n"
            f"Request used by this test:\n"
            f"  {client.dashboard_search_request()}\n\n"
            f"If this is a local run, verify the cluster is reachable and Grafana is running."
        )

    return client


# ---------------------------------------------------------------------------
# Reference data
# ---------------------------------------------------------------------------


@pytest.fixture(scope="session")
def dashboard_reference(live_config: LiveTestConfig) -> DashboardReference:
    """Load the static dashboard reference dataset.

    Fails with actionable message if the file doesn't exist.
    """
    path = live_config.reference_dashboards_path
    if not path.exists():
        pytest.fail(
            f"Reference file not found: {path}\n\n"
            f"Restore the static reference snapshot under tests/reference/."
        )
    return DashboardReference.load(path)
