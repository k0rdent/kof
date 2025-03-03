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

package k0rdentmirantiscom

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kcmv1alpha1 "github.com/K0rdent/kcm/api/v1alpha1"
)

const DEFAULT_NAMESPACE = "default"

var _ = Describe("ClusterDeployment Controller", func() {
	Context("When reconciling a resource", func() {
		const clusterName = "test-resource"
		const clusterSecretName = "test-promxy-secret"
		const credentialsSecretName = "test-cluster-credentials"

		ctx := context.Background()

		clusterDeploymentNamespacedName := types.NamespacedName{
			Name:      clusterName,
			Namespace: DEFAULT_NAMESPACE,
		}
		clusterDeployment := &kcmv1alpha1.ClusterDeployment{}

		credentialsSecretNamespacesName := types.NamespacedName{
			Name:      credentialsSecretName,
			Namespace: DEFAULT_NAMESPACE,
		}
		credentialsSecret := &coreV1.Secret{}

		remoteSecretNamespacedName := types.NamespacedName{
			Name:      clusterSecretName,
			Namespace: DEFAULT_NAMESPACE,
		}

		var controllerReconciler *ClusterDeploymentReconciler

		BeforeEach(func() {
			controllerReconciler = &ClusterDeploymentReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			By("creating the resource for the Kind ClusterDeployment")
			err := k8sClient.Get(ctx, clusterDeploymentNamespacedName, clusterDeployment)
			if err != nil && errors.IsNotFound(err) {
				resource := &kcmv1alpha1.ClusterDeployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      clusterName,
						Namespace: DEFAULT_NAMESPACE,
						Labels: map[string]string{
							"k0rdent.mirantis.com/istio-role": "child",
						},
					},
					Status: kcmv1alpha1.ClusterDeploymentStatus{
						Conditions: []metav1.Condition{
							{
								Type:               kcmv1alpha1.ReadyCondition,
								Status:             metav1.ConditionTrue,
								LastTransitionTime: metav1.Time{Time: time.Now()},
								Reason:             "ClusterReady",
								Message:            "Cluster is ready",
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}

			By("creating the fake Secret for the cluster deployment kubeconfig")
			err = k8sClient.Get(ctx, credentialsSecretNamespacesName, credentialsSecret)
			if err != nil && errors.IsNotFound(err) {
				resource := &coreV1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      credentialsSecretName,
						Namespace: DEFAULT_NAMESPACE,
						Labels:    map[string]string{},
					},
					StringData: map[string]string{
						"value": "aGVsbG9fd29ybGQ=",
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

			credentialsSecret := &coreV1.Secret{}
			if err := k8sClient.Get(ctx, credentialsSecretNamespacesName, credentialsSecret); err == nil {
				By("Cleanup the Kubeconfig Secret")
				Expect(k8sClient.Delete(ctx, credentialsSecret)).To(Succeed())
			}

			remoteSecret := &coreV1.Secret{}
			if err := k8sClient.Get(ctx, remoteSecretNamespacedName, remoteSecret); err == nil {
				By("Cleanup the Remote Secret")
				Expect(k8sClient.Delete(ctx, remoteSecret)).To(Succeed())
			}

		})

		It("should successfully reconcile the resource if deleted", func() {
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

		It("should successfully reconcile the resource if not ready", func() {
			By("Reconciling the not ready resource")
			clusterDeployment := &kcmv1alpha1.ClusterDeployment{}
			err := k8sClient.Get(ctx, clusterDeploymentNamespacedName, clusterDeployment)
			Expect(err).NotTo(HaveOccurred())

			for _, condition := range clusterDeployment.Status.Conditions {
				if condition.Type == kcmv1alpha1.ReadyCondition {
					condition.Status = metav1.ConditionFalse
					break
				}
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

		It("should successfully reconcile the resource if special label not provided", func() {
			By("Reconciling the resource without labels")
			clusterDeployment := &kcmv1alpha1.ClusterDeployment{}
			err := k8sClient.Get(ctx, clusterDeploymentNamespacedName, clusterDeployment)
			Expect(err).NotTo(HaveOccurred())

			// Delete all labels including special label
			for k := range clusterDeployment.Labels {
				delete(clusterDeployment.Labels, k)
			}

			// Update ClusterDeployment with deleted labels
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
	})
})
