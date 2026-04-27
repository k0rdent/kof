package vmuser

import (
	"context"
	"testing"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"github.com/k0rdent/kof/kof-operator/internal/controller/record"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	addoncontrollerv1beta1 "github.com/projectsveltos/addon-controller/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	k8sevents "k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestVMUserManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "VMUser Manager Suite")
}

var _ = BeforeSuite(func() {
	record.DefaultRecorder = new(k8sevents.FakeRecorder)
})

var _ = Describe("VMUser Manager - MCS Update Tests", func() {
	var (
		ctx        context.Context
		fakeClient client.Client
		manager    *Manager
	)

	BeforeEach(func() {
		// Setup scheme
		s := scheme.Scheme
		err := kcmv1beta1.AddToScheme(s)
		Expect(err).NotTo(HaveOccurred())
		err = addoncontrollerv1beta1.AddToScheme(s)
		Expect(err).NotTo(HaveOccurred())
		err = corev1.AddToScheme(s)
		Expect(err).NotTo(HaveOccurred())

		// Create fake client with scheme
		fakeClient = fake.NewClientBuilder().
			WithScheme(s).
			Build()

		manager = NewManager(fakeClient)
		ctx = context.Background()
	})

	It("should create MCS if it does not exist", func() {
		opts := &CreateOptions{
			Name:      "test-cluster",
			Namespace: "test-ns",
			MCSConfig: &MCSConfig{
				ClusterSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"role": "regional",
					},
				},
			},
		}

		err := manager.createOrUpdatePropagationMCS(ctx, opts)
		Expect(err).NotTo(HaveOccurred())

		// Verify MCS was created
		mcs := &kcmv1beta1.MultiClusterService{}
		err = fakeClient.Get(ctx, client.ObjectKey{Name: BuildMCSName(opts.Name)}, mcs)
		Expect(err).NotTo(HaveOccurred())
		Expect(mcs.Spec.ClusterSelector.MatchLabels).To(Equal(opts.MCSConfig.ClusterSelector.MatchLabels))
	})

	It("should not update MCS if spec has not changed", func() {
		opts := &CreateOptions{
			Name:      "test-cluster",
			Namespace: "test-ns",
			MCSConfig: &MCSConfig{
				ClusterSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"role": "regional",
					},
				},
			},
		}

		// Create initial MCS
		err := manager.createOrUpdatePropagationMCS(ctx, opts)
		Expect(err).NotTo(HaveOccurred())

		// Get the created MCS
		mcs := &kcmv1beta1.MultiClusterService{}
		err = fakeClient.Get(ctx, client.ObjectKey{Name: BuildMCSName(opts.Name)}, mcs)
		Expect(err).NotTo(HaveOccurred())
		originalResourceVersion := mcs.ResourceVersion

		// Call createOrUpdatePropagationMCS again with same opts
		err = manager.createOrUpdatePropagationMCS(ctx, opts)
		Expect(err).NotTo(HaveOccurred())

		// Verify MCS was not updated (resource version should remain the same)
		mcs = &kcmv1beta1.MultiClusterService{}
		err = fakeClient.Get(ctx, client.ObjectKey{Name: BuildMCSName(opts.Name)}, mcs)
		Expect(err).NotTo(HaveOccurred())
		Expect(mcs.ResourceVersion).To(Equal(originalResourceVersion), "MCS should not be updated when spec has not changed")
	})

	It("should update MCS when the spec changes", func() {
		mcsName := BuildMCSName("test-cluster")

		// Create MCS with initial spec
		initialMCS := &kcmv1beta1.MultiClusterService{
			ObjectMeta: metav1.ObjectMeta{
				Name: mcsName,
			},
			Spec: kcmv1beta1.MultiClusterServiceSpec{
				ClusterSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"role": "regional",
					},
				},
				ServiceSpec: kcmv1beta1.ServiceSpec{
					Services: []kcmv1beta1.Service{
						{
							Name:      "service1",
							Namespace: "ns1",
							Template:  "template1",
						},
					},
					TemplateResourceRefs: []addoncontrollerv1beta1.TemplateResourceRef{
						{
							Identifier: "resource1",
							Resource: corev1.ObjectReference{
								Name:      "resource1",
								Namespace: "ns1",
							},
						},
					},
				},
			},
		}

		// Create the MCS in the fake client
		err := fakeClient.Create(ctx, initialMCS)
		Expect(err).NotTo(HaveOccurred())

		originalResourceVersion := initialMCS.ResourceVersion

		// Now call manager with completely different spec
		opts := &CreateOptions{
			Name:      "test-cluster",
			Namespace: "updated-ns",
			MCSConfig: &MCSConfig{
				ClusterSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"role": "child",
					},
				},
			},
		}

		err = manager.createOrUpdatePropagationMCS(ctx, opts)
		Expect(err).NotTo(HaveOccurred())

		// Verify MCS was updated
		updatedMCS := &kcmv1beta1.MultiClusterService{}
		err = fakeClient.Get(ctx, client.ObjectKey{Name: mcsName}, updatedMCS)
		Expect(err).NotTo(HaveOccurred())

		// Check that spec changed
		Expect(updatedMCS.ResourceVersion).NotTo(Equal(originalResourceVersion))
		Expect(updatedMCS.Spec.ClusterSelector.MatchLabels).To(Equal(map[string]string{
			"role": "child",
		}))
		Expect(updatedMCS.Spec.ServiceSpec.Services[0].Name).NotTo(Equal(initialMCS.Spec.ServiceSpec.Services[0].Name))
		Expect(updatedMCS.Spec.ServiceSpec.Services[0].Namespace).NotTo(Equal(initialMCS.Spec.ServiceSpec.Services[0].Namespace))
		Expect(updatedMCS.Spec.ServiceSpec.Services[0].Template).NotTo(Equal(initialMCS.Spec.ServiceSpec.Services[0].Template))
	})
})
