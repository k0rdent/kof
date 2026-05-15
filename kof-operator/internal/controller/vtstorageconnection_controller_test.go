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

	vmv1 "github.com/VictoriaMetrics/operator/api/operator/v1"
	kofv1beta1 "github.com/k0rdent/kof/kof-operator/api/v1beta1"
	"github.com/k0rdent/kof/kof-operator/internal/models/labels"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	vtClusterName = "vtcluster"
	ns            = "default"
	connName      = "conn"
	addr          = "vtstorage.example.com:8400"
)

func newVTStorageConnectionReconciler() *VTStorageConnectionReconciler {
	return &VTStorageConnectionReconciler{
		Client: k8sClient,
		Scheme: k8sClient.Scheme(),
	}
}

// newVTCluster creates a minimal VTCluster with an initialised Select so the
// controller's map operations on Spec.Select.ExtraArgs never nil-deref.
func newVTCluster() *vmv1.VTCluster {
	return &vmv1.VTCluster{
		ObjectMeta: metav1.ObjectMeta{Name: vtClusterName, Namespace: ns},
		Spec:       vmv1.VTClusterSpec{Select: &vmv1.VTSelect{}},
	}
}

// newConn creates a VTStorageConnection with the given name pointing at the shared VTCluster.
func newConn(name, address string) *kofv1beta1.VTStorageConnection {
	return &kofv1beta1.VTStorageConnection{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec: kofv1beta1.VTStorageConnectionSpec{
			VTClusterRef: kofv1beta1.VTClusterRef{
				Name:      vtClusterName,
				Namespace: ns,
			},
			TargetStorageNode: kofv1beta1.TargetStorageNode{Address: address},
		},
	}
}

// newConnWithFinalizer returns a connection that has already passed the
// first reconcile (finalizer and index label are present).
func newConnWithFinalizer(name, address string) *kofv1beta1.VTStorageConnection {
	c := newConn(name, address)
	c.Finalizers = []string{vtStorageConnectionFinalizer}
	c.Labels = map[string]string{labels.VtClusterNameLabelKey: vtClusterName}
	return c
}

func doReconcile(r *VTStorageConnectionReconciler, name string) error {
	_, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: name, Namespace: ns},
	})
	return err
}

