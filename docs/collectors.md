# Opentelemetry Collectors Custom Configuration

The Collector is used to gather metrics, logs, and traces from pods running in a Kubernetes nodes.

## How to Customize Collectors?

There are two ways to customize the collector:

- Through the `kof-child` Helm chart values.
- Using the annotation `k0rdent.mirantis.com/kof-collectors-values` in a ClusterDeployment resource.

### Preferred Method

The first method (using the kof-child chart values) is the preferred approach as it centralizes configuration and avoids overloading the ClusterDeployment. However, if you need to apply different configurations for individual child, you can use the annotation method.

**Note**: Annotation values take precedence. The configuration is initially merged from the `kof-child` chart, and then the annotation values are applied to override or extend those values.

## Steps to Customize Collectors Through `kof-child` Values

To customize Collectors using `kof-child` [values](https://github.com/k0rdent/kof/blob/main/charts/kof-child/values.yaml), follow these steps:

1. Open the `charts/kof-child/values.yaml` file.
2. Locate or add the `collectors` section.
3. Modify or add the desired configuration for the collectors.

### Example

Below is an example of how the `values.yaml` file should look if you want to customize Collectors:

```yaml
kcm:
  namespace: kcm-system
collectors:
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
```

**Note:** Ensure that the configuration is placed inside the collectors section.

## Example of Collectors Customization via Annotation

The example below demonstrates how to configure Collectors via an annotation in a ClusterDeployment. In this example, Collectors are set up to collect logs from the system log file `/var/log/messages` using a syslog parser.

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

**Note**: If you want nodes collectors to have full access, be sure to enable the `run_as_root` flag.

All default values for Collectors can be reviewed [here](https://github.com/k0rdent/kof/blob/main/charts/kof-collectors/values.yaml).
You can find all available configuration options for the OpenTelemetry Collector [here](https://opentelemetry.io/docs/collector/configuration).
