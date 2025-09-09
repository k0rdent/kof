package handlers

import (
	"fmt"
	"net/http"

	"github.com/k0rdent/kof/kof-operator/internal/k8s"
	"github.com/k0rdent/kof/kof-operator/internal/server"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type K8sObjectsResponse struct {
	Items map[string]client.Object `json:"items"`
}

func K8sObjectsHandler[ObjListT client.ObjectList](res *server.Response, req *http.Request) {
	ctx := req.Context()

	kubeClient, err := k8s.NewClient()
	if err != nil {
		res.Logger.Error(err, "Failed to create kube client")
		res.Fail(server.BasicInternalErrorMessage, http.StatusInternalServerError)
		return
	}

	objectList, err := k8s.GetObjectsList[ObjListT](ctx, kubeClient.Client)
	if err != nil {
		res.Logger.Error(err, "Failed to get objects list")
		res.Fail(server.BasicInternalErrorMessage, http.StatusInternalServerError)
		return
	}

	objectsMap := make(map[string]client.Object)

	if err := meta.EachListItem(objectList, func(o runtime.Object) error {
		obj, ok := o.(client.Object)
		if !ok {
			return fmt.Errorf("object is not a client.Object")
		}
		// Clear managed fields to reduce payload size
		obj.SetManagedFields(nil)
		path := getObjectPath(obj)
		objectsMap[path] = obj
		return nil
	}); err != nil {
		res.Logger.Error(err, "Failed to iterate over list items")
		res.Fail(server.BasicInternalErrorMessage, http.StatusInternalServerError)
		return
	}

	res.Send(&K8sObjectsResponse{
		Items: objectsMap,
	}, http.StatusOK)
}

func getObjectPath(obj client.Object) string {
	var path string
	namespace := obj.GetNamespace()
	name := obj.GetName()
	if namespace == "" {
		path = name
	} else {
		path = fmt.Sprintf("%s/%s", namespace, name)
	}
	return path
}
