from __future__ import annotations

import base64
import subprocess
from typing import ClassVar


class KubectlError(RuntimeError):
    """Raised when a kubectl command fails."""


class KubectlClient:
    """Minimal kubectl wrapper for live test setup.

    All commands enforce a timeout to prevent indefinite hangs when the
    cluster is unreachable or a port-forward stalls.
    """

    DEFAULT_TIMEOUT_SECONDS: ClassVar[int] = 30

    def __init__(
        self,
        command: str = "kubectl",
        context: str | None = None,
        timeout_seconds: int = DEFAULT_TIMEOUT_SECONDS,
    ) -> None:
        self._command = command
        self._context = context
        self._timeout = timeout_seconds

    def get_secret_value(self, namespace: str, name: str, key: str) -> str:
        """Decode a single key from a Kubernetes Secret.

        Raises:
            KubectlError: if the secret or key does not exist.
        """
        encoded = self.run(
            "-n", namespace,
            "get", "secret", name,
            "-o", f"jsonpath={{.data.{key}}}",
        ).strip()
        if not encoded:
            raise KubectlError(
                f"Secret '{namespace}/{name}' is missing key '{key}'. "
                f"Ensure the secret exists and contains the expected data."
            )
        return base64.b64decode(encoded).decode()

    def resource_exists(self, namespace: str, resource_type: str, name: str) -> bool:
        """Check whether a resource exists without raising on not-found."""
        try:
            self.run("-n", namespace, "get", resource_type, name, "-o", "name")
            return True
        except KubectlError:
            return False

    def run(self, *args: str) -> str:
        """Execute a kubectl command and return stdout.

        Raises:
            KubectlError: on non-zero exit code or timeout.
        """
        command = [self._command]
        if self._context:
            command.extend(["--context", self._context])
        command.extend(args)

        try:
            result = subprocess.run(
                command,
                capture_output=True,
                text=True,
                timeout=self._timeout,
            )
        except subprocess.TimeoutExpired as exc:
            raise KubectlError(
                f"kubectl command timed out after {self._timeout}s: {' '.join(command)}"
            ) from exc
        except FileNotFoundError as exc:
            raise KubectlError(
                f"kubectl binary not found at '{self._command}'. "
                f"Ensure kubectl is installed and on PATH."
            ) from exc

        if result.returncode != 0:
            stderr = result.stderr.strip()
            raise KubectlError(
                f"kubectl command failed (exit {result.returncode}): "
                f"{' '.join(command)}\n{stderr}"
            )
        return result.stdout