var _ = Describe("VTStorageConnection Controller", func() {
	AfterEach(func() {
		vtscList := &kofv1beta1.VTStorageConnectionList{}
		if err := k8sClient.List(ctx, vtscList); err == nil {
			for i := range vtscList.Items {
				if controllerutil.ContainsFinalizer(&vtscList.Items[i], vtStorageConnectionFinalizer) {
					controllerutil.RemoveFinalizer(&vtscList.Items[i], vtStorageConnectionFinalizer)
					Expect(k8sClient.Update(ctx, &vtscList.Items[i])).To(Succeed())
				}
			}
		}
	})

	BeforeEach(func() {
		Expect(os.Setenv("KOF_VT_CLUSTER_NAME", vtClusterName)).To(Succeed())
	})

	It("adds the finalizer and the vtcluster-name label", func() {
		conn := newConn(connName, addr)
		Expect(k8sClient.Create(ctx, conn)).To(Succeed())

		vtCluster := newVTCluster()
		Expect(k8sClient.Create(ctx, vtCluster)).To(Succeed())

		r := newVTStorageConnectionReconciler()
		Expect(doReconcile(r, connName)).To(Succeed())

		got := &kofv1beta1.VTStorageConnection{}
		Expect(r.Get(ctx, types.NamespacedName{Name: connName, Namespace: ns}, got)).To(Succeed())
		Expect(got.Finalizers).To(ContainElement(vtStorageConnectionFinalizer))
		Expect(got.Labels[labels.VtClusterNameLabelKey]).To(Equal(vtClusterName))
	})

	It("sets storageNode and TLS ExtraArgs", func() {
		conn := newConnWithFinalizer(connName, addr)
		Expect(k8sClient.Create(ctx, conn)).To(Succeed())
		Expect(k8sClient.Create(ctx, newVTCluster())).To(Succeed())

		r := newVTStorageConnectionReconciler()
		Expect(doReconcile(r, connName)).To(Succeed())

		got := &vmv1.VTCluster{}
		Expect(r.Get(ctx, types.NamespacedName{Name: vtClusterName, Namespace: ns}, got)).To(Succeed())
		Expect(got.Spec.Select.ExtraArgs[storageNodeArg]).To(Equal(addr))
		Expect(got.Spec.Select.ExtraArgs[storageNodeTLSArg]).To(Equal("false"))
		Expect(got.Spec.Select.ExtraArgs[storageNodeTLSInsecureSkipVerify]).To(Equal("false"))
	})

	It("preserves ExtraArgs not owned by this controller", func() {
		vtc := newVTCluster()
		vtc.Spec.Select.ExtraArgs = map[string]string{"customFlag": "customValue"}
		Expect(k8sClient.Create(ctx, vtc)).To(Succeed())

		conn := newConnWithFinalizer(connName, addr)
		Expect(k8sClient.Create(ctx, conn)).To(Succeed())

		r := newVTStorageConnectionReconciler()
		Expect(doReconcile(r, connName)).To(Succeed())

		got := &vmv1.VTCluster{}
		Expect(r.Get(ctx, types.NamespacedName{Name: vtClusterName, Namespace: ns}, got)).To(Succeed())
		Expect(got.Spec.Select.ExtraArgs["customFlag"]).To(Equal("customValue"))
		Expect(got.Spec.Select.ExtraArgs[storageNodeArg]).To(Equal(addr))
	})

	It("sets TLS=true and tlsInsecureSkipVerify=true when configured", func() {
		conn := newConnWithFinalizer(connName, addr)
		conn.Spec.TargetStorageNode.TLSConfig = kofv1beta1.TLSStorageConfig{
			Enabled:            true,
			InsecureSkipVerify: true,
		}
		Expect(k8sClient.Create(ctx, conn)).To(Succeed())
		Expect(k8sClient.Create(ctx, newVTCluster())).To(Succeed())

		r := newVTStorageConnectionReconciler()
		Expect(doReconcile(r, connName)).To(Succeed())

		got := &vmv1.VTCluster{}
		Expect(r.Get(ctx, types.NamespacedName{Name: vtClusterName, Namespace: ns}, got)).To(Succeed())
		Expect(got.Spec.Select.ExtraArgs[storageNodeTLSArg]).To(Equal("true"))
		Expect(got.Spec.Select.ExtraArgs[storageNodeTLSInsecureSkipVerify]).To(Equal("true"))
	})

	It("sets secret mount paths and mounts the secret", func() {
		conn := newConnWithFinalizer(connName, addr)
		conn.Spec.TargetStorageNode.Secret = kofv1beta1.SecretRef{
			Name:        "auth-secret",
			UsernameKey: "username",
			PasswordKey: "password",
		}
		Expect(k8sClient.Create(ctx, conn)).To(Succeed())
		Expect(k8sClient.Create(ctx, newVTCluster())).To(Succeed())

		r := newVTStorageConnectionReconciler()
		Expect(doReconcile(r, connName)).To(Succeed())

		got := &vmv1.VTCluster{}
		Expect(r.Get(ctx, types.NamespacedName{Name: vtClusterName, Namespace: ns}, got)).To(Succeed())
		Expect(got.Spec.Select.Secrets).To(ConsistOf("auth-secret"))
		Expect(got.Spec.Select.ExtraArgs[storageNodeUsernameFileArg]).To(Equal("/etc/vm/secrets/auth-secret/username"))
		Expect(got.Spec.Select.ExtraArgs[storageNodePasswordFileArg]).To(Equal("/etc/vm/secrets/auth-secret/password"))
	})

	It("does not set usernameFile or passwordFile when the secret has no keys", func() {
		conn := newConnWithFinalizer(connName, addr)
		conn.Spec.TargetStorageNode.Secret = kofv1beta1.SecretRef{Name: "auth-secret"}
		Expect(k8sClient.Create(ctx, conn)).To(Succeed())
		Expect(k8sClient.Create(ctx, newVTCluster())).To(Succeed())

		r := newVTStorageConnectionReconciler()
		Expect(doReconcile(r, connName)).To(Succeed())

		got := &vmv1.VTCluster{}
		Expect(r.Get(ctx, types.NamespacedName{Name: vtClusterName, Namespace: ns}, got)).To(Succeed())
		Expect(got.Spec.Select.ExtraArgs).NotTo(HaveKey(storageNodeUsernameFileArg))
		Expect(got.Spec.Select.ExtraArgs).NotTo(HaveKey(storageNodePasswordFileArg))
	})

	Describe("multiple connections", func() {
		// The API server returns List results in lexicographic order by name,
		// so "conn-a" (index 0) is always before "conn-b" (index 1).
		It("comma-joins addresses from all active connections", func() {
			Expect(k8sClient.Create(ctx, newConnWithFinalizer("conn-a", "storage-a:8400"))).To(Succeed())
			Expect(k8sClient.Create(ctx, newConnWithFinalizer("conn-b", "storage-b:8400"))).To(Succeed())
			Expect(k8sClient.Create(ctx, newVTCluster())).To(Succeed())

			r := newVTStorageConnectionReconciler()
			Expect(doReconcile(r, "conn-a")).To(Succeed())

			got := &vmv1.VTCluster{}
			Expect(r.Get(ctx, types.NamespacedName{Name: vtClusterName, Namespace: ns}, got)).To(Succeed())
			Expect(got.Spec.Select.ExtraArgs[storageNodeArg]).To(ContainSubstring("storage-a:8400"))
			Expect(got.Spec.Select.ExtraArgs[storageNodeArg]).To(ContainSubstring("storage-b:8400"))
		})

		It("preserves positional correspondence with empty placeholders for missing credentials", func() {
			// conn-a has no secret → empty placeholder at position 0
			// conn-b has a secret  → path at position 1
			Expect(k8sClient.Create(ctx, newConnWithFinalizer("conn-a", "storage-a:8400"))).To(Succeed())

			connB := newConnWithFinalizer("conn-b", "storage-b:8400")
			connB.Spec.TargetStorageNode.Secret = kofv1beta1.SecretRef{
				Name:        "sec-b",
				UsernameKey: "user",
				PasswordKey: "pass",
			}
			Expect(k8sClient.Create(ctx, connB)).To(Succeed())
			Expect(k8sClient.Create(ctx, newVTCluster())).To(Succeed())

			r := newVTStorageConnectionReconciler()
			Expect(doReconcile(r, "conn-a")).To(Succeed())

			got := &vmv1.VTCluster{}
			Expect(r.Get(ctx, types.NamespacedName{Name: vtClusterName, Namespace: ns}, got)).To(Succeed())
			Expect(got.Spec.Select.ExtraArgs[storageNodeUsernameFileArg]).To(Equal(",/etc/vm/secrets/sec-b/user"))
			Expect(got.Spec.Select.ExtraArgs[storageNodePasswordFileArg]).To(Equal(",/etc/vm/secrets/sec-b/pass"))
		})
	})

	It("removes the storage node from VTCluster and drops the finalizer", func() {
		r := newVTStorageConnectionReconciler()

		vtc := newVTCluster()
		Expect(k8sClient.Create(ctx, vtc)).To(Succeed())

		conn := newConnWithFinalizer(connName, addr)
		Expect(k8sClient.Create(ctx, conn)).To(Succeed())
		Expect(doReconcile(r, connName)).To(Succeed())

		vtcBefore := &vmv1.VTCluster{}
		Expect(r.Get(ctx, types.NamespacedName{Name: vtClusterName, Namespace: ns}, vtcBefore)).To(Succeed())
		Expect(vtcBefore.Spec.Select.ExtraArgs[storageNodeArg]).To(Equal(addr))

		Expect(k8sClient.Delete(ctx, conn)).To(Succeed())
		Expect(doReconcile(r, connName)).To(Succeed())

		gotVTC := &vmv1.VTCluster{}
		Expect(r.Get(ctx, types.NamespacedName{Name: vtClusterName, Namespace: ns}, gotVTC)).To(Succeed())
		Expect(gotVTC.Spec.Select.ExtraArgs).NotTo(HaveKey(storageNodeArg))

		// After the finalizer is removed the API server deletes the object
		// (DeletionTimestamp was already set), so Get returns NotFound.
		gotConn := &kofv1beta1.VTStorageConnection{}
		err := r.Get(ctx, types.NamespacedName{Name: connName, Namespace: ns}, gotConn)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("not found"))
	})

	It("drops the finalizer immediately when the VTCluster is already gone", func() {
		conn := newConnWithFinalizer(connName, addr)
		Expect(k8sClient.Create(ctx, conn)).To(Succeed())

		r := newVTStorageConnectionReconciler()
		Expect(k8sClient.Delete(ctx, conn)).To(Succeed())
		Expect(doReconcile(r, connName)).To(Succeed())

		// Object is fully deleted after the finalizer is removed.
		gotConn := &kofv1beta1.VTStorageConnection{}
		err := r.Get(ctx, types.NamespacedName{Name: connName, Namespace: ns}, gotConn)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("not found"))
	})

	It("returns an error when the referenced VTCluster does not exist", func() {
		conn := newConnWithFinalizer(connName, addr)
		Expect(k8sClient.Create(ctx, conn)).To(Succeed())

		r := newVTStorageConnectionReconciler()
		err := doReconcile(r, connName)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("not found"))
	})
})
