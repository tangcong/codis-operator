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
	"log"
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

func (pm *dashboardManager) Reconcile(cc *v1alpha1.CodisCluster) error {
	// Reconcile Codis Dashboard Service
	if err := pm.syncCodisDashboardService(cc); err != nil {
		return err
	}

	// Reconcile Codis Dashboard StatefulSet
	return pm.syncCodisDashboardStatefulSet(cc)
}

func (pm *dashboardManager) getSvcName(ccName string) string {
	return ccName + "-dashboard"
}

func (pm *dashboardManager) getStatefulSetName(ccName string) string {
	return ccName + "-dashboard"
}

func (pm *dashboardManager) recordServiceEvent(verb string, cc *v1alpha1.CodisCluster, svc *corev1.Service, err error) {
	ccName := cc.Name
	svcName := svc.Name
	if err == nil {
		reason := fmt.Sprintf("Successful %s", strings.Title(verb))
		msg := fmt.Sprintf("%s Service %s in CodisCluster %s successful",
			strings.ToLower(verb), svcName, ccName)
		pm.recorder.Event(cc, corev1.EventTypeNormal, reason, msg)
		log.Printf("%s,%s", reason, msg)
	} else {
		reason := fmt.Sprintf("Failed %s", strings.Title(verb))
		msg := fmt.Sprintf("%s Service %s in CodisCluster %s failed error: %s",
			strings.ToLower(verb), svcName, ccName, err)
		pm.recorder.Event(cc, corev1.EventTypeWarning, reason, msg)
		log.Printf("%s,%s", reason, msg)
	}
}

func (pm *dashboardManager) recordStatefulSetEvent(verb string, cc *v1alpha1.CodisCluster, svc *apps.StatefulSet, err error) {
	ccName := cc.Name
	svcName := svc.Name
	if err == nil {
		reason := fmt.Sprintf("Successful %s", strings.Title(verb))
		msg := fmt.Sprintf("%s StatefulSet %s in CodisCluster %s successful",
			strings.ToLower(verb), svcName, ccName)
		pm.recorder.Event(cc, corev1.EventTypeNormal, reason, msg)
		log.Printf("%s,%s", reason, msg)
	} else {
		reason := fmt.Sprintf("Failed %s", strings.Title(verb))
		msg := fmt.Sprintf("%s StatefulSet %s in CodisCluster %s failed error: %s",
			strings.ToLower(verb), svcName, ccName, err)
		pm.recorder.Event(cc, corev1.EventTypeWarning, reason, msg)
		log.Printf("%s,%s", reason, msg)
	}
}

func (pm *dashboardManager) createService(cc *v1alpha1.CodisCluster, svc *corev1.Service) error {
	if err := pm.client.Create(context.TODO(), svc); err != nil {
		pm.recordServiceEvent("create", cc, svc, err)
		return err
	} else {
		pm.recordServiceEvent("create", cc, svc, err)
		return nil
	}
}

func (pm *dashboardManager) createStatefulSet(cc *v1alpha1.CodisCluster, deploy *apps.StatefulSet) error {
	if err := pm.client.Create(context.TODO(), deploy); err != nil {
		pm.recordStatefulSetEvent("create", cc, deploy, err)
		return err
	} else {
		pm.recordStatefulSetEvent("create", cc, deploy, err)
		return nil
	}
}

func (pm *dashboardManager) syncCodisDashboardService(cc *v1alpha1.CodisCluster) error {
	ns := cc.GetNamespace()
	ccName := cc.GetName()

	newSvc := pm.getNewCodisDashboardService(cc)

	oldSvc := &corev1.Service{}
	if err := pm.client.Get(context.TODO(), types.NamespacedName{Name: pm.getSvcName(ccName), Namespace: ns}, oldSvc); err != nil {
		if errors.IsNotFound(err) {
			return pm.createService(cc, newSvc)
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

func (pm *dashboardManager) populateEnvVar(cc *v1alpha1.CodisCluster) []corev1.EnvVar {
	var envVarList []corev1.EnvVar
	envVarList = append(envVarList, corev1.EnvVar{Name: "CODIS_PATH", Value: "/gopath/src/github.com/CodisLabs/codis"})
	envVarList = append(envVarList, corev1.EnvVar{Name: "PRODUCT_NAME", Value: cc.Spec.ClusterName})
	envVarList = append(envVarList, corev1.EnvVar{Name: "POD_IP", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "status.podIP"}}})
	envVarList = append(envVarList, corev1.EnvVar{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}})
	return envVarList
}

func (pm *dashboardManager) getNewCodisDashboardStatefulSet(cc *v1alpha1.CodisCluster) *apps.StatefulSet {
	ns := cc.GetNamespace()
	ccName := cc.GetName()

	codisDashboardLabels := map[string]string{
		"component":   "codis-dashboard",
		"clusterName": ccName,
	}

	envVarList := pm.populateEnvVar(cc)

	sts := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            pm.getStatefulSetName(ccName),
			Namespace:       ns,
			OwnerReferences: []metav1.OwnerReference{utils.GetOwnerRef(cc)},
		},
		Spec: apps.StatefulSetSpec{
			Replicas: &cc.Spec.CodisDashboard.Replicas,
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
							Args:            []string{"-c", "$(CODIS_PATH)/config/dashboard.toml", "--host-admin", "$(POD_IP):18000", "--product_name", "$(PRODUCT_NAME)", "--product_auth", cc.Spec.CodisDashboard.ProductAuth},
							Env:             envVarList,
							Ports:           []corev1.ContainerPort{{Name: "dashboard", ContainerPort: 18080}},
						},
					},
				},
			},
		},
	}
	log.Printf("codis dashboard image:%s", cc.Spec.CodisDashboard.Image)
	return sts
}

func (pm *dashboardManager) syncCodisDashboardStatefulSet(cc *v1alpha1.CodisCluster) error {
	ns := cc.GetNamespace()
	ccName := cc.GetName()

	newCodisDashboardStatefulSet := pm.getNewCodisDashboardStatefulSet(cc)
	oldCodisDashboardStatefulSet := &apps.StatefulSet{}
	if err := pm.client.Get(context.TODO(), types.NamespacedName{Name: pm.getStatefulSetName(ccName), Namespace: ns}, oldCodisDashboardStatefulSet); err != nil {
		if errors.IsNotFound(err) {
			return pm.createStatefulSet(cc, newCodisDashboardStatefulSet)
		} else {
			log.Printf("ns:%s,ccName:%s,get svc err:%s", ns, ccName, err)
			return err
		}
	} else {
		log.Printf("ns:%s,ccName:%s,get svc info:%+v", ns, ccName, oldCodisDashboardStatefulSet)
	}
	//to do
	return nil
}

func (pm *dashboardManager) getNewCodisDashboardService(cc *v1alpha1.CodisCluster) *corev1.Service {
	ns := cc.Namespace
	ccName := cc.Name

	codisDashboardLabels := map[string]string{
		"component":   "codis-dashboard",
		"clusterName": ccName,
	}
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            pm.getSvcName(ccName),
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
