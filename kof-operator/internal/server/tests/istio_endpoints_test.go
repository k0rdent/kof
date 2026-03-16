package tests

import (
	"github.com/k0rdent/kof/kof-operator/internal/server/handlers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"istio.io/istio/pilot/pkg/model"
)

var _ = Describe("ShardKeyToClusterID", func() {
	DescribeTable("extracts cluster ID from shard key",
		func(shardKey, expected string) {
			Expect(handlers.ShardKeyToClusterID(shardKey)).To(Equal(expected))
		},
		Entry("standard format", "Kubernetes/my-cluster", "my-cluster"),
		Entry("regional cluster", "Kubernetes/regional-1", "regional-1"),
		Entry("no slash falls back to full key", "no-slash-key", "no-slash-key"),
		Entry("empty string", "", ""),
		Entry("multiple slashes keeps remainder", "a/b/c", "b/c"),
	)
})

var _ = Describe("BuildClusterConnectivity", func() {
	It("returns empty connected clusters for an empty mesh", func() {
		result := handlers.BuildClusterConnectivity("source-cluster", "kof", handlers.IstioMesh{})
		Expect(result.SourceCluster).To(Equal("source-cluster"))
		Expect(result.SourceClusterNamespace).To(Equal("kof"))
		Expect(result.ConnectedClusters).To(BeEmpty())
	})

	It("excludes self-cluster endpoints", func() {
		mesh := handlers.IstioMesh{
			"my-service.default.svc.cluster.local": handlers.NamespaceMap{
				"default": handlers.NamespaceEntry{
					Shards: map[string][]handlers.Endpoint{
						"Kubernetes/source-cluster": {
							{WorkloadName: "my-pod", EndpointPort: 8080, HealthStatus: model.Healthy},
						},
					},
				},
			},
		}
		result := handlers.BuildClusterConnectivity("source-cluster", "kof", mesh)
		Expect(result.ConnectedClusters).To(BeEmpty())
	})

	It("includes remote cluster endpoints", func() {
		mesh := handlers.IstioMesh{
			"svc.default.svc.cluster.local": handlers.NamespaceMap{
				"default": handlers.NamespaceEntry{
					Shards: map[string][]handlers.Endpoint{
						"Kubernetes/remote-cluster": {
							{
								WorkloadName:   "remote-pod",
								EndpointPort:   9090,
								Addresses:      []string{"10.0.0.1"},
								ServiceAccount: "default",
								HealthStatus:   model.Healthy,
							},
						},
					},
				},
			},
		}
		result := handlers.BuildClusterConnectivity("source-cluster", "kof", mesh)
		Expect(result.ConnectedClusters).To(HaveLen(1))
		cc := result.ConnectedClusters[0]
		Expect(cc.ClusterID).To(Equal("remote-cluster"))
		Expect(cc.Services).To(HaveLen(1))
		svc := cc.Services[0]
		Expect(svc.WorkloadName).To(Equal("remote-pod"))
		Expect(svc.Port).To(Equal(9090))
		Expect(svc.Healthy).To(BeTrue())
	})

	It("marks unhealthy endpoints correctly", func() {
		mesh := handlers.IstioMesh{
			"svc.default.svc.cluster.local": handlers.NamespaceMap{
				"default": handlers.NamespaceEntry{
					Shards: map[string][]handlers.Endpoint{
						"Kubernetes/remote-cluster": {
							{EndpointPort: 8080, HealthStatus: model.UnHealthy},
						},
					},
				},
			},
		}
		result := handlers.BuildClusterConnectivity("source-cluster", "kof", mesh)
		Expect(result.ConnectedClusters).To(HaveLen(1))
		Expect(result.ConnectedClusters[0].Services[0].Healthy).To(BeFalse())
	})
})
