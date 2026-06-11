from __future__ import annotations

import base64
from dataclasses import dataclass
from typing import Any
from urllib.parse import quote

import requests


@dataclass(frozen=True)
class GrafanaAuth:
    """HTTP Authorization header value for Grafana API requests."""

    header_value: str

    @classmethod
    def bearer(cls, token: str) -> GrafanaAuth:
        return cls(f"Bearer {token}")

    @classmethod
    def basic(cls, username: str, password: str) -> GrafanaAuth:
        token = base64.b64encode(f"{username}:{password}".encode()).decode()
        return cls(f"Basic {token}")


@dataclass(frozen=True)
class GrafanaDashboard:
    title: str
    uid: str
    folder_title: str


@dataclass(frozen=True)
class GrafanaDatasource:
    name: str
    uid: str
    type: str
    is_default: bool


class GrafanaClient:
    """Small Grafana HTTP API client for live KOF tests.

    The client owns request construction, authentication headers, response
    parsing, and user-facing error messages. Test code gets a simple method
    call and does not need to know how Grafana's HTTP layer behaves.
    """

    def __init__(
        self,
        base_url: str,
        auth: GrafanaAuth,
        timeout_seconds: int = 10,
        session: requests.Session | None = None,
    ) -> None:
        self._base_url = base_url.rstrip("/")
        self._timeout_seconds = timeout_seconds
        self._session = session or requests.Session()
        self._session.headers.update(
            {
                "Accept": "application/json",
                "Authorization": auth.header_value,
            }
        )

    @property
    def base_url(self) -> str:
        return self._base_url

    def health_check(self) -> bool:
        """Return True if Grafana responds to /api/health."""
        try:
            self._get_json("/api/health")
            return True
        except RuntimeError:
            return False

    def dashboard_search_request(self) -> str:
        """Return a redacted request line for dashboard inventory diagnostics."""
        return (
            "GET "
            f"{self._url('/api/search', {'type': 'dash-db', 'limit': '5000'})}"
        )

    def list_dashboards(self) -> list[GrafanaDashboard]:
        """Fetch all dashboards from Grafana search API."""
        data = self._get_json(
            "/api/search",
            query={"type": "dash-db", "limit": "5000"},
        )
        if not isinstance(data, list):
            raise RuntimeError(
                f"Grafana /api/search returned {type(data).__name__}, expected list. "
                f"Response: {str(data)[:200]}"
            )

        dashboards: list[GrafanaDashboard] = []
        for item in data:
            if not isinstance(item, dict):
                continue
            if item.get("type") != "dash-db":
                continue
            dashboards.append(
                GrafanaDashboard(
                    title=str(item.get("title") or ""),
                    uid=str(item.get("uid") or ""),
                    folder_title=str(item.get("folderTitle") or ""),
                )
            )
        return dashboards

    def get_dashboard_json(self, uid: str) -> dict[str, Any]:
        """Fetch the dashboard model for a single dashboard UID."""
        data = self._get_json(f"/api/dashboards/uid/{quote(uid, safe='')}")
        if not isinstance(data, dict) or not isinstance(data.get("dashboard"), dict):
            raise RuntimeError(
                f"Grafana returned an invalid dashboard payload for UID {uid!r}. "
                f"Response: {str(data)[:200]}"
            )
        return data["dashboard"]

    def list_datasources(self) -> list[GrafanaDatasource]:
        """Fetch all Grafana datasources."""
        data = self._get_json("/api/datasources")
        if not isinstance(data, list):
            raise RuntimeError(
                f"Grafana /api/datasources returned {type(data).__name__}, "
                f"expected list. Response: {str(data)[:200]}"
            )

        datasources: list[GrafanaDatasource] = []
        for item in data:
            if not isinstance(item, dict):
                continue
            datasources.append(
                GrafanaDatasource(
                    name=str(item.get("name") or ""),
                    uid=str(item.get("uid") or ""),
                    type=str(item.get("type") or ""),
                    is_default=bool(item.get("isDefault")),
                )
            )
        return datasources

    def datasource_proxy_get(
        self,
        datasource_uid: str,
        path: str,
        query: dict[str, str | list[str]] | None = None,
    ) -> Any:
        """GET a datasource API path through Grafana's datasource proxy.

        Grafana appends the sub-path to the datasource's configured URL.
        For example, if datasource URL is http://host:9091/metrics and
        path is /api/v1/query, Grafana proxies to
        http://host:9091/metrics/api/v1/query.

        Callers should use standard Prometheus paths (/api/v1/query, etc.)
        without worrying about the datasource URL prefix.
        """
        encoded_uid = quote(datasource_uid, safe="")
        if not path.startswith("/"):
            path = f"/{path}"
        return self._get_json(
            f"/api/datasources/proxy/uid/{encoded_uid}{path}",
            query=query,
        )

    def query_datasource(self, payload: dict[str, Any]) -> dict[str, Any]:
        """Execute a Grafana /api/ds/query request."""
        data = self._request_json("POST", "/api/ds/query", json=payload)
        if not isinstance(data, dict):
            raise RuntimeError(
                f"Grafana /api/ds/query returned {type(data).__name__}, "
                f"expected object. Response: {str(data)[:200]}"
            )
        return data

    def _get_json(
        self,
        path: str,
        query: dict[str, str | list[str]] | None = None,
    ) -> Any:
        return self._request_json("GET", path, query=query)

    def _request_json(
        self,
        method: str,
        path: str,
        query: dict[str, str | list[str]] | None = None,
        json: dict[str, Any] | None = None,
    ) -> Any:
        url = self._url(path, query)
        try:
            response = self._session.request(
                method,
                url,
                json=json,
                timeout=self._timeout_seconds,
            )
        except requests.exceptions.Timeout as exc:
            raise RuntimeError(
                f"Timed out after {self._timeout_seconds}s while requesting Grafana: "
                f"{method} {url}. The port-forward may be stale; try restarting it."
            ) from exc
        except requests.exceptions.ConnectionError as exc:
            raise RuntimeError(
                f"Cannot connect to Grafana at {self._base_url}. "
                f"For local runs, ensure the configured Grafana port-forward is running."
            ) from exc
        except requests.exceptions.RequestException as exc:
            raise RuntimeError(f"Grafana request failed: {method} {url}: {exc}") from exc

        if response.status_code in (401, 403):
            raise RuntimeError(
                f"Grafana rejected credentials with HTTP {response.status_code}. "
                f"Check GRAFANA_USER/GRAFANA_PASSWORD or GRAFANA_API_TOKEN. "
                f"Response: {_response_excerpt(response)}"
            )

        if not response.ok:
            raise RuntimeError(
                f"Grafana API returned HTTP {response.status_code} for {method} {url}. "
                f"Response: {_response_excerpt(response)}"
            )

        try:
            return response.json()
        except ValueError as exc:
            raise RuntimeError(
                f"Grafana returned a non-JSON response for {method} {url}. "
                f"Response: {_response_excerpt(response)}"
            ) from exc

    def _url(
        self,
        path: str,
        query: dict[str, str | list[str]] | None = None,
    ) -> str:
        request = requests.Request("GET", f"{self._base_url}{path}", params=query)
        prepared = request.prepare()
        if prepared.url is None:
            raise RuntimeError(f"Failed to build Grafana URL for path {path!r}")
        return prepared.url


def _response_excerpt(response: requests.Response, limit: int = 500) -> str:
    text = response.text.strip()
    return text[:limit] if text else "(empty)"
