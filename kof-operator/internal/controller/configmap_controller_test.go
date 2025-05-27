/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	"github.com/k0rdent/kof/kof-operator/internal/controller/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("ConfigMap controller", func() {
	Context("when reconciling ConfigMaps", func() {
		ctx := context.Background()
		var controllerReconciler *ConfigMapReconciler

		const prometheusRuleName = "test-prometheus-rule"

		// Alert rules.
		const defaultAlertConfigMapName = "test-promxy-rules-default"
		const clusterAlertConfigMapName = "test-promxy-rules-cluster-cluster1"
		const promxyRulesConfigMapName = ReleaseName + "-promxy-rules"

		// Record rules.
		const defaultRecordConfigMapName = "test-record-rules-default"
		const clusterRecordConfigMapName = "test-record-rules-cluster-regional1"
		const recordVMRulesConfigMapName = "kof-record-vmrules-regional1"

		BeforeEach(func() {
			By("creating reconciler")
			controllerReconciler = &ConfigMapReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			By("creating PrometheusRule")
			duration := promv1.Duration("15m")
			prometheusRule := &promv1.PrometheusRule{
				ObjectMeta: metav1.ObjectMeta{
					Name:      prometheusRuleName,
					Namespace: ReleaseNamespace,
					Labels: map[string]string{
						ReleaseNameLabel: ReleaseName,
					},
				},
				Spec: promv1.PrometheusRuleSpec{
					Groups: []promv1.RuleGroup{
						{
							Name: "kubernetes-resources",
							Rules: []promv1.Rule{
								{
									Record: "instance:node_vmstat_pgmajfault:rate5m",
									Expr:   intstr.FromString(`rate(node_vmstat_pgmajfault{job="node-exporter"}[5m])`),
								},
								{
									Alert: "CPUThrottlingHigh",
									Annotations: map[string]string{
										"description": "{{ $value | humanizePercentage }} throttling of CPU in namespace {{ $labels.namespace }} for container {{ $labels.container }} in pod {{ $labels.pod }} on cluster {{ $labels.cluster }}.",
										"runbook_url": "https://runbooks.prometheus-operator.dev/runbooks/kubernetes/cputhrottlinghigh",
										"summary":     "Processes experience elevated CPU throttling.",
									},
									Expr: intstr.FromString(`sum(increase(container_cpu_cfs_throttled_periods_total{container!="", job="kubelet", metrics_path="/metrics/cadvisor", }[5m])) without (id, metrics_path, name, image, endpoint, job, node)
  / on (cluster, namespace, pod, container, instance) group_left
sum(increase(container_cpu_cfs_periods_total{job="kubelet", metrics_path="/metrics/cadvisor", }[5m])) without (id, metrics_path, name, image, endpoint, job, node)
  > ( 25 / 100 )`),
									For:    &duration,
									Labels: map[string]string{"severity": "info"},
								},
							},
						},
						{
							Name: "record-group0",
							Rules: []promv1.Rule{
								{
									Expr:   intstr.FromString(`count (up == 0)`),
									Record: "count:up0_from_prometheus_rule",
								},
							},
						},
						{
							Name: "record-group1",
							Rules: []promv1.Rule{
								{
									Expr:   intstr.FromString(`count (up == 1)`),
									Record: "count:up1_from_prometheus_rule",
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, prometheusRule)).To(Succeed())

			By("creating default alert ConfigMap")
			defaultAlertConfigMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultAlertConfigMapName,
					Namespace: ReleaseNamespace,
					Labels: map[string]string{
						KofAlertRulesClusterNameLabel: "",
					},
				},
				Data: map[string]string{
					"kubernetes-resources": `CPUThrottlingHigh:
  expr: |-
    sum(increase(container_cpu_cfs_throttled_periods_total{cluster!~"^cluster1$|^cluster10$", container!="", job="kubelet", metrics_path="/metrics/cadvisor", }[5m])) without (id, metrics_path, name, image, endpoint, job, node)
      / on (cluster, namespace, pod, container, instance) group_left
    sum(increase(container_cpu_cfs_periods_total{cluster!~"^cluster1$|^cluster10$", job="kubelet", metrics_path="/metrics/cadvisor", }[5m])) without (id, metrics_path, name, image, endpoint, job, node)
      > ( 25 / 100 )
  for: 10m`,
				},
			}
			Expect(k8sClient.Create(ctx, defaultAlertConfigMap)).To(Succeed())

			By("creating cluster alert ConfigMap")
			clusterAlertConfigMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterAlertConfigMapName,
					Namespace: ReleaseNamespace,
					Labels: map[string]string{
						KofAlertRulesClusterNameLabel: "cluster1",
					},
				},
				Data: map[string]string{
					"kubernetes-resources": `CPUThrottlingHigh:
  expr: |-
    sum(increase(container_cpu_cfs_throttled_periods_total{cluster="cluster1", container!="", job="kubelet", metrics_path="/metrics/cadvisor", }[5m])) without (id, metrics_path, name, image, endpoint, job, node)
      / on (cluster, namespace, pod, container, instance) group_left
    sum(increase(container_cpu_cfs_periods_total{cluster="cluster1", job="kubelet", metrics_path="/metrics/cadvisor", }[5m])) without (id, metrics_path, name, image, endpoint, job, node)
      > ( 42 / 100 )`,
				},
			}
			Expect(k8sClient.Create(ctx, clusterAlertConfigMap)).To(Succeed())

			By("creating promxy rules ConfigMap")
			promxyRulesConfigMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      promxyRulesConfigMapName,
					Namespace: ReleaseNamespace,
					Labels: map[string]string{
						utils.KofGeneratedLabel: "true",
					},
					Annotations: map[string]string{
						ReleaseNameAnnotation: ReleaseName,
					},
				},
			}
			Expect(k8sClient.Create(ctx, promxyRulesConfigMap)).To(Succeed())

			By("creating default record ConfigMap")
			defaultRecordConfigMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultRecordConfigMapName,
					Namespace: ReleaseNamespace,
					Labels: map[string]string{
						KofRecordRulesClusterNameLabel: "",
					},
				},
				Data: map[string]string{
					"record-group1": `- expr: count (up == 1)
  record: count:default_up1`,
					"record-group10": `- expr: count (up >= 0)
  record: count:default_up10`,
				},
			}
			Expect(k8sClient.Create(ctx, defaultRecordConfigMap)).To(Succeed())

			By("creating cluster record ConfigMap")
			clusterRecordConfigMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterRecordConfigMapName,
					Namespace: ReleaseNamespace,
					Labels: map[string]string{
						KofRecordRulesClusterNameLabel: "regional1",
					},
				},
				Data: map[string]string{
					"record-group1": `- expr: count (up{cluster="child2"} == 1)
  record: count:child2_up1
- expr: count (up{cluster="child3"} == 1)
  record: count:child3_up1`,
				},
			}
			Expect(k8sClient.Create(ctx, clusterRecordConfigMap)).To(Succeed())

			By("creating record VMules ConfigMap")
			recordVMRulesConfigMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      recordVMRulesConfigMapName,
					Namespace: ReleaseNamespace,
					Labels: map[string]string{
						KofRecordVMRulesClusterNameLabel: "regional1",
						utils.KofGeneratedLabel:          "true",
					},
				},
			}
			Expect(k8sClient.Create(ctx, recordVMRulesConfigMap)).To(Succeed())
		})

		AfterEach(func() {
			configMap := &corev1.ConfigMap{}
			prometheusRule := &promv1.PrometheusRule{}

			if err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      prometheusRuleName,
				Namespace: ReleaseNamespace,
			}, prometheusRule); err == nil {
				By("deleting PrometheusRule")
				Expect(k8sClient.Delete(ctx, prometheusRule)).To(Succeed())
			}

			if err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      defaultAlertConfigMapName,
				Namespace: ReleaseNamespace,
			}, configMap); err == nil {
				By("deleting default alert ConfigMap")
				Expect(k8sClient.Delete(ctx, configMap)).To(Succeed())
			}

			if err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      clusterAlertConfigMapName,
				Namespace: ReleaseNamespace,
			}, configMap); err == nil {
				By("deleting cluster alert ConfigMap")
				Expect(k8sClient.Delete(ctx, configMap)).To(Succeed())
			}

			if err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      promxyRulesConfigMapName,
				Namespace: ReleaseNamespace,
			}, configMap); err == nil {
				By("deleting promxy rules ConfigMap")
				Expect(k8sClient.Delete(ctx, configMap)).To(Succeed())
			}

			if err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      defaultRecordConfigMapName,
				Namespace: ReleaseNamespace,
			}, configMap); err == nil {
				By("deleting default record ConfigMap")
				Expect(k8sClient.Delete(ctx, configMap)).To(Succeed())
			}

			if err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      clusterRecordConfigMapName,
				Namespace: ReleaseNamespace,
			}, configMap); err == nil {
				By("deleting cluster record ConfigMap")
				Expect(k8sClient.Delete(ctx, configMap)).To(Succeed())
			}

			if err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      recordVMRulesConfigMapName,
				Namespace: ReleaseNamespace,
			}, configMap); err == nil {
				By("deleting record VMules ConfigMap")
				Expect(k8sClient.Delete(ctx, configMap)).To(Succeed())
			}
		})

		It("should successfully reconcile ConfigMaps", func() {
			By("reconciling")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      defaultAlertConfigMapName,
					Namespace: ReleaseNamespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("checking the promxy rules ConfigMap")
			promxyRulesConfigMap := &corev1.ConfigMap{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      promxyRulesConfigMapName,
				Namespace: ReleaseNamespace,
			}, promxyRulesConfigMap)).To(Succeed())
			Expect(promxyRulesConfigMap.Data).To(Equal(map[string]string{
				"__cluster1__kubernetes-resources.yaml": `groups:
- name: kubernetes-resources
  rules:
  - alert: CPUThrottlingHigh
    annotations:
      description: '{{ $value | humanizePercentage }} throttling of CPU in namespace
        {{ $labels.namespace }} for container {{ $labels.container }} in pod {{ $labels.pod
        }} on cluster {{ $labels.cluster }}.'
      runbook_url: https://runbooks.prometheus-operator.dev/runbooks/kubernetes/cputhrottlinghigh
      summary: Processes experience elevated CPU throttling.
    expr: |-
      sum(increase(container_cpu_cfs_throttled_periods_total{cluster="cluster1", container!="", job="kubelet", metrics_path="/metrics/cadvisor", }[5m])) without (id, metrics_path, name, image, endpoint, job, node)
        / on (cluster, namespace, pod, container, instance) group_left
      sum(increase(container_cpu_cfs_periods_total{cluster="cluster1", job="kubelet", metrics_path="/metrics/cadvisor", }[5m])) without (id, metrics_path, name, image, endpoint, job, node)
        > ( 42 / 100 )
    for: 10m
    labels:
      alertgroup: kubernetes-resources
      severity: info
`,
				// Note `expr: ...cluster="cluster1" ...( 42 / 100 )` is from cluster `ConfigMap`,
				// `for: 10m` is from default `ConfigMap`,
				// `alertgroup: kubernetes-resources` label is added by the operator,
				// and the rest is from alert rule of `PrometheusRule`,
				// skipping the record rules found in `PrometheusRule`.

				"kubernetes-resources.yaml": `groups:
- name: kubernetes-resources
  rules:
  - alert: CPUThrottlingHigh
    annotations:
      description: '{{ $value | humanizePercentage }} throttling of CPU in namespace
        {{ $labels.namespace }} for container {{ $labels.container }} in pod {{ $labels.pod
        }} on cluster {{ $labels.cluster }}.'
      runbook_url: https://runbooks.prometheus-operator.dev/runbooks/kubernetes/cputhrottlinghigh
      summary: Processes experience elevated CPU throttling.
    expr: |-
      sum(increase(container_cpu_cfs_throttled_periods_total{cluster!~"^cluster1$|^cluster10$", container!="", job="kubelet", metrics_path="/metrics/cadvisor", }[5m])) without (id, metrics_path, name, image, endpoint, job, node)
        / on (cluster, namespace, pod, container, instance) group_left
      sum(increase(container_cpu_cfs_periods_total{cluster!~"^cluster1$|^cluster10$", job="kubelet", metrics_path="/metrics/cadvisor", }[5m])) without (id, metrics_path, name, image, endpoint, job, node)
        > ( 25 / 100 )
    for: 10m
    labels:
      alertgroup: kubernetes-resources
      severity: info
`,
				// Note `expr: ...cluster!~"^cluster1$|^cluster10$"`
				// and `for: 10m` are from default `ConfigMap`,
				// `alertgroup: kubernetes-resources` label is added by the operator,
				// and the rest is from alert rule of `PrometheusRule`,
				// skipping the record rules found in `PrometheusRule`.
			}))

			By("checking the `kof-record-vmrules-$regional_cluster_name` ConfigMap")
			recordVMRulesConfigMap := &corev1.ConfigMap{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      recordVMRulesConfigMapName,
				Namespace: ReleaseNamespace,
			}, recordVMRulesConfigMap)).To(Succeed())
			Expect(recordVMRulesConfigMap.Data).To(Equal(map[string]string{
				"values": `vmrules:
  groups:
    kubernetes-resources:
    - expr: rate(node_vmstat_pgmajfault{job="node-exporter"}[5m])
      record: instance:node_vmstat_pgmajfault:rate5m
    record-group0:
    - expr: count (up == 0)
      record: count:up0_from_prometheus_rule
    record-group1:
    - expr: count (up{cluster="child2"} == 1)
      record: count:child2_up1
    - expr: count (up{cluster="child3"} == 1)
      record: count:child3_up1
    record-group10:
    - expr: count (up >= 0)
      record: count:default_up10
`,
				// Note `kubernetes-resources` and `record-group0` are from `PrometheusRule`,
				// `record-group10` with `default_up10` is from `defaultRecordConfigMap`,
				// and `record-group1` with `child*_up1` is from `clusterRecordConfigMap`.
			}))

			// As we want to check the **update** of the same output `recordVMRulesConfigMap`,
			// we need to avoid its deletion in `AfterEach()`,
			// so we don't move the next test to a separate `It()`.
			By("checking updated output ConfigMap after update/deletion of input ConfigMaps")

			defaultRecordConfigMap := &corev1.ConfigMap{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      defaultRecordConfigMapName,
				Namespace: ReleaseNamespace,
			}, defaultRecordConfigMap)).To(Succeed())
			defaultRecordConfigMap.Data["record-group10"] = `- expr: count (up <= 1)
  record: count:default_up10`
			Expect(k8sClient.Update(ctx, defaultRecordConfigMap)).To(Succeed())

			clusterRecordConfigMap := &corev1.ConfigMap{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      clusterRecordConfigMapName,
				Namespace: ReleaseNamespace,
			}, clusterRecordConfigMap)).To(Succeed())
			Expect(k8sClient.Delete(ctx, clusterRecordConfigMap)).To(Succeed())

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      clusterRecordConfigMapName,
					Namespace: ReleaseNamespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      recordVMRulesConfigMapName,
				Namespace: ReleaseNamespace,
			}, recordVMRulesConfigMap)).To(Succeed())
			expectedData := map[string]string{
				"values": `vmrules:
  groups:
    kubernetes-resources:
    - expr: rate(node_vmstat_pgmajfault{job="node-exporter"}[5m])
      record: instance:node_vmstat_pgmajfault:rate5m
    record-group0:
    - expr: count (up == 0)
      record: count:up0_from_prometheus_rule
    record-group1:
    - expr: count (up == 1)
      record: count:default_up1
    record-group10:
    - expr: count (up <= 1)
      record: count:default_up10
`,
				// Note `record-group1` and `record-group10` are from updated `defaultRecordConfigMap`,
				// there are no rules from deleted `clusterRecordConfigMap`,
				// and the rest is from `PrometheusRule`.
			}
			Expect(recordVMRulesConfigMap.Data).To(Equal(expectedData))

			By("checking output ConfigMap is not updated given invalid input ConfigMap")

			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      defaultRecordConfigMapName,
				Namespace: ReleaseNamespace,
			}, defaultRecordConfigMap)).To(Succeed())
			defaultRecordConfigMap.Data["record-group10"] = "INVALID YAML: -"
			Expect(k8sClient.Update(ctx, defaultRecordConfigMap)).To(Succeed())

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      defaultRecordConfigMapName,
					Namespace: ReleaseNamespace,
				},
			})
			Expect(err).To(MatchError(
				ContainSubstring("error converting YAML to JSON: yaml: " +
					"block sequence entries are not allowed in this context"),
			))

			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      recordVMRulesConfigMapName,
				Namespace: ReleaseNamespace,
			}, recordVMRulesConfigMap)).To(Succeed())
			Expect(recordVMRulesConfigMap.Data).To(Equal(expectedData))
			// Old working version of rules is kept without any changes.
		})
	})
})
