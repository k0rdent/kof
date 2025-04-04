# Collector

The Collector is used to gather metrics, logs, and traces from pods running in a Kubernetes nodes.

## How to Customize the Collector?

There are two ways to customize the collector:

- Through the `kof-child` Helm chart values.
- Using the annotation `k0rdent.mirantis.com/kof-collectors-values` in a ClusterDeployment resource.

### Preferred Method

The first method (using the kof-child chart values) is the preferred approach as it centralizes configuration and avoids overloading the ClusterDeployment. However, if you need to apply different configurations for individual child, you can use the annotation method.

**Note**: Annotation values take precedence. The configuration is initially merged from the `kof-child` chart, and then the annotation values are applied to override or extend those settings.

## Example of Collector Customization via Annotation

The example below demonstrates how to configure the Collector via an annotation in a ClusterDeployment. In this example, the Collector is set up to collect logs from the system log file `/var/log/messages` using a syslog parser.

```yaml
apiVersion: k0rdent.mirantis.com/v1alpha1
kind: ClusterDeployment
metadata:
  name: aws-ue2-istio-child
  namespace: kcm-system
  labels:
    k0rdent.mirantis.com/istio-role: child
    k0rdent.mirantis.com/kof-cluster-role: child
spec:
  template: aws-standalone-cp-0-2-0
  credential: aws-cluster-identity-cred
  config:
    clusterAnnotations:
      k0rdent.mirantis.com/kof-collectors-values: |
        collectors:
          node:
            run_as_root: true
            receivers:
              filelog/sys:
                include:
                  - /var/log/messages
                include_file_name: false
                include_file_path: true
                operators:
                  - id: syslog_parser
                    type: syslog_parser
                    protocol: rfc3164
                    on_error: send_quiet
            service:
              pipelines:
                logs:
                  receivers:
                    - filelog/sys

    clusterIdentity:
      name: aws-cluster-identity
      namespace: kcm-system
    controlPlane:
      instanceType: t3.large
    controlPlaneNumber: 1
    publicIP: false
    region: us-east-2
    worker:
      instanceType: t3.medium
    workersNumber: 3
```

**Note**: If you want the nodes collector to have full access, be sure to enable the `run_as_root` flag.

All default values for the Collector can be reviewed [here](https://github.com/k0rdent/kof/blob/main/charts/kof-collectors/values.yaml).
You can find all available configuration options for the OpenTelemetry Collector [here](https://github.com/open-telemetry/opentelemetry-helm-charts/blob/main/charts/opentelemetry-collector/values.yaml).
