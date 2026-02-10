from __future__ import annotations

import argparse
import json
import os
import re
import sys
import tarfile
import tempfile
import shutil
import datetime as dt
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Dict, List, Optional, Tuple

try:
    import yaml
except Exception:
    yaml = None


ELLIPSIS = "…"

_TMP_DIRS: list[Path] = []


def _register_tmp_dir(tmp_dir: Path) -> Path:
    _TMP_DIRS.append(tmp_dir)
    return tmp_dir


def cleanup_tmp_dirs() -> None:
    for tmp_dir in reversed(_TMP_DIRS):
        shutil.rmtree(tmp_dir, ignore_errors=True)
    _TMP_DIRS.clear()


# -----------------------------
# Output
# -----------------------------

class Output:
    """
    - console: write to stdout
    - github: write to stdout (same logs) AND write a pretty report to $GITHUB_STEP_SUMMARY
    """
    def __init__(self, mode: str = "auto") -> None:
        if mode not in {"auto", "console", "github"}:
            raise ValueError(f"invalid output mode: {mode}")
        if mode == "auto":
            self.mode = "github" if os.getenv("GITHUB_ACTIONS", "").lower() == "true" else "console"
        else:
            self.mode = mode

        self._summary_path = os.getenv("GITHUB_STEP_SUMMARY") if self.mode == "github" else None
        self._summary_md: List[str] = []

    def line(self, line_text: str = "") -> None:
        sys.stdout.write(line_text + "\n")

    def err(self, error_text: str) -> None:
        sys.stderr.write(error_text + "\n")

    def section(self, title: str) -> None:
        self.line()
        self.line(title)

        # Add nice summary formatting for GitHub (doesn't affect logs):
        if self._summary_path:
            # title already includes leading \n in caller sometimes; normalize:
            normalized_title = title.strip("\n")
            if normalized_title:
                self._summary_md.append(f"## {normalized_title}\n")

    def summary_text(self, text: str) -> None:
        if not self._summary_path:
            return
        if text is None:
            return
        self._summary_md.append(text + "\n")

    def summary_codeblock(self, text: str, lang: str = "") -> None:
        if not self._summary_path:
            return
        self._summary_md.append(f"```{lang}\n{text}\n```\n")

    def finalize(self) -> None:
        if not self._summary_path:
            return
        try:
            Path(self._summary_path).write_text("".join(self._summary_md), encoding="utf-8")
        except Exception:
            # Never fail the analyzer because summary writing failed.
            pass


def trunc(text: str, n: int) -> str:
    normalized_text = (text or "").replace("\n", " ").replace("\r", " ")
    if len(normalized_text) <= n:
        return normalized_text
    return normalized_text[: max(0, n - 1)] + ELLIPSIS


def iso_time(ts: str) -> str:
    return ts or ""


def parse_k8s_time(ts: str) -> Optional[dt.datetime]:
    """
    Parses Kubernetes timestamps like:
      2026-02-04T16:23:34Z
      2026-02-04T16:23:34.123Z
      2026-02-04T16:23:34+00:00
    Returns aware datetime or None.
    """
    if not ts:
        return None
    stripped_ts = ts.strip()
    if stripped_ts.endswith("Z"):
        stripped_ts = stripped_ts[:-1] + "+00:00"
    try:
        return dt.datetime.fromisoformat(stripped_ts)
    except Exception:
        return None


def safe_get(d: Any, path: str, default=None):
    cur = d
    for part in path.split("."):
        if cur is None:
            return default
        if isinstance(cur, dict):
            cur = cur.get(part)
        else:
            return default
    return cur if cur is not None else default


def pad_table(rows: List[List[str]], headers: List[str]) -> str:
    cols = len(headers)
    widths = [len(header) for header in headers]
    for row in rows:
        for col_idx in range(cols):
            widths[col_idx] = max(widths[col_idx], len(row[col_idx] if col_idx < len(row) else ""))
    header_line = "  ".join(header.ljust(widths[col_idx]) for col_idx, header in enumerate(headers))
    sep_line = "  ".join("-" * widths[col_idx] for col_idx in range(cols))
    out_lines = [header_line, sep_line]
    for row in rows:
        out_lines.append("  ".join((row[col_idx] if col_idx < len(row) else "").ljust(widths[col_idx]) for col_idx in range(cols)))
    return "\n".join(out_lines)


# -----------------------------
# Bundle loading
# -----------------------------

def is_tar_path(path_obj: Path) -> bool:
    return path_obj.is_file() and (path_obj.suffix in {".tar", ".tgz"} or path_obj.name.endswith(".tar.gz"))


def _is_within_dir(base: Path, target: Path) -> bool:
    try:
        target.relative_to(base)
        return True
    except ValueError:
        return False


