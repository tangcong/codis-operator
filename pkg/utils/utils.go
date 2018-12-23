package utils

import (
	"encoding/json"
	"errors"
	log "github.com/golang/glog"
	"github.com/tangcong/codis-operator/pkg/apis/codis/v1alpha1"
	apps "k8s.io/api/apps/v1"
	as "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// controllerKind contains the schema.GroupVersionKind for this controller type.
	controllerKind       = v1alpha1.SchemeGroupVersion.WithKind("CodisCluster")
	ErrNoLastApplyConfig = errors.New("last apply config is not found!")
)

const (
	LastAppliedConfigKey = "codis.k8s.io/last-applied-config"
)

func encode(obj interface{}) (string, error) {
	b, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// GetOwnerRef returns CodisCluster's OwnerReference
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

// GetDashboardSvr returns CodisCluster's dashboard addr
// for example,sample-dashboard.codis-operator-system.svc.cluster.local
func GetDashboardSvr(cc *v1alpha1.CodisCluster) string {
	ns := cc.GetNamespace()
	ccName := cc.GetName()
	return ccName + "-dashboard." + ns + ".svc.cluster.local"
}

func SetStsLastAppliedConfig(sts *apps.StatefulSet) error {
	stsSpec, err := encode(sts.Spec)
	if err != nil {
		return err
	}
	if sts.Annotations == nil {
		sts.Annotations = map[string]string{}
	}
	sts.Annotations[LastAppliedConfigKey] = stsSpec
	return nil
}

func GetStsLastAppliedConfig(sts *apps.StatefulSet) (*apps.StatefulSetSpec, error) {
	stsSpec, ok := sts.Annotations[LastAppliedConfigKey]
	if !ok {
		log.Errorf("ns:%s,name:%s,err is %v", sts.GetNamespace(), sts.GetName(), ErrNoLastApplyConfig)
		return nil, ErrNoLastApplyConfig
	}
	spec := &apps.StatefulSetSpec{}
	err := json.Unmarshal([]byte(stsSpec), spec)
	if err != nil {
		return nil, err
	}
	return spec, nil
}

func SetServiceLastAppliedConfig(svc *corev1.Service) error {
	svcSpec, err := encode(svc.Spec)
	if err != nil {
		return err
	}
	if svc.Annotations == nil {
		svc.Annotations = map[string]string{}
	}
	svc.Annotations[LastAppliedConfigKey] = svcSpec
	return nil
}

func SetDeploymentLastAppliedConfig(deploy *apps.Deployment) error {
	deploySpec, err := encode(deploy.Spec)
	if err != nil {
		return err
	}
	if deploy.Annotations == nil {
		deploy.Annotations = map[string]string{}
	}
	deploy.Annotations[LastAppliedConfigKey] = deploySpec
	return nil
}

func SetHPALastAppliedConfig(hpa *as.HorizontalPodAutoscaler) error {
	hpaSpec, err := encode(hpa.Spec)
	if err != nil {
		return err
	}
	if hpa.Annotations == nil {
		hpa.Annotations = map[string]string{}
	}
	hpa.Annotations[LastAppliedConfigKey] = hpaSpec
	return nil
}

func StatefulSetEqual(new *apps.StatefulSet, old *apps.StatefulSet) (bool, error) {
	oldSpec := apps.StatefulSetSpec{}
	if lastAppliedConfig, ok := old.Annotations[LastAppliedConfigKey]; ok {
		err := json.Unmarshal([]byte(lastAppliedConfig), &oldSpec)
		if err != nil {
			log.Errorf("ns:%s,name:%s,unmarshal statefulset err is %v", old.GetNamespace(), old.GetName(), err)
			return false, err
		}
		return apiequality.Semantic.DeepEqual(oldSpec.Replicas, new.Spec.Replicas) &&
			apiequality.Semantic.DeepEqual(oldSpec.Template, new.Spec.Template) &&
			apiequality.Semantic.DeepEqual(oldSpec.UpdateStrategy, new.Spec.UpdateStrategy), nil
	}
	return false, nil
}

func ServiceEqual(new, old *corev1.Service) (bool, error) {
	oldSpec := corev1.ServiceSpec{}
	if lastAppliedConfig, ok := old.Annotations[LastAppliedConfigKey]; ok {
		err := json.Unmarshal([]byte(lastAppliedConfig), &oldSpec)
		if err != nil {
			log.Errorf("ns:%s,name:%s,unmarshal service err is %v", old.GetNamespace(), old.GetName(), err)
			return false, err
		}
		return apiequality.Semantic.DeepEqual(oldSpec.Selector, new.Spec.Selector), nil
	}
	return false, nil
}

func DeploymentEqual(new, old *apps.Deployment) (bool, error) {
	oldSpec := apps.DeploymentSpec{}
	if lastAppliedConfig, ok := old.Annotations[LastAppliedConfigKey]; ok {
		err := json.Unmarshal([]byte(lastAppliedConfig), &oldSpec)
		if err != nil {
			log.Errorf("ns:%s,name:%s,unmarshal deployment err is %v", old.GetNamespace(), old.GetName(), err)
			return false, err
		}
		return apiequality.Semantic.DeepEqual(oldSpec, new.Spec), nil
	}
	return false, nil
}

func HPAEqual(new, old *as.HorizontalPodAutoscaler) (bool, error) {
	oldSpec := as.HorizontalPodAutoscalerSpec{}
	if lastAppliedConfig, ok := old.Annotations[LastAppliedConfigKey]; ok {
		err := json.Unmarshal([]byte(lastAppliedConfig), &oldSpec)
		if err != nil {
			log.Errorf("ns:%s,name:%s,unmarshal hpa err is %v", old.GetNamespace(), old.GetName(), err)
			return false, err
		}
		return apiequality.Semantic.DeepEqual(oldSpec, new.Spec), nil
	}
	return false, nil
}

func ResourceRequirement(spec v1alpha1.ContainerSpec, isPvc bool) corev1.ResourceRequirements {
	rr := corev1.ResourceRequirements{}
	if spec.Requests != nil {
		if rr.Requests == nil {
			rr.Requests = make(map[corev1.ResourceName]resource.Quantity)
		}
		if spec.Requests.CPU != "" && isPvc == false {
			if q, err := resource.ParseQuantity(spec.Requests.CPU); err != nil {
				log.Errorf("failed to parse CPU resource %s to quantity: %v", spec.Requests.CPU, err)
			} else {
				rr.Requests[corev1.ResourceCPU] = q
			}
		}
		if spec.Requests.Memory != "" && isPvc == false {
			if q, err := resource.ParseQuantity(spec.Requests.Memory); err != nil {
				log.Errorf("failed to parse memory resource %s to quantity: %v", spec.Requests.Memory, err)
			} else {
				rr.Requests[corev1.ResourceMemory] = q
			}
		}

		if spec.Requests.Storage != "" && isPvc {
			if q, err := resource.ParseQuantity(spec.Requests.Storage); err != nil {
				log.Errorf("failed to parse storage resource %s to quantity: %v", spec.Requests.Storage, err)
			} else {
				rr.Requests[corev1.ResourceStorage] = q
			}
		}
	}
	if spec.Limits != nil {
		if rr.Limits == nil {
			if isPvc && spec.Limits.Storage == "" {
				return rr
			}
			rr.Limits = make(map[corev1.ResourceName]resource.Quantity)
		}
		if spec.Limits.CPU != "" && isPvc == false {
			if q, err := resource.ParseQuantity(spec.Limits.CPU); err != nil {
				log.Errorf("failed to parse CPU resource %s to quantity: %v", spec.Limits.CPU, err)
			} else {
				rr.Limits[corev1.ResourceCPU] = q
			}
		}
		if spec.Limits.Memory != "" && isPvc == false {
			if q, err := resource.ParseQuantity(spec.Limits.Memory); err != nil {
				log.Errorf("failed to parse memory resource %s to quantity: %v", spec.Limits.Memory, err)
			} else {
				rr.Limits[corev1.ResourceMemory] = q
			}
		}

		if spec.Limits.Storage != "" && isPvc {
			if q, err := resource.ParseQuantity(spec.Limits.Storage); err != nil {
				log.Errorf("failed to parse storage resource %s to quantity: %v", spec.Limits.Storage, err)
			} else {
				rr.Limits[corev1.ResourceStorage] = q
			}
		}
	}
	return rr
}
