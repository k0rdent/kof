package handlers

import (
	"fmt"
)

func GetResourcePath(clusterName, namespace, name string) string {
	if namespace == "" {
		return fmt.Sprintf("%s/%s", clusterName, name)
	}
	return fmt.Sprintf("%s/%s/%s", clusterName, namespace, name)
}