def extract_tar_to_dir(tar_path, dst) -> None:
    dst = dst.resolve()
    dst.mkdir(parents=True, exist_ok=True)

    mode = "r:gz" if tar_path.name.endswith(".tar.gz") or tar_path.suffix == ".tgz" else "r"

    with tarfile.open(tar_path, mode) as tar_file:
        members = tar_file.getmembers()

        safe_members: list[tarfile.TarInfo] = []
        for tar_member in members:
            # 1) deny symlinks/hardlinks
            if tar_member.issym() or tar_member.islnk():
                continue

            # 2) deny absolute paths (unix + windows style)
            member_name = tar_member.name
            if member_name.startswith(("/", "\\")) or os.path.isabs(member_name):
                continue

            # 3) normalize target path and ensure it stays within dst
            target_path = (dst / member_name).resolve()

            if not _is_within_dir(dst, target_path):
                continue

            safe_members.append(tar_member)

        tar_file.extractall(dst, members=safe_members)


def list_support_bundle_like_dirs(root: Path) -> List[Path]:
    dirs = []
    if root.is_dir():
        if re.match(r"support-bundle-\d{4}-\d{2}-\d{2}T", root.name):
            return [root]
        for child_path in root.iterdir():
            if child_path.is_dir() and re.match(r"support-bundle-\d{4}-\d{2}-\d{2}T", child_path.name):
                dirs.append(child_path)
    return sorted(dirs)


def list_support_bundle_tars_in_dir(root: Path) -> List[Path]:
    tars = []
    if root.is_dir():
        for child_path in root.iterdir():
            if child_path.is_file() and re.match(r"support-bundle-.*\.tar(\.gz)?$", child_path.name):
                tars.append(child_path)
    return sorted(tars)


def load_text(path: Path) -> str:
    return path.read_text(errors="replace")


def load_json(path: Path) -> Any:
    return json.loads(load_text(path))


def load_yaml(path: Path) -> Any:
    if yaml is None:
        raise RuntimeError("pyyaml not installed; cannot parse yaml")
    return yaml.safe_load(load_text(path))


def load_k8s_list_from_file(path: Path) -> List[Dict[str, Any]]:
    obj = load_json(path) if path.suffix == ".json" else load_yaml(path)
    if obj is None:
        return []
    if isinstance(obj, dict) and "items" in obj and isinstance(obj["items"], list):
        return obj["items"]
    if isinstance(obj, list):
        return obj
    if isinstance(obj, dict) and obj.get("kind") and obj.get("metadata"):
        return [obj]
    return []


def load_k8s_list_from_dir(dir_path: Path) -> List[Dict[str, Any]]:
    """
    Loads k8s objects from a directory containing many *.json/*.yaml files.
    Each file may be:
      - { "items": [...] }
      - a single object
      - a list of objects
    """
    items: List[Dict[str, Any]] = []
    if not dir_path.exists() or not dir_path.is_dir():
        return items

    files = (
        sorted(dir_path.rglob("*.json"))
        + sorted(dir_path.rglob("*.yaml"))
        + sorted(dir_path.rglob("*.yml"))
    )

    for file_path in files:
        try:
            items.extend(load_k8s_list_from_file(file_path))
        except Exception:
            # ignore bad files; support bundles can be messy
            continue

    return items


def find_first_existing(root: Path, patterns: List[str]) -> Optional[Path]:
    for pattern in patterns:
        candidate_path = root / pattern
        if candidate_path.exists():
            return candidate_path
        matches = list(root.rglob(pattern))
        if matches:
            matches.sort(key=lambda match_path: len(str(match_path)))
            return matches[0]
    return None


# -----------------------------
# Domain models
# -----------------------------

@dataclass
class PodProblem:
    ns: str
    name: str
    phase: str
    node: str
    reason: str
    message: str
    image: str


@dataclass
class FluxObjProblem:
    ns: str
    kind: str
    name: str
    status: str
    message: str


@dataclass
class EventRow:
    type: str
    time: str
    reason: str
    ns: str
    obj: str
    message: str


@dataclass
class EventAgg:
    type: str
    time: str
    reason: str
    obj: str
    message: str
    count: int


# -----------------------------
# Events discovery + loading
# -----------------------------

