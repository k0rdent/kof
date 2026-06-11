from __future__ import annotations

import logging
import socket
import subprocess
import tempfile
import time
from dataclasses import dataclass, field
from typing import BinaryIO, ClassVar

logger = logging.getLogger(__name__)


class PortForwardError(RuntimeError):
    """Raised when port-forward setup fails."""


@dataclass
class PortForwardProcess:
    """Manages a kubectl port-forward subprocess lifecycle.

    Usage as context manager ensures cleanup on exit:

        with PortForwardProcess.start(...) as pf:
            # port-forward is ready
            ...
        # port-forward is terminated
    """

    process: subprocess.Popen
    local_port: int
    namespace: str
    service: str
    _stderr_file: BinaryIO | None = field(repr=False)

    STARTUP_TIMEOUT_SECONDS: ClassVar[float] = 10.0
    STARTUP_POLL_INTERVAL: ClassVar[float] = 0.3

    @classmethod
    def start(
        cls,
        kubectl: str,
        context: str | None,
        namespace: str,
        service: str,
        local_port: int,
    ) -> PortForwardProcess:
        """Start a port-forward subprocess and wait until the port is open.

        Raises:
            PortForwardError: if port-forward fails to start within timeout.
        """
        cmd = [kubectl]
        if context:
            cmd.extend(["--context", context])
        cmd.extend([
            "port-forward", "-n", namespace,
            service, f"{local_port}:{local_port}",
        ])

        logger.info("starting port-forward: %s", " ".join(cmd))

        stderr_file = tempfile.TemporaryFile()
        try:
            process = subprocess.Popen(
                cmd,
                stdout=subprocess.DEVNULL,
                stderr=stderr_file,
            )
        except FileNotFoundError as exc:
            stderr_file.close()
            raise PortForwardError(
                f"kubectl binary not found at '{kubectl}'. "
                f"Ensure kubectl is installed and on PATH."
            ) from exc
        except OSError:
            stderr_file.close()
            raise

        pf = cls(
            process=process,
            local_port=local_port,
            namespace=namespace,
            service=service,
            _stderr_file=stderr_file,
        )

        # Wait for port to become reachable
        if not pf._wait_for_port():
            stderr = pf.stop()
            raise PortForwardError(
                f"Port-forward to {service} in namespace '{namespace}' "
                f"failed to become ready within {cls.STARTUP_TIMEOUT_SECONDS}s. "
                f"Stderr: {stderr or '(empty)'}"
            )

        logger.info("port-forward ready on localhost:%d", local_port)
        return pf

    def stop(self) -> str:
        """Terminate the port-forward process and return captured stderr."""
        if self.process.poll() is None:
            self.process.terminate()
            try:
                self.process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                self.process.kill()
                self.process.wait(timeout=2)
            logger.info("port-forward terminated (pid=%d)", self.process.pid)

        stderr = ""
        if self._stderr_file is not None:
            self._stderr_file.seek(0)
            stderr = self._stderr_file.read(500).decode(errors="replace")
            self._stderr_file.close()
            self._stderr_file = None
        return stderr

    def is_alive(self) -> bool:
        """Check if the port-forward process is still running."""
        return self.process.poll() is None

    def __enter__(self) -> PortForwardProcess:
        return self

    def __exit__(self, *_: object) -> None:
        self.stop()

    def _wait_for_port(self) -> bool:
        """Poll until localhost:port accepts connections."""
        deadline = time.monotonic() + self.STARTUP_TIMEOUT_SECONDS
        while time.monotonic() < deadline:
            # Check process hasn't died
            if self.process.poll() is not None:
                return False

            try:
                with socket.create_connection(
                    ("127.0.0.1", self.local_port), timeout=1
                ):
                    return True
            except (ConnectionRefusedError, OSError):
                time.sleep(self.STARTUP_POLL_INTERVAL)

        return False
