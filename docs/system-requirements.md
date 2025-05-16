# System Requirements

## System Overview

Let's assume the `kof` regional storage cluster consists of three nodes configured for fault tolerance.
All nodes must have identical hardware configurations to guarantee consistent performance.

## Hardware Requirements

Each node in the cluster must meet the following hardware specifications:

**Minimal requirements:**
For development and testing purposes.

| Component   | Requirement |
| ----------- | ----------- |
| **CPU**     | 2 Cores     |
| **RAM**     | 4 GB        |
| **Storage** | 25 GB       |

**Recommended Requirements:**
For production usage.

| Component   | Requirement |
| ----------- | ----------- |
| **CPU**     | 3 Cores     |
| **RAM**     | 5 GB        |
| **Storage** | 30 GB       |

### Storage Requirements

Storage capacity may need to be expanded depending on the volume of logs and metrics collected. The estimates below provide guidance for the Victoria components:

#### Victoria Logs Storage

For Victoria Logs storage, every **1 million logs** is estimated to require approximately **25 MB** of storage in the `emptyDir` volume of the `victoria-logs-single` pod.

#### Victoria Metrics Storage

For Victoria Metrics storage, every **100 million metrics** is estimated to require roughly **50 MB** of storage in the `vmstorage-db` PVC.

**Note**: These estimates are approximate and may vary based on workload and environmental factors. To ensure stability, consider provisioning an additional storage margin.

#### Persistent Volume Claims (PVC) Details

By default, the following PVCs are deployed across the nodes:

* **vmstorage-db**: 10Gi per node
Each node is provisioned with its own `vmstorage-db` PVC for storing Victoria Metrics data.
* **vmselect-cachedir**: 2Gi per node
Each node has a dedicated `vmselect-cachedir` PVC for caching in VMSelect.
* **grafana-vm-pvc**: 1Gi
Used for regional components such as Grafana.

## Recommendations for Cluster Scaling

To ensure optimal performance of your cluster under high load, it is recommended to:

* **Resource Monitoring**: Regularly track key metrics such as CPU usage, memory consumption, disk I/O, and data ingestion rates to promptly identify the need for scaling.

* **Review Documentation and Benchmarks**: Consult the official documentation and performance test results to determine the optimal resources for your workload.

Detailed recommendations on resource planning and scaling are available in the official VictoriaMetrics documentation:

* [Understanding your setup size](https://docs.victoriametrics.com/guides/understand-your-setup-size/)

* [Cluster resizing and scalability](https://docs.victoriametrics.com/cluster-victoriametrics/#cluster-resizing-and-scalability)

* [Benchmarks](https://docs.victoriametrics.com/articles/#benchmarks)