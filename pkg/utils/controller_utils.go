package utils

import (
	"github.com/tangcong/codis-operator/pkg/apis/codis/v1alpha1"
	//corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// controllerKind contains the schema.GroupVersionKind for this controller type.
	controllerKind = v1alpha1.SchemeGroupVersion.WithKind("CodisCluster")
)

// GetOwnerRef returns TidbCluster's OwnerReference
func GetOwnerRef(cc *v1alpha1.CodisCluster) metav1.OwnerReference {
	controller := true
	blockOwnerDeletion := true
	return metav1.OwnerReference{
		APIVersion:         controllerKind.GroupVersion().String(),
		Kind:               controllerKind.Kind,
		Name:               cc.GetName(),
		UID:                cc.GetUID(),
		Controller:         &controller,
		BlockOwnerDeletion: &blockOwnerDeletion,
	}
}
