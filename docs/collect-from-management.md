# Options to collect data from DEV management cluster

* Non-dev version will be added to https://docs.k0rdent.io/next/admin/kof/kof-install/

## Export to the same management cluster

* This option uses:
  * Grafana and VictoriaMetrics provided by `kof-mothership`, disabled in `kof-storage`.
  * VictoriaLogs and Jaeger provided by `kof-storage`.
* Run:
  ```bash
  make dev-storage-deploy
  make dev-collectors-deploy
  ```

## Export to a regional cluster

* This option assumes you did not enable Istio,
  and you have a regional cluster with the name and domain below.
* Run:
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

  helm upgrade -i --wait -n kof kof-collectors ./charts/kof-collectors -f dev/collectors-values.yaml
  ```

## Export to a regional cluster with Istio

* This option assumes you have Istio enabled,
  and you have a regional cluster with the name below.
* Run:
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

  helm upgrade -i --wait -n kof kof-collectors ./charts/kof-collectors -f dev/collectors-values.yaml
  ```

## Export to a third-party storage like CloudWatch

* This option uses the [AWS CloudWatch Logs Exporter](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/exporter/awscloudwatchlogsexporter) as an example of a third-party storage.
* You should use the most secure option to [specify AWS credentials](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials),
  but to keep this demo simple we will use environment variables stored in a k8s secret.
* Create AWS IAM user with access to CloudWatch Logs,
  e.g. by allowing `"Action": "logs:*"` in the inline policy.
* Create access key and save it to the `dev/cloudwatch-credentials` file:
  ```
  AWS_ACCESS_KEY_ID=REDACTED
  AWS_SECRET_ACCESS_KEY=REDACTED
  ```
* Create the `cloudwatch-credentials` secret:
  ```bash
  kubectl create secret generic -n kof cloudwatch-credentials \
    --from-env-file=dev/cloudwatch-credentials
  ```
* Run:
  ```bash
  REGIONAL_CLUSTER_NAME=$USER-aws-standalone-regional

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
          log_group_name: $REGIONAL_CLUSTER_NAME
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

  helm upgrade -i --wait -n kof kof-collectors ./charts/kof-collectors -f dev/collectors-values.yaml
  ```
* Verify:
  ```bash
  aws configure
    # Use the same access key

  aws logs get-log-events \
    --region us-east-2 \
    --log-group-name $REGIONAL_CLUSTER_NAME \
    --log-stream-name logs \
    --limit 1
  ```
* Example of the output:
  ```
  {"events": [{
    "timestamp": 1744305535107,
    "message": "{\"body\":\"10.244.0.1 - - [10/Apr/2025 17:18:55] \\\"GET /-/ready HTTP/1.1 200 ...
  ```
