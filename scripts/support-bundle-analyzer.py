from __future__ import annotations

import argparse
import json
import os
import re
import sys
import tarfile
import tempfile
import datetime as dt
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Dict, List, Optional, Tuple

try:
    import yaml
except Exception:
    yaml = None


ELLIPSIS = "…"


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

    def line(self, s: str = "") -> None:
        sys.stdout.write(s + "\n")

    def err(self, s: str) -> None:
        sys.stderr.write(s + "\n")

    def section(self, title: str) -> None:
        self.line()
        self.line(title)

        # Add nice summary formatting for GitHub (doesn't affect logs):
        if self._summary_path:
            # title already includes leading \n in caller sometimes; normalize:
            t = title.strip("\n")
            if t:
                self._summary_md.append(f"## {t}\n")

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


def trunc(s: str, n: int) -> str:
    s = (s or "").replace("\n", " ").replace("\r", " ")
    if len(s) <= n:
        return s
    return s[: max(0, n - 1)] + ELLIPSIS


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
    s = ts.strip()
    if s.endswith("Z"):
        s = s[:-1] + "+00:00"
    try:
        return dt.datetime.fromisoformat(s)
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
    widths = [len(h) for h in headers]
    for r in rows:
        for i in range(cols):
            widths[i] = max(widths[i], len(r[i] if i < len(r) else ""))
    line = "  ".join(h.ljust(widths[i]) for i, h in enumerate(headers))
    sep = "  ".join("-" * widths[i] for i in range(cols))
    out = [line, sep]
    for r in rows:
        out.append("  ".join((r[i] if i < len(r) else "").ljust(widths[i]) for i in range(cols)))
    return "\n".join(out)


# -----------------------------
# Bundle loading
# -----------------------------

def is_tar_path(p: Path) -> bool:
    return p.is_file() and (p.suffix in {".tar", ".tgz"} or p.name.endswith(".tar.gz"))


def extract_tar_to_dir(tar_path: Path, dst: Path) -> None:
    mode = "r:gz" if tar_path.name.endswith(".tar.gz") or tar_path.suffix == ".tgz" else "r"
    with tarfile.open(tar_path, mode) as tf:
        tf.extractall(dst)


def list_support_bundle_like_dirs(root: Path) -> List[Path]:
    dirs = []
    if root.is_dir():
        if re.match(r"support-bundle-\d{4}-\d{2}-\d{2}T", root.name):
            return [root]
        for p in root.iterdir():
            if p.is_dir() and re.match(r"support-bundle-\d{4}-\d{2}-\d{2}T", p.name):
                dirs.append(p)
    return sorted(dirs)


def list_support_bundle_tars_in_dir(root: Path) -> List[Path]:
    tars = []
    if root.is_dir():
        for p in root.iterdir():
            if p.is_file() and re.match(r"support-bundle-.*\.tar(\.gz)?$", p.name):
                tars.append(p)
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

    for f in files:
        try:
            items.extend(load_k8s_list_from_file(f))
        except Exception:
            # ignore bad files; support bundles can be messy
            continue

    return items


def find_first_existing(root: Path, patterns: List[str]) -> Optional[Path]:
    for pat in patterns:
        p = root / pat
        if p.exists():
            return p
        matches = list(root.rglob(pat))
        if matches:
            matches.sort(key=lambda x: len(str(x)))
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
    for rel in exact:
        p = bundle_dir / rel
        if p.exists() and p.is_file():
            candidates.append(p)

    # 2) common dirs
    dirs = [
        bundle_dir / "cluster-resources" / "events",
        bundle_dir / "events",
    ]
    for d in dirs:
        if d.exists() and d.is_dir():
            for p in sorted(d.glob("*.json")) + sorted(d.glob("*.yaml")) + sorted(d.glob("*.yml")):
                if p.is_file():
                    candidates.append(p)

    # 3) bounded rglob fallback (avoid scanning huge trees forever)
    rglob_hits: List[Path] = []
    for pat in ["events*.json", "events*.yaml", "events*.yml"]:
        rglob_hits.extend(bundle_dir.rglob(pat))

    filtered = []
    for p in rglob_hits:
        sp = str(p).lower()
        if "cluster-resources" in sp or "/events" in sp or sp.endswith("/events.json") or sp.endswith("/events.yaml"):
            filtered.append(p)

    filtered = sorted(set(filtered), key=lambda x: len(str(x)))
    candidates.extend(filtered[:50])  # cap hard

    out: List[Path] = []
    seen = set()
    for p in candidates:
        if p in seen:
            continue
        seen.add(p)
        out.append(p)
    return out


