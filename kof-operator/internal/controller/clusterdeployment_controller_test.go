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
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("ClusterDeployment Controller", func() {
	Context("When reconciling a resource", func() {

		const clusterDeploymentName = "test-resource"
		const clusterCertificateName = "kof-istio-test-resource-ca"
		const clusterLabels = `{"clusterLabels": {"k0rdent.mirantis.com/istio-role": "child"} }`

		ctx := context.Background()

		clusterDeploymentNamespacedName := types.NamespacedName{
			Name:      clusterDeploymentName,
			Namespace: "default",
		}

		clusterCertificateNamespacedName := types.NamespacedName{
			Name:      clusterCertificateName,
			Namespace: istioCANamespace,
		}
		clusterDeployment := &kcmv1alpha1.ClusterDeployment{}
		var controllerReconciler *ClusterDeploymentReconciler

		BeforeEach(func() {
			controllerReconciler = &ClusterDeploymentReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
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
						Namespace: "default",
						Labels:    make(map[string]string),
					},
					Spec: kcmv1alpha1.ClusterDeploymentSpec{
						Template: "test-cluster-template",
						Config:   &apiextensionsv1.JSON{Raw: []byte(clusterLabels)},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			clusterDeployment := &kcmv1alpha1.ClusterDeployment{}
			err := k8sClient.Get(ctx, clusterDeploymentNamespacedName, clusterDeployment)
			if err == nil {
				By("Cleanup the ClusterDeployment")
				Expect(k8sClient.Delete(ctx, clusterDeployment)).To(Succeed())
			}

			cert := &cmv1.Certificate{}
			err = k8sClient.Get(ctx, clusterCertificateNamespacedName, cert)
			if err == nil {
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
	})
})
