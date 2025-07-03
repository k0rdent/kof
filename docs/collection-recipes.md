# Data Collection Recipes

Here is a general guidance of how the Opentelemtery Collector configuration can be extended to collect extra data, not included into the default KOF setup.

## Collect metrics from the AWS Cloudwatch

That could be useful to utilize the same [Alerting](https://docs.k0rdent.io/next/admin/kof/kof-alerts/) setup.

1. Select any KOF cluster, where you have kof-collectors helm chart installed. It's recommended to select the regional cluster running in the same AWS Account as you will export metrics from, for better security and connectivity.

1. Install and configure [Prometheus CloudWatch Exporter](https://github.com/prometheus-community/helm-charts/tree/main/charts/prometheus-cloudwatch-exporter) helm chart in the selected KOF cluster. Follow the official [cloudwatch exporter docs](https://github.com/prometheus/cloudwatch_exporter) to select metrics and period to export, as it is implies extra cost of using Cloudwatch API.

1. Make sure that you have [enabled service monitor](https://github.com/prometheus-community/helm-charts/blob/dbe51c19ee2003ce7d268efa0486b9fa4027fb85/charts/prometheus-cloudwatch-exporter/values.yaml#L159). Opentelemtery Collector will automatically scrape metrics from the Prometheus Cloudwatch Exporter.

