package handlers

import (
	"net/http"

	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	"github.com/k0rdent/kof/kof-operator/internal/server"
	addoncontrollerv1beta1 "github.com/projectsveltos/addon-controller/api/v1beta1"
)

type ClusterSummariesHandlerResponse struct {
	ClusterSummaries map[string]*addoncontrollerv1beta1.ClusterSummary `json:"cluster_summaries"`
}

func ClusterSummariesHandler(res *server.Response, req *http.Request) {
	ctx := req.Context()

	kubeClient, err := k8s.NewClient()
	if err != nil {
		res.Logger.Error(err, "Failed to create kube client")
		res.Fail(server.BasicInternalErrorMessage, http.StatusInternalServerError)
		return
	}

	summaries, err := k8s.GetClusterSummaries(ctx, kubeClient.Client)
	if err != nil {
		res.Logger.Error(err, "Failed to get cluster summaries")
		res.Fail(server.BasicInternalErrorMessage, http.StatusInternalServerError)
		return
	}

	summariesMap := make(map[string]*addoncontrollerv1beta1.ClusterSummary, len(summaries.Items))
	for _, summary := range summaries.Items {
		// Remove managed fields before sending to reduce payload size and avoid unnecessary data
		summary.SetManagedFields(nil)
		summariesMap[summary.Name] = &summary
	}

	res.Send(&ClusterSummariesHandlerResponse{
		ClusterSummaries: summariesMap,
	}, http.StatusOK)
}
