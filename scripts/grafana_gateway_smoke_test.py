import pytest
import os
import subprocess
import time

# Config (overridable via environment variables)

NAMESPACE   = "kof"
GATEWAY     = os.environ.get("GATEWAY", "mothership-gateway")
HOSTNAME    = "grafana.example.com"
TLS_SECRET  = "kof-https"
CERT        = "kof-https"
KIND_NODE   = f"{os.environ.get('KIND_CLUSTER', 'kcm-dev')}-control-plane"
TIMEOUT     = int(os.environ.get("SMOKE_TIMEOUT", "300"))
POLL        = 10

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

# Setup fixture (runs once per test session)

@pytest.fixture(scope="module", autouse=True)
def setup_tls() -> None:
    """Wait for cert-manager to issue the certificate and for the Gateway to be programmed."""
    def cert_ready() -> None:
        out = kubectl("-n", NAMESPACE, "get", "certificate", CERT,
                      "-o", "jsonpath={.status.conditions[?(@.type=='Ready')].status}")
        assert out.strip() == "True", "not Ready yet"

    wait_for(f"Certificate {CERT!r}", cert_ready)

    def gateway_programmed() -> None:
        out = kubectl("-n", NAMESPACE, "get", "gateway", GATEWAY,
                      "-o", "jsonpath={.status.conditions[?(@.type=='Programmed')].status}")
        assert out.strip() == "True", "not Programmed yet"

    wait_for(f"Gateway {GATEWAY!r}", gateway_programmed)


@pytest.fixture(scope="module")
def gateway_addr() -> str:
    addr = kubectl("-n", NAMESPACE, "get", "gateway", GATEWAY,
                   "-o", "jsonpath={.status.addresses[0].value}").strip()
    assert addr, "Gateway has no address"
    return addr

# Tests

def test_tls_secret_exists() -> None:
    """cert-manager populated the TLS secret referenced by the Gateway listener."""
    kubectl("-n", NAMESPACE, "get", "secret", TLS_SECRET)

def test_grafana_https(gateway_addr: str) -> None:
    """Grafana responds over HTTPS through the Envoy Gateway (200 or 302 → /login)."""
    def check() -> None:
        r = subprocess.run(
            ["docker", "exec", KIND_NODE,
             "curl", "-skI",
             "--resolve", f"{HOSTNAME}:8443:{gateway_addr}",
             f"https://{HOSTNAME}:8443/"],
            capture_output=True, text=True, timeout=30,
        )
        first = r.stdout.splitlines()[0] if r.stdout.strip() else r.stderr.strip()
        assert "HTTP/" in first, f"no HTTP response: {first!r}"
        assert any(c in first for c in ("200", "302")), f"unexpected status: {first!r}"
        if "302" in first:
            assert "location: /login" in r.stdout.lower(), f"302 not to /login:\n{r.stdout}"

    wait_for("Grafana HTTPS", check, timeout=60)
