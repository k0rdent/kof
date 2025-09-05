package handlers

import (
	"net/http"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	"github.com/k0rdent/kof/kof-operator/internal/server"
)

type MultiClusterServicesHandlerResponse struct {
	MultiClusterServices map[string]*kcmv1beta1.MultiClusterService `json:"items"`
}

func MultiClusterServicesHandler(res *server.Response, req *http.Request) {
	ctx := req.Context()

	kubeClient, err := k8s.NewClient()
	if err != nil {
		res.Logger.Error(err, "Failed to create kube client")
		res.Fail(server.BasicInternalErrorMessage, http.StatusInternalServerError)
		return
	}

	summaries, err := k8s.GetMultiClusterService(ctx, kubeClient.Client)
	if err != nil {
		res.Logger.Error(err, "Failed to get cluster summaries")
		res.Fail(server.BasicInternalErrorMessage, http.StatusInternalServerError)
		return
	}

	clusterServicesMap := make(map[string]*kcmv1beta1.MultiClusterService, len(summaries.Items))
	for _, multiClusterService := range summaries.Items {
		// Remove managed fields before sending to reduce payload size and avoid unnecessary data
		multiClusterService.SetManagedFields(nil)
		clusterServicesMap[multiClusterService.Name] = &multiClusterService
	}

	res.Send(&MultiClusterServicesHandlerResponse{
		MultiClusterServices: clusterServicesMap,
	}, http.StatusOK)
}
