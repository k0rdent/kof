#!/usr/bin/env python3
"""
Step 13 — Pod log error scan.

Scans all pod logs in the bundle for known crash / error patterns and prints
the relevant lines.  Useful for diagnosing CrashLoopBackOff root causes after
step 12 identifies the failing pod.

Usage:
    python3 step13_pod_logs.py <bundle-dir> [<namespace> ...]

Default namespaces checked: kof, kcm-system
Exits 0 if no error patterns found, 1 if any found.
"""
import sys
import os
import re

sys.path.insert(0, os.path.dirname(__file__))

DEFAULT_NAMESPACES = ["kof", "kcm-system"]

# Patterns that indicate a meaningful error in a log file.
# Each entry is (label, compiled_regex).
# Ordered from most specific / severe to most general so the first match wins.
ERROR_PATTERNS = [
    # Hard crashes / exits
    ("panic",        re.compile(r'\bpanic\b',                          re.IGNORECASE)),
    ("OOM",          re.compile(r'OOMKilled|out of memory',            re.IGNORECASE)),
    ("CrashLoop",    re.compile(r'CrashLoop',                         re.IGNORECASE)),
    # Connectivity problems (specific — avoids matching "dial" in normal info messages)
    ("dial-fail",    re.compile(r'dial tcp.*(?:connection refused|no such host|i/o timeout)', re.IGNORECASE)),
    # TLS / certificate errors
    ("tls",          re.compile(r'(?:tls|x509):.*(?:failed|invalid|unknown|expired|cannot)', re.IGNORECASE)),
    # OIDC initialisation failure (the key ACL signal)
    ("OIDC-fail",    re.compile(r'Failed to initialize OIDC|oidc.*error|openid.*error', re.IGNORECASE)),
    # Auth rejections at HTTP level
    ("auth-fail",    re.compile(r'(?:^|\s)(?:unauthorized|forbidden)\b', re.IGNORECASE)),
    # Deadline / hard timeouts (not just the word "timeout" which appears in normal helm logs)
    ("deadline",     re.compile(r'context deadline exceeded',          re.IGNORECASE)),
    # Structured-log ERROR level — matches JSON {"level":"error"} and plain "ERROR" prefix
    ("ERROR",        re.compile(r'(?:"level"\s*:\s*"error"|^\S*\s+ERROR\b|\bERROR\b\s+\S+\s+Failed)', re.IGNORECASE)),
    # Fatal
    ("FATAL",        re.compile(r'\bFATAL\b',                         re.IGNORECASE)),
]

# Log suffixes to prefer: -previous.log surfaces last crash, .log is current run.
LOG_PRIORITY = ["-previous.log", ".log"]


def scan_logs(bundle_dir, namespaces):
    logs_dir = os.path.join(bundle_dir, "logs")
    if not os.path.isdir(logs_dir):
        print("  [WARN] No logs/ directory in bundle")
        return True

    found_errors = False

    for ns in namespaces:
        ns_dir = os.path.join(logs_dir, ns)
        if not os.path.isdir(ns_dir):
            continue

        for pod_name in sorted(os.listdir(ns_dir)):
            pod_dir = os.path.join(ns_dir, pod_name)
            if not os.path.isdir(pod_dir):
                continue

            log_files = os.listdir(pod_dir)

            # Collect all log files sorted: previous first, then current
            ordered = []
            for suffix in LOG_PRIORITY:
                ordered += sorted(f for f in log_files if f.endswith(suffix))
            # Any remaining files not matched above
            ordered += sorted(f for f in log_files if f not in ordered)

            for log_file in ordered:
                log_path = os.path.join(pod_dir, log_file)
                try:
                    with open(log_path, errors="replace") as fh:
                        lines = fh.readlines()
                except OSError:
                    continue

                matched_lines = []
                for lineno, line in enumerate(lines, 1):
                    for label, pat in ERROR_PATTERNS:
                        if pat.search(line):
                            matched_lines.append((lineno, label, line.rstrip()))
                            break  # one label per line is enough

                if matched_lines:
                    found_errors = True
                    print(f"  [FAIL] {ns}/{pod_name}/{log_file} — {len(matched_lines)} error line(s)")
                    # Print up to 15 lines to keep output manageable
                    for lineno, label, text in matched_lines[:15]:
                        print(f"    L{lineno} [{label}] {text[:200]}")
                    if len(matched_lines) > 15:
                        print(f"    ... ({len(matched_lines) - 15} more lines omitted)")

    if not found_errors:
        print("  [OK  ] No error patterns found in pod logs")
    return not found_errors


def main():
    if len(sys.argv) < 2:
        print(f"Usage: {sys.argv[0]} <bundle-dir> [<namespace> ...]")
        sys.exit(2)
    bundle_dir = sys.argv[1]
    namespaces = sys.argv[2:] if len(sys.argv) > 2 else DEFAULT_NAMESPACES

    print(f"=== Step 13: Pod log error scan (namespaces: {', '.join(namespaces)}) ===")
    ok = scan_logs(bundle_dir, namespaces)
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
