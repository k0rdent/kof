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
	"os"
	"strconv"
	"time"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	grafanav1beta1 "github.com/grafana/grafana-operator/v5/api/v1beta1"
	kofv1beta1 "github.com/k0rdent/kof/kof-operator/api/v1beta1"
	"github.com/k0rdent/kof/kof-operator/internal/controller/utils"
	"github.com/k0rdent/kof/kof-operator/internal/controller/vmuser"
	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("RegionalConfigMap Controller", func() {
	Context("When reconciling a resource", func() {
		ctx := context.Background()
		var clusterDeploymentReconciler *ClusterDeploymentReconciler
		var regionalClusterConfigmapReconciler *RegionalClusterConfigMapReconciler

		const regionalClusterDeploymentName = "test-regional-cm"

		regionalClusterConfigmapNamespacedName := types.NamespacedName{
			Name:      GetRegionalClusterConfigMapName(regionalClusterDeploymentName),
			Namespace: defaultNamespace,
		}

		// child ClusterDeployment

		const childClusterDeploymentName = "test-child-cm"

		childClusterDeploymentNamespacedName := types.NamespacedName{
			Name:      childClusterDeploymentName,
			Namespace: defaultNamespace,
		}

		childClusterDeploymentLabels := map[string]string{
			IstioRoleLabel:              "member",
			KofClusterRoleLabel:         "child",
			KofRegionalClusterNameLabel: regionalClusterDeploymentName,
			utils.ClusterNameLabel:      childClusterDeploymentName,
		}

		childClusterDeploymentAnnotations := map[string]string{}

		const childClusterDeploymentConfig = `{"region": "us-east-2"}`

		// child cluster ConfigMap

		childClusterConfigMapNamespacedName := types.NamespacedName{
			Name:      "kof-cluster-config-test-child-cm", // prefix + child cluster name
			Namespace: defaultNamespace,
		}

		const secretName = "test-child-cm-kubeconfig"

		const clusterTemplateName = "aws-cluster-template"

		// createClusterTemplate

		createClusterTemplate := func(name string, namespace string) {
			clusterTemplate := &kcmv1beta1.ClusterTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: kcmv1beta1.ClusterTemplateSpec{
					Helm: kcmv1beta1.HelmSpec{
						ChartSpec: &sourcev1.HelmChartSpec{
							Chart: "aws-standalone-cp",
							SourceRef: sourcev1.LocalHelmChartSourceReference{
								Name: "kcm-templates",
								Kind: "HelmRepository",
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, clusterTemplate)).To(Succeed())

			clusterTemplate.Status = kcmv1beta1.ClusterTemplateStatus{
				Providers: []string{"infrastructure-aws"},
			}
			Expect(k8sClient.Status().Update(ctx, clusterTemplate)).To(Succeed())
		}

		// create regional cluster configmap

		createRegionalClusterConfigMap := func(
			name,
			namespace,
			istioRole,
			readMetricsEndpoint,
			writeMetricsEndpoint,
			readLogsEndpoint,
			writeLogsEndpoint,
			readTracesEndpoint,
			writeTracesEndpoint,
			cloud,
			awsRegion,
			azureLocation,
			openstackRegion,
			vshereDatacenter string,
		) {
			configMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      GetRegionalClusterConfigMapName(name),
					Namespace: namespace,
					Labels: map[string]string{
						KofClusterRoleLabel: KofRoleRegional,
					},
				},
				Data: map[string]string{
					RegionalClusterNameKey:      name,
					RegionalClusterNamespaceKey: defaultNamespace,
					RegionalIstioRoleKey:        istioRole,
					RegionalClusterCloudKey:     cloud,
					ReadMetricsKey:              readMetricsEndpoint,
					WriteMetricsKey:             writeMetricsEndpoint,
					ReadLogsKey:                 readLogsEndpoint,
					ReadTracesKey:               readTracesEndpoint,
					WriteLogsKey:                writeLogsEndpoint,
					WriteTracesKey:              writeTracesEndpoint,
					AwsRegionKey:                awsRegion,
					AzureLocationKey:            azureLocation,
					OpenstackRegionKey:          openstackRegion,
					VSphereDatacenterKey:        vshereDatacenter,
				},
			}
			Expect(k8sClient.Create(ctx, configMap)).To(Succeed())
		}
		// createClusterDeployment

		createClusterDeployment := func(
			name string,
			namespace string,
			labels map[string]string,
			annotations map[string]string,
			config string,
		) {
			clusterDeployment := &kcmv1beta1.ClusterDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					Namespace:   namespace,
					Labels:      labels,
					Annotations: annotations,
				},
				Spec: kcmv1beta1.ClusterDeploymentSpec{
					Template: "aws-cluster-template",
					Config:   &apiextensionsv1.JSON{Raw: []byte(config)},
				},
			}
			Expect(k8sClient.Create(ctx, clusterDeployment)).To(Succeed())

			clusterDeployment.Status = kcmv1beta1.ClusterDeploymentStatus{
				Conditions: []metav1.Condition{
					{
						Type:               kcmv1beta1.ReadyCondition,
						Status:             metav1.ConditionTrue,
						LastTransitionTime: metav1.Time{Time: time.Now()},
						Reason:             "ClusterReady",
						Message:            "Cluster is ready",
					},
					{
						Type:               kcmv1beta1.CAPIClusterSummaryCondition,
						Status:             metav1.ConditionTrue,
						LastTransitionTime: metav1.Time{Time: time.Now()},
						Reason:             "InfrastructureReady",
						Message:            "Infrastructure is ready",
					},
				},
			}
			Expect(k8sClient.Status().Update(ctx, clusterDeployment)).To(Succeed())
		}

		// createSecret

		createSecret := func(secretName string) {
			kubeconfigSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: k8s.DefaultSystemNamespace,
					Labels:    map[string]string{},
				},
				Data: map[string][]byte{"value": []byte("")},
			}
			Expect(k8sClient.Create(ctx, kubeconfigSecret)).To(Succeed())
		}

		// before each test case

		BeforeEach(func() {
			clusterDeploymentReconciler = &ClusterDeploymentReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			regionalClusterConfigmapReconciler = &RegionalClusterConfigMapReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			By("creating ClusterTemplate in default namespace")
			createClusterTemplate(clusterTemplateName, defaultNamespace)

			By("creating ClusterTemplate in release namespace")
			createClusterTemplate(clusterTemplateName, ReleaseNamespace)

			By("creating regional cluster configMap")
			createRegionalClusterConfigMap(
				regionalClusterDeploymentName,
				defaultNamespace,
				"",
				"https://vmauth.test-aws-ue2.kof.example.com/vm/select/0/prometheus",
				"https://vmauth.test-aws-ue2.kof.example.com/vm/insert/0/prometheus/api/v1/write",
				"https://vmauth.test-aws-ue2.kof.example.com/vls/select/opentelemetry/v1/logs",
				"https://vmauth.test-aws-ue2.kof.example.com/vli/insert/opentelemetry/v1/logs",
				"https://vmauth.test-aws-ue2.kof.example.com/vts/select/jaeger",
				"https://vmauth.test-aws-ue2.kof.example.com/vti/insert/opentelemetry/v1/traces",
				"aws",
				"us-east-2",
				"", "", "",
			)

			By("creating child ClusterDeployment")
			createClusterDeployment(
				childClusterDeploymentName,
				defaultNamespace,
				childClusterDeploymentLabels,
				childClusterDeploymentAnnotations,
				childClusterDeploymentConfig,
			)

			By("creating the fake Secret for the cluster deployment kubeconfig")
			createSecret(secretName)
		})

		It("Should create PromxyServerGroup and GrafanaDatasource for regional cluster", func() {
			promxyServerGroupNamespacedName := types.NamespacedName{
				Name:      regionalClusterDeploymentName + "-metrics",
				Namespace: defaultNamespace,
			}

			grafanaDatasourceNamespacedName := types.NamespacedName{
				Name:      regionalClusterDeploymentName + "-logs",
				Namespace: defaultNamespace,
			}

			By("reconciling regional cluster ConfigMap")
			_, err := regionalClusterConfigmapReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: regionalClusterConfigmapNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			By("reading PromxyServerGroup")
			promxyServerGroup := &kofv1beta1.PromxyServerGroup{}
			err = k8sClient.Get(ctx, promxyServerGroupNamespacedName, promxyServerGroup)
			Expect(err).NotTo(HaveOccurred())
			Expect(promxyServerGroup.Spec.Scheme).To(Equal("https"))
			Expect(promxyServerGroup.Spec.Targets).To(Equal([]string{"vmauth.test-aws-ue2.kof.example.com:443"}))
			Expect(promxyServerGroup.Spec.PathPrefix).To(Equal("/vm/select/0/prometheus"))
			Expect(promxyServerGroup.Spec.HttpClient).To(Equal(kofv1beta1.HTTPClientConfig{
				DialTimeout: defaultDialTimeout,
				TLSConfig: kofv1beta1.TLSConfig{
					InsecureSkipVerify: false,
				},
				BasicAuth: kofv1beta1.BasicAuth{
					CredentialsSecretName: vmuser.BuildSecretName(GetVMUserAdminName(
						regionalClusterConfigmapNamespacedName.Name,
						regionalClusterConfigmapNamespacedName.Namespace,
					)),
					UsernameKey: vmuser.UsernameKey,
					PasswordKey: vmuser.PasswordKey,
				},
			}))

			By("reading GrafanaDatasource")
			grafanaDatasource := &grafanav1beta1.GrafanaDatasource{}
			err = k8sClient.Get(ctx, grafanaDatasourceNamespacedName, grafanaDatasource)
			Expect(err).NotTo(HaveOccurred())
			Expect(grafanaDatasource.Spec.Datasource.URL).To(Equal("https://vmauth.test-aws-ue2.kof.example.com/vls/select/opentelemetry/v1/logs"))
		})

		It("should create ConfigMap for child cluster", func() {
			By("reconciling child ClusterDeployment")
			_, err := clusterDeploymentReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: childClusterDeploymentNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("reading created ConfigMap")
			configMap := &corev1.ConfigMap{}
			err = k8sClient.Get(ctx, childClusterConfigMapNamespacedName, configMap)
			Expect(err).NotTo(HaveOccurred())
			Expect(configMap.Data[RegionalClusterNameKey]).To(Equal(regionalClusterDeploymentName))
			Expect(configMap.Data[ReadMetricsKey]).To(Equal(
				"https://vmauth.test-aws-ue2.kof.example.com/vm/select/0/prometheus",
			))
			Expect(configMap.Data[WriteMetricsKey]).To(Equal(
				"https://vmauth.test-aws-ue2.kof.example.com/vm/insert/0/prometheus/api/v1/write",
			))
			Expect(configMap.Data[WriteLogsKey]).To(Equal(
				"https://vmauth.test-aws-ue2.kof.example.com/vli/insert/opentelemetry/v1/logs",
			))
			Expect(configMap.Data[WriteTracesKey]).To(Equal(
				"https://vmauth.test-aws-ue2.kof.example.com/vti/insert/opentelemetry/v1/traces",
			))
		})

		DescribeTable("should discover regional cluster by AWS region or label", func(
			withLabel bool,
			crossNamespace bool,
			childNamespace string,
			shouldSucceed bool,
		) {

			By("setting CROSS_NAMESPACE environment variable")
			err := os.Setenv("CROSS_NAMESPACE", strconv.FormatBool(crossNamespace))
			Expect(err).NotTo(HaveOccurred())

			const childClusterDeploymentName = "test-child-aws"
			const regionalClusterName = "test-regional-aws"

			childClusterDeploymentNamespacedName := types.NamespacedName{
				Name:      childClusterDeploymentName,
				Namespace: childNamespace,
			}

			childClusterConfigMapNamespacedName := types.NamespacedName{
				Name:      "kof-cluster-config-" + childClusterDeploymentName,
				Namespace: childNamespace,
			}

			regionalClusterConfigmapNamespacedName := types.NamespacedName{
				Name:      GetRegionalClusterConfigMapName(regionalClusterName),
				Namespace: defaultNamespace,
			}

			DeferCleanup(func() {
				err := os.Unsetenv("CROSS_NAMESPACE")
				Expect(err).NotTo(HaveOccurred())

				childClusterDeployment := &kcmv1beta1.ClusterDeployment{}
				if err := k8sClient.Get(ctx, childClusterDeploymentNamespacedName, childClusterDeployment); err == nil {
					By("cleanup child ClusterDeployment")
					Expect(k8sClient.Delete(ctx, childClusterDeployment)).To(Succeed())
				}

				configMap := &corev1.ConfigMap{}
				if err := k8sClient.Get(ctx, childClusterConfigMapNamespacedName, configMap); err == nil {
					By("cleanup child cluster ConfigMap")
					Expect(k8sClient.Delete(ctx, configMap)).To(Succeed())
				}

				regionalClusterConfigMap := &corev1.ConfigMap{}
				if err := k8sClient.Get(ctx, regionalClusterConfigmapNamespacedName, regionalClusterConfigMap); err == nil {
					By("cleanup regional cluster configMap")
					Expect(k8sClient.Delete(ctx, regionalClusterConfigMap)).To(Succeed())
				}
			})

			By("creating child ClusterDeployment without kof-regional-cluster-name label")

			childClusterDeploymentLabels := map[string]string{
				KofClusterRoleLabel:    "child",
				utils.ClusterNameLabel: childClusterDeploymentName,
				// Note no `KofRegionalClusterNameLabel` here, it will be auto-discovered!
			}
			if withLabel {
				childClusterDeploymentLabels[KofRegionalClusterNameLabel] = regionalClusterName
				if crossNamespace {
					childClusterDeploymentLabels[KofRegionalClusterNamespaceLabel] = "default"
				}
			}

			childClusterDeploymentAnnotations := map[string]string{}

			createRegionalClusterConfigMap(
				regionalClusterName,
				regionalClusterConfigmapNamespacedName.Namespace,
				"", "", "",
				"", "", "",
				"", "aws",
				"us-east-2",
				"", "", "",
			)

			createClusterDeployment(
				childClusterDeploymentName,
				childNamespace,
				childClusterDeploymentLabels,
				childClusterDeploymentAnnotations,
				childClusterDeploymentConfig,
			)

			createSecret(childClusterDeploymentName + "-kubeconfig")

			By("reconciling child ClusterDeployment")
			_, err = clusterDeploymentReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: childClusterDeploymentNamespacedName,
			})
			if shouldSucceed {
				Expect(err).NotTo(HaveOccurred())
			} else {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not found"))
			}

			By("reading created ConfigMap")
			configMap := &corev1.ConfigMap{}
			err = k8sClient.Get(ctx, childClusterConfigMapNamespacedName, configMap)
			if shouldSucceed {
				Expect(err).NotTo(HaveOccurred())
				Expect(configMap.Data[RegionalClusterNameKey]).To(Equal(regionalClusterName))
			} else {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not found"))
			}
		},
			EntryDescription("withLabel=%t, crossNamespace=%t, childNamespace=%s, ok=%t"),
			Entry(nil, false, false, defaultNamespace, true),
			Entry(nil, false, false, ReleaseNamespace, false),
			Entry(nil, false, true, defaultNamespace, true),
			Entry(nil, false, true, ReleaseNamespace, true),
			Entry(nil, true, false, defaultNamespace, true),
			Entry(nil, true, false, ReleaseNamespace, false),
			Entry(nil, true, true, defaultNamespace, true),
			Entry(nil, true, true, ReleaseNamespace, true),
		)
	})
})
