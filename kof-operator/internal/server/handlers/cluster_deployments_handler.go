package handlers

import (
	"net/http"

	kcmv1beta1 "github.com/K0rdent/kcm/api/v1beta1"
	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	"github.com/k0rdent/kof/kof-operator/internal/server"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClusterDeploymentDTO struct {
	Name              string                              `json:"name"`
	Namespace         string                              `json:"namespace"`
	Generation        int64                               `json:"generation"`
	Labels            map[string]string                   `json:"labels"`
	Annotations       map[string]string                   `json:"annotations"`
	Spec              *kcmv1beta1.ClusterDeploymentSpec   `json:"spec"`
	Status            *kcmv1beta1.ClusterDeploymentStatus `json:"status"`
	CreationTimestamp *metav1.Time                        `json:"creation_time"`
	DeletionTimestamp *metav1.Time                        `json:"deletion_time"`
}

type response struct {
	ClusterDeployments map[string]*ClusterDeploymentDTO `json:"cluster_deployments"`
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

	cdsMap := make(map[string]*ClusterDeploymentDTO, len(cds.Items))
	for _, cd := range cds.Items {
		cdsMap[cd.Name] = &ClusterDeploymentDTO{
			Name:              cd.Name,
			Namespace:         cd.Namespace,
			Generation:        cd.Generation,
			Labels:            cd.GetLabels(),
			Annotations:       cd.GetAnnotations(),
			Spec:              &cd.Spec,
			Status:            &cd.Status,
			CreationTimestamp: &cd.CreationTimestamp,
			DeletionTimestamp: cd.DeletionTimestamp,
		}
	}

	res.Send(&response{
		ClusterDeployments: cdsMap,
	}, http.StatusOK)
}
