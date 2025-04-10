# Alert System

## Overview

The alerting system is designed to help receive and manage alerts effectively. It consists of two main components:

* **VM Alert:** This component evaluates alerting rules based on metric data and sends alerts to the VMAlertmanager for further processing.
* **VM Alert Manager:** This component routes and sends alerts to various receivers. It handles the aggregation and routing of alerts triggered by VMAlert.

Both objects are created only if both flags below are enabled in the values file:

* ```victoriametrics.enabled```
* ```victoriametrics.vmalert.enabled```

**Data source-managed Alert Rules:**
The system uses data source-managed alert rules. These rules are stored and evaluated within the data source, reducing the load on Grafana and allowing evaluations to run closer to the data. This approach improves scalability and performance when handling large volumes of alerts. For more details, please refer to Grafana’s [Alert Rules documentation](https://grafana.com/docs/grafana/latest/alerting/fundamentals/alert-rules/).

## Receivers Configuration

To configure the alerting system, you need to set the appropriate parameters in the [mothership](https://github.com/k0rdent/kof/blob/main/charts/kof-mothership/values.yaml) values file. This configuration defines how alerts are processed, routed, and delivered to the desired notification channels.

### Example

In this example, the configuration enables VMAlert and sets up a receiver for webhook notifications:

```yaml
vmalert:
  enabled: true
  remoteRead: ""
  vmalertmanager: 
    config: |
      global:
        resolve_timeout: 5m
      route:
        receiver: webhook_receiver
      receivers:
        - name: webhook_receiver
          webhook_configs:
            - url: '<INSERT-YOUR-WEBHOOK>'
              send_resolved: false
```

For more detailed information on configuring other receivers and further customizations, please refer to the [Alertmanager configuration specification](https://github.com/VictoriaMetrics/VictoriaMetrics/blob/master/docs/victoriametrics-cloud/alertmanager-setup-for-deployment.md#alertmanager-config-specification).

## Example of Received Alert

Below is an example of an alert message generated by the system.

```text
Alert Firing:
Labels:
    alertname = KubeAPIDown
    alertgroup = kubernetes-system-apiserver
    severity = critical
Annotations:
    description = KubeAPI has disappeared from Prometheus target discovery.
    runbook_url = https://runbooks.prometheus-operator.dev/runbooks/kubernetes/kubeapidown
    summary = Target disappeared from Prometheus target discovery.
Source: http://vmalert-cluster-844f5c8756-c4z86:8080/vmalert/alert?group_id=1257292949034549654&alert_id=5362419348168359053
```

## How to Test the Alert System

To ensure that your alerting pipeline is working correctly, follow these steps to test the alert system:

1. Set up alert manager with the configuration provided below.
2. Visit [Webhook.site](https://webhook.site/) to generate a unique URL. This URL will be used to receive alert notifications.
3. Replace <YOUR_WEBHOOK_URL_HERE> in the configuration with the URL you obtained from [Webhook.site](https://webhook.site/).
4. Apply the updated configuration. Once the alert is triggered, you should see a notification on the [Webhook.site](https://webhook.site/) page.

**Test configuration:**

```yaml
vmalert:
  enabled: true
  remoteRead: ""
  vmalertmanager: 
    config: |
      global:
        resolve_timeout: 5m
      route:
        receiver: test_notifier
        group_interval: 5m
        repeat_interval: 12h
      receivers:
        - name: test_notifier
          webhook_configs:
            - url: <YOUR_WEBHOOK_URL_HERE>

```

If everything is set up correctly, you will receive a "**Watchdog**" notification. This confirms that the entire alerting pipeline is functional.
