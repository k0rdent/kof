package objects

import (
	"maps"
	"net/http"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
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

	regions := new(kcmv1beta1.RegionList)
	if err := k8s.LocalKubeClient.Client.List(ctx, regions); err != nil {
		res.Logger.Error(err, "Failed to get regions")
		res.Fail(server.BasicInternalErrorMessage, http.StatusInternalServerError)
		return
	}

	for _, region := range regions.Items {
		if region.Spec.KubeConfig == nil {
			continue
		}

		kubeconfigSecretName := region.Spec.KubeConfig.Name
		clusterName := k8s.GetClusterNameByKubeconfigSecretName(kubeconfigSecretName)
		if clusterName == "" {
			res.Logger.Error(nil, "Failed to get cluster name from kubeconfig secret name", "kubeconfigSecretName", kubeconfigSecretName)
			continue
		}

		regionKubeClient, err := k8s.NewKubeClientFromSecret(ctx, k8s.LocalKubeClient.Client, kubeconfigSecretName, k8s.DefaultSystemNamespace)
		if err != nil {
			res.Logger.Error(err, "Failed to create kube client for region", "region", region.Name)
			continue
		}

		regionClusterSummariesMap, err := GetObjectsMap[*sveltosv1beta1.ClusterSummaryList](ctx, regionKubeClient, clusterName)
		if err != nil {
			res.Logger.Error(err, "Failed to get cluster summaries for region", "region", region.Name)
			continue
		}

		maps.Copy(clusterSummariesMap, regionClusterSummariesMap)
	}

	res.Send(&K8sObjectsResponse{
		Items: clusterSummariesMap,
	}, http.StatusOK)
}
