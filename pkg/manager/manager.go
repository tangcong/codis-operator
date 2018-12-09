package manager

import (
	"github.com/tangcong/codis-operator/pkg/apis/codis/v1alpha1"
)

type Manager interface {
	Reconcile(*v1alpha1.CodisCluster) error
}
