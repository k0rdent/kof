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
		const defaultConfigMapName = "test-promxy-rules-default"
		const clusterConfigMapName = "test-promxy-rules-cluster-cluster1"
		promxyRulesConfigMapName := ReleaseName + "-promxy-rules"

		prometheusRuleNamespacedName := types.NamespacedName{
			Name:      prometheusRuleName,
			Namespace: ReleaseNamespace,
		}

		defaultConfigMapNamespacedName := types.NamespacedName{
			Name:      defaultConfigMapName,
			Namespace: ReleaseNamespace,
		}

		clusterConfigMapNamespacedName := types.NamespacedName{
			Name:      clusterConfigMapName,
			Namespace: ReleaseNamespace,
		}

		promxyRulesConfigMapNamespacedName := types.NamespacedName{
			Name:      promxyRulesConfigMapName,
			Namespace: ReleaseNamespace,
		}

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
					},
				},
			}
			Expect(k8sClient.Create(ctx, prometheusRule)).To(Succeed())

			By("creating default ConfigMap")
			defaultConfigMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultConfigMapName,
					Namespace: ReleaseNamespace,
					Labels: map[string]string{
						KofRulesClusterNameLabel: "",
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
			Expect(k8sClient.Create(ctx, defaultConfigMap)).To(Succeed())

			By("creating cluster ConfigMap")
			clusterConfigMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterConfigMapName,
					Namespace: ReleaseNamespace,
					Labels: map[string]string{
						KofRulesClusterNameLabel: "cluster1",
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
			Expect(k8sClient.Create(ctx, clusterConfigMap)).To(Succeed())

			By("creating promxy rules ConfigMap")
			promxyRulesConfigMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      promxyRulesConfigMapName,
					Namespace: ReleaseNamespace,
					Annotations: map[string]string{
						ReleaseNameAnnotation: ReleaseName,
					},
				},
			}
			Expect(k8sClient.Create(ctx, promxyRulesConfigMap)).To(Succeed())
		})

		AfterEach(func() {
			configMap := &corev1.ConfigMap{}
			prometheusRule := &promv1.PrometheusRule{}

			if err := k8sClient.Get(ctx, prometheusRuleNamespacedName, prometheusRule); err == nil {
				By("deleting PrometheusRule")
				Expect(k8sClient.Delete(ctx, prometheusRule)).To(Succeed())
			}

			if err := k8sClient.Get(ctx, defaultConfigMapNamespacedName, configMap); err == nil {
				By("deleting default ConfigMap")
				Expect(k8sClient.Delete(ctx, configMap)).To(Succeed())
			}

			if err := k8sClient.Get(ctx, clusterConfigMapNamespacedName, configMap); err == nil {
				By("deleting cluster ConfigMap")
				Expect(k8sClient.Delete(ctx, configMap)).To(Succeed())
			}

			if err := k8sClient.Get(ctx, promxyRulesConfigMapNamespacedName, configMap); err == nil {
				By("deleting promxy rules ConfigMap")
				Expect(k8sClient.Delete(ctx, configMap)).To(Succeed())
			}
		})

		It("should successfully reconcile ConfigMaps", func() {
			By("reconciling")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: clusterConfigMapNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("checking the promxy rules ConfigMap")
			configMap := &corev1.ConfigMap{}
			Expect(k8sClient.Get(ctx, promxyRulesConfigMapNamespacedName, configMap)).To(Succeed())
			Expect(configMap.Data).To(Equal(map[string]string{
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
				// Note `expr: ...cluster="cluster1" ...( 42 / 100 )` is from cluster ConfigMap,
				// `for: 10m` is from default ConfigMap,
				// `alertgroup: kubernetes-resources` label is added by the operator,
				// and the rest is from alerting rule of PrometheusRule,
				// skipping the recording rule.

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
				// and `for: 10m` are from default ConfigMap,
				// `alertgroup: kubernetes-resources` label is added by the operator,
				// and the rest is from alerting rule of PrometheusRule,
				// skipping the recording rule.
			}))
		})
	})
})
