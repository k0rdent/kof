#!/usr/bin/env python
"""
Script checking metrics from promxy API that should be populated for
child and regional adopted clusters
"""

import sys
import time
import argparse

import requests

parser = argparse.ArgumentParser()

parser.add_argument(
    "--child-labels",
    type=str,
    help="Additional label selectors to filter metrics in child cluster queries",
)

parser.add_argument(
    "--regional-labels",
    type=str,
    help="Additional label selectors to filter metrics in regional cluster queries",
)

args = parser.parse_args()

REGIONAL_CLUSTER = "regional-adopted"
CHILD_CLUSTER = "child-adopted"

CHILD_LABELS = f", {args.child_labels}" if args.child_labels else ""
REGIONAL_LABELS = f", {args.regional_labels}" if args.regional_labels else ""

metrics = [
    f'up{{job=~"kubernetes-apiservers|apiserver", cluster="{CHILD_CLUSTER}"{CHILD_LABELS}}}',
    f'up{{job=~"kubernetes-apiservers|apiserver", cluster="{REGIONAL_CLUSTER}"{REGIONAL_LABELS}}}',
    f'up{{app_kubernetes_io_name="kof-collectors-daemon-collector", cluster="{CHILD_CLUSTER}"{CHILD_LABELS}}}',
    f'up{{app_kubernetes_io_name="kof-collectors-daemon-collector", cluster="{REGIONAL_CLUSTER}"{REGIONAL_LABELS}}}',
    f'vm_app_uptime_seconds{{cluster="{REGIONAL_CLUSTER}"{REGIONAL_LABELS}}}',
    f'sum(node_total_hourly_cost{{cluster="{CHILD_CLUSTER}"{CHILD_LABELS}}})',
    f'sum(node_total_hourly_cost{{cluster="{REGIONAL_CLUSTER}"{REGIONAL_LABELS}}})',
]

from_timestamp = time.time() - 60

for metric in metrics:
    print(f"Checking promxy metric {metric}:", end=" ")
    r = requests.get(
        f"http://localhost:8082/api/v1/query?query={metric}&time={from_timestamp:.3f}"
    )
    response = r.json()
    if response["status"] == "success":
        if not response["data"]["result"]:
            print("response has no data")
            sys.exit(1)
    else:
        print(f"failed to get: {response}")
        sys.exit(1)
    print("OK")
