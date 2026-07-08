import json
import os
import subprocess


def read_min_attempts():
    raw = os.getenv("MIN_ATTEMPTS", "3")
    try:
        if (value := int(raw)) >= 2:
            return value
    except ValueError:
        pass
    print(f"MIN_ATTEMPTS must be an integer >= 2, got: {raw}; using 3.")
    return 3


def read_pods(kubectl, namespace, context):
    try:
        result = subprocess.run(kubectl + ["-n", namespace, "get", "pods", "-o", "json"],
                                check=True, capture_output=True, text=True)
        return json.loads(result.stdout)
    except FileNotFoundError:
        print(f"Cannot report KOF pod restarts: {kubectl[0]} is not installed.")
    except subprocess.CalledProcessError as error:
        output = error.stderr or error.stdout or str(error)
        print(f"Cannot read pods in namespace {namespace} for context {context}:")
        print(output.strip())
    except json.JSONDecodeError as error:
        print(f"Cannot parse pod JSON in namespace {namespace} for context {context}:")
        print(error)
    return None


def previous_log(kubectl, namespace, pod_name, container_name):
    try:
        result = subprocess.run(kubectl + ["-n", namespace, "logs", pod_name, "-c", container_name, "--previous", "--tail=20"],
                                check=True, capture_output=True, text=True)
    except (FileNotFoundError, subprocess.CalledProcessError):
        return "-"

    lines = [line.strip() for line in result.stdout.splitlines() if line.strip()]
    if not lines:
        return "-"
    return " ".join(lines[-1].split())[:180]


def main() -> int:
    namespace = os.getenv("NAMESPACE", "kof")
    minimum = read_min_attempts()
    context_name = os.getenv("KUBECTL_CONTEXT", "")
    context = context_name or "current"
    kubectl = [os.getenv("KUBECTL", "kubectl")] + (["--context", context_name] if context_name else [])

    print(f"KOF pod startup attempts report: namespace={namespace} context={context} minAttempts={minimum}")
    pods = read_pods(kubectl, namespace, context)
    if pods is None:
        return 0

    rows = []
    for pod in pods.get("items", []):
        pod_name = pod["metadata"]["name"]
        for container in pod.get("status", {}).get("containerStatuses", []):
            restarts = container.get("restartCount", 0)
            attempts = restarts + 1
            if attempts >= minimum:
                terminated = container.get("lastState", {}).get("terminated", {})
                reason = terminated.get("reason", "-")
                exit_code = terminated.get("exitCode", "-")
                container_name = container["name"]
                log = previous_log(kubectl, namespace, pod_name, container_name)
                rows.append((attempts, pod_name, container_name, f"{reason}/{exit_code}", log))
    rows.sort(key=lambda row: (-row[0], row[1], row[2]))

    if not rows:
        print(f"No KOF pod containers with startup attempts >= {minimum}.")
        return 0

    print(f"{'ATTEMPTS':<8} {'POD':<64} {'CONTAINER':<32} {'LAST_FAILURE':<14} PREVIOUS_LOG")
    for attempts, pod_name, container_name, last_failure, log in rows:
        print(f"{attempts:<8} {pod_name:<64} {container_name:<32} {last_failure:<14} {log}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
