#!/usr/bin/env python
"""
Script checking metrics from promxy API that should be populated for
child and regional adopted clusters
"""

import sys
import time

import requests

REGIONAL_CLUSTER = "regional-adopted"
CHILD_CLUSTER = "child-adopted"

metrics = [
    f'up{{job=~"kubernetes-apiservers|apiserver", cluster="{CHILD_CLUSTER}"}}',
    f'up{{job=~"kubernetes-apiservers|apiserver", cluster="{REGIONAL_CLUSTER}"}}',
    f'vm_app_uptime_seconds{{cluster="{REGIONAL_CLUSTER}"}}',
    f'sum(node_total_hourly_cost{{cluster="{CHILD_CLUSTER}"}})',
    f'sum(node_total_hourly_cost{{cluster="{REGIONAL_CLUSTER}"}})',
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
