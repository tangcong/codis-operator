package redis

import (
	"context"
	"fmt"
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
	//"sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"log"
	"strings"
)

type redisManager struct {
	client   client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
}

func NewRedisManager(client client.Client, scheme *runtime.Scheme, record record.EventRecorder) manager.Manager {
	return &redisManager{client, scheme, record}
}

func (rm *redisManager) Reconcile(cc *v1alpha1.CodisCluster) error {
	// Reconcile Codis Redis Service
	if err := rm.syncCodisServerService(cc); err != nil {
		return err
	}

	// Reconcile Codis Redis StatefulSet
	return rm.syncCodisServerStatefulSet(cc)
}

func (rm *redisManager) getSvcName(ccName string) string {
	return ccName + "-redis"
}

func (rm *redisManager) getStatefulSetName(ccName string) string {
	return ccName + "-redis"
}

func (rm *redisManager) recordServiceEvent(verb string, cc *v1alpha1.CodisCluster, svc *corev1.Service, err error) {
	ccName := cc.Name
	svcName := svc.Name
	if err == nil {
		reason := fmt.Sprintf("Successful %s", strings.Title(verb))
		msg := fmt.Sprintf("%s Service %s in CodisCluster %s successful",
			strings.ToLower(verb), svcName, ccName)
		rm.recorder.Event(cc, corev1.EventTypeNormal, reason, msg)
		log.Printf("%s,%s", reason, msg)
	} else {
		reason := fmt.Sprintf("Failed %s", strings.Title(verb))
		msg := fmt.Sprintf("%s Service %s in CodisCluster %s failed error: %s",
			strings.ToLower(verb), svcName, ccName, err)
		rm.recorder.Event(cc, corev1.EventTypeWarning, reason, msg)
		log.Printf("%s,%s", reason, msg)
	}
}

func (rm *redisManager) recordStatefulSetEvent(verb string, cc *v1alpha1.CodisCluster, svc *apps.StatefulSet, err error) {
	ccName := cc.Name
	svcName := svc.Name
	if err == nil {
		reason := fmt.Sprintf("Successful %s", strings.Title(verb))
		msg := fmt.Sprintf("%s StatefulSet %s in CodisCluster %s successful",
			strings.ToLower(verb), svcName, ccName)
		rm.recorder.Event(cc, corev1.EventTypeNormal, reason, msg)
		log.Printf("%s,%s", reason, msg)
	} else {
		reason := fmt.Sprintf("Failed %s", strings.Title(verb))
		msg := fmt.Sprintf("%s StatefulSet %s in CodisCluster %s failed error: %s",
			strings.ToLower(verb), svcName, ccName, err)
		rm.recorder.Event(cc, corev1.EventTypeWarning, reason, msg)
		log.Printf("%s,%s", reason, msg)
	}
}

func (rm *redisManager) createService(cc *v1alpha1.CodisCluster, svc *corev1.Service) error {
	if err := rm.client.Create(context.TODO(), svc); err != nil {
		rm.recordServiceEvent("create", cc, svc, err)
		return err
	} else {
		rm.recordServiceEvent("create", cc, svc, err)
		return nil
	}
}

func (rm *redisManager) createStatefulSet(cc *v1alpha1.CodisCluster, deploy *apps.StatefulSet) error {
	if err := rm.client.Create(context.TODO(), deploy); err != nil {
		rm.recordStatefulSetEvent("create", cc, deploy, err)
		return err
	} else {
		rm.recordStatefulSetEvent("create", cc, deploy, err)
		return nil
	}
}

func (rm *redisManager) syncCodisServerService(cc *v1alpha1.CodisCluster) error {
	ns := cc.GetNamespace()
	ccName := cc.GetName()

	newSvc := rm.getNewCodisServerService(cc)

	oldSvc := &corev1.Service{}
	if err := rm.client.Get(context.TODO(), types.NamespacedName{Name: rm.getSvcName(ccName), Namespace: ns}, oldSvc); err != nil {
		if errors.IsNotFound(err) {
			return rm.createService(cc, newSvc)
		} else {
			log.Printf("ns:%s,ccName:%s,get svc err:%s", ns, ccName, err)
			return err
		}
	} else {
		log.Printf("ns:%s,ccName:%s,get svc ok", ns, ccName)
	}
	//to do
	return nil
}