def load_all_events(bundle_dir: Path) -> Tuple[List[Dict[str, Any]], List[Path], List[Tuple[Path, str]]]:
    """
    Returns: (events_items, files_used, errors)
    """
    files = discover_event_files(bundle_dir)
    items: List[Dict[str, Any]] = []
    errors: List[Tuple[Path, str]] = []

    for f in files:
        try:
            items.extend(load_k8s_list_from_file(f))
        except Exception as e:
            errors.append((f, str(e)))

    if not items:
        extra = []
        for pat in ["event*.json", "event*.yaml", "event*.yml"]:
            extra.extend(bundle_dir.rglob(pat))
        extra = sorted(set(extra), key=lambda x: len(str(x)))[:30]
        for f in extra:
            try:
                items.extend(load_k8s_list_from_file(f))
                if f not in files:
                    files.append(f)
            except Exception as e:
                errors.append((f, str(e)))

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
    for p in pods:
        ns = safe_get(p, "metadata.namespace", "") or ""
        name = safe_get(p, "metadata.name", "") or ""
        st = safe_get(p, "status.startTime", "") or ""
        if ns and name:
            idx[(ns, name)] = parse_k8s_time(st)
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

    for cs in safe_get(pod, "status.containerStatuses", []) or []:
        img = cs.get("image", "") or ""
        st = cs.get("state") or {}
        if "waiting" in st:
            w = st["waiting"] or {}
            consider(w.get("reason", "") or "", w.get("message", "") or "", img)
        if "terminated" in st:
            t = st["terminated"] or {}
            consider(t.get("reason", "") or "", t.get("message", "") or "", img)

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
    for p in problems:
        out[p.reason] = out.get(p.reason, 0) + 1
    return dict(sorted(out.items(), key=lambda kv: (-kv[1], kv[0])))


def group_by_image(problems: List[PodProblem], reason: str) -> List[Tuple[str, int]]:
    counts: Dict[str, int] = {}
    for p in problems:
        if p.reason != reason:
            continue
        img = p.image or "(image unknown)"
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
        for c in conds:
            ctype = c.get("type", "")
            status = c.get("status", "")
            if ctype in {"Ready", "Released"} and status in {"False", "Unknown"}:
                msg = c.get("message", "") or ""
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
        t = ev.get("lastTimestamp") or ev.get("eventTime") or ev.get("firstTimestamp") or ""
        involved = ev.get("involvedObject") or {}
        ns = involved.get("namespace", "") or ""
        obj = normalize_event_obj(ev)
        out.append(EventRow(type=etype, time=t, reason=reason, ns=ns, obj=obj, message=msg))
    return out


def filter_events(rows: List[EventRow], include_normal: bool = False) -> List[EventRow]:
    out: List[EventRow] = []
    for r in rows:
        if r.type == "Warning":
            out.append(r)
            continue

        if include_normal and r.type == "Normal":
            if r.reason in {"UpgradeFailed", "InstallFailed", "UpgradeSucceeded"}:
                out.append(r)
    return out


def agg_events(rows: List[EventRow]) -> List[EventAgg]:
    def norm_msg(m: str) -> str:
        m = (m or "").strip()
        m = re.sub(r"\s+", " ", m)
        return m

    buckets: Dict[Tuple[str, str, str, str], EventAgg] = {}
    for r in rows:
        key = (r.type, r.reason, r.obj, norm_msg(r.message))
        if key not in buckets:
            buckets[key] = EventAgg(type=r.type, time=r.time, reason=r.reason, obj=r.obj, message=r.message, count=1)
        else:
            buckets[key].count += 1
            if buckets[key].time and r.time and r.time < buckets[key].time:
                buckets[key].time = r.time
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
    severe_exists = any(event_score(e) >= 40 and not (e.reason == "BadConfig" and e.obj.startswith("CertificateRequest/")) for e in events)
    if not severe_exists:
        return events, 0
    kept = []
    suppressed = 0
    for e in events:
        if e.reason == "BadConfig" and e.obj.startswith("CertificateRequest/"):
            suppressed += 1
            continue
        kept.append(e)
    return kept, suppressed


