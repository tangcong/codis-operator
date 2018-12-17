package sentinel

import (
	"context"
	"fmt"
	log "github.com/golang/glog"
	"github.com/tangcong/codis-operator/pkg/apis/codis/v1alpha1"
	"github.com/tangcong/codis-operator/pkg/manager"
	"github.com/tangcong/codis-operator/pkg/utils"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type sentinelManager struct {
	client   client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
}

func NewSentinelManager(client client.Client, scheme *runtime.Scheme, record record.EventRecorder) manager.Manager {
	return &sentinelManager{client, scheme, record}
}

func (sm *sentinelManager) Reconcile(cc *v1alpha1.CodisCluster) error {
	// Reconcile Sentinel Service
	if err := sm.syncSentinelService(cc); err != nil {
		return err
	}

	// Reconcile Sentinel StatefulSet
	return sm.syncSentinelStatefulSet(cc)
}

func (sm *sentinelManager) getSvcName(ccName string) string {
	return ccName + "-redis-sentinel"
}

func (sm *sentinelManager) getStatefulSetName(ccName string) string {
	return ccName + "-redis-sentinel"
}

func (sm *sentinelManager) recordServiceEvent(verb string, cc *v1alpha1.CodisCluster, svc *corev1.Service, err error) {
	ccName := cc.Name
	svcName := svc.Name
	if err == nil {
		reason := fmt.Sprintf("Successful %s", strings.Title(verb))
		msg := fmt.Sprintf("%s Service %s in CodisCluster %s successful",
			strings.ToLower(verb), svcName, ccName)
		sm.recorder.Event(cc, corev1.EventTypeNormal, reason, msg)
		log.Infof("%s,%s", reason, msg)
	} else {
		reason := fmt.Sprintf("Failed %s", strings.Title(verb))
		msg := fmt.Sprintf("%s Service %s in CodisCluster %s failed error: %s",
			strings.ToLower(verb), svcName, ccName, err)
		sm.recorder.Event(cc, corev1.EventTypeWarning, reason, msg)
		log.Infof("%s,%s", reason, msg)
	}
}

func (sm *sentinelManager) recordStatefulSetEvent(verb string, cc *v1alpha1.CodisCluster, svc *apps.StatefulSet, err error) {
	ccName := cc.Name
	svcName := svc.Name
	if err == nil {
		reason := fmt.Sprintf("Successful %s", strings.Title(verb))
		msg := fmt.Sprintf("%s StatefulSet %s in CodisCluster %s successful",
			strings.ToLower(verb), svcName, ccName)
		sm.recorder.Event(cc, corev1.EventTypeNormal, reason, msg)
		log.Infof("%s,%s", reason, msg)
	} else {
		reason := fmt.Sprintf("Failed %s", strings.Title(verb))
		msg := fmt.Sprintf("%s StatefulSet %s in CodisCluster %s failed error: %s",
			strings.ToLower(verb), svcName, ccName, err)
		sm.recorder.Event(cc, corev1.EventTypeWarning, reason, msg)
		log.Infof("%s,%s", reason, msg)
	}
}

func (sm *sentinelManager) createService(cc *v1alpha1.CodisCluster, svc *corev1.Service) error {
	if err := sm.client.Create(context.TODO(), svc); err != nil {
		sm.recordServiceEvent("create", cc, svc, err)
		return err
	} else {
		sm.recordServiceEvent("create", cc, svc, err)
		return nil
	}
}

func (sm *sentinelManager) createStatefulSet(cc *v1alpha1.CodisCluster, sts *apps.StatefulSet) error {
	if err := sm.client.Create(context.TODO(), sts); err != nil {
		sm.recordStatefulSetEvent("create", cc, sts, err)
		return err
	} else {
		sm.recordStatefulSetEvent("create", cc, sts, err)
		return nil
	}
}

func (sm *sentinelManager) updateStatefulSet(cc *v1alpha1.CodisCluster, sts *apps.StatefulSet) error {
	if err := sm.client.Update(context.TODO(), sts); err != nil {
		sm.recordStatefulSetEvent("update", cc, sts, err)
		return err
	} else {
		sm.recordStatefulSetEvent("update", cc, sts, err)
		return nil
	}
}

func (sm *sentinelManager) syncSentinelService(cc *v1alpha1.CodisCluster) error {
	ns := cc.GetNamespace()
	ccName := cc.GetName()

	newSvc := sm.getNewSentinelService(cc)

	oldSvc := &corev1.Service{}
	if err := sm.client.Get(context.TODO(), types.NamespacedName{Name: sm.getSvcName(ccName), Namespace: ns}, oldSvc); err != nil {
		if errors.IsNotFound(err) {
			return sm.createService(cc, newSvc)
		} else {
			log.Infof("ns:%s,ccName:%s,get sentinel svc err:%s", ns, ccName, err)
			return err
		}
	} else {
		log.Infof("ns:%s,ccName:%s,get sentinel svc ok", ns, ccName)
	}
	//to do
	return nil
}

func (sm *sentinelManager) populateEnvVar(cc *v1alpha1.CodisCluster) []corev1.EnvVar {
	ccName := cc.GetName()
	var envVarList []corev1.EnvVar
	envVarList = append(envVarList, corev1.EnvVar{Name: "CODIS_PATH", Value: "/gopath/src/github.com/CodisLabs/codis"})
	envVarList = append(envVarList, corev1.EnvVar{Name: "PRODUCT_NAME", Value: ccName})
	envVarList = append(envVarList, corev1.EnvVar{Name: "POD_IP", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "status.podIP"}}})
	envVarList = append(envVarList, corev1.EnvVar{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}})
	return envVarList
}

