package dashboard

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
	log "github.com/golang/glog"
	"strings"
)

type dashboardManager struct {
	client   client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
}

func NewDashboardManager(client client.Client, scheme *runtime.Scheme, record record.EventRecorder) manager.Manager {
	return &dashboardManager{client, scheme, record}
}

func (dm *dashboardManager) Reconcile(cc *v1alpha1.CodisCluster) error {
	// Reconcile Codis Dashboard Service
	if err := dm.syncCodisDashboardService(cc); err != nil {
		return err
	}

	// Reconcile Codis Dashboard StatefulSet
	return dm.syncCodisDashboardStatefulSet(cc)
}

func (dm *dashboardManager) getSvcName(ccName string) string {
	return ccName + "-dashboard"
}

func (dm *dashboardManager) getStatefulSetName(ccName string) string {
	return ccName + "-dashboard"
}

func (dm *dashboardManager) recordServiceEvent(verb string, cc *v1alpha1.CodisCluster, svc *corev1.Service, err error) {
	ccName := cc.Name
	svcName := svc.Name
	if err == nil {
		reason := fmt.Sprintf("Successful %s", strings.Title(verb))
		msg := fmt.Sprintf("%s Service %s in CodisCluster %s successful",
			strings.ToLower(verb), svcName, ccName)
		dm.recorder.Event(cc, corev1.EventTypeNormal, reason, msg)
		log.Infof("%s,%s", reason, msg)
	} else {
		reason := fmt.Sprintf("Failed %s", strings.Title(verb))
		msg := fmt.Sprintf("%s Service %s in CodisCluster %s failed error: %s",
			strings.ToLower(verb), svcName, ccName, err)
		dm.recorder.Event(cc, corev1.EventTypeWarning, reason, msg)
		log.Infof("%s,%s", reason, msg)
	}
}

func (dm *dashboardManager) recordStatefulSetEvent(verb string, cc *v1alpha1.CodisCluster, svc *apps.StatefulSet, err error) {
	ccName := cc.Name
	svcName := svc.Name
	if err == nil {
		reason := fmt.Sprintf("Successful %s", strings.Title(verb))
		msg := fmt.Sprintf("%s StatefulSet %s in CodisCluster %s successful",
			strings.ToLower(verb), svcName, ccName)
		dm.recorder.Event(cc, corev1.EventTypeNormal, reason, msg)
		log.Infof("%s,%s", reason, msg)
	} else {
		reason := fmt.Sprintf("Failed %s", strings.Title(verb))
		msg := fmt.Sprintf("%s StatefulSet %s in CodisCluster %s failed error: %s",
			strings.ToLower(verb), svcName, ccName, err)
		dm.recorder.Event(cc, corev1.EventTypeWarning, reason, msg)
		log.Infof("%s,%s", reason, msg)
	}
}

func (dm *dashboardManager) createService(cc *v1alpha1.CodisCluster, svc *corev1.Service) error {
	if err := dm.client.Create(context.TODO(), svc); err != nil {
		dm.recordServiceEvent("create", cc, svc, err)
		return err
	} else {
		dm.recordServiceEvent("create", cc, svc, err)
		return nil
	}
}

func (dm *dashboardManager) createStatefulSet(cc *v1alpha1.CodisCluster, sts *apps.StatefulSet) error {
	if err := dm.client.Create(context.TODO(), sts); err != nil {
		dm.recordStatefulSetEvent("create", cc, sts, err)
		return err
	}
	dm.recordStatefulSetEvent("create", cc, sts, nil)
	return nil
}

func (dm *dashboardManager) updateStatefulSet(cc *v1alpha1.CodisCluster, sts *apps.StatefulSet) error {
	if err := dm.client.Update(context.TODO(), sts); err != nil {
		dm.recordStatefulSetEvent("update", cc, sts, err)
		return err
	}
	dm.recordStatefulSetEvent("update", cc, sts, nil)
	return nil
}

func (dm *dashboardManager) syncCodisDashboardService(cc *v1alpha1.CodisCluster) error {
	ns := cc.GetNamespace()
	ccName := cc.GetName()

	newSvc := dm.getNewCodisDashboardService(cc)

	oldSvc := &corev1.Service{}
	if err := dm.client.Get(context.TODO(), types.NamespacedName{Name: dm.getSvcName(ccName), Namespace: ns}, oldSvc); err != nil {
		if errors.IsNotFound(err) {
			return dm.createService(cc, newSvc)
		} else {
			log.Infof("ns:%s,ccName:%s,get dashboard svc err:%s", ns, ccName, err)
			return err
		}
	} else {
		log.Infof("ns:%s,ccName:%s,get dashboard svc ok", ns, ccName)
	}
	//to do
	return nil
}