def collapse_one_best_per_object(events: List[EventAgg], max_events: int) -> List[EventAgg]:
    best_by_obj: Dict[str, EventAgg] = {}
    for e in events:
        obj = e.obj or "(unknown)"
        if obj not in best_by_obj:
            best_by_obj[obj] = e
            continue
        if event_score(e) > event_score(best_by_obj[obj]):
            best_by_obj[obj] = e
        elif event_score(e) == event_score(best_by_obj[obj]):
            if e.type == "Warning" and best_by_obj[obj].type != "Warning":
                best_by_obj[obj] = e
            elif len(e.message or "") > len(best_by_obj[obj].message or ""):
                best_by_obj[obj] = e

    chosen = list(best_by_obj.values())
    chosen.sort(key=lambda e: (-event_score(e), e.time or ""))
    return chosen[:max_events]


def choose_best_rootcause_event(events: List[EventAgg], preferred_objects: List[str]) -> Optional[EventAgg]:
    if not events:
        return None
    preferred_set = set(preferred_objects)

    def pref_rank(e: EventAgg) -> int:
        return 0 if e.obj in preferred_set else 1

    events_sorted = sorted(events, key=lambda e: (pref_rank(e), -event_score(e), e.time or ""))
    return events_sorted[0] if events_sorted else None


def build_helm_webhook_chain(top_flux: FluxObjProblem, root_event: Optional[EventAgg], all_events: List[EventAgg]) -> List[str]:
    chain: List[str] = []
    msg = (top_flux.message or "")
    if "failed calling webhook" not in msg.lower():
        return chain

    chain.append(f"HelmRelease {top_flux.ns}/{top_flux.name}: {top_flux.status} — {trunc(top_flux.message, 180)}")

    svc = ""
    svc_ns = top_flux.ns
    m = WEBHOOK_URL_RE.search(msg)
    if m:
        svc, svc_ns = m.group(1), m.group(2)
        chain.append(f"Webhook endpoint: Service {svc_ns}/{svc} (from URL)")

    def obj_is_related(o: str) -> bool:
        if not o.startswith("Pod/"):
            return False
        n = o.split("/", 1)[1].lower()
        return ("cluster-api-operator" in n) or ("capi-operator" in n) or ("webhook" in n)

    related = [e for e in all_events if e.obj and obj_is_related(e.obj)]
    related.sort(key=lambda e: (-event_score(e), e.time or ""))

    pod_health = next((e for e in related if e.reason in {"Unhealthy", "BackOff"}), None)
    if pod_health:
        chain.append(f"{pod_health.obj}: {pod_health.reason} — {trunc(pod_health.message, 160)}")

    pod_mount = next((e for e in related if e.reason == "FailedMount" and "secret" in (e.message or "").lower()), None)
    if pod_mount:
        sec = ""
        sm = SECRET_NOT_FOUND_RE.search(pod_mount.message or "")
        if sm:
            sec = sm.group(1)
        chain.append(f'{pod_mount.obj}: FailedMount — missing Secret "{sec}"' if sec else f"{pod_mount.obj}: FailedMount — {trunc(pod_mount.message, 160)}")

    if len(chain) == 2 and root_event:
        chain.append(f"{root_event.obj}: {root_event.reason} — {trunc(root_event.message, 160)}")

    return chain


# -----------------------------
# Report
# -----------------------------

