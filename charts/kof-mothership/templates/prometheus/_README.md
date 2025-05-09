* This is a small part of the upstream [kube-prometheus-stack](https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack) chart.
* We did not add this chart as a dependency,
  because that would install a lot of things we don't need,
  and some of them may not be covered with 107 `enabled:` in its [values.yaml](https://github.com/prometheus-community/helm-charts/blob/main/charts/kube-prometheus-stack/values.yaml).
* To get an updaded version of `prometheus/rules` from the upstream:
  * Run:
    ```
    rm -R rules
    git clone --depth=1 https://github.com/prometheus-community/helm-charts
    mv helm-charts/charts/kube-prometheus-stack/templates/prometheus/rules* rules
    mv helm-charts/charts/kube-prometheus-stack/templates/_helpers.tpl .
    ```
  * Copy `customRules` and `defaultRules`
    from `helm-charts/charts/kube-prometheus-stack/values.yaml`
    to `../../values.yaml` of `kof-mothership`.
  * Add these empty keys for compatibility:
    ```
    kube-state-metrics: {}
    kubeApiServer: {}
    kubeControllerManager: {}
    kubeEtcd: {}
    kubelet: {}
    kubeProxy: {}
    kubeScheduler: {}
    prometheusOperator:
      kubeletService: {}
    windowsMonitoring: {}
    ```
  * Run: `rm -R helm-charts`