def discover_event_files(bundle_dir: Path) -> List[Path]:
    """
    Support bundles vary wildly. We load events from *multiple* sources.
    Priority:
      1) canonical single file locations
      2) directories with many event files
      3) bounded rglob fallback for events*.json/yaml
    """
    candidates: List[Path] = []

    # 1) common exact paths
    exact = [
        "cluster-resources/events.json",
        "cluster-resources/events.yaml",
        "events.json",
        "events.yaml",
    ]
    for rel_path in exact:
        candidate_path = bundle_dir / rel_path
        if candidate_path.exists() and candidate_path.is_file():
            candidates.append(candidate_path)

    # 2) common dirs
    dirs = [
        bundle_dir / "cluster-resources" / "events",
        bundle_dir / "events",
    ]
    for events_dir in dirs:
        if events_dir.exists() and events_dir.is_dir():
            for candidate_path in sorted(events_dir.glob("*.json")) + sorted(events_dir.glob("*.yaml")) + sorted(events_dir.glob("*.yml")):
                if candidate_path.is_file():
                    candidates.append(candidate_path)

    # 3) single-pass fallback scan (avoid multiple full-tree rglob traversals)
    rglob_hits: List[Path] = []
    for root_dir, _, file_names in os.walk(bundle_dir):
        for file_name in file_names:
            # match events*.json / events*.yaml / events*.yml
            if not file_name.startswith("events"):
                continue
            lower_name = file_name.lower()
            if not (lower_name.endswith(".json") or lower_name.endswith(".yaml") or lower_name.endswith(".yml")):
                continue
            rglob_hits.append(Path(root_dir) / file_name)

    filtered = []
    for candidate_path in rglob_hits:
        candidate_path_str = str(candidate_path).lower()
        if "cluster-resources" in candidate_path_str or "/events" in candidate_path_str or candidate_path_str.endswith("/events.json") or candidate_path_str.endswith("/events.yaml"):
            filtered.append(candidate_path)

    filtered = sorted(set(filtered), key=lambda match_path: len(str(match_path)))
    candidates.extend(filtered[:50])  # cap hard

    out_paths: List[Path] = []
    seen = set()
    for candidate_path in candidates:
        if candidate_path in seen:
            continue
        seen.add(candidate_path)
        out_paths.append(candidate_path)
    return out_paths


def load_all_events(bundle_dir: Path) -> Tuple[List[Dict[str, Any]], List[Path], List[Tuple[Path, str]]]:
    """
    Returns: (events_items, files_used, errors)
    """
    files = discover_event_files(bundle_dir)
    items: List[Dict[str, Any]] = []
    errors: List[Tuple[Path, str]] = []

    for file_path in files:
        try:
            items.extend(load_k8s_list_from_file(file_path))
        except Exception as exc:
            errors.append((file_path, str(exc)))

    if not items:
        extra = []
        for pattern in ["event*.json", "event*.yaml", "event*.yml"]:
            extra.extend(bundle_dir.rglob(pattern))
        extra = sorted(set(extra), key=lambda match_path: len(str(match_path)))[:30]
        for file_path in extra:
            try:
                items.extend(load_k8s_list_from_file(file_path))
                if file_path not in files:
                    files.append(file_path)
            except Exception as exc:
                errors.append((file_path, str(exc)))

    return items, files, errors


# -----------------------------
# Pod problems
# -----------------------------

BAD_PHASES = {"Pending", "Failed", "Unknown"}
GOOD_PHASES = {"Running", "Succeeded"}

WAIT_REASONS = {
    "ImagePullBackOff",
    "ErrImagePull",
    "CrashLoopBackOff",
    "CreateContainerConfigError",
    "CreateContainerError",
    "RunContainerError",
    "ContainerCannotRun",
    "InvalidImageName",
    "ErrImageNeverPull",
    "OOMKilled",
}


def build_pod_start_index(pods: List[Dict[str, Any]]) -> Dict[Tuple[str, str], Optional[dt.datetime]]:
    """
    Key: (namespace, pod_name) -> pod.status.startTime as datetime (or None)
    """
    idx: Dict[Tuple[str, str], Optional[dt.datetime]] = {}
    for pod_obj in pods:
        ns = safe_get(pod_obj, "metadata.namespace", "") or ""
        name = safe_get(pod_obj, "metadata.name", "") or ""
        start_time = safe_get(pod_obj, "status.startTime", "") or ""
        if ns and name:
            idx[(ns, name)] = parse_k8s_time(start_time)
    return idx


def extract_pod_issue(pod: Dict[str, Any]) -> Optional[PodProblem]:
    ns = safe_get(pod, "metadata.namespace", "") or ""
    name = safe_get(pod, "metadata.name", "") or ""
    phase = safe_get(pod, "status.phase", "") or ""
    node = safe_get(pod, "spec.nodeName", "") or ""

    best_reason = ""
    best_msg = ""
    best_img = ""

    def consider(reason: str, msg: str, img: str):
        nonlocal best_reason, best_msg, best_img
        if not reason:
            return
        score = 2 if reason in WAIT_REASONS else 1
        best_score = 2 if best_reason in WAIT_REASONS else (1 if best_reason else 0)
        if score > best_score:
            best_reason, best_msg, best_img = reason, msg, img
        elif score == best_score and len(msg or "") > len(best_msg or ""):
            best_reason, best_msg, best_img = reason, msg, img

    for container_status in safe_get(pod, "status.containerStatuses", []) or []:
        img = container_status.get("image", "") or ""
        state = container_status.get("state") or {}
        if "waiting" in state:
            waiting_state = state["waiting"] or {}
            consider(waiting_state.get("reason", "") or "", waiting_state.get("message", "") or "", img)
        if "terminated" in state:
            terminated_state = state["terminated"] or {}
            consider(terminated_state.get("reason", "") or "", terminated_state.get("message", "") or "", img)

    consider(safe_get(pod, "status.reason", "") or "", safe_get(pod, "status.message", "") or "", "")

    is_bad_phase = phase in BAD_PHASES
    is_bad_reason = best_reason in WAIT_REASONS
    if not (is_bad_phase or is_bad_reason):
        return None

    reason = best_reason or (phase if is_bad_phase else "Unknown")
    msg = best_msg or ""
    return PodProblem(ns=ns, name=name, phase=phase, node=node, reason=reason, message=msg, image=best_img)


