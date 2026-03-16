package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	"github.com/k0rdent/kof/kof-operator/internal/models/labels"
	"github.com/k0rdent/kof/kof-operator/internal/server"
	"istio.io/istio/pkg/cluster"
	"istio.io/istio/pkg/config/constants"
	"istio.io/istio/pkg/kube"
)

func IstioMeshHandler(res *server.Response, req *http.Request) {
	ctx := req.Context()
	graph, err := buildMeshGraph(ctx, res)
	if err != nil {
		res.Logger.Error(err, "Failed to build Istio mesh graph")
		res.Fail(server.BasicInternalErrorMessage, http.StatusInternalServerError)
		return
	}
	res.SendObj(graph, http.StatusOK)
}

func buildMeshGraph(ctx context.Context, res *server.Response) (*MeshGraph, error) {
	wg := sync.WaitGroup{}
	nodesSet := &sync.Map{}
	linksSet := &sync.Map{}

	addMeshNode(nodesSet, ManagementClusterName, "", "management")

	if err := collectLinksFromCluster(ctx, res, k8s.LocalKubeClient, ManagementClusterName, linksSet); err != nil {
		res.Logger.Error(err, "Failed to collect Istio remote secrets from management cluster")
	}

	cdList, err := k8s.GetIstioClusterDeployments(ctx, k8s.LocalKubeClient.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to list ClusterDeployments: %w", err)
	}

	for i := range cdList.Items {
		wg.Go(func() {
			cd := &cdList.Items[i]
			role := meshClusterRole(cd)
			addMeshNode(nodesSet, cd.Name, cd.Namespace, role)

			remoteClient, err := k8s.NewKubeClientFromClusterDeployment(ctx, k8s.LocalKubeClient.Client, cd)
			if err != nil {
				res.Logger.Error(err, "Failed to create kube client for cluster", "cluster", cd.Name)
				return
			}

			if err := collectLinksFromCluster(ctx, res, remoteClient, cd.Name, linksSet); err != nil {
				res.Logger.Error(err, "Failed to collect Istio remote secrets", "cluster", cd.Name)
			}
		})
	}

	wg.Wait()

	graph := &MeshGraph{
		Nodes: make([]MeshNode, 0),
		Links: make([]MeshLink, 0),
	}
	nodesSet.Range(func(key, value any) bool {
		graph.Nodes = append(graph.Nodes, value.(MeshNode))
		return true
	})
	linksSet.Range(func(key, value any) bool {
		graph.Links = append(graph.Links, value.(MeshLink))
		return true
	})

	return graph, nil
}

func addMeshNode(nodesSet *sync.Map, id, namespace, role string) {
	nodesSet.LoadOrStore(id, MeshNode{ID: id, Name: id, Namespace: namespace, Role: role})
}

func meshClusterRole(cd *kcmv1beta1.ClusterDeployment) string {
	if role, ok := cd.Labels[labels.KofClusterRoleLabel]; ok {
		return role
	}
	return "child"
}

func collectLinksFromCluster(
	ctx context.Context,
	res *server.Response,
	kubeClient *k8s.KubeClient,
	sourceCluster string,
	linksSet *sync.Map,
) error {
	istioClient, err := kube.NewCLIClient(kubeClient.Config)
	if err != nil {
		return fmt.Errorf("failed to create Istio CLI client for %s: %w", sourceCluster, err)
	}

	results, err := istioClient.AllDiscoveryDo(ctx, constants.IstioSystemNamespace, "debug/clusterz")
	if err != nil {
		return fmt.Errorf("istio AllDiscoveryDo for %s: %w", sourceCluster, err)
	}

	for _, data := range results {
		var infos []cluster.DebugInfo
		if err := json.Unmarshal(data, &infos); err != nil {
			res.Logger.Error(err, "Failed to unmarshal clusterz response", "cluster", sourceCluster)
			continue
		}

		for _, info := range infos {
			if info.SecretName == "" {
				continue
			}

			targetCluster := info.ID.String()
			if targetCluster == "" || targetCluster == sourceCluster {
				continue
			}

			edgeKey := sourceCluster + "->" + targetCluster
			secretName := info.SecretName
			if parts := strings.SplitN(info.SecretName, "/", 2); len(parts) == 2 {
				secretName = parts[1]
			}

			linksSet.LoadOrStore(edgeKey, MeshLink{
				Source:     sourceCluster,
				Target:     targetCluster,
				SecretName: secretName,
			})
		}
	}

	return nil
}