def detect_focus_namespaces(pod_problems: List[PodProblem], flux_problems: List[FluxObjProblem]) -> List[str]:
    counts: Dict[str, int] = {}
    for p in pod_problems:
        counts[p.ns] = counts.get(p.ns, 0) + 3
    for f in flux_problems:
        counts[f.ns] = counts.get(f.ns, 0) + 2
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
    for r in rows:
        # only apply to Pod events
        if not r.obj.startswith("Pod/"):
            out.append(r)
            continue

        pod_name = r.obj.split("/", 1)[1]
        pod_start = pod_start_idx.get((r.ns, pod_name))

        # if pod not found or startTime unknown -> keep (safe)
        if pod_start is None:
            out.append(r)
            continue

        et = parse_k8s_time(r.time)
        # if event time missing/unparseable -> keep (safe)
        if et is None:
            out.append(r)
            continue

        if et < pod_start:
            # stale event for a previous pod incarnation -> drop
            continue

        out.append(r)

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
        for p in sorted(hr_root.rglob("*.json")) + sorted(hr_root.rglob("*.yaml")) + sorted(hr_root.rglob("*.yml")):
            try:
                helmreleases.extend(load_k8s_list_from_file(p))
            except Exception:
                continue

    pod_problems = [pp for pp in (extract_pod_issue(p) for p in pods) if pp is not None]  # type: ignore
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
        key=lambda e: parse_k8s_time(e.time) or dt.datetime.min.replace(tzinfo=dt.timezone.utc),
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
        for p in pod_problems:
            affected_ns[p.ns] = affected_ns.get(p.ns, 0) + 1
        top_ns = sorted(affected_ns.items(), key=lambda kv: (-kv[1], kv[0]))[:3]
        samples = ", ".join([f"{p.ns}/{p.name}" for p in pod_problems[:3]])

        out.line(f"Top pod issue: {top_reason} — {top_count} pod(s)")
        if top_ns:
            out.line("Affected namespaces (top): " + ", ".join(f"{ns}({c})" for ns, c in top_ns))
        out.line(f"Sample pods: {samples}")

        best = sorted(pod_problems, key=lambda p: (0 if p.reason in WAIT_REASONS else 1, -len(p.message or "")))[0]
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
        best_any = sorted(ev_collapsed, key=lambda e: (-event_score(e), e.time or ""))[0] if ev_collapsed else None
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
            preferred_objects = []
            root_event = None

    # --- Pods (!= Running/Succeeded) ---
    out.section("\nPods (!= Running/Succeeded):")
    if not pods:
        out.line("(no pods data found in bundle)")
        out.summary_codeblock("(no pods data found in bundle)")
    else:
        non_ok = []
        for p in pods:
            ns = safe_get(p, "metadata.namespace", "") or ""
            name = safe_get(p, "metadata.name", "") or ""
            phase = safe_get(p, "status.phase", "") or ""
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
        for n in nodes:
            name = safe_get(n, "metadata.name", "") or ""
            conds = safe_get(n, "status.conditions", []) or []
            ready = ""
            for c in conds:
                if c.get("type") == "Ready":
                    ready = "True" if c.get("status") == "True" else (c.get("status") or "")
            taints = safe_get(n, "spec.taints", []) or []
            taint_str = ",".join(f"{t.get('key')}:{t.get('effect')}" for t in taints if isinstance(t, dict))
            alloc_cpu = safe_get(n, "status.allocatable.cpu", "") or ""
            alloc_mem = safe_get(n, "status.allocatable.memory", "") or ""
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
            for e in ev_collapsed:
                rows.append([e.type, iso_time(e.time), e.reason, e.obj, f"x{e.count}", trunc(e.message, max_msg_len)])
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
        for i, step in enumerate(causal_chain, 1):
            s = f"{i}. {step}"
            out.line(s)
            lines.append(s)
        out.summary_codeblock("\n".join(lines))


# -----------------------------
# Entrypoint / multi-bundle support
# -----------------------------

def resolve_bundles(input_path: Path) -> List[Tuple[str, Path]]:
    bundles: List[Tuple[str, Path]] = []

    if input_path.is_dir():
        dirs = list_support_bundle_like_dirs(input_path)
        if dirs:
            for d in dirs:
                bundles.append((d.name, d))
            return bundles
        bundles.append((input_path.name, input_path))
        return bundles

    if is_tar_path(input_path):
        tmp = Path(tempfile.mkdtemp(prefix="support-bundle-analyzer-"))
        extract_tar_to_dir(input_path, tmp)

        dirs = list_support_bundle_like_dirs(tmp)
        if dirs:
            for d in dirs:
                bundles.append((d.name, d))
            return bundles

        inner_tars = list_support_bundle_tars_in_dir(tmp)
        if inner_tars:
            for it in inner_tars:
                inner_tmp = Path(tempfile.mkdtemp(prefix="support-bundle-inner-"))
                extract_tar_to_dir(it, inner_tmp)
                inner_dirs = list_support_bundle_like_dirs(inner_tmp)
                if inner_dirs:
                    bundles.append((it.stem, inner_dirs[0]))
                else:
                    bundles.append((it.stem, inner_tmp))
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

    for idx, (label, bdir) in enumerate(bundles, 1):
        if len(bundles) > 1:
            sep = f"\n\n==================== BUNDLE {idx}/{len(bundles)}: {label} ====================\n"
            out.line(sep)
            out.summary_text(f"# BUNDLE {idx}/{len(bundles)}: {label}\n")
        analyze_bundle(bdir, details=args.details, out=out)

    out.finalize()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
