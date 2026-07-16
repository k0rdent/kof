from __future__ import annotations

import os
from dataclasses import dataclass
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[3]


def _env_bool(name: str, default: bool = False) -> bool:
    """Parse a boolean from an environment variable."""
    value = os.environ.get(name)
    if value is None:
        return default
    return value.strip().lower() in {"1", "true", "yes", "y", "on"}


def _env_int(name: str, default: int) -> int:
    """Parse an integer from an environment variable with validation."""
    raw = os.environ.get(name)
    if raw is None:
        return default
    try:
        value = int(raw)
    except ValueError as exc:
        raise ValueError(
            f"Environment variable {name}={raw!r} is not a valid integer"
        ) from exc
    if value <= 0:
        raise ValueError(f"Environment variable {name}={value} must be positive")
    return value


def _env_float(name: str, default: float) -> float:
    """Parse a float from an environment variable with validation."""
    raw = os.environ.get(name)
    if raw is None:
        return default
    try:
        value = float(raw)
    except ValueError as exc:
        raise ValueError(
            f"Environment variable {name}={raw!r} is not a valid number"
        ) from exc
    if value <= 0:
        raise ValueError(f"Environment variable {name}={value} must be positive")
    return value


@dataclass(frozen=True)
class LiveTestConfig:
    """Runtime configuration shared by all live validation tests.

    All values are sourced from environment variables with sensible defaults
    for the standard local kind deployment (make dev-deploy).

    Environment variables:
        KOF_NAMESPACE               Namespace where KOF is installed (default: kof)
        KUBECTL                     kubectl binary path (default: kubectl)
        KUBECTL_CONTEXT             Kubernetes context (default: current context)
        GRAFANA_URL                 Grafana base URL (default: http://localhost:3000)
        GRAFANA_API_TOKEN           Bearer token for Grafana API auth
        GRAFANA_USER                Basic auth username (fallback to K8s secret)
        GRAFANA_PASSWORD            Basic auth password (fallback to K8s secret)
        GRAFANA_CREDENTIALS_SECRET  Secret name for auto-discovery (default: grafana-admin-credentials)
        GRAFANA_SERVICE             Service name for port-forward (default: svc/grafana-vm-service)
        GRAFANA_PORT                Local port for Grafana (default: 3000)
        KOF_REFERENCE_DASHBOARDS    Path to reference dashboards.yaml
        LIVE_TEST_TIMEOUT           Max wait time in seconds (default: 300)
        LIVE_TEST_POLL_INTERVAL     Poll interval in seconds (default: 10)
        LIVE_TEST_PRINT_DIAGNOSTICS Print live inventory summary (default: true)
        GRAFANA_ALLOW_EXTRA_DASHBOARDS  Allow dashboards not in reference (default: false)
        LIVE_TEST_FAST_RETRY        Cap dashboard query retries to a short
                                     budget for fast local iteration (default: false)
    """

    namespace: str
    kubectl: str
    kubectl_context: str | None
    grafana_url: str
    grafana_api_token: str | None
    grafana_username: str | None
    grafana_password: str | None
    grafana_credentials_secret: str
    grafana_service: str
    grafana_port: int
    reference_dashboards_path: Path
    timeout_seconds: int
    poll_interval_seconds: float
    print_diagnostics: bool
    allow_extra_dashboards: bool
    fast_retry: bool

    @classmethod
    def from_environment(cls) -> LiveTestConfig:
        """Build configuration from environment variables."""
        return cls(
            namespace=os.environ.get("KOF_NAMESPACE", "kof"),
            kubectl=os.environ.get("KUBECTL", "kubectl"),
            kubectl_context=os.environ.get("KUBECTL_CONTEXT") or None,
            grafana_url=os.environ.get("GRAFANA_URL", "http://localhost:3000").rstrip("/"),
            grafana_api_token=os.environ.get("GRAFANA_API_TOKEN") or None,
            grafana_username=os.environ.get("GRAFANA_USER") or None,
            grafana_password=os.environ.get("GRAFANA_PASSWORD") or None,
            grafana_credentials_secret=os.environ.get(
                "GRAFANA_CREDENTIALS_SECRET", "grafana-admin-credentials",
            ),
            grafana_service=os.environ.get("GRAFANA_SERVICE", "svc/grafana-vm-service"),
            grafana_port=_env_int("GRAFANA_PORT", 3000),
            reference_dashboards_path=Path(
                os.environ.get(
                    "KOF_REFERENCE_DASHBOARDS",
                    str(REPO_ROOT / "tests" / "reference" / "dashboards.yaml"),
                )
            ),
            timeout_seconds=_env_int("LIVE_TEST_TIMEOUT", 300),
            poll_interval_seconds=_env_float("LIVE_TEST_POLL_INTERVAL", 10),
            print_diagnostics=_env_bool("LIVE_TEST_PRINT_DIAGNOSTICS", True),
            allow_extra_dashboards=_env_bool("GRAFANA_ALLOW_EXTRA_DASHBOARDS"),
            fast_retry=_env_bool("LIVE_TEST_FAST_RETRY"),
        )
