package remotesecret

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Istio Suite")
}

var _ = Describe("RemoteSecretProfileName", func() {
	It("should generate profile name no longer than 63 characters", func() {
		Expect(CopyRemoteSecretProfileName("cluster-deployment-name")).To(Equal("cluster-deployment-name-istio-remote-secret"))
		Expect(CopyRemoteSecretProfileName("looooooooooooooooooong-cluster-deployment-name")).To(Equal("looooooooooooooooooong-cluster-deploym-6c6f-istio-remote-secret"))
	})
})
