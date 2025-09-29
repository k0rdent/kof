import os
import sys
import json


def main():
    if len(sys.argv) != 2:
        print("Usage: python support-bundle-analyzer.py <support_bundle_root>")
        sys.exit(1)

    root_folder = sys.argv[1]
    pods_dir = os.path.join(root_folder, "cluster-resources", "pods")
    cr_dir = os.path.join(root_folder, "cluster-resources", "custom-resources")

    if not os.path.isdir(pods_dir):
        print(f"Pods directory not found: {pods_dir}")
        sys.exit(1)

    pod_rows = []
    for filename in os.listdir(pods_dir):
        if filename.endswith(".json"):
            file_path = os.path.join(pods_dir, filename)
            with open(file_path, "r") as f:
                pod_list = json.load(f)
                # Expecting pod_list to be a dict with "items" key
                pod_list_items = pod_list.get("items", [])
                if not pod_list_items:
                    continue
                for pod in pod_list_items:
                    metadata = pod.get("metadata", {})
                    status = pod.get("status", {})
                    namespace = metadata.get("namespace", "")
                    name = metadata.get("name", "")
                    phase = status.get("phase", "")
                    pod_rows.append((namespace, phase, name))

    print("Pods:")
    print("-" * 75)
    print(f"{'namespace':<20} {'phase':<15} {'name':<40}")
    print("-" * 75)
    for row in pod_rows:
        print(f"{row[0]:<20} {row[1]:<14} {row[2]:<40}")

    cr_rows = []
    for dirpath, _, filenames in os.walk(cr_dir):
        for filename in filenames:
            if filename.endswith(".json"):
                file_path = os.path.join(dirpath, filename)
                with open(file_path, "r") as f:
                    cr_list = json.load(f)
                    for cr in cr_list:
                        metadata = cr.get("metadata", {})
                        status = cr.get("status", {})
                        namespace = metadata.get("namespace", "")
                        name = metadata.get("name", "")
                        conditions = status.get("conditions", [])
                        if conditions:
                            for c in conditions:
                                if c.get("status", "") != "True":
                                    cr_rows.append(
                                        (
                                            namespace,
                                            name,
                                            f"{c.get('type', '')}:{c.get('status', '')}:{c.get('message', '')}",
                                        )
                                    )

    print("")
    print("Custom resources conditions:")
    print("-" * 75)
    print(f"{'namespace':<20} {'name':<40} {'status':<20}")
    print("-" * 75)
    for row in cr_rows:
        print(f"{row[0]:<20} {row[1]:<40} {row[2]:<20}")


if __name__ == "__main__":
    main()
