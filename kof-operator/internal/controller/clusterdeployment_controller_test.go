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

	kcmv1alpha1 "github.com/K0rdent/kcm/api/v1alpha1"
	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	coreV1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const DEFAULT_NAMESPACE = "default"

var _ = Describe("ClusterDeployment Controller", func() {
	Context("When reconciling a resource", func() {

		const clusterDeploymentName = "test-resource"
		const clusterCertificateName = "kof-istio-test-resource-ca"
		const clusterLabels = `{"clusterLabels": {"k0rdent.mirantis.com/istio-role": "child"} }`
		const SecretName = "test-resource-kubeconfig"

		ctx := context.Background()

		clusterDeploymentNamespacedName := types.NamespacedName{
			Name:      clusterDeploymentName,
			Namespace: "default",
		}

		clusterCertificateNamespacedName := types.NamespacedName{
			Name:      clusterCertificateName,
			Namespace: istioCANamespace,
		}

		kubeconfigSecretNamespacesName := types.NamespacedName{
			Name:      SecretName,
			Namespace: DEFAULT_NAMESPACE,
		}

		remoteSecretNamespacedName := types.NamespacedName{
			Name:      clusterName,
			Namespace: DEFAULT_NAMESPACE,
		}

		clusterDeployment := &kcmv1alpha1.ClusterDeployment{}
		kubeconfigSecret := &coreV1.Secret{}
		var controllerReconciler *ClusterDeploymentReconciler

		BeforeEach(func() {
			controllerReconciler = &ClusterDeploymentReconciler{
				Client:              k8sClient,
				Scheme:              k8sClient.Scheme(),
				RemoteSecretManager: remotesecret.NewFakeManager(k8sClient),
			}

			By(fmt.Sprintf("creating the %s namespace", istioCANamespace))
			certNamespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: istioCANamespace,
				},
			}
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      istioCANamespace,
				Namespace: istioCANamespace,
			}, certNamespace)
			if err != nil && errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, certNamespace)).To(Succeed())
			}

			By("creating the custom resource for the Kind ClusterDeployment")
			err = k8sClient.Get(ctx, clusterDeploymentNamespacedName, clusterDeployment)
			if err != nil && errors.IsNotFound(err) {
				resource := &kcmv1alpha1.ClusterDeployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      clusterDeploymentName,
						Namespace: DEFAULT_NAMESPACE,
						Labels: map[string]string{
							"k0rdent.mirantis.com/istio-role": "child",
						},
					},
					Spec: kcmv1alpha1.ClusterDeploymentSpec{
						Template: "test-cluster-template",
						Config:   &apiextensionsv1.JSON{Raw: []byte(clusterLabels)},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())

				resource.Status = kcmv1alpha1.ClusterDeploymentStatus{
					Conditions: []metav1.Condition{
						{
							Type:               kcmv1alpha1.ReadyCondition,
							Status:             metav1.ConditionTrue,
							LastTransitionTime: metav1.Time{Time: time.Now()},
							Reason:             "ClusterReady",
							Message:            "Cluster is ready",
						},
					},
				}
				k8sClient.Status().Update(ctx, resource)
			}

			By("creating the fake Secret for the cluster deployment kubeconfig")
			err = k8sClient.Get(ctx, kubeconfigSecretNamespacesName, kubeconfigSecret)
			if err != nil && errors.IsNotFound(err) {
				resource := &coreV1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      SecretName,
						Namespace: DEFAULT_NAMESPACE,
						Labels:    map[string]string{},
					},
					StringData: map[string]string{
						"value": "hello-world",
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			cd := &kcmv1alpha1.ClusterDeployment{}
			if err := k8sClient.Get(ctx, clusterDeploymentNamespacedName, cd); err == nil {
				By("Cleanup the ClusterDeployment")
				Expect(k8sClient.Delete(ctx, cd)).To(Succeed())
			}

			kubeconfigSecret := &coreV1.Secret{}
			if err := k8sClient.Get(ctx, kubeconfigSecretNamespacesName, kubeconfigSecret); err == nil {
				By("Cleanup the Kubeconfig Secret")
				Expect(k8sClient.Delete(ctx, credentialsSecret)).To(Succeed())
			}

			remoteSecret := &coreV1.Secret{}
			if err := k8sClient.Get(ctx, remoteSecretNamespacedName, remoteSecret); err == nil {
				By("Cleanup the Remote Secret")
				Expect(k8sClient.Delete(ctx, remoteSecret)).To(Succeed())
			}

			cert := &cmv1.Certificate{}
			if err = k8sClient.Get(ctx, clusterCertificateNamespacedName, cert); err == nil {
				By("Cleanup the Certificate")
				Expect(k8sClient.Delete(ctx, cert)).To(Succeed())
			}
		})

		It("should successfully reconcile the resource", func() {

			By("Reconciling the created resource")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: clusterDeploymentNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			cert := &cmv1.Certificate{}
			err = k8sClient.Get(ctx, clusterCertificateNamespacedName, cert)
			Expect(err).NotTo(HaveOccurred())
			Expect(cert.Spec.CommonName).To(Equal(fmt.Sprintf("%s CA", clusterDeploymentName)))
		})

		It("should successfully reconcile the resource when deleted", func() {
			By("Reconciling the deleted resource")
			clusterDeployment := &kcmv1alpha1.ClusterDeployment{}
			err := k8sClient.Get(ctx, clusterDeploymentNamespacedName, clusterDeployment)
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sClient.Delete(ctx, clusterDeployment)).To(Succeed())

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: clusterDeploymentNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			secret := &coreV1.Secret{}
			err = k8sClient.Get(ctx, remoteSecretNamespacedName, secret)
			Expect(errors.IsNotFound(err)).To(BeTrue())
		})

		It("should successfully reconcile the resource when not ready", func() {
			By("Reconciling the not ready resource")
			clusterDeployment := &kcmv1alpha1.ClusterDeployment{}
			err := k8sClient.Get(ctx, clusterDeploymentNamespacedName, clusterDeployment)
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
				NamespacedName: clusterDeploymentNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			secret := &coreV1.Secret{}
			err = k8sClient.Get(ctx, remoteSecretNamespacedName, secret)
			Expect(errors.IsNotFound(err)).To(BeTrue())
		})

		It("should successfully reconcile the resource if special label not provided", func() {
			By("Reconciling the resource without labels")
			clusterDeployment := &kcmv1alpha1.ClusterDeployment{}
			err := k8sClient.Get(ctx, clusterDeploymentNamespacedName, clusterDeployment)
			Expect(err).NotTo(HaveOccurred())

			for k := range clusterDeployment.Labels {
				delete(clusterDeployment.Labels, k)
			}

			err = k8sClient.Update(ctx, clusterDeployment)
			Expect(err).NotTo(HaveOccurred())

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: clusterDeploymentNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			secret := &coreV1.Secret{}
			err = k8sClient.Get(ctx, remoteSecretNamespacedName, secret)
			Expect(errors.IsNotFound(err)).To(BeTrue())
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			clusterDeployment := &kcmv1alpha1.ClusterDeployment{}
			err := k8sClient.Get(ctx, clusterDeploymentNamespacedName, clusterDeployment)
			Expect(err).NotTo(HaveOccurred())

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: clusterDeploymentNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			secret := &coreV1.Secret{}
			err = k8sClient.Get(ctx, remoteSecretNamespacedName, secret)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
