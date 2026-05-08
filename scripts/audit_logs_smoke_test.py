import pytest
import os
import subprocess
import base64
import json
import urllib.request
import urllib.parse
import urllib.error
import time

# Config (overridable via environment variables)

NAMESPACE       = "kof"
GRAFANA_URL     = os.environ.get("GRAFANA_URL", "http://localhost:3000")
CREDENTIALS_SECRET = "grafana-admin-credentials"
DATASOURCE_NAME = os.environ.get("LOGS_DATASOURCE", "kof-logs")
AUDIT_LOG_QUERY = 'log_type:="k8s_audit" AND `k8s.cluster.name`:="{cluster}"'
TIMEOUT         = int(os.environ.get("SMOKE_TIMEOUT", "300"))
POLL            = 10

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
            print(f"  [{deadline - time.monotonic():.0f}s] waiting for {description}...")
            time.sleep(POLL)


# Fixtures

@pytest.fixture(scope="module")
def grafana_credentials() -> tuple[str, str]:
    """Read Grafana admin credentials from the Kubernetes secret."""
    username = kubectl(
        "-n", NAMESPACE, "get", "secret", CREDENTIALS_SECRET,
        "-o", "jsonpath={.data.GF_SECURITY_ADMIN_USER}",
    ).strip()
    password = kubectl(
        "-n", NAMESPACE, "get", "secret", CREDENTIALS_SECRET,
        "-o", "jsonpath={.data.GF_SECURITY_ADMIN_PASSWORD}",
    ).strip()
    assert username and password, f"Secret {CREDENTIALS_SECRET!r} is missing username or password"
    decoded_user = base64.b64decode(username).decode()
    decoded_pass = base64.b64decode(password).decode()
    return decoded_user, decoded_pass


@pytest.fixture(scope="module")
def logs_datasource_uid(grafana_credentials: tuple[str, str]) -> str:
    """Resolve the UID of the logs datasource from Grafana."""
    user, password = grafana_credentials
    token = base64.b64encode(f"{user}:{password}".encode()).decode()
    url = f"{GRAFANA_URL}/api/datasources/name/{urllib.parse.quote(DATASOURCE_NAME)}"
    req = urllib.request.Request(url, headers={"Authorization": f"Basic {token}"})
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            data = json.loads(resp.read())
    except urllib.error.HTTPError as e:
        pytest.fail(f"Failed to fetch datasource {DATASOURCE_NAME!r}: HTTP {e.code} {e.reason}")
    uid = data.get("uid", "")
    assert uid, f"Datasource {DATASOURCE_NAME!r} has no UID in response: {data}"
    return uid


# Tests

def test_credentials_secret_exists() -> None:
    """The Grafana admin credentials secret exists in the namespace."""
    kubectl("-n", NAMESPACE, "get", "secret", CREDENTIALS_SECRET)


@pytest.mark.parametrize("cluster", ["regional-adopted", "child-adopted"])
def test_audit_logs_present(
    grafana_credentials: tuple[str, str],
    logs_datasource_uid: str,
    cluster: str,
) -> None:
    """Audit log records with log_type=k8s_audit are present for the given cluster."""
    user, password = grafana_credentials
    token = base64.b64encode(f"{user}:{password}".encode()).decode()
    query = AUDIT_LOG_QUERY.format(cluster=cluster)

    def check() -> None:
        now = time.time()
        params = urllib.parse.urlencode({
            "query": query,
            "limit": "1",
            "start": str(int(now - 3600)),
            "end": str(int(now)),
        })
        url = (
            f"{GRAFANA_URL}/api/datasources/proxy/uid/{logs_datasource_uid}"
            f"/select/logsql/query?{params}"
        )
        req = urllib.request.Request(url, headers={"Authorization": f"Basic {token}"})
        try:
            with urllib.request.urlopen(req, timeout=10) as resp:
                body = resp.read().decode()
        except urllib.error.HTTPError as e:
            raise AssertionError(f"HTTP {e.code} {e.reason}: {e.read().decode()[:200]}")
        except (urllib.error.URLError, TimeoutError) as e:
            raise AssertionError(f"Connection error: {e}")

        # VictoriaLogs returns newline-delimited JSON; each line is one log entry
        lines = [l for l in body.splitlines() if l.strip()]
        assert lines, (
            f"No audit log records returned for query {query!r}.\n"
            f"Response body: {body[:500]!r}"
        )

    wait_for(f"audit log records ({query!r})", check)
