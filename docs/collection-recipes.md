# Data Collection Recipes

Here is a general guidance of how the Opentelemtery Collector configuration can be extended to collect extra data, not included into the default KOF setup.

## Collect metrics from the AWS Cloudwatch

That could be useful to utilize the same [Alerting](https://docs.k0rdent.io/next/admin/kof/kof-alerts/) setup.

1. Select any KOF cluster, where you have kof-collectors helm chart installed. It's recommended to select the regional cluster running in the same AWS Account as you will export metrics from, for better security and connectivity.

1. Install and configure [Prometheus CloudWatch Exporter](https://github.com/prometheus-community/helm-charts/tree/main/charts/prometheus-cloudwatch-exporter) helm chart in the selected KOF cluster. Follow the official [cloudwatch exporter docs](https://github.com/prometheus/cloudwatch_exporter) to select metrics and period to export, as it is implies extra cost of using Cloudwatch API.

1. Make sure that you have [enabled service monitor](https://github.com/prometheus-community/helm-charts/blob/dbe51c19ee2003ce7d268efa0486b9fa4027fb85/charts/prometheus-cloudwatch-exporter/values.yaml#L159). Opentelemtery Collector will automatically scrape metrics from the Prometheus Cloudwatch Exporter.

## Customizing data sending

There are scenarios where you might need to additionally send data to a "cold" storage, like AWS S3 or Kafka for some pre-processing.

### Extra destination

By default collectors send data over http using these two exporters configured:

  - [otlp/http](https://github.com/open-telemetry/opentelemetry-collector/tree/main/exporter/otlphttpexporter)
  - [prometheusremotewrite](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/exporter/prometheusremotewriteexporter)

Please consider the following exporters for some extra data storage:

  - [aws/kinesis](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/exporter/awskinesisexporter/) to sending data to AWS Kinesis for a [real-time analytics patterns](https://aws.amazon.com/blogs/big-data/architectural-patterns-for-real-time-analytics-using-amazon-kinesis-data-streams-part-1/)
  - [kafka](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/exporter/kafkaexporter) to sending data to [Kafka](https://kafka.apache.org/) for high-performance data pipelines, streaming analytics, data integration
  - [etc](https://github.com/open-telemetry/opentelemetry-collector/blob/main/exporter/README.md) - explore available exporters configuration.

### Sending frequency and sizing

To controll the frequency and size of sending data please refer to the [batch processor](https://github.com/open-telemetry/opentelemetry-collector/tree/main/processor/batchprocessor) documentation.

The default configuration is a good compromise between network overhead of sending data too frequently and risk of data lost if data is collected too long before sending.
