package redis

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

func (rm *redisManager) getCodisServerVolumeName(ccName string) string {
	return ccName + "-redis-volume"
}

func (rm *redisManager) recordServiceEvent(verb string, cc *v1alpha1.CodisCluster, svc *corev1.Service, err error) {
	ccName := cc.Name
	svcName := svc.Name
	if err == nil {
		reason := fmt.Sprintf("Successful %s", strings.Title(verb))
		msg := fmt.Sprintf("%s Service %s in CodisCluster %s successful",
			strings.ToLower(verb), svcName, ccName)
		rm.recorder.Event(cc, corev1.EventTypeNormal, reason, msg)
		log.Infof("%s,%s", reason, msg)
	} else {
		reason := fmt.Sprintf("Failed %s", strings.Title(verb))
		msg := fmt.Sprintf("%s Service %s in CodisCluster %s failed error: %s",
			strings.ToLower(verb), svcName, ccName, err)
		rm.recorder.Event(cc, corev1.EventTypeWarning, reason, msg)
		log.Infof("%s,%s", reason, msg)
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
		log.Infof("%s,%s", reason, msg)
	} else {
		reason := fmt.Sprintf("Failed %s", strings.Title(verb))
		msg := fmt.Sprintf("%s StatefulSet %s in CodisCluster %s failed error: %s",
			strings.ToLower(verb), svcName, ccName, err)
		rm.recorder.Event(cc, corev1.EventTypeWarning, reason, msg)
		log.Infof("%s,%s", reason, msg)
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

func (rm *redisManager) createStatefulSet(cc *v1alpha1.CodisCluster, sts *apps.StatefulSet) error {
	if err := rm.client.Create(context.TODO(), sts); err != nil {
		rm.recordStatefulSetEvent("create", cc, sts, err)
		return err
	} else {
		rm.recordStatefulSetEvent("create", cc, sts, err)
		return nil
	}
}

func (rm *redisManager) updateStatefulSet(cc *v1alpha1.CodisCluster, sts *apps.StatefulSet) error {
	if err := rm.client.Update(context.TODO(), sts); err != nil {
		rm.recordStatefulSetEvent("update", cc, sts, err)
		return err
	} else {
		rm.recordStatefulSetEvent("update", cc, sts, err)
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
			log.Infof("ns:%s,ccName:%s,get codis server svc err:%s", ns, ccName, err)
			return err
		}
	} else {
		log.Infof("ns:%s,ccName:%s,get codis server svc ok", ns, ccName)
	}
	//to do
	return nil
}

func (rm *redisManager) populateEnvVar(cc *v1alpha1.CodisCluster) []corev1.EnvVar {
	ccName := cc.GetName()
	var envVarList []corev1.EnvVar
	envVarList = append(envVarList, corev1.EnvVar{Name: "CODIS_PATH", Value: "/gopath/src/github.com/CodisLabs/codis"})
	envVarList = append(envVarList, corev1.EnvVar{Name: "PRODUCT_NAME", Value: ccName})
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

	//VolumeClaimTemplates []v1.PersistentVolumeClaim `json:"volumeClaimTemplates,omitempty" protobuf:"bytes,4,rep,name=volumeClaimTemplates"`
	pvcList := []corev1.PersistentVolumeClaim{}
	volumeList := []corev1.Volume{}
	//Volumes: []corev1.Volume{corev1.Volume{Name: rm.getCodisServerVolumeName(cc), EmptyDir: &corev1.EmptyDirVolumeSource{}}},
	if cc.Spec.CodisServer.StorageClassName != nil {
		pvc := corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:            rm.getCodisServerVolumeName(ccName),
				Namespace:       ns,
				OwnerReferences: []metav1.OwnerReference{utils.GetOwnerRef(cc)},
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes:      []corev1.PersistentVolumeAccessMode{"ReadWriteOnce"},
				Resources:        utils.ResourceRequirement(cc.Spec.CodisServer.ContainerSpec, true),
				StorageClassName: cc.Spec.CodisServer.StorageClassName,
			},
		}
		pvcList = append(pvcList, pvc)
	} else {
		// EmptyDir represents a temporary directory that shares a pod's lifetime.
		volume := corev1.Volume{Name: rm.getCodisServerVolumeName(ccName), VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}
		volumeList = append(volumeList, volume)
	}
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
			VolumeClaimTemplates: pvcList,
			//RollingUpdateStatefulSetStrategyType = "RollingUpdate"
			//RollingUpdate *RollingUpdateStatefulSetStrategy `json:"rollingUpdate,omitempty" protobuf:"bytes,2,opt,name=rollingUpdate"`
			UpdateStrategy: apps.StatefulSetUpdateStrategy{Type: apps.RollingUpdateStatefulSetStrategyType, RollingUpdate: &apps.RollingUpdateStatefulSetStrategy{Partition: &cc.Spec.CodisServer.Partition}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: codisServerLabels},
				Spec: corev1.PodSpec{
					Volumes: volumeList,
					Containers: []corev1.Container{
						{
							Name:            "codis-server",
							Image:           cc.Spec.CodisServer.Image,
							ImagePullPolicy: "IfNotPresent",
							Command:         []string{"codis-server"},
							Args:            []string{"$(CODIS_PATH)/config/redis.conf"},
							Env:             envVarList,
							Resources:       utils.ResourceRequirement(cc.Spec.CodisServer.ContainerSpec, false),
							Ports:           []corev1.ContainerPort{{Name: "redis", ContainerPort: 6379}},
							VolumeMounts:    []corev1.VolumeMount{corev1.VolumeMount{Name: rm.getCodisServerVolumeName(ccName), MountPath: "/data"}},
						},
					},
				},
			},
		},
	}
	log.Infof("deploy codis-server image:%s", cc.Spec.CodisServer.Image)
	return sts
}

