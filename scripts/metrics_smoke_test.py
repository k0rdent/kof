import base64
import json
import os
import subprocess
import time
import urllib.error
import urllib.parse
import urllib.request

import pytest

# Config (overridable via environment variables)

NAMESPACE = "kof"
GRAFANA_URL = os.environ.get("GRAFANA_URL", "http://localhost:3000")
CREDENTIALS_SECRET = "grafana-admin-credentials"
DATASOURCE_NAME = os.environ.get("METRICS_DATASOURCE", "kof-metrics")
TIMEOUT = int(os.environ.get("SMOKE_TIMEOUT", "300"))
POLL = 10

MGMT_CLUSTER = os.environ.get("MGMT_CLUSTER", "mothership")
REGIONAL_CLUSTER = os.environ.get("REGIONAL_CLUSTER", "regional-adopted")
CHILD_CLUSTER = os.environ.get("CHILD_CLUSTER", "child-adopted")

get_labels = lambda cluster, env_var: f'cluster="{cluster}"' + (
    f", {labels}" if (labels := os.environ.get(env_var, "")) else ""
)

MGMT_LABELS = get_labels(MGMT_CLUSTER, "MGMT_LABELS")
REGIONAL_LABELS = get_labels(REGIONAL_CLUSTER, "REGIONAL_LABELS")
CHILD_LABELS = get_labels(CHILD_CLUSTER, "CHILD_LABELS")

METRICS = [
    metric % labels
    for metric in [
        'up{job=~"kubernetes-apiservers|apiserver", %s}',
        'up{app_kubernetes_io_name="kof-collectors-daemon-collector", %s}',
        'sum(node_total_hourly_cost{%s})',
    ]
    for labels in [MGMT_LABELS, REGIONAL_LABELS, CHILD_LABELS]
] + [
    'vm_app_uptime_seconds{%s}' % REGIONAL_LABELS,
]

# Helpers


def kubectl(*args: str, stdin: str | None = None) -> str:
    r = subprocess.run(["kubectl", *args], input=stdin, capture_output=True, text=True)
    assert r.returncode == 0, r.stderr.strip()
    return r.stdout


def wait_for(description: str, fn, timeout: int = TIMEOUT) -> None:
    deadline = time.monotonic() + timeout
    while True:
        try:
            fn()
            return
        except AssertionError as exc:
            if time.monotonic() >= deadline:
                pytest.fail(f"Timed out ({timeout}s) waiting for {description}: {exc}")
            print(
                f"  [{deadline - time.monotonic():.0f}s] waiting for {description}..."
            )
            time.sleep(POLL)


# Fixtures


@pytest.fixture(scope="module")
def grafana_credentials() -> tuple[str, str]:
    """Read Grafana admin credentials from the Kubernetes secret."""
    username = kubectl(
        "-n",
        NAMESPACE,
        "get",
        "secret",
        CREDENTIALS_SECRET,
        "-o",
        "jsonpath={.data.GF_SECURITY_ADMIN_USER}",
    ).strip()
    password = kubectl(
        "-n",
        NAMESPACE,
        "get",
        "secret",
        CREDENTIALS_SECRET,
        "-o",
        "jsonpath={.data.GF_SECURITY_ADMIN_PASSWORD}",
    ).strip()
    assert username and password, (
        f"Secret {CREDENTIALS_SECRET!r} is missing credentials"
    )
    return base64.b64decode(username).decode(), base64.b64decode(password).decode()


@pytest.fixture(scope="module")
def metrics_datasource_uid(grafana_credentials: tuple[str, str]) -> str:
    """Resolve the UID of the metrics datasource from Grafana."""
    user, password = grafana_credentials
    token = base64.b64encode(f"{user}:{password}".encode()).decode()
    url = f"{GRAFANA_URL}/api/datasources/name/{urllib.parse.quote(DATASOURCE_NAME)}"
    req = urllib.request.Request(url, headers={"Authorization": f"Basic {token}"})
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            data = json.loads(resp.read())
    except urllib.error.HTTPError as e:
        pytest.fail(
            f"Failed to fetch datasource {DATASOURCE_NAME!r}: HTTP {e.code} {e.reason}"
        )
    uid = data.get("uid", "")
    assert uid, f"Datasource {DATASOURCE_NAME!r} has no UID in response: {data}"
    return uid


# Tests


def test_credentials_secret_exists() -> None:
    """The Grafana admin credentials secret exists in the namespace."""
    kubectl("-n", NAMESPACE, "get", "secret", CREDENTIALS_SECRET)


@pytest.mark.parametrize("metric", METRICS)
def test_metric_has_data(
    grafana_credentials: tuple[str, str],
    metrics_datasource_uid: str,
    metric: str,
) -> None:
    """Each metric query returns at least one result via the Grafana datasource proxy."""
    user, password = grafana_credentials
    token = base64.b64encode(f"{user}:{password}".encode()).decode()

    def check() -> None:
        query_time = f"{time.time() - 60:.3f}"
        params = urllib.parse.urlencode({"query": metric, "time": query_time})
        url = (
            f"{GRAFANA_URL}/api/datasources/proxy/uid/{metrics_datasource_uid}"
            f"/api/v1/query?{params}"
        )
        req = urllib.request.Request(url, headers={"Authorization": f"Basic {token}"})
        try:
            with urllib.request.urlopen(req, timeout=10) as resp:
                response = json.loads(resp.read())
        except urllib.error.HTTPError as e:
            raise AssertionError(f"HTTP {e.code} {e.reason}: {e.read().decode()[:200]}")
        except (urllib.error.URLError, TimeoutError) as e:
            raise AssertionError(f"Connection error: {e}")

        assert response.get("status") == "success", f"Query failed: {response}"
        result = response.get("data", {}).get("result", [])
        assert result, f"No data returned for metric {metric!r}"

    wait_for(f"metric data ({metric!r})", check)
