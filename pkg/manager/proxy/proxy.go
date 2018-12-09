package proxy

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

type proxyManager struct {
	client   client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
}

func NewProxyManager(client client.Client, scheme *runtime.Scheme, record record.EventRecorder) manager.Manager {
	return &proxyManager{client, scheme, record}
}

func (pm *proxyManager) Reconcile(cc *v1alpha1.CodisCluster) error {
	// Reconcile Codis Proxy Service
	if err := pm.syncCodisProxyService(cc); err != nil {
		return err
	}

	// Reconcile Codis Proxy Deployment
	return pm.syncCodisProxyDeployment(cc)
}

func (pm *proxyManager) getSvcName(ccName string) string {
	return ccName + "-proxy"
}

func (pm *proxyManager) getDeployName(ccName string) string {
	return ccName + "-proxy"
}

func (pm *proxyManager) recordServiceEvent(verb string, cc *v1alpha1.CodisCluster, svc *corev1.Service, err error) {
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

func (pm *proxyManager) recordDeployEvent(verb string, cc *v1alpha1.CodisCluster, svc *apps.Deployment, err error) {
	ccName := cc.Name
	svcName := svc.Name
	if err == nil {
		reason := fmt.Sprintf("Successful %s", strings.Title(verb))
		msg := fmt.Sprintf("%s Deploy %s in CodisCluster %s successful",
			strings.ToLower(verb), svcName, ccName)
		pm.recorder.Event(cc, corev1.EventTypeNormal, reason, msg)
		log.Printf("%s,%s", reason, msg)
	} else {
		reason := fmt.Sprintf("Failed %s", strings.Title(verb))
		msg := fmt.Sprintf("%s Deploy %s in CodisCluster %s failed error: %s",
			strings.ToLower(verb), svcName, ccName, err)
		pm.recorder.Event(cc, corev1.EventTypeWarning, reason, msg)
		log.Printf("%s,%s", reason, msg)
	}
}

func (pm *proxyManager) createService(cc *v1alpha1.CodisCluster, svc *corev1.Service) error {
	if err := pm.client.Create(context.TODO(), svc); err != nil {
		pm.recordServiceEvent("create", cc, svc, err)
		return err
	} else {
		pm.recordServiceEvent("create", cc, svc, err)
		return nil
	}
}

func (pm *proxyManager) createDeploy(cc *v1alpha1.CodisCluster, deploy *apps.Deployment) error {
	if err := pm.client.Create(context.TODO(), deploy); err != nil {
		pm.recordDeployEvent("create", cc, deploy, err)
		return err
	} else {
		pm.recordDeployEvent("create", cc, deploy, err)
		return nil
	}
}

func (pm *proxyManager) syncCodisProxyService(cc *v1alpha1.CodisCluster) error {
	ns := cc.GetNamespace()
	ccName := cc.GetName()

	newSvc := pm.getNewCodisProxyService(cc)

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

func (pm *proxyManager) getNewCodisProxyDeployment(cc *v1alpha1.CodisCluster) *apps.Deployment {
	ns := cc.GetNamespace()
	ccName := cc.GetName()

	codisProxyLabels := map[string]string{
		"component":   "codis-proxy",
		"clusterName": ccName,
	}

	deploy := &apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:            pm.getDeployName(ccName),
			Namespace:       ns,
			OwnerReferences: []metav1.OwnerReference{utils.GetOwnerRef(cc)},
		},
		Spec: apps.DeploymentSpec{
			Replicas: &cc.Spec.CodisProxy.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: codisProxyLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: codisProxyLabels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "codis-proxy",
							Image:           cc.Spec.CodisProxy.Image,
							ImagePullPolicy: "IfNotPresent",
						},
					},
				},
			},
		},
	}
	log.Printf("codis proxy image:%s", cc.Spec.CodisProxy.Image)
	return deploy
}

func (pm *proxyManager) syncCodisProxyDeployment(cc *v1alpha1.CodisCluster) error {
	ns := cc.GetNamespace()
	ccName := cc.GetName()

	newCodisProxyDeploy := pm.getNewCodisProxyDeployment(cc)
	oldCodisProxyDeploy := &apps.Deployment{}
	if err := pm.client.Get(context.TODO(), types.NamespacedName{Name: pm.getDeployName(ccName), Namespace: ns}, oldCodisProxyDeploy); err != nil {
		if errors.IsNotFound(err) {
			return pm.createDeploy(cc, newCodisProxyDeploy)
		} else {
			log.Printf("ns:%s,ccName:%s,get svc err:%s", ns, ccName, err)
			return err
		}
	} else {
		log.Printf("ns:%s,ccName:%s,get svc info:%+v", ns, ccName, oldCodisProxyDeploy)
	}
	//to do
	return nil
}

func (pm *proxyManager) getNewCodisProxyService(cc *v1alpha1.CodisCluster) *corev1.Service {
	ns := cc.Namespace
	ccName := cc.Name

	codisProxyLabels := map[string]string{
		"component":   "codis-proxy",
		"clusterName": ccName,
	}
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            pm.getSvcName(ccName),
			Namespace:       ns,
			Labels:          codisProxyLabels,
			OwnerReferences: []metav1.OwnerReference{utils.GetOwnerRef(cc)},
		},
		Spec: corev1.ServiceSpec{
			Type: "NodePort",
			Ports: []corev1.ServicePort{
				{
					Name:       "proxy-port",
					Port:       19000,
					TargetPort: intstr.FromInt(19000),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: codisProxyLabels,
		},
	}
}
