package controller

import (
	kcmv1alpha1 "github.com/K0rdent/kcm/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func GetOwnerReference(ownerName string, ownerUID types.UID) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: kcmv1alpha1.GroupVersion.String(),
		Kind:       kcmv1alpha1.ClusterDeploymentKind,
		Name:       ownerName,
		UID:        ownerUID,
	}
}
