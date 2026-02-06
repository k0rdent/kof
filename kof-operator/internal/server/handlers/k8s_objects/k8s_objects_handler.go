package objects

import (
	"context"
	"fmt"
	"net/http"

	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	"github.com/k0rdent/kof/kof-operator/internal/server"
	"github.com/k0rdent/kof/kof-operator/internal/server/handlers"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type K8sObjectsResponse struct {
	Items map[string]client.Object `json:"items"`
}

func K8sObjectsHandler[ObjListT client.ObjectList](res *server.Response, req *http.Request) {
	ctx := req.Context()

	objectsMap, err := GetObjectsMap[ObjListT](ctx, k8s.LocalKubeClient, handlers.MothershipClusterName)
	if err != nil {
		res.Logger.Error(err, "Failed to get objects map")
		res.Fail(server.BasicInternalErrorMessage, http.StatusInternalServerError)
		return
	}

	res.SendObj(&K8sObjectsResponse{
		Items: objectsMap,
	}, http.StatusOK)
}

func GetObjectsMap[ObjListT client.ObjectList](ctx context.Context, kubeClient *k8s.KubeClient, clusterName string) (map[string]client.Object, error) {
	objectList, err := k8s.GetObjectsList[ObjListT](ctx, kubeClient.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to get objects list: %v", err)
	}

	objectsMap := make(map[string]client.Object)

	if err := meta.EachListItem(objectList, func(o runtime.Object) error {
		obj, ok := o.(client.Object)
		if !ok {
			return fmt.Errorf("object is not a client.Object")
		}
		// Clear managed fields to reduce payload size
		obj.SetManagedFields(nil)
		path := getObjectPath(clusterName, obj)
		objectsMap[path] = obj
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to iterate over list items: %v", err)
	}
	return objectsMap, nil
}

func getObjectPath(clusterName string, obj client.Object) string {
	return handlers.GetResourcePath(clusterName, obj.GetNamespace(), obj.GetName())
}
