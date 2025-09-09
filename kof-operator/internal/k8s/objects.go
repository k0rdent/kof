package k8s

import (
	"context"
	"fmt"
	"reflect"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetObjectsList[ObjListT client.ObjectList](ctx context.Context, c client.Client) (ObjListT, error) {
	var objList ObjListT
	objType := reflect.TypeOf(objList)
	if objType.Kind() != reflect.Ptr || objType.Elem().Kind() != reflect.Struct {
		var zero ObjListT
		return zero, fmt.Errorf("ObjListT must be a pointer to a struct that implements client.ObjectList, got %v", objType)
	}
	listPtr := reflect.New(objType.Elem()).Interface().(client.ObjectList)
	err := c.List(ctx, listPtr)
	return listPtr.(ObjListT), err
}