func (rm *redisManager) populateEnvVar(cc *v1alpha1.CodisCluster) []corev1.EnvVar {
	var envVarList []corev1.EnvVar
	envVarList = append(envVarList, corev1.EnvVar{Name: "CODIS_PATH", Value: "/gopath/src/github.com/CodisLabs/codis"})
	envVarList = append(envVarList, corev1.EnvVar{Name: "PRODUCT_NAME", Value: cc.Spec.ClusterName})
	envVarList = append(envVarList, corev1.EnvVar{Name: "POD_IP", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "status.podIP"}}})
	envVarList = append(envVarList, corev1.EnvVar{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}})
	return envVarList
}

func (rm *redisManager) getNewCodisServerStatefulSet(cc *v1alpha1.CodisCluster) *apps.StatefulSet {
	ns := cc.GetNamespace()
	ccName := cc.GetName()

	codisServerLabels := map[string]string{
		"component":   "codis-server",
		"clusterName": ccName,
	}

	envVarList := rm.populateEnvVar(cc)

	sts := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            rm.getStatefulSetName(ccName),
			Namespace:       ns,
			OwnerReferences: []metav1.OwnerReference{utils.GetOwnerRef(cc)},
		},
		Spec: apps.StatefulSetSpec{
			Replicas:    &cc.Spec.CodisServer.Replicas,
			ServiceName: rm.getSvcName(ccName),
			Selector: &metav1.LabelSelector{
				MatchLabels: codisServerLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: codisServerLabels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "codis-server",
							Image:           cc.Spec.CodisServer.Image,
							ImagePullPolicy: "IfNotPresent",
							Command:         []string{"codis-server"},
							Args:            []string{"$(CODIS_PATH)/config/redis.conf"},
							Env:             envVarList,
							Ports:           []corev1.ContainerPort{{Name: "redis", ContainerPort: 6379}},
						},
					},
				},
			},
		},
	}
	log.Printf("codis redis image:%s", cc.Spec.CodisServer.Image)
	return sts
}

func (rm *redisManager) syncCodisServerStatefulSet(cc *v1alpha1.CodisCluster) error {
	ns := cc.GetNamespace()
	ccName := cc.GetName()

	newCodisServerStatefulSet := rm.getNewCodisServerStatefulSet(cc)
	oldCodisServerStatefulSet := &apps.StatefulSet{}
	if err := rm.client.Get(context.TODO(), types.NamespacedName{Name: rm.getStatefulSetName(ccName), Namespace: ns}, oldCodisServerStatefulSet); err != nil {
		if errors.IsNotFound(err) {
			return rm.createStatefulSet(cc, newCodisServerStatefulSet)
		} else {
			log.Printf("ns:%s,ccName:%s,get svc err:%s", ns, ccName, err)
			return err
		}
	} else {
		log.Printf("ns:%s,ccName:%s,get svc info:%+v", ns, ccName, oldCodisServerStatefulSet)
	}
	//to do
	return nil
}

func (rm *redisManager) getNewCodisServerService(cc *v1alpha1.CodisCluster) *corev1.Service {
	ns := cc.Namespace
	ccName := cc.Name

	codisServerLabels := map[string]string{
		"component":   "codis-server",
		"clusterName": ccName,
	}
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            rm.getSvcName(ccName),
			Namespace:       ns,
			Labels:          codisServerLabels,
			OwnerReferences: []metav1.OwnerReference{utils.GetOwnerRef(cc)},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Ports: []corev1.ServicePort{
				{
					Name:       "redis-port",
					Port:       6379,
					TargetPort: intstr.FromInt(6379),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: codisServerLabels,
		},
	}
}