def summarize_pod_problems(problems: List[PodProblem]) -> Dict[str, int]:
    out: Dict[str, int] = {}
    for pod_problem in problems:
        out[pod_problem.reason] = out.get(pod_problem.reason, 0) + 1
    return dict(sorted(out.items(), key=lambda kv: (-kv[1], kv[0])))


def group_by_image(problems: List[PodProblem], reason: str) -> List[Tuple[str, int]]:
    counts: Dict[str, int] = {}
    for pod_problem in problems:
        if pod_problem.reason != reason:
            continue
        img = pod_problem.image or "(image unknown)"
        counts[img] = counts.get(img, 0) + 1
    return sorted(counts.items(), key=lambda kv: (-kv[1], kv[0]))


# -----------------------------
# Flux / Helm (HelmRelease)
# -----------------------------

def extract_flux_helm_problems(helmreleases: List[Dict[str, Any]]) -> List[FluxObjProblem]:
    out: List[FluxObjProblem] = []
    for hr in helmreleases:
        ns = safe_get(hr, "metadata.namespace", "") or ""
        name = safe_get(hr, "metadata.name", "") or ""
        kind = safe_get(hr, "kind", "HelmRelease") or "HelmRelease"

        conds = safe_get(hr, "status.conditions", []) or []
        for cond in conds:
            ctype = cond.get("type", "")
            status = cond.get("status", "")
            if ctype in {"Ready", "Released"} and status in {"False", "Unknown"}:
                msg = cond.get("message", "") or ""
                out.append(FluxObjProblem(ns=ns, kind=kind, name=name, status=f"{ctype}:{status}", message=msg))
    return out


# -----------------------------
# Events parsing + cleaning
# -----------------------------

EVENT_REASON_SCORE = {
    # Hard failures first
    "Failed": 110,
    "InstallFailed": 105,
    "UpgradeFailed": 100,

    # Scheduling / infra blockers
    "FailedMount": 95,
    "FailedScheduling": 90,
    "Unhealthy": 85,
    "FailedCreatePodSandBox": 80,

    # Pull/backoff
    "ImagePullBackOff": 75,
    "ErrImagePull": 75,
    "FailedPull": 70,
    "BackOff": 65,

    # Lower signal
    "ExternalProvisioning": 50,
    "ProvisioningFailed": 50,
    "WaitForFirstConsumer": 45,
    "DependencyNotReady": 40,
    "BadConfig": 10,
}

WEBHOOK_URL_RE = re.compile(r'https://([a-z0-9-]+)\.([a-z0-9-]+)\.svc(?::\d+)?', re.IGNORECASE)
SECRET_NOT_FOUND_RE = re.compile(r'secret\s+"([^"]+)"\s+not found', re.IGNORECASE)


def normalize_event_obj(ev: Dict[str, Any]) -> str:
    involved = ev.get("involvedObject") or {}
    kind = involved.get("kind", "") or ""
    name = involved.get("name", "") or ""
    return f"{kind}/{name}" if kind and name else ""


def extract_events(events: List[Dict[str, Any]]) -> List[EventRow]:
    out: List[EventRow] = []
    for ev in events:
        etype = ev.get("type", "") or ""
        reason = ev.get("reason", "") or ""
        msg = ev.get("message", "") or ""
        event_time = ev.get("lastTimestamp") or ev.get("eventTime") or ev.get("firstTimestamp") or ""
        involved = ev.get("involvedObject") or {}
        ns = involved.get("namespace", "") or ""
        obj = normalize_event_obj(ev)
        out.append(EventRow(type=etype, time=event_time, reason=reason, ns=ns, obj=obj, message=msg))
    return out


def filter_events(rows: List[EventRow], include_normal: bool = False) -> List[EventRow]:
    out: List[EventRow] = []
    for event_row in rows:
        if event_row.type == "Warning":
            out.append(event_row)
            continue

        if include_normal and event_row.type == "Normal":
            if event_row.reason in {"UpgradeFailed", "InstallFailed", "UpgradeSucceeded"}:
                out.append(event_row)
    return out


def agg_events(rows: List[EventRow]) -> List[EventAgg]:
    def norm_msg(m: str) -> str:
        m = (m or "").strip()
        m = re.sub(r"\s+", " ", m)
        return m

    buckets: Dict[Tuple[str, str, str, str], EventAgg] = {}
    for event_row in rows:
        key = (event_row.type, event_row.reason, event_row.obj, norm_msg(event_row.message))
        if key not in buckets:
            buckets[key] = EventAgg(type=event_row.type, time=event_row.time, reason=event_row.reason, obj=event_row.obj, message=event_row.message, count=1)
        else:
            buckets[key].count += 1
            if buckets[key].time and event_row.time and event_row.time < buckets[key].time:
                buckets[key].time = event_row.time
    return list(buckets.values())


