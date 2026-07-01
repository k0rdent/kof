from __future__ import annotations

import logging
import time
from collections.abc import Callable

logger = logging.getLogger(__name__)


class WaitTimeoutError(AssertionError):
    """Raised when wait_for exceeds its deadline.

    Inherits from AssertionError so pytest treats it as a test failure
    with a clean message rather than an unexpected exception.
    """


def wait_for(
    description: str,
    check: Callable[[], None],
    timeout_seconds: int,
    poll_interval_seconds: float,
) -> None:
    """Poll *check* until it returns without raising, or timeout expires.

    The *check* callable should raise AssertionError with a descriptive
    message if the condition is not yet met. Any other exception type
    propagates immediately (fail-fast for unexpected errors).

    Args:
        description: Human-readable description of what we're waiting for.
        check: Zero-arg callable; raises AssertionError while condition unmet.
        timeout_seconds: Maximum wall-clock seconds to wait.
        poll_interval_seconds: Seconds between retries.

    Raises:
        WaitTimeoutError: if deadline exceeded (includes last check error).
    """
    deadline = time.monotonic() + timeout_seconds
    last_error: AssertionError | None = None
    attempts = 0

    while time.monotonic() < deadline:
        attempts += 1
        try:
            check()
            if attempts > 1:
                logger.info("condition met after %d attempts: %s", attempts, description)
            return
        except AssertionError as exc:
            last_error = exc
            remaining = max(0, int(deadline - time.monotonic()))
            logger.info(
                "[%ds remaining] waiting for %s (attempt %d)",
                remaining, description, attempts,
            )
            time.sleep(poll_interval_seconds)

    detail = f"\nLast check error: {last_error}" if last_error else ""
    raise WaitTimeoutError(
        f"Timed out after {timeout_seconds}s ({attempts} attempts) "
        f"waiting for: {description}{detail}"
    )
