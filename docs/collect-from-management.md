# Options to collect data from DEV management cluster

* Full version: https://docs.k0rdent.io/next/admin/kof/kof-storing/
* Current file contains just the DEV versions of the commands.

## From Management to Management

```bash
make dev-storage-deploy
make dev-collectors-deploy
```

## From Management to Regional

```bash
REGIONAL_CLUSTER_NAME=$USER-aws-standalone-regional
REGIONAL_DOMAIN=$REGIONAL_CLUSTER_NAME.$KOF_DNS

cat >dev/collectors-values.yaml <<EOF
kof:
  logs:
    endpoint: https://vmauth.$REGIONAL_DOMAIN/vls/insert/opentelemetry/v1/logs
  metrics:
    endpoint: https://vmauth.$REGIONAL_DOMAIN/vm/insert/0/prometheus/api/v1/write
  traces:
    endpoint: https://jaeger.$REGIONAL_DOMAIN/collector
opencost:
  opencost:
    prometheus:
      external:
        url: https://vmauth.$REGIONAL_DOMAIN/vm/select/0/prometheus
EOF

helm upgrade -i --reset-values --wait -n kof kof-collectors ./charts/kof-collectors \
  -f dev/collectors-values.yaml
```

## From Management to Regional with Istio

```bash
REGIONAL_CLUSTER_NAME=$USER-aws-standalone-regional

cat >dev/collectors-values.yaml <<EOF
kof:
  basic_auth: false
  logs:
    endpoint: http://$REGIONAL_CLUSTER_NAME-logs:9428/insert/opentelemetry/v1/logs
  metrics:
    endpoint: http://$REGIONAL_CLUSTER_NAME-vminsert:8480/insert/0/prometheus/api/v1/write
  traces:
    endpoint: http://$REGIONAL_CLUSTER_NAME-jaeger-collector:4318
opencost:
  opencost:
    prometheus:
      existingSecretName: ""
      external:
        url: http://$REGIONAL_CLUSTER_NAME-vmselect:8481/select/0/prometheus
EOF

helm upgrade -i --reset-values --wait -n kof kof-collectors ./charts/kof-collectors \
  -f dev/collectors-values.yaml
```

## From Management to Third-party

```bash
cat >dev/cloudwatch-credentials <<EOF
AWS_ACCESS_KEY_ID=REDACTED
AWS_SECRET_ACCESS_KEY=REDACTED
EOF

kubectl create secret generic -n kof cloudwatch-credentials \
  --from-env-file=dev/cloudwatch-credentials

COLLECTOR_CONFIG="
    env:
      - name: AWS_ACCESS_KEY_ID
        valueFrom:
          secretKeyRef:
            name: cloudwatch-credentials
            key: AWS_ACCESS_KEY_ID
      - name: AWS_SECRET_ACCESS_KEY
        valueFrom:
          secretKeyRef:
            name: cloudwatch-credentials
            key: AWS_SECRET_ACCESS_KEY
    exporters:
      awscloudwatchlogs:
        region: us-east-2
        log_group_name: management
        log_stream_name: logs"

cat >dev/collectors-values.yaml <<EOF
kof:
  logs:
    endpoint: ""
  metrics:
    endpoint: ""
  traces:
    endpoint: ""
collectors:
  k8scluster:$COLLECTOR_CONFIG
    service:
      pipelines:
        logs:
          exporters:
            - awscloudwatchlogs
            - debug
        metrics:
          exporters:
            - debug
  node:$COLLECTOR_CONFIG
    service:
      pipelines:
        logs:
          exporters:
            - awscloudwatchlogs
            - debug
        metrics:
          exporters:
            - debug
        traces:
          exporters:
            - debug
EOF

helm upgrade -i --reset-values --wait -n kof kof-collectors ./charts/kof-collectors \
  -f dev/collectors-values.yaml

aws configure
  # Use the same access key

aws logs get-log-events \
  --region us-east-2 \
  --log-group-name management \
  --log-stream-name logs \
  --limit 1
```