def event_score(e: EventAgg) -> int:
    base = EVENT_REASON_SCORE.get(e.reason, 20 if e.type == "Warning" else 5)
    if e.type == "Warning":
        base += 5
    msg = (e.message or "").strip()
    if msg.startswith("Error:"):
        base -= 10
    m = msg.lower()
    if "connect: connection refused" in m or "connection reset" in m:
        base += 10
    if "401 unauthorized" in m or "unauthorized" in m:
        base += 10
    return base


def suppress_badconfig(events: List[EventAgg]) -> Tuple[List[EventAgg], int]:
    severe_exists = any(event_score(event_agg) >= 40 and not (event_agg.reason == "BadConfig" and event_agg.obj.startswith("CertificateRequest/")) for event_agg in events)
    if not severe_exists:
        return events, 0
    kept = []
    suppressed = 0
    for event_agg in events:
        if event_agg.reason == "BadConfig" and event_agg.obj.startswith("CertificateRequest/"):
            suppressed += 1
            continue
        kept.append(event_agg)
    return kept, suppressed


def collapse_one_best_per_object(events: List[EventAgg], max_events: int) -> List[EventAgg]:
    best_by_obj: Dict[str, EventAgg] = {}
    for event_agg in events:
        obj = event_agg.obj or "(unknown)"
        if obj not in best_by_obj:
            best_by_obj[obj] = event_agg
            continue
        if event_score(event_agg) > event_score(best_by_obj[obj]):
            best_by_obj[obj] = event_agg
        elif event_score(event_agg) == event_score(best_by_obj[obj]):
            if event_agg.type == "Warning" and best_by_obj[obj].type != "Warning":
                best_by_obj[obj] = event_agg
            elif len(event_agg.message or "") > len(best_by_obj[obj].message or ""):
                best_by_obj[obj] = event_agg

    chosen = list(best_by_obj.values())
    chosen.sort(key=lambda event_agg: (-event_score(event_agg), event_agg.time or ""))
    return chosen[:max_events]


def choose_best_rootcause_event(events: List[EventAgg], preferred_objects: List[str]) -> Optional[EventAgg]:
    if not events:
        return None
    preferred_set = set(preferred_objects)

    def pref_rank(event_agg: EventAgg) -> int:
        return 0 if event_agg.obj in preferred_set else 1

    events_sorted = sorted(events, key=lambda event_agg: (pref_rank(event_agg), -event_score(event_agg), event_agg.time or ""))
    return events_sorted[0] if events_sorted else None


def build_helm_webhook_chain(top_flux: FluxObjProblem, root_event: Optional[EventAgg], all_events: List[EventAgg]) -> List[str]:
    chain: List[str] = []
    msg = (top_flux.message or "")
    if "failed calling webhook" not in msg.lower():
        return chain

    chain.append(f"HelmRelease {top_flux.ns}/{top_flux.name}: {top_flux.status} — {trunc(top_flux.message, 180)}")

    match = WEBHOOK_URL_RE.search(msg)
    if match:
        svc, svc_ns = match.group(1), match.group(2)
        chain.append(f"Webhook endpoint: Service {svc_ns}/{svc} (from URL)")

    def obj_is_related(object_ref: str) -> bool:
        if not object_ref.startswith("Pod/"):
            return False
        n = object_ref.split("/", 1)[1].lower()
        return ("cluster-api-operator" in n) or ("capi-operator" in n) or ("webhook" in n)

    related = [event_agg for event_agg in all_events if event_agg.obj and obj_is_related(event_agg.obj)]
    related.sort(key=lambda event_agg: (-event_score(event_agg), event_agg.time or ""))

    pod_health = next((event_agg for event_agg in related if event_agg.reason in {"Unhealthy", "BackOff"}), None)
    if pod_health:
        chain.append(f"{pod_health.obj}: {pod_health.reason} — {trunc(pod_health.message, 160)}")

    pod_mount = next((event_agg for event_agg in related if event_agg.reason == "FailedMount" and "secret" in (event_agg.message or "").lower()), None)
    if pod_mount:
        sec = ""
        secret_match = SECRET_NOT_FOUND_RE.search(pod_mount.message or "")
        if secret_match:
            sec = secret_match.group(1)
        chain.append(f'{pod_mount.obj}: FailedMount — missing Secret "{sec}"' if sec else f"{pod_mount.obj}: FailedMount — {trunc(pod_mount.message, 160)}")

    if len(chain) == 2 and root_event:
        chain.append(f"{root_event.obj}: {root_event.reason} — {trunc(root_event.message, 160)}")

    return chain


# -----------------------------
# Report
# -----------------------------

def detect_focus_namespaces(pod_problems: List[PodProblem], flux_problems: List[FluxObjProblem]) -> List[str]:
    counts: Dict[str, int] = {}
    for pod_problem in pod_problems:
        counts[pod_problem.ns] = counts.get(pod_problem.ns, 0) + 3
    for flux_problem in flux_problems:
        counts[flux_problem.ns] = counts.get(flux_problem.ns, 0) + 2
    return [ns for ns, _ in sorted(counts.items(), key=lambda kv: (-kv[1], kv[0]))[:3]]


