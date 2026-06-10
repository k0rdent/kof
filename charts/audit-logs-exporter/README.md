# audit-logs-exporter

![Version: 1.11.0-rc0](https://img.shields.io/badge/Version-1.11.0--rc0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 1.11.0-rc0](https://img.shields.io/badge/AppVersion-1.11.0--rc0-informational?style=flat-square)

Hourly CronJob that exports audit events from VictoriaLogs to an S3-compatible object store.

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` | Affinity rules for the Job pods. |
| catchupHours | string | `"24"` | How many hours back to look for un-exported windows on each run. |
| complianceMode | string | `"false"` | Compliance mode. When true, the exporter aborts if the S3 bucket does not have WORM/Object-lock enabled. |
| concurrencyPolicy | string | `"Forbid"` | Kubernetes concurrency policy. Forbid prevents overlapping runs, which satisfies the spec requirement that concurrent runs against the same window must not produce duplicates. |
| exportDelay | string | `"5m"` | How long after the hour boundary to wait before exporting a window. Absorbs late/out-of-order events.  Value is a Go duration string. |
| failedJobsHistoryLimit | int | `5` | Number of failed jobs to retain in history. |
| image.pullPolicy | string | `"IfNotPresent"` |  |
| image.repository | string | `"ghcr.io/k0rdent/kof/kof-audit-logs-exporter"` |  |
| image.tag | string | `""` |  |
| kms.keyID | string | `"local-dev-key"` | Key reference passed to the signer. For the built-in LocalSigner this is the HMAC key (base64-encoded or raw string). For AWS KMS set this to the key ARN / alias. |
| nodeSelector | object | `{}` | Node selector for the Job pods. |
| podAnnotations | object | `{}` |  |
| podLabels | object | `{}` |  |
| producerName | string | `"audit-logs-exporter"` | Producer name embedded in every manifest. |
| producerVersion | string | `""` | Producer version embedded in every manifest.  Defaults to chart appVersion. |
| resources | object | `{"limits":{"cpu":"500m","memory":"256Mi"},"requests":{"cpu":"50m","memory":"64Mi"}}` | Resource requests/limits for the exporter container. |
| s3.bucket | string | `""` | Target bucket name. |
| s3.credentials.accessKey | string | `""` | S3 access key.  Written to the Secret when existingSecret is empty. |
| s3.credentials.secretKey | string | `""` | S3 secret key.  Written to the Secret when existingSecret is empty. |
| s3.endpoint | string | `""` | S3-compatible endpoint URL (e.g. http://minio.minio.svc.cluster.local:9000). |
| s3.existingSecret | string | `""` | Credentials are stored in a Secret referenced below. If existingSecret is set, the chart will reference that Secret instead of creating a new one. |
| s3.forceHTTP | string | `"false"` | Skip TLS certificate verification. Set to "true" only in development. |
| s3.prefix | string | `"audit"` | Key prefix inside the bucket (no leading/trailing slash). |
| s3.region | string | `"us-east-1"` | AWS/S3 region. |
| s3.secretName | string | `"audit-logs-exporter-s3-credentials"` |  |
| s3.usePathStyle | string | `"true"` | Use path-style S3 addressing. Required for MinIO and most self-hosted S3. |
| schedule | string | `"5 * * * *"` | CronJob schedule (UTC). Default: 5 minutes past every hour. Running at :05 with exportDelay=5m means the job looks back to exactly :00, capturing the full previous-hour window without missing the last few minutes. |
| serviceAccount.annotations | object | `{}` |  |
| serviceAccount.create | bool | `true` | Create a dedicated ServiceAccount for the CronJob. |
| serviceAccount.name | string | `""` |  |
| streams | string | `"tenant-audit-log,platform-audit-log"` | Comma-separated list of audit streams to export. Default: both streams defined in the spec. |
| successfulJobsHistoryLimit | int | `3` | Number of completed successful jobs to retain in history. Reduce to save etcd space in long-running clusters. |
| tenants | string | `""` | Comma-separated list of tenant IDs for tenant-audit-log. Leave empty to auto-discover tenants from VictoriaLogs. |
| timeZone | string | `"UTC"` | Time zone for the CronJob schedule (requires k8s >=1.27). |
| tolerations | list | `[]` | Tolerations for the Job pods. |
| vlogsURL | string | `"http://vlselect-audit-logs:9471"` | VictoriaLogs base URL. |

