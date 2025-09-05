package handlers

import (
	"net/http"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	"github.com/k0rdent/kof/kof-operator/internal/server"
)

type ClusterDeploymentResponse struct {
	ClusterDeployments map[string]*kcmv1beta1.ClusterDeployment `json:"items"`
}

func ClusterDeploymentHandler(res *server.Response, req *http.Request) {
	ctx := req.Context()

	kubeClient, err := k8s.NewClient()
	if err != nil {
		res.Logger.Error(err, "Failed to create kube client")
		res.Fail(server.BasicInternalErrorMessage, http.StatusInternalServerError)
		return
	}

	cds, err := k8s.GetClusterDeployments(ctx, kubeClient.Client)
	if err != nil {
		res.Logger.Error(err, "Failed to get cluster deployments")
		res.Fail(server.BasicInternalErrorMessage, http.StatusInternalServerError)
		return
	}

	cdsMap := make(map[string]*kcmv1beta1.ClusterDeployment, len(cds.Items))
	for _, cd := range cds.Items {

		// Remove managed fields before sending to reduce payload size and avoid unnecessary data
		cd.SetManagedFields(nil)
		cdsMap[cd.Name] = &cd
	}

	res.Send(&ClusterDeploymentResponse{
		ClusterDeployments: cdsMap,
	}, http.StatusOK)
}