def filter_stale_pod_events_by_start_time(
    rows: List[EventRow],
    pod_start_idx: Dict[Tuple[str, str], Optional[dt.datetime]],
) -> List[EventRow]:
    """
    Rule:
    - For Pod/<name> events: if pod exists in snapshot and event_time < pod.status.startTime => drop.
    """
    out: List[EventRow] = []
    for event_row in rows:
        # only apply to Pod events
        if not event_row.obj.startswith("Pod/"):
            out.append(event_row)
            continue

        pod_name = event_row.obj.split("/", 1)[1]
        pod_start = pod_start_idx.get((event_row.ns, pod_name))

        # if pod not found or startTime unknown -> keep (safe)
        if pod_start is None:
            out.append(event_row)
            continue

        event_time = parse_k8s_time(event_row.time)
        # if event time missing/unparseable -> keep (safe)
        if event_time is None:
            out.append(event_row)
            continue

        if event_time < pod_start:
            # stale event for a previous pod incarnation -> drop
            continue

        out.append(event_row)

    return out


def analyze_bundle(bundle_dir: Path, details: bool, out: Output) -> None:
    nodes_file = find_first_existing(
        bundle_dir,
        [
            "cluster-resources/nodes.json",
            "cluster-resources/nodes.yaml",
            "nodes.json",
            "nodes.yaml",
            "*nodes*.json",
            "*nodes*.yaml",
        ],
    )

    pods: List[Dict[str, Any]] = []
    pods_dir = bundle_dir / "cluster-resources" / "pods"
    if pods_dir.exists() and pods_dir.is_dir():
        pods = load_k8s_list_from_dir(pods_dir)

    nodes: List[Dict[str, Any]] = load_k8s_list_from_file(nodes_file) if nodes_file else []

    # HelmRelease CRs
    helmreleases: List[Dict[str, Any]] = []
    hr_root = bundle_dir / "cluster-resources" / "custom-resources" / "helmreleases.helm.toolkit.fluxcd.io"
    if hr_root.exists() and hr_root.is_dir():
        for file_path in sorted(hr_root.rglob("*.json")) + sorted(hr_root.rglob("*.yaml")) + sorted(hr_root.rglob("*.yml")):
            try:
                helmreleases.extend(load_k8s_list_from_file(file_path))
            except Exception:
                continue

    pod_problems = [pod_problem for pod_problem in (extract_pod_issue(pod_obj) for pod_obj in pods) if pod_problem is not None]  # type: ignore
    flux_problems = extract_flux_helm_problems(helmreleases)
    focus_namespaces = detect_focus_namespaces(pod_problems, flux_problems)

    # Events
    events_items, events_files_used, events_errors = load_all_events(bundle_dir)
    ev_rows = extract_events(events_items)
    ev_rows = filter_events(ev_rows)

    pod_start_idx = build_pod_start_index(pods)
    ev_rows = filter_stale_pod_events_by_start_time(ev_rows, pod_start_idx)

    ev_ag = agg_events(ev_rows)
    ev_ag, suppressed_badconfig = suppress_badconfig(ev_ag)
    max_events = 25 if details else 15
    ev_collapsed = collapse_one_best_per_object(ev_ag, max_events=max_events)
    ev_collapsed.sort(
        key=lambda event_agg: parse_k8s_time(event_agg.time) or dt.datetime.min.replace(tzinfo=dt.timezone.utc),
        reverse=True
    )
    # --- QUICK DIAGNOSIS (same text output) ---
    out.line("=== QUICK DIAGNOSIS ===\n")
    out.summary_text("### QUICK DIAGNOSIS")

    causal_chain: List[str] = []

    if pod_problems:
        summary = summarize_pod_problems(pod_problems)
        top_reason, top_count = next(iter(summary.items()))
        affected_ns: Dict[str, int] = {}
        for pod_problem in pod_problems:
            affected_ns[pod_problem.ns] = affected_ns.get(pod_problem.ns, 0) + 1
        top_ns = sorted(affected_ns.items(), key=lambda kv: (-kv[1], kv[0]))[:3]
        samples = ", ".join([f"{pod_problem.ns}/{pod_problem.name}" for pod_problem in pod_problems[:3]])

        out.line(f"Top pod issue: {top_reason} — {top_count} pod(s)")
        if top_ns:
            out.line("Affected namespaces (top): " + ", ".join(f"{ns}({c})" for ns, c in top_ns))
        out.line(f"Sample pods: {samples}")

        best = sorted(pod_problems, key=lambda pod_problem: (0 if pod_problem.reason in WAIT_REASONS else 1, -len(pod_problem.message or "")))[0]
        if best.image:
            out.line(f"Example image: {best.image}")
        if best.message:
            out.line("Example message (full-ish):")
            out.line(trunc(best.message, 520 if details else 360))

        preferred_objects = [f"Pod/{best.name}"]
        root_event = choose_best_rootcause_event(ev_ag, preferred_objects)

    elif flux_problems:
        def flux_rank(fp: FluxObjProblem) -> Tuple[int, int]:
            if fp.status.startswith("Ready:False"):
                pri = 0
            elif fp.status.startswith("Ready:Unknown"):
                pri = 1
            elif fp.status.startswith("Released:False"):
                pri = 2
            else:
                pri = 3
            return (pri, -len(fp.message or ""))

        top_flux = sorted(flux_problems, key=flux_rank)[0]
        out.line("Top issue: Flux/Helm object not ready")
        out.line(f"Example: {top_flux.kind} {top_flux.ns}/{top_flux.name} => {top_flux.status}")
        if top_flux.message:
            out.line("Message:")
            out.line(trunc(top_flux.message, 620 if details else 420))

        preferred_objects = [f"{top_flux.kind}/{top_flux.name}"]
        root_event = choose_best_rootcause_event(ev_ag, preferred_objects)
        causal_chain = build_helm_webhook_chain(top_flux, root_event, ev_ag)

    else:
        best_any = sorted(ev_collapsed, key=lambda event_agg: (-event_score(event_agg), event_agg.time or ""))[0] if ev_collapsed else None
        if best_any:
            msg_l = (best_any.message or "").lower()
            if best_any.reason == "FailedScheduling" and "not-ready" in msg_l and "untolerated taint" in msg_l:
                out.line("Top issue: Scheduling blocked by NotReady taint / node not ready")
            else:
                out.line("Top issue: Warning event detected")

            out.line(f"Example: {best_any.type} {iso_time(best_any.time)} {best_any.reason} {best_any.obj}")
            out.line("Message:")
            out.line(trunc(best_any.message, 520 if details else 360))
            preferred_objects = [best_any.obj]
            root_event = choose_best_rootcause_event(ev_ag, preferred_objects)
        else:
            out.line("No obvious root-cause detected by current heuristics.")
            root_event = None

    # --- Pods (!= Running/Succeeded) ---
    out.section("\nPods (!= Running/Succeeded):")
    if not pods:
        out.line("(no pods data found in bundle)")
        out.summary_codeblock("(no pods data found in bundle)")
    else:
        non_ok = []
        for pod_obj in pods:
            ns = safe_get(pod_obj, "metadata.namespace", "") or ""
            name = safe_get(pod_obj, "metadata.name", "") or ""
            phase = safe_get(pod_obj, "status.phase", "") or ""
            if phase not in GOOD_PHASES:
                non_ok.append([ns, phase, name])
        text = pad_table(non_ok, ["namespace", "phase", "name"]) if non_ok else "All pods appear Running/Succeeded."
        out.line(text)
        out.summary_codeblock(text)

    # --- Problem pods summary ---
    out.section("\nProblem pods summary (by reason):")
    if pod_problems:
        summ = summarize_pod_problems(pod_problems)
        rows = [[k, str(v)] for k, v in summ.items()]
        text = pad_table(rows, ["reason", "count"])
        out.line(text)
        out.summary_codeblock(text)

        top_reason = next(iter(summ.keys()))
        if top_reason in {"ImagePullBackOff", "ErrImagePull"}:
            grouped = group_by_image(pod_problems, reason=top_reason)
            out.section(f"\nTop reason '{top_reason}' grouped by image (top 10):")
            rows2 = [[img, str(c)] for img, c in grouped[:10]]
            text2 = pad_table(rows2, ["image", "count"])
            out.line(text2)
            out.summary_codeblock(text2)
    else:
        out.line("Problem pods: none detected by current heuristics.")
        out.summary_codeblock("Problem pods: none detected by current heuristics.")

    # --- Nodes ---
    out.section("\nNodes:")
    if nodes:
        rows = []
        for node_obj in nodes:
            name = safe_get(node_obj, "metadata.name", "") or ""
            conds = safe_get(node_obj, "status.conditions", []) or []
            ready = ""
            for cond in conds:
                if cond.get("type") == "Ready":
                    ready = "True" if cond.get("status") == "True" else (cond.get("status") or "")
            taints = safe_get(node_obj, "spec.taints", []) or []
            taint_str = ",".join(f"{taint.get('key')}:{taint.get('effect')}" for taint in taints if isinstance(taint, dict))
            alloc_cpu = safe_get(node_obj, "status.allocatable.cpu", "") or ""
            alloc_mem = safe_get(node_obj, "status.allocatable.memory", "") or ""
            rows.append([name, ready, taint_str, alloc_cpu, alloc_mem, ""])
        text = pad_table(rows, ["name", "Ready", "taints", "alloc(cpu)", "alloc(mem)", "pressures"])
        out.line(text)
        out.summary_codeblock(text)
    else:
        out.line("(no nodes data found)")
        out.summary_codeblock("(no nodes data found)")

    # --- Flux/Helm not ready ---
    if flux_problems:
        out.section("\nFlux/Helm objects not ready (focus namespaces):")
        rows = []
        for fp in flux_problems:
            if focus_namespaces and fp.ns not in focus_namespaces:
                continue
            rows.append([fp.ns, fp.kind, fp.name, fp.status, trunc(fp.message, 360 if details else 220)])
        if rows:
            text = pad_table(rows, ["ns", "kind", "name", "status", "message"])
            out.line(text)
            out.summary_codeblock(text)

    # --- Events ---
    out.section(f"\nEvents (focus namespaces) — filtered + dedup (top {25 if details else 15}):")
    if not events_files_used and not events_items:
        out.line("(events source not found in bundle)")
        out.summary_codeblock("(events source not found in bundle)")
    else:
        if not ev_collapsed:
            out.line("(no relevant events after filtering)")
            out.summary_codeblock("(no relevant events after filtering)")
        else:
            rows = []
            max_msg_len = 620 if details else 240
            for event_agg in ev_collapsed:
                rows.append([event_agg.type, iso_time(event_agg.time), event_agg.reason, event_agg.obj, f"x{event_agg.count}", trunc(event_agg.message, max_msg_len)])
            text = pad_table(rows, ["type", "time", "reason", "object", "count", "message"])
            out.line(text)
            out.summary_codeblock(text)

    if suppressed_badconfig:
        out.line(f"\nNote: suppressed {suppressed_badconfig} CertificateRequest BadConfig warning(s) (low signal).")
        out.summary_text(f"**Note:** suppressed {suppressed_badconfig} CertificateRequest BadConfig warning(s) (low signal).")

    if root_event:
        out.section("\nBest root-cause candidate event (full message):")
        out.line(f"{root_event.type} time={iso_time(root_event.time)} reason={root_event.reason} object={root_event.obj}")
        out.line(trunc(root_event.message, 2000))
        out.summary_codeblock(
            f"{root_event.type} time={iso_time(root_event.time)} reason={root_event.reason} object={root_event.obj}\n"
            f"{trunc(root_event.message, 2000)}"
        )

    if causal_chain:
        out.section("\nCausal chain (Helm/webhook):")
        lines = []
        for step_index, step in enumerate(causal_chain, 1):
            step_line = f"{step_index}. {step}"
            out.line(step_line)
            lines.append(step_line)
        out.summary_codeblock("\n".join(lines))