func (dm *dashboardManager) populateEnvVar(cc *v1alpha1.CodisCluster) []corev1.EnvVar {
	var envVarList []corev1.EnvVar
	ccName := cc.GetName()
	envVarList = append(envVarList, corev1.EnvVar{Name: "CODIS_PATH", Value: "/gopath/src/github.com/CodisLabs/codis"})
	envVarList = append(envVarList, corev1.EnvVar{Name: "PRODUCT_NAME", Value: ccName})
	envVarList = append(envVarList, corev1.EnvVar{Name: "COORDINATOR_NAME", Value: cc.Spec.CoordinatorName})
	envVarList = append(envVarList, corev1.EnvVar{Name: "COORDINATOR_ADDR", Value: cc.Spec.CoordinatorAddr})
	envVarList = append(envVarList, corev1.EnvVar{Name: "POD_IP", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "status.podIP"}}})
	envVarList = append(envVarList, corev1.EnvVar{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}})
	return envVarList
}

func (dm *dashboardManager) getNewCodisDashboardStatefulSet(cc *v1alpha1.CodisCluster) *apps.StatefulSet {
	ns := cc.GetNamespace()
	ccName := cc.GetName()

	codisDashboardLabels := map[string]string{
		"component":   "codis-dashboard",
		"clusterName": ccName,
	}

	envVarList := dm.populateEnvVar(cc)

	sts := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            dm.getStatefulSetName(ccName),
			Namespace:       ns,
			OwnerReferences: []metav1.OwnerReference{utils.GetOwnerRef(cc)},
		},
		Spec: apps.StatefulSetSpec{
			Replicas:    &cc.Spec.CodisDashboard.Replicas,
			ServiceName: dm.getSvcName(ccName),
			Selector: &metav1.LabelSelector{
				MatchLabels: codisDashboardLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: codisDashboardLabels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "codis-dashboard",
							Image:           cc.Spec.CodisDashboard.Image,
							ImagePullPolicy: "IfNotPresent",
							Command:         []string{"codis-dashboard"},
							Args:            []string{"--$(COORDINATOR_NAME)", "$(COORDINATOR_ADDR)", "-c", "$(CODIS_PATH)/config/dashboard.toml", "--host-admin", "$(POD_IP):18080", "--product_name", "$(PRODUCT_NAME)", "--product_auth", cc.Spec.CodisDashboard.ProductAuth},
							Env:             envVarList,
							Ports:           []corev1.ContainerPort{{Name: "dashboard", ContainerPort: 18080}},
						},
					},
				},
			},
		},
	}
	log.Infof("deploy codis dashboard image:%s", cc.Spec.CodisDashboard.Image)
	return sts
}

func (dm *dashboardManager) syncCodisDashboardStatefulSet(cc *v1alpha1.CodisCluster) error {
	ns := cc.GetNamespace()
	ccName := cc.GetName()

	newCodisDashboardStatefulSet := dm.getNewCodisDashboardStatefulSet(cc)
	oldCodisDashboardStatefulSet := &apps.StatefulSet{}
	if err := dm.client.Get(context.TODO(), types.NamespacedName{Name: dm.getStatefulSetName(ccName), Namespace: ns}, oldCodisDashboardStatefulSet); err != nil {
		if errors.IsNotFound(err) {
			err = utils.SetStsLastAppliedConfig(newCodisDashboardStatefulSet)
			if err != nil {
				return nil
			}
			return dm.createStatefulSet(cc, newCodisDashboardStatefulSet)
		} else {
			log.Infof("ns:%s,ccName:%s,get dashboard statefulset err:%s", ns, ccName, err)
			return err
		}
	}
	log.Infof("ns:%s,ccName:%s,get dashboard statefulset info succ", ns, ccName)
	if equal, err := utils.StatefulSetEqual(newCodisDashboardStatefulSet, oldCodisDashboardStatefulSet); err != nil {
		log.Errorf("ns:%s,ccName:%s,statefulset equal err:%v", ns, ccName, err)
		return err
	} else {
		if !equal {
			sts := *oldCodisDashboardStatefulSet
			sts.Spec = newCodisDashboardStatefulSet.Spec
			if err = utils.SetStsLastAppliedConfig(&sts); err != nil {
				log.Errorf("ns:%s,ccName:%s,set statefulset annotation err:%v", ns, ccName, err)
				return err
			}
			if err = dm.updateStatefulSet(cc, &sts); err != nil {
				log.Errorf("ns:%s,ccName:%s,update statefulset err:%v", ns, ccName, err)
				return err
			}
			log.Errorf("ns:%s,ccName:%s,statefulset change,update succ", ns, ccName)
		}
	}
	return nil
}

func (dm *dashboardManager) getNewCodisDashboardService(cc *v1alpha1.CodisCluster) *corev1.Service {
	ns := cc.Namespace
	ccName := cc.Name

	codisDashboardLabels := map[string]string{
		"component":   "codis-dashboard",
		"clusterName": ccName,
	}
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            dm.getSvcName(ccName),
			Namespace:       ns,
			Labels:          codisDashboardLabels,
			OwnerReferences: []metav1.OwnerReference{utils.GetOwnerRef(cc)},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Ports: []corev1.ServicePort{
				{
					Name:       "dashboard-port",
					Port:       18080,
					TargetPort: intstr.FromInt(18080),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: codisDashboardLabels,
		},
	}
}
