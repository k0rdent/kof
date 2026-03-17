package tests

import (
	"encoding/json"

	"github.com/k0rdent/kof/kof-operator/internal/server/handlers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"istio.io/istio/pkg/cluster"

	"github.com/go-logr/logr"
)

var _ = Describe("ParseIstioSecretsStatus", func() {
	var logger logr.Logger

	BeforeEach(func() {
		logger = logr.Discard()
	})

	makeRaw := func(infos []cluster.DebugInfo) []byte {
		data, _ := json.Marshal(infos)
		return data
	}

	It("returns empty list for empty input", func() {
		result, err := handlers.ParseIstioSecretsStatus(&logger, map[string][]byte{})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(BeEmpty())
	})

	It("skips entries with empty secret name", func() {
		input := map[string][]byte{
			"pod1": makeRaw([]cluster.DebugInfo{
				{ID: "cluster-a", SecretName: "", SyncStatus: "synced"},
			}),
		}
		result, err := handlers.ParseIstioSecretsStatus(&logger, input)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(BeEmpty())
	})

	It("parses a valid secret correctly", func() {
		input := map[string][]byte{
			"pod1": makeRaw([]cluster.DebugInfo{
				{ID: "cluster-a", SecretName: "istio-system/remote-secret-cluster-a", SyncStatus: "synced"},
			}),
		}
		result, err := handlers.ParseIstioSecretsStatus(&logger, input)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(HaveLen(1))
		Expect(result[0]).To(Equal(handlers.Secret{
			Namespace:   "istio-system",
			Name:        "remote-secret-cluster-a",
			SyncStatus:  "synced",
			ClusterName: "cluster-a",
		}))
	})

	It("parses multiple secrets from multiple pods", func() {
		input := map[string][]byte{
			"pod1": makeRaw([]cluster.DebugInfo{
				{ID: "cluster-a", SecretName: "istio-system/secret-a", SyncStatus: "synced"},
			}),
			"pod2": makeRaw([]cluster.DebugInfo{
				{ID: "cluster-b", SecretName: "istio-system/secret-b", SyncStatus: "timeout"},
			}),
		}
		result, err := handlers.ParseIstioSecretsStatus(&logger, input)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(HaveLen(2))
	})

	It("returns an error on invalid JSON", func() {
		input := map[string][]byte{
			"pod1": []byte("not-json"),
		}
		_, err := handlers.ParseIstioSecretsStatus(&logger, input)
		Expect(err).To(HaveOccurred())
	})
})
