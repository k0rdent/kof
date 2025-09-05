package handlers

import (
	"net/http"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	"github.com/k0rdent/kof/kof-operator/internal/server"
)

type StateManagementProvidersResponse struct {
	StateManagementProviders map[string]*kcmv1beta1.StateManagementProvider `json:"items"`
}

func StateManagementProvidersHandler(res *server.Response, req *http.Request) {
	ctx := req.Context()

	kubeClient, err := k8s.NewClient()
	if err != nil {
		res.Logger.Error(err, "Failed to create kube client")
		res.Fail(server.BasicInternalErrorMessage, http.StatusInternalServerError)
		return
	}

	cds, err := k8s.GetStateManagementProviders(ctx, kubeClient.Client)
	if err != nil {
		res.Logger.Error(err, "Failed to get state management providers")
		res.Fail(server.BasicInternalErrorMessage, http.StatusInternalServerError)
		return
	}

	cdsMap := make(map[string]*kcmv1beta1.StateManagementProvider, len(cds.Items))
	for _, cd := range cds.Items {

		// Remove managed fields before sending to reduce payload size and avoid unnecessary data
		cd.SetManagedFields(nil)
		cdsMap[cd.Name] = &cd
	}

	res.Send(&StateManagementProvidersResponse{
		StateManagementProviders: cdsMap,
	}, http.StatusOK)
}
