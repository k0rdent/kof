# cold-storage-exporter

![Version: 1.11.0-rc0](https://img.shields.io/badge/Version-1.11.0--rc0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 1.11.0-rc0](https://img.shields.io/badge/AppVersion-1.11.0--rc0-informational?style=flat-square)

Hourly CronJob that exports metrics, logs, and traces from VictoriaMetrics/VictoriaLogs/VictoriaTraces to Parquet on S3-compatible object storage.

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` | Affinity rules for the Job pods. |
| catchupHours | string | `"24"` | How many hours back to look for un-exported windows on each run. |
| clusters | string | `""` | Comma-separated list of cluster names to export. Leave empty to auto-discover clusters from the source. |
| concurrencyPolicy | string | `"Forbid"` | Kubernetes concurrency policy. Forbid prevents overlapping runs. |
| exportDelay | string | `"5m"` | How long after the hour boundary to wait before exporting a window. Absorbs late/out-of-order events. Value is a Go duration string. |
| failedJobsHistoryLimit | int | `5` | Number of failed jobs to retain in history. |
| image.pullPolicy | string | `"IfNotPresent"` |  |
| image.repository | string | `"ghcr.io/k0rdent/kof/kof-cold-storage-exporter"` |  |
| image.tag | string | `""` |  |
| nodeSelector | object | `{}` | Node selector for the Job pods. |
| podAnnotations | object | `{}` |  |
| podLabels | object | `{}` |  |
| resources | object | `{"limits":{"cpu":"1000m","memory":"1Gi"},"requests":{"cpu":"100m","memory":"256Mi"}}` | Resource requests/limits for the exporter container. |
| s3.bucket | string | `""` | Target bucket name. |
| s3.credentials.accessKey | string | `""` | S3 access key. Written to the Secret when existingSecret is empty. |
| s3.credentials.secretKey | string | `""` | S3 secret key. Written to the Secret when existingSecret is empty. |
| s3.endpoint | string | `""` | S3-compatible endpoint URL (e.g. http://minio.minio.svc.cluster.local:9000). |
| s3.existingSecret | string | `""` | If existingSecret is set, the chart will reference that Secret instead of creating a new one. |
| s3.forceHTTP | string | `"false"` | Skip TLS certificate verification. Set to "true" only in development. |
| s3.prefix | string | `"telemetry"` | Key prefix inside the bucket (no leading/trailing slash). |
| s3.region | string | `"us-east-1"` | AWS/S3 region. |
| s3.secretName | string | `"cold-storage-exporter-s3-credentials"` |  |
| s3.usePathStyle | string | `"true"` | Use path-style S3 addressing. Required for MinIO and most self-hosted S3. |
| schedule | string | `"5 * * * *"` | CronJob schedule (UTC). Default: 5 minutes past every hour. |
| serviceAccount.annotations | object | `{}` |  |
| serviceAccount.create | bool | `true` | Create a dedicated ServiceAccount for the CronJob. |
| serviceAccount.name | string | `""` |  |
| sources | string | `"metrics,logs"` | Sources to export. Comma-separated list of: metrics, logs, traces. |
| successfulJobsHistoryLimit | int | `3` | Number of completed successful jobs to retain in history. |
| tenants | string | `""` | Comma-separated list of tenant IDs to export. Leave empty to auto-discover tenants from the source. |
| timeZone | string | `"UTC"` | Time zone for the CronJob schedule (requires k8s >=1.27). |
| tolerations | list | `[]` | Tolerations for the Job pods. |
| vlogsURL | string | `"http://kof-storage-victoria-logs-cluster-vlselect.kof.svc.cluster.local:9471"` | VictoriaLogs base URL (for logs export). |
| vmURL | string | `"http://vmselect-cluster.kof.svc.cluster.local:8481/select/0/prometheus"` | VictoriaMetrics base URL (for metrics export). |
| vtracesURL | string | `"http://kof-storage-vt-cluster-vtselect.kof.svc.cluster.local:10471"` | VictoriaTraces base URL (LogsQL API on port 10471; for traces export). |