# -----------------------------
# Entrypoint / multi-bundle support
# -----------------------------

def resolve_bundles(input_path: Path) -> List[Tuple[str, Path]]:
    bundles: List[Tuple[str, Path]] = []

    if input_path.is_dir():
        dirs = list_support_bundle_like_dirs(input_path)
        if dirs:
            for bundle_dir in dirs:
                bundles.append((bundle_dir.name, bundle_dir))
            return bundles
        bundles.append((input_path.name, input_path))
        return bundles

    if is_tar_path(input_path):
        tmp = _register_tmp_dir(Path(tempfile.mkdtemp(prefix="support-bundle-analyzer-")))
        extract_tar_to_dir(input_path, tmp)

        dirs = list_support_bundle_like_dirs(tmp)
        if dirs:
            for bundle_dir in dirs:
                bundles.append((bundle_dir.name, bundle_dir))
            return bundles

        inner_tars = list_support_bundle_tars_in_dir(tmp)
        if inner_tars:
            for inner_tar in inner_tars:
                inner_tmp = _register_tmp_dir(Path(tempfile.mkdtemp(prefix="support-bundle-inner-")))
                extract_tar_to_dir(inner_tar, inner_tmp)
                inner_dirs = list_support_bundle_like_dirs(inner_tmp)
                if inner_dirs:
                    bundles.append((inner_tar.stem, inner_dirs[0]))
                else:
                    bundles.append((inner_tar.stem, inner_tmp))
            return bundles

        bundles.append((input_path.stem, tmp))
        return bundles

    bundles.append((input_path.name, input_path))
    return bundles


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("path", help="support-bundle directory OR .tar/.tar.gz (also supports outer tar containing multiple bundles)")
    ap.add_argument("--details", action="store_true", help="print deep-dive sections")
    ap.add_argument("--output", default="auto", choices=["auto", "console", "github"], help="output mode (default: auto)")
    args = ap.parse_args()

    out = Output(mode=args.output)

    input_path = Path(args.path).expanduser()
    if not input_path.exists():
        out.err(f"ERROR: path not found: {input_path}")
        out.finalize()
        return 2

    bundles = resolve_bundles(input_path)
    if not bundles:
        out.err("ERROR: no bundles detected")
        out.finalize()
        return 2

    try:
        for idx, (label, bdir) in enumerate(bundles, 1):
            if len(bundles) > 1:
                sep = f"\n\n==================== BUNDLE {idx}/{len(bundles)}: {label} ====================\n"
                out.line(sep)
                out.summary_text(f"# BUNDLE {idx}/{len(bundles)}: {label}\n")
            analyze_bundle(bdir, details=args.details, out=out)
    finally:
        cleanup_tmp_dirs()

    out.finalize()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
