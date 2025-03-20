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
	"fmt"
	"strconv"
	"time"

	kcmv1alpha1 "github.com/K0rdent/kcm/api/v1alpha1"
	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	istio "github.com/k0rdent/kof/kof-operator/internal/controller/istio"
	"github.com/k0rdent/kof/kof-operator/internal/controller/istio/cert"
	remotesecret "github.com/k0rdent/kof/kof-operator/internal/controller/istio/remote-secret"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	sveltosv1beta1 "github.com/projectsveltos/addon-controller/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clusterapiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const DEFAULT_NAMESPACE = "default"

var _ = Describe("ClusterDeployment Controller", func() {
	Context("When reconciling a resource", func() {
		ctx := context.Background()
		var controllerReconciler *ClusterDeploymentReconciler

		// regional ClusterDeployment

		const regionalClusterDeploymentName = "test-regional"

		regionalClusterDeploymentNamespacedName := types.NamespacedName{
			Name:      regionalClusterDeploymentName,
			Namespace: DEFAULT_NAMESPACE,
		}

		regionalClusterDeploymentLabels := map[string]string{
			KofClusterRoleLabel:    "regional",
			KofRegionalDomainLabel: "test-aws-ue2.kof.example.com",
		}

		// child ClusterDeployment

		const childClusterDeploymentName = "test-child"

		childClusterDeploymentNamespacedName := types.NamespacedName{
			Name:      childClusterDeploymentName,
			Namespace: DEFAULT_NAMESPACE,
		}

		childClusterDeploymentLabels := map[string]string{
			IstioRoleLabel:              "child",
			KofClusterRoleLabel:         "child",
			KofRegionalClusterNameLabel: "test-regional",
		}

		// child cluster ConfigMap

		childClusterConfigMapNamespacedName := types.NamespacedName{
			Name:      "kof-cluster-config-test-child", // prefix + child cluster name
			Namespace: DEFAULT_NAMESPACE,
		}

		// istio child

		const clusterCertificateName = "kof-istio-test-child-ca"

		clusterCertificateNamespacedName := types.NamespacedName{
			Name:      clusterCertificateName,
			Namespace: istio.IstioSystemNamespace,
		}

		const secretName = "test-child-kubeconfig"

		kubeconfigSecretNamespacedName := types.NamespacedName{
			Name:      secretName,
			Namespace: DEFAULT_NAMESPACE,
		}

		remoteSecretNamespacedName := types.NamespacedName{
			Name:      remotesecret.RemoteSecretNameFromClusterName(childClusterDeploymentName),
			Namespace: istio.IstioSystemNamespace,
		}

		profileDeploymentName := types.NamespacedName{
			Name:      remotesecret.RemoteSecretNameFromClusterName(regionalClusterDeploymentName),
			Namespace: DEFAULT_NAMESPACE,
		}

		// createClusterDeployment

		createClusterDeployment := func(
			name string,
			labels map[string]string,
		) *kcmv1alpha1.ClusterDeployment {
			clusterDeployment := &kcmv1alpha1.ClusterDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: DEFAULT_NAMESPACE,
					Labels:    labels,
				},
				Spec: kcmv1alpha1.ClusterDeploymentSpec{
					Template: "aws-test-cluster-template",
					Config:   &apiextensionsv1.JSON{Raw: []byte(`{"region": "us-east-2"}`)},
				},
			}
			Expect(k8sClient.Create(ctx, clusterDeployment)).To(Succeed())

			clusterDeployment.Status = kcmv1alpha1.ClusterDeploymentStatus{
				Conditions: []metav1.Condition{
					{
						Type:               kcmv1alpha1.ReadyCondition,
						Status:             metav1.ConditionTrue,
						LastTransitionTime: metav1.Time{Time: time.Now()},
						Reason:             "ClusterReady",
						Message:            "Cluster is ready",
					},
					{
						Type:               string(clusterapiv1beta1.InfrastructureReadyCondition),
						Status:             metav1.ConditionTrue,
						LastTransitionTime: metav1.Time{Time: time.Now()},
						Reason:             "InfrastructureReady",
						Message:            "Infrastructure is ready",
					},
				},
			}
			Expect(k8sClient.Status().Update(ctx, clusterDeployment)).To(Succeed())

			return clusterDeployment
		}

		// before each test case

		BeforeEach(func() {
			controllerReconciler = &ClusterDeploymentReconciler{
				Client:              k8sClient,
				Scheme:              k8sClient.Scheme(),
				RemoteSecretManager: remotesecret.NewFakeManager(k8sClient),
				IstioCertManager:    cert.New(k8sClient),
			}

			By(fmt.Sprintf("creating the %s namespace", istio.IstioSystemNamespace))
			certNamespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: istio.IstioSystemNamespace,
				},
			}
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      istio.IstioSystemNamespace,
				Namespace: istio.IstioSystemNamespace,
			}, certNamespace)
			if err != nil && errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, certNamespace)).To(Succeed())
			}

			By("creating regional ClusterDeployment")
			createClusterDeployment(regionalClusterDeploymentName, regionalClusterDeploymentLabels)

			By("creating child ClusterDeployment")
			createClusterDeployment(childClusterDeploymentName, childClusterDeploymentLabels)

			By("creating the fake Secret for the cluster deployment kubeconfig")
			kubeconfigSecret := &corev1.Secret{}
			err = k8sClient.Get(ctx, kubeconfigSecretNamespacedName, kubeconfigSecret)
			if err != nil && errors.IsNotFound(err) {
				resource := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName,
						Namespace: DEFAULT_NAMESPACE,
						Labels:    map[string]string{},
					},
					Data: map[string][]byte{
						"value": []byte(""),
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		// after each test case

		AfterEach(func() {
			cd := &kcmv1alpha1.ClusterDeployment{}

			if err := k8sClient.Get(ctx, regionalClusterDeploymentNamespacedName, cd); err == nil {
				By("Cleanup regional ClusterDeployment")
				Expect(k8sClient.Delete(ctx, cd)).To(Succeed())
			}

			if err := k8sClient.Get(ctx, childClusterDeploymentNamespacedName, cd); err == nil {
				By("Cleanup child ClusterDeployment")
				Expect(k8sClient.Delete(ctx, cd)).To(Succeed())
			}

			configMap := &corev1.ConfigMap{}
			if err := k8sClient.Get(ctx, childClusterConfigMapNamespacedName, configMap); err == nil {
				By("Cleanup child cluster ConfigMap")
				Expect(k8sClient.Delete(ctx, configMap)).To(Succeed())
			}

			kubeconfigSecret := &corev1.Secret{}
			if err := k8sClient.Get(ctx, kubeconfigSecretNamespacedName, kubeconfigSecret); err == nil {
				By("Cleanup the Kubeconfig Secret")
				Expect(k8sClient.Delete(ctx, kubeconfigSecret)).To(Succeed())
			}

			remoteSecret := &corev1.Secret{}
			if err := k8sClient.Get(ctx, remoteSecretNamespacedName, remoteSecret); err == nil {
				By("Cleanup the Remote Secret")
				Expect(k8sClient.Delete(ctx, remoteSecret)).To(Succeed())
			}

			cert := &cmv1.Certificate{}
			if err := k8sClient.Get(ctx, clusterCertificateNamespacedName, cert); err == nil {
				By("Cleanup the Certificate")
				Expect(k8sClient.Delete(ctx, cert)).To(Succeed())
			}
		})

		// test cases

		It("should successfully reconcile the CA resource", func() {
			By("Reconciling the created resource")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: childClusterDeploymentNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			cert := &cmv1.Certificate{}
			err = k8sClient.Get(ctx, clusterCertificateNamespacedName, cert)
			Expect(err).NotTo(HaveOccurred())
			Expect(cert.Spec.CommonName).To(Equal(fmt.Sprintf("%s CA", childClusterDeploymentName)))
		})

		It("should successfully reconcile the resource when deleted", func() {
			By("Reconciling the deleted resource")
			clusterDeployment := &kcmv1alpha1.ClusterDeployment{}
			err := k8sClient.Get(ctx, childClusterDeploymentNamespacedName, clusterDeployment)
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sClient.Delete(ctx, clusterDeployment)).To(Succeed())

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: childClusterDeploymentNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			secret := &corev1.Secret{}
			err = k8sClient.Get(ctx, remoteSecretNamespacedName, secret)
			Expect(errors.IsNotFound(err)).To(BeTrue())
		})

		It("should successfully reconcile the resource when not ready", func() {
			By("Reconciling the not ready resource")
			clusterDeployment := &kcmv1alpha1.ClusterDeployment{}
			err := k8sClient.Get(ctx, childClusterDeploymentNamespacedName, clusterDeployment)
			Expect(err).NotTo(HaveOccurred())

			for i := range clusterDeployment.Status.Conditions {
				if clusterDeployment.Status.Conditions[i].Type == kcmv1alpha1.ReadyCondition {
					clusterDeployment.Status.Conditions[i].Status = metav1.ConditionFalse
					break
				}
			}

			err = k8sClient.Status().Update(ctx, clusterDeployment)
			Expect(err).NotTo(HaveOccurred())

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: childClusterDeploymentNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			secret := &corev1.Secret{}
			err = k8sClient.Get(ctx, remoteSecretNamespacedName, secret)
			Expect(errors.IsNotFound(err)).To(BeTrue())
		})

		It("should successfully reconcile the resource if special label not provided", func() {
			By("Reconciling the resource without labels")
			clusterDeployment := &kcmv1alpha1.ClusterDeployment{}
			err := k8sClient.Get(ctx, childClusterDeploymentNamespacedName, clusterDeployment)
			Expect(err).NotTo(HaveOccurred())

			clusterDeployment.Labels = map[string]string{}

			err = k8sClient.Update(ctx, clusterDeployment)
			Expect(err).NotTo(HaveOccurred())

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: childClusterDeploymentNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			secret := &corev1.Secret{}
			err = k8sClient.Get(ctx, remoteSecretNamespacedName, secret)
			Expect(errors.IsNotFound(err)).To(BeTrue())
		})

		It("should successfully reconcile when remote secret already exists", func() {
			By("Reconciling the resource with existed remote secret")
			clusterDeployment := &kcmv1alpha1.ClusterDeployment{}
			err := k8sClient.Get(ctx, childClusterDeploymentNamespacedName, clusterDeployment)
			Expect(err).NotTo(HaveOccurred())

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: childClusterDeploymentNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: childClusterDeploymentNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			secret := &corev1.Secret{}
			err = k8sClient.Get(ctx, remoteSecretNamespacedName, secret)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should successfully reconcile after creating and deleting resource", func() {
			By("Verifying resource reconciliation after creation and deletion")
			cd := &kcmv1alpha1.ClusterDeployment{}
			err := k8sClient.Get(ctx, childClusterDeploymentNamespacedName, cd)
			Expect(err).NotTo(HaveOccurred())
			cdUID := cd.GetUID()

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: childClusterDeploymentNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Delete(ctx, cd)).To(Succeed())

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: childClusterDeploymentNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			secret := &corev1.Secret{}
			err = k8sClient.Get(ctx, remoteSecretNamespacedName, secret)
			Expect(errors.IsNotFound(err)).To(BeTrue())

			configMap := &corev1.ConfigMap{}
			err = k8sClient.Get(ctx, childClusterConfigMapNamespacedName, configMap)
			Expect(err).NotTo(HaveOccurred())
			// There is no garbage collector in the `envtest`,
			// so we should test that `OwnerReference` is set correctly,
			// and assume that Kubernetes garbage collection works:
			// https://github.com/kubernetes-sigs/controller-runtime/issues/626#issuecomment-538529534
			owner := configMap.OwnerReferences[0]
			Expect(owner.APIVersion).To(Equal("k0rdent.mirantis.com/v1alpha1"))
			Expect(owner.Kind).To(Equal("ClusterDeployment"))
			Expect(owner.Name).To(Equal(childClusterDeploymentName))
			Expect(owner.UID).To(Equal(cdUID))

			cert := &cmv1.Certificate{}
			err = k8sClient.Get(ctx, clusterCertificateNamespacedName, cert)
			Expect(err).NotTo(HaveOccurred())
			owner = configMap.OwnerReferences[0]
			Expect(owner.APIVersion).To(Equal("k0rdent.mirantis.com/v1alpha1"))
			Expect(owner.Kind).To(Equal("ClusterDeployment"))
			Expect(owner.Name).To(Equal(childClusterDeploymentName))
			Expect(owner.UID).To(Equal(cdUID))
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			clusterDeployment := &kcmv1alpha1.ClusterDeployment{}
			err := k8sClient.Get(ctx, childClusterDeploymentNamespacedName, clusterDeployment)
			Expect(err).NotTo(HaveOccurred())

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: childClusterDeploymentNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			remoteSecret := &corev1.Secret{}
			err = k8sClient.Get(ctx, remoteSecretNamespacedName, remoteSecret)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create and update ConfigMap for child cluster", func() {
			By("reconciling child ClusterDeployment")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: childClusterDeploymentNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("reading child ClusterDeployment")
			clusterDeployment := &kcmv1alpha1.ClusterDeployment{}
			err = k8sClient.Get(ctx, childClusterDeploymentNamespacedName, clusterDeployment)
			Expect(err).NotTo(HaveOccurred())
			initialClusterDeploymentGeneration := clusterDeployment.Generation

			By("reading created ConfigMap")
			configMap := &corev1.ConfigMap{}
			err = k8sClient.Get(ctx, childClusterConfigMapNamespacedName, configMap)
			Expect(err).NotTo(HaveOccurred())
			Expect(configMap.Data["regional_cluster_name"]).To(Equal("test-regional"))
			Expect(configMap.Data["regional_domain"]).To(Equal("test-aws-ue2.kof.example.com"))
			configMapCDGeneration, err := strconv.Atoi(configMap.Data["cluster_deployment_generation"])
			Expect(err).NotTo(HaveOccurred())
			Expect(configMapCDGeneration).To(BeNumerically("==", initialClusterDeploymentGeneration))
			initialConfigMapResourceVersion := configMap.ResourceVersion

			// status update

			By("updating the status of child ClusterDeployment")
			clusterDeployment.Status.KubernetesVersion = "v1.32.0"
			err = k8sClient.Update(ctx, clusterDeployment)
			Expect(err).NotTo(HaveOccurred())

			By("reconciling child ClusterDeployment after status update")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: childClusterDeploymentNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("reading unchanged ConfigMap")
			err = k8sClient.Get(ctx, childClusterConfigMapNamespacedName, configMap)
			Expect(err).NotTo(HaveOccurred())
			configMapCDGeneration, err = strconv.Atoi(configMap.Data["cluster_deployment_generation"])
			Expect(err).NotTo(HaveOccurred())
			Expect(configMapCDGeneration).To(BeNumerically("==", initialClusterDeploymentGeneration))
			Expect(configMap.ResourceVersion).To(Equal(initialConfigMapResourceVersion))

			// spec update

			By("updating the spec of child ClusterDeployment")
			clusterDeployment.Spec.Template += "-updated"
			err = k8sClient.Update(ctx, clusterDeployment)
			Expect(err).NotTo(HaveOccurred())

			By("reconciling child ClusterDeployment after spec update")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: childClusterDeploymentNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("reading updated ConfigMap")
			err = k8sClient.Get(ctx, childClusterConfigMapNamespacedName, configMap)
			Expect(err).NotTo(HaveOccurred())
			configMapCDGeneration, err = strconv.Atoi(configMap.Data["cluster_deployment_generation"])
			Expect(err).NotTo(HaveOccurred())
			Expect(configMapCDGeneration).To(BeNumerically(">", initialClusterDeploymentGeneration))
		})

		It("should discover regional cluster by AWS region", func() {
			By("creating child ClusterDeployment without kof-regional-cluster-name label")
			const childClusterDeploymentName = "test-child-aws"

			childClusterDeploymentNamespacedName := types.NamespacedName{
				Name:      childClusterDeploymentName,
				Namespace: DEFAULT_NAMESPACE,
			}

			childClusterConfigMapNamespacedName := types.NamespacedName{
				Name:      "kof-cluster-config-" + childClusterDeploymentName,
				Namespace: DEFAULT_NAMESPACE,
			}

			childClusterDeploymentLabels := map[string]string{
				KofClusterRoleLabel: "child",
				// Note no `KofRegionalClusterNameLabel` here, it will be auto-discovered!
			}

			createClusterDeployment(childClusterDeploymentName, childClusterDeploymentLabels)

			DeferCleanup(func() {
				childClusterDeployment := &kcmv1alpha1.ClusterDeployment{}
				if err := k8sClient.Get(ctx, childClusterDeploymentNamespacedName, childClusterDeployment); err == nil {
					By("cleanup child ClusterDeployment")
					Expect(k8sClient.Delete(ctx, childClusterDeployment)).To(Succeed())
				}

				configMap := &corev1.ConfigMap{}
				if err := k8sClient.Get(ctx, childClusterConfigMapNamespacedName, configMap); err == nil {
					By("cleanup child cluster ConfigMap")
					Expect(k8sClient.Delete(ctx, configMap)).To(Succeed())
				}
			})

			By("reconciling child ClusterDeployment")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: childClusterDeploymentNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("reading created ConfigMap")
			configMap := &corev1.ConfigMap{}
			err = k8sClient.Get(ctx, childClusterConfigMapNamespacedName, configMap)
			Expect(err).NotTo(HaveOccurred())
			Expect(configMap.Data["regional_cluster_name"]).To(Equal("test-regional"))
			Expect(configMap.Data["regional_domain"]).To(Equal("test-aws-ue2.kof.example.com"))
		})

		It("should create profile", func() {
			By("reading child ClusterDeployment")
			clusterDeployment := &kcmv1alpha1.ClusterDeployment{}
			err := k8sClient.Get(ctx, childClusterDeploymentNamespacedName, clusterDeployment)
			Expect(err).NotTo(HaveOccurred())

			By("reconciling child ClusterDeployment")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: childClusterDeploymentNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("reading profile")
			profile := &sveltosv1beta1.Profile{}
			err = k8sClient.Get(ctx, profileDeploymentName, profile)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