func (rm *redisManager) syncCodisServerStatefulSet(cc *v1alpha1.CodisCluster) error {
	ns := cc.GetNamespace()
	ccName := cc.GetName()

	newCodisServerStatefulSet := rm.getNewCodisServerStatefulSet(cc)
	oldCodisServerStatefulSet := &apps.StatefulSet{}
	if err := rm.client.Get(context.TODO(), types.NamespacedName{Name: rm.getStatefulSetName(ccName), Namespace: ns}, oldCodisServerStatefulSet); err != nil {
		if errors.IsNotFound(err) {
			err = utils.SetStsLastAppliedConfig(newCodisServerStatefulSet)
			if err != nil {
				return nil
			}
			return rm.createStatefulSet(cc, newCodisServerStatefulSet)
		} else {
			log.Infof("ns:%s,ccName:%s,get codis server statefulset err:%s", ns, ccName, err)
			return err
		}
	}
	log.Infof("ns:%s,ccName:%s,get codis server statefulset succ", ns, ccName)
	if equal, err := utils.StatefulSetEqual(newCodisServerStatefulSet, oldCodisServerStatefulSet); err != nil {
		log.Errorf("ns:%s,ccName:%s,statefulset equal err:%v", ns, ccName, err)
		return err
	} else {
		if !equal {
			sts := *oldCodisServerStatefulSet
			sts.Spec = newCodisServerStatefulSet.Spec
			if err = utils.SetStsLastAppliedConfig(&sts); err != nil {
				log.Errorf("ns:%s,ccName:%s,set statefulset annotation err:%v", ns, ccName, err)
				return err
			}
			if err = rm.updateStatefulSet(cc, &sts); err != nil {
				log.Errorf("ns:%s,ccName:%s,update statefulset err:%v", ns, ccName, err)
				return err
			}
			log.Errorf("ns:%s,ccName:%s,statefulset change,update succ", ns, ccName)
		}
	}
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
