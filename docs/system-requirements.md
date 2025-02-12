# System Requirements

## System Overview

The `kof` cluster consists of three nodes configured for fault tolerance. All nodes must have identical hardware configurations to guarantee consistent performance.

## Hardware Requirements

Each node in the cluster must meet the following hardware specifications:

**Minimal requirements:**
These requirements reflect the resource specifications that were used during development and testing.

| Component   | Requirement |
| ----------- | ----------- |
| **CPU**     | 2 Cores     |
| **RAM**     | 4 GB        |
| **Storage** | 25 GB       |

**Recommended Requirements:**
These recommendations are based on preset resource limits for each service. However, since some services do not have explicit limits, they may consume additional resources under heavy load.

| Component   | Requirement |
| ----------- | ----------- |
| **CPU**     | 3 Cores     |
| **RAM**     | 5 GB        |
| **Storage** | 30 GB       |

### Storage Requirements

Storage capacity may need to be expanded depending on the volume of logs and metrics collected. The estimates below provide guidance for the Victoria components:

#### Victoria Logs Storage

For Victoria Logs storage, every **1 million logs** is estimated to require approximately **25 MB** of storage in the `victoria-logs-single` pod.

#### Victoria Metrics Storage

For Victoria Metrics storage, every **100 million metrics** is estimated to require roughly **50 MB** of storage in the `vmstorage-cluster` pod.

**Note**: These estimates are approximate and may vary based on workload and environmental factors. To ensure stability, consider provisioning an additional storage margin.