func (sm *sentinelManager) getNewSentinelStatefulSet(cc *v1alpha1.CodisCluster) *apps.StatefulSet {
	ns := cc.GetNamespace()
	ccName := cc.GetName()

	sentinelLabels := map[string]string{
		"component":   "redis-sentinel",
		"clusterName": ccName,
	}

	envVarList := sm.populateEnvVar(cc)

	sts := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            sm.getStatefulSetName(ccName),
			Namespace:       ns,
			OwnerReferences: []metav1.OwnerReference{utils.GetOwnerRef(cc)},
		},
		Spec: apps.StatefulSetSpec{
			Replicas:    &cc.Spec.Sentinel.Replicas,
			ServiceName: sm.getSvcName(ccName),
			Selector: &metav1.LabelSelector{
				MatchLabels: sentinelLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: sentinelLabels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "redis-sentinel",
							Image:           cc.Spec.Sentinel.Image,
							ImagePullPolicy: "IfNotPresent",
							Command:         []string{"codis-server"},
							Args:            []string{"$(CODIS_PATH)/config/sentinel.conf", "--sentinel"},
							Env:             envVarList,
							Ports:           []corev1.ContainerPort{{Name: "sentinel-port", ContainerPort: 26379}},
						},
					},
				},
			},
		},
	}
	log.Infof("deploy redis-sentinel image:%s", cc.Spec.Sentinel.Image)
	return sts
}

func (sm *sentinelManager) syncSentinelStatefulSet(cc *v1alpha1.CodisCluster) error {
	ns := cc.GetNamespace()
	ccName := cc.GetName()

	newSentinelStatefulSet := sm.getNewSentinelStatefulSet(cc)
	oldSentinelStatefulSet := &apps.StatefulSet{}
	if err := sm.client.Get(context.TODO(), types.NamespacedName{Name: sm.getStatefulSetName(ccName), Namespace: ns}, oldSentinelStatefulSet); err != nil {
		if errors.IsNotFound(err) {
			err = utils.SetStsLastAppliedConfig(newSentinelStatefulSet)
			if err != nil {
				return nil
			}
			return sm.createStatefulSet(cc, newSentinelStatefulSet)
		} else {
			log.Infof("ns:%s,ccName:%s,get sentinel statefulset err:%s", ns, ccName, err)
			return err
		}
	}
	log.Infof("ns:%s,ccName:%s,get sentinel statefulset succ", ns, ccName)
	if equal, err := utils.StatefulSetEqual(newSentinelStatefulSet, oldSentinelStatefulSet); err != nil {
		log.Errorf("ns:%s,ccName:%s,statefulset equal err:%v", ns, ccName, err)
		return err
	} else {
		if !equal {
			sts := *oldSentinelStatefulSet
			sts.Spec = newSentinelStatefulSet.Spec
			if err = utils.SetStsLastAppliedConfig(&sts); err != nil {
				log.Errorf("ns:%s,ccName:%s,set statefulset annotation err:%v", ns, ccName, err)
				return err
			}
			if err = sm.updateStatefulSet(cc, &sts); err != nil {
				log.Errorf("ns:%s,ccName:%s,update statefulset err:%v", ns, ccName, err)
				return err
			}
			log.Errorf("ns:%s,ccName:%s,statefulset change,update succ", ns, ccName)
		}
	}
	return nil
}

func (sm *sentinelManager) getNewSentinelService(cc *v1alpha1.CodisCluster) *corev1.Service {
	ns := cc.Namespace
	ccName := cc.Name

	sentinelLabels := map[string]string{
		"component":   "redis-sentinel",
		"clusterName": ccName,
	}
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            sm.getSvcName(ccName),
			Namespace:       ns,
			Labels:          sentinelLabels,
			OwnerReferences: []metav1.OwnerReference{utils.GetOwnerRef(cc)},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Ports: []corev1.ServicePort{
				{
					Name:       "sentinel-port",
					Port:       6379,
					TargetPort: intstr.FromInt(6379),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: sentinelLabels,
		},
	}
}
