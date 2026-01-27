package objects

import (
	"maps"
	"net/http"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	"github.com/k0rdent/kof/kof-operator/internal/server"
	"github.com/k0rdent/kof/kof-operator/internal/server/handlers"
	sveltosv1beta1 "github.com/projectsveltos/addon-controller/api/v1beta1"
	"k8s.io/apimachinery/pkg/types"
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
		var kubeconfigSecretName string
		var clusterName string

		if region.Spec.KubeConfig != nil {
			kubeconfigSecretName = region.Spec.KubeConfig.Name
			config, err := k8s.GetKubeconfigFromSecret(ctx, k8s.LocalKubeClient.Client, region.Spec.KubeConfig.Name, region.Namespace)
			if err != nil {
				res.Logger.Error(err, "Failed to get kubeconfig from secret", "kubeconfigSecretName", kubeconfigSecretName)
				continue
			}

			clusterName = config.CurrentContext
			if clusterName == "" {
				res.Logger.Error(nil, "Failed to get cluster name from kubeconfig", "kubeconfigSecretName", kubeconfigSecretName)
				continue
			}
		}

		if region.Spec.ClusterDeployment != nil {
			clusterName = region.Spec.ClusterDeployment.Name
			cd := new(kcmv1beta1.ClusterDeployment)
			err := k8s.LocalKubeClient.Client.Get(ctx, types.NamespacedName{
				Name:      region.Spec.ClusterDeployment.Name,
				Namespace: region.Spec.ClusterDeployment.Namespace,
			}, cd)
			if err != nil {
				res.Logger.Error(err, "Failed to get cluster deployment", "clusterDeployment", region.Spec.ClusterDeployment.Name)
				continue
			}

			kubeconfigSecretName, err = k8s.GetSecretName(ctx, k8s.LocalKubeClient.Client, cd)
			if err != nil {
				res.Logger.Error(err, "Failed to get secret name for cluster deployment", "clusterDeployment", region.Spec.ClusterDeployment.Name)
				continue
			}
		}

		if kubeconfigSecretName == "" || clusterName == "" {
			res.Logger.Error(nil, "Region is missing kubeconfig and cluster deployment", "region", region.Name)
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
