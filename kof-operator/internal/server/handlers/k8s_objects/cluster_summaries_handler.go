package objects

import (
	"maps"
	"net/http"

	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	"github.com/k0rdent/kof/kof-operator/internal/server"
	"github.com/k0rdent/kof/kof-operator/internal/server/handlers"
	sveltosv1beta1 "github.com/projectsveltos/addon-controller/api/v1beta1"
)

func ClusterSummariesHandler(res *server.Response, req *http.Request) {
	ctx := req.Context()

	clusterSummariesMap, err := GetObjectsMap[*sveltosv1beta1.ClusterSummaryList](ctx, k8s.LocalKubeClient, handlers.MothershipClusterName)
	if err != nil {
		res.Logger.Error(err, "Failed to get cluster summaries")
		res.Fail(server.BasicInternalErrorMessage, http.StatusInternalServerError)
		return
	}

	regionClusters, err := k8s.GetKcmRegionClusters(ctx, k8s.LocalKubeClient.Client)
	if err != nil {
		res.Logger.Error(err, "Failed to get KCM regions")
		res.Fail(server.BasicInternalErrorMessage, http.StatusInternalServerError)
		return
	}

	for _, cluster := range regionClusters {
		kubeconfigSecretName, err := k8s.GetKubeconfigSecretName(ctx, k8s.LocalKubeClient.Client, cluster)
		if err != nil {
			res.Logger.Error(err, "Failed to get secret name for cluster deployment", "clusterDeployment", cluster.Name)
			continue
		}

		regionKubeClient, err := k8s.NewKubeClientFromSecret(ctx, k8s.LocalKubeClient.Client, kubeconfigSecretName, k8s.DefaultSystemNamespace)
		if err != nil {
			res.Logger.Error(err, "Failed to create kube client for region", "region", cluster.Name)
			continue
		}

		regionClusterSummariesMap, err := GetObjectsMap[*sveltosv1beta1.ClusterSummaryList](ctx, regionKubeClient, cluster.Name)
		if err != nil {
			res.Logger.Error(err, "Failed to get cluster summaries for region", "region", cluster.Name)
			continue
		}

		maps.Copy(clusterSummariesMap, regionClusterSummariesMap)
	}

	res.Send(&K8sObjectsResponse{
		Items: clusterSummariesMap,
	}, http.StatusOK)
}
