"""KOF live test framework.

Provides reusable components for validating a running KOF deployment:

- config: Environment-based test configuration
- grafana: Grafana API client with clear pytest-facing failures
- kubernetes: kubectl wrapper with timeouts
- port_forward: Managed port-forward subprocess lifecycle
- prometheus: Prometheus-only Grafana dashboard probing helpers
- reference: Static reference dataset loaders
- runtime: Shared live-test runtime setup helpers
- waiting: Polling/retry utility
"""
