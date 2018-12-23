package proxy

import (
	"context"
	"fmt"
	log "github.com/golang/glog"
	"github.com/tangcong/codis-operator/pkg/apis/codis/v1alpha1"
	"github.com/tangcong/codis-operator/pkg/manager"
	"github.com/tangcong/codis-operator/pkg/utils"
	apps "k8s.io/api/apps/v1"
	as "k8s.io/api/autoscaling/v1"
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
	if err := pm.syncCodisProxyDeployment(cc); err != nil {
		return err
	}

	if cc.Spec.CodisProxy.HpaSpec.MinReplicas != 0 {
		if err := pm.syncCodisProxyHPA(cc); err != nil {
			return err
		}
	}
	return nil
}

func (pm *proxyManager) getSvcName(ccName string) string {
	return ccName + "-proxy"
}

func (pm *proxyManager) getDeployName(ccName string) string {
	return ccName + "-proxy"
}

func (pm *proxyManager) getHPAName(ccName string) string {
	return ccName + "-hpa"
}

func (pm *proxyManager) recordServiceEvent(verb string, cc *v1alpha1.CodisCluster, svc *corev1.Service, err error) {
	ccName := cc.Name
	svcName := svc.Name
	if err == nil {
		reason := fmt.Sprintf("Successful %s", strings.Title(verb))
		msg := fmt.Sprintf("%s Service %s in CodisCluster %s successful",
			strings.ToLower(verb), svcName, ccName)
		pm.recorder.Event(cc, corev1.EventTypeNormal, reason, msg)
		log.Infof("%s,%s", reason, msg)
	} else {
		reason := fmt.Sprintf("Failed %s", strings.Title(verb))
		msg := fmt.Sprintf("%s Service %s in CodisCluster %s failed error: %s",
			strings.ToLower(verb), svcName, ccName, err)
		pm.recorder.Event(cc, corev1.EventTypeWarning, reason, msg)
		log.Infof("%s,%s", reason, msg)
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
		log.Infof("%s,%s", reason, msg)
	} else {
		reason := fmt.Sprintf("Failed %s", strings.Title(verb))
		msg := fmt.Sprintf("%s Deploy %s in CodisCluster %s failed error: %s",
			strings.ToLower(verb), svcName, ccName, err)
		pm.recorder.Event(cc, corev1.EventTypeWarning, reason, msg)
		log.Infof("%s,%s", reason, msg)
	}
}

func (pm *proxyManager) recordHPAEvent(verb string, cc *v1alpha1.CodisCluster, hpa *as.HorizontalPodAutoscaler, err error) {
	ccName := cc.Name
	svcName := hpa.Name
	if err == nil {
		reason := fmt.Sprintf("Successful %s", strings.Title(verb))
		msg := fmt.Sprintf("%s HPA %s in CodisCluster %s successful",
			strings.ToLower(verb), svcName, ccName)
		pm.recorder.Event(cc, corev1.EventTypeNormal, reason, msg)
		log.Infof("%s,%s", reason, msg)
	} else {
		reason := fmt.Sprintf("Failed %s", strings.Title(verb))
		msg := fmt.Sprintf("%s HPA %s in CodisCluster %s failed error: %s",
			strings.ToLower(verb), svcName, ccName, err)
		pm.recorder.Event(cc, corev1.EventTypeWarning, reason, msg)
		log.Infof("%s,%s", reason, msg)
	}
}

func (pm *proxyManager) createService(cc *v1alpha1.CodisCluster, svc *corev1.Service) error {
	if err := pm.client.Create(context.TODO(), svc); err != nil {
		pm.recordServiceEvent("create", cc, svc, err)
		return err
	}
	pm.recordServiceEvent("create", cc, svc, nil)
	return nil
}

func (pm *proxyManager) updateService(cc *v1alpha1.CodisCluster, svc *corev1.Service) error {
	if err := pm.client.Update(context.TODO(), svc); err != nil {
		pm.recordServiceEvent("update", cc, svc, err)
		return err
	}
	pm.recordServiceEvent("update", cc, svc, nil)
	return nil
}

func (pm *proxyManager) createDeploy(cc *v1alpha1.CodisCluster, deploy *apps.Deployment) error {
	if err := pm.client.Create(context.TODO(), deploy); err != nil {
		pm.recordDeployEvent("create", cc, deploy, err)
		return err
	}
	pm.recordDeployEvent("create", cc, deploy, nil)
	return nil
}

func (pm *proxyManager) createHPA(cc *v1alpha1.CodisCluster, hpa *as.HorizontalPodAutoscaler) error {
	if err := pm.client.Create(context.TODO(), hpa); err != nil {
		pm.recordHPAEvent("create", cc, hpa, err)
		return err
	}
	pm.recordHPAEvent("create", cc, hpa, nil)
	return nil
}

func (pm *proxyManager) updateDeploy(cc *v1alpha1.CodisCluster, deploy *apps.Deployment) error {
	if err := pm.client.Update(context.TODO(), deploy); err != nil {
		pm.recordDeployEvent("update", cc, deploy, err)
		return err
	}
	pm.recordDeployEvent("update", cc, deploy, nil)
	return nil
}

func (pm *proxyManager) updateHPA(cc *v1alpha1.CodisCluster, hpa *as.HorizontalPodAutoscaler) error {
	if err := pm.client.Update(context.TODO(), hpa); err != nil {
		pm.recordHPAEvent("update", cc, hpa, err)
		return err
	}
	pm.recordHPAEvent("update", cc, hpa, nil)
	return nil
}

func (pm *proxyManager) syncCodisProxyService(cc *v1alpha1.CodisCluster) error {
	ns := cc.GetNamespace()
	ccName := cc.GetName()

	newSvc := pm.getNewCodisProxyService(cc)

	oldSvc := &corev1.Service{}
	if err := pm.client.Get(context.TODO(), types.NamespacedName{Name: pm.getSvcName(ccName), Namespace: ns}, oldSvc); err != nil {
		if errors.IsNotFound(err) {
			err = utils.SetServiceLastAppliedConfig(newSvc)
			if err != nil {
				return nil
			}
			return pm.createService(cc, newSvc)
		} else {
			log.Infof("ns:%s,ccName:%s,get proxy svc err:%s", ns, ccName, err)
			return err
		}
	}
	log.Infof("ns:%s,ccName:%s,get proxy svc ok", ns, ccName)
	//warning: if type is NodePort,UpdateService will make node port change,so serviceEqual only compares selector label.
	if equal, err := utils.ServiceEqual(newSvc, oldSvc); err != nil {
		log.Errorf("ns:%s,ccName:%s,proxy svc equal err:%s", ns, ccName, err)
		return err
	} else {
		if !equal {
			svc := *oldSvc
			svc.Spec = newSvc.Spec
			//ensure svc cluster ip sticky
			svc.Spec.ClusterIP = oldSvc.Spec.ClusterIP
			if err = utils.SetServiceLastAppliedConfig(&svc); err != nil {
				log.Errorf("ns:%s,ccName:%s,set proxy svc annotation err:%v", ns, ccName, err)
				return err
			}
			if err = pm.updateService(cc, &svc); err != nil {
				log.Errorf("ns:%s,ccName:%s,update svc err:%v", ns, ccName, err)
				return err
			}
			log.Infof("ns:%s,ccName:%s,update proxy svc succ", ns, ccName)
		}
	}
	return nil
}

func (pm *proxyManager) populateEnvVar(cc *v1alpha1.CodisCluster) []corev1.EnvVar {
	ccName := cc.GetName()
	var envVarList []corev1.EnvVar
	envVarList = append(envVarList, corev1.EnvVar{Name: "CODIS_PATH", Value: "/gopath/src/github.com/CodisLabs/codis"})
	envVarList = append(envVarList, corev1.EnvVar{Name: "PRODUCT_NAME", Value: ccName})
	envVarList = append(envVarList, corev1.EnvVar{Name: "POD_IP", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "status.podIP"}}})
	envVarList = append(envVarList, corev1.EnvVar{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}})
	envVarList = append(envVarList, corev1.EnvVar{Name: "DASHBOARD", Value: utils.GetDashboardSvr(cc)})
	//fix bug,we have to recreate pod when dashboard backend storage addr changed.
	envVarList = append(envVarList, corev1.EnvVar{Name: "COORDINATOR_ADDR", Value: cc.Spec.CoordinatorAddr})
	envVarList = append(envVarList, corev1.EnvVar{Name: "COORDINATOR_NAME", Value: cc.Spec.CoordinatorName})
	return envVarList
}

func (pm *proxyManager) getNewCodisProxyDeployment(cc *v1alpha1.CodisCluster) *apps.Deployment {
	ns := cc.GetNamespace()
	ccName := cc.GetName()

	codisProxyLabels := map[string]string{
		"component":   "codis-proxy",
		"clusterName": ccName,
	}

	envVarList := pm.populateEnvVar(cc)
	maxUnavailable := intstr.FromInt(cc.Spec.CodisProxy.MaxUnavailable)
	maxSurge := intstr.FromInt(cc.Spec.CodisProxy.MaxSurge)
	//Strategy: apps.DeploymentStrategy{Type: apps.RollingUpdateDeploymentStrategyType, RollingUpdate: apps.RollingUpdateDeployment{MaxUnavailable: &MaxUnavailable, MaxSurge: &(intstr.FromInt(cc.Spec.CodisProxy.MaxSurge))}},

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

			//MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty" protobuf:"bytes,1,opt,name=maxUnavailable"`
			//MaxSurge *intstr.IntOrString `json:"maxSurge,omitempty" protobuf:"bytes,2,opt,name=maxSurge"`
			Strategy: apps.DeploymentStrategy{Type: apps.RollingUpdateDeploymentStrategyType, RollingUpdate: &apps.RollingUpdateDeployment{MaxUnavailable: &maxUnavailable, MaxSurge: &maxSurge}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: codisProxyLabels},
				Spec: corev1.PodSpec{
					NodeSelector: cc.Spec.CodisProxy.NodeSelector,
					Tolerations:  cc.Spec.CodisProxy.Tolerations,
					Containers: []corev1.Container{
						{
							Name:            "codis-proxy",
							Image:           cc.Spec.CodisProxy.Image,
							ImagePullPolicy: "IfNotPresent",

							Command: []string{"codis-proxy"},
							Args:    []string{"-c", "$(CODIS_PATH)/config/proxy.toml", "--dashboard", "$(DASHBOARD):18080", "--host-admin", "$(POD_IP):11080", "--host-proxy", "$(POD_IP):19000", "--product_name", "$(PRODUCT_NAME)", "--product_auth", cc.Spec.CodisDashboard.ProductAuth, "--session_auth", cc.Spec.CodisProxy.SessionAuth},
							Env:     envVarList,
							Ports: []corev1.ContainerPort{
								{Name: "admin-port", ContainerPort: 11080},
								{Name: "proxy-port", ContainerPort: 19000},
							},
							Resources: utils.ResourceRequirement(cc.Spec.CodisProxy.ContainerSpec, false),
						},
					},
				},
			},
		},
	}
	log.Infof("deploy codis proxy image:%s", cc.Spec.CodisProxy.Image)
	return deploy
}

func (pm *proxyManager) syncCodisProxyDeployment(cc *v1alpha1.CodisCluster) error {
	ns := cc.GetNamespace()
	ccName := cc.GetName()

	newCodisProxyDeploy := pm.getNewCodisProxyDeployment(cc)
	oldCodisProxyDeploy := &apps.Deployment{}
	if err := pm.client.Get(context.TODO(), types.NamespacedName{Name: pm.getDeployName(ccName), Namespace: ns}, oldCodisProxyDeploy); err != nil {
		if errors.IsNotFound(err) {
			err = utils.SetDeploymentLastAppliedConfig(newCodisProxyDeploy)
			if err != nil {
				return nil
			}
			return pm.createDeploy(cc, newCodisProxyDeploy)
		} else {
			log.Infof("ns:%s,ccName:%s,get proxy deployment err:%s", ns, ccName, err)
			return err
		}
	}
	log.Infof("ns:%s,ccName:%s,get proxy deployment info succ", ns, ccName)
	if equal, err := utils.DeploymentEqual(newCodisProxyDeploy, oldCodisProxyDeploy); err != nil {
		log.Errorf("ns:%s,ccName:%s,deployment equal err:%v", ns, ccName, err)
		return err
	} else {
		if !equal {
			deploy := *oldCodisProxyDeploy
			deploy.Spec = newCodisProxyDeploy.Spec
			if err = utils.SetDeploymentLastAppliedConfig(&deploy); err != nil {
				log.Errorf("ns:%s,ccName:%s,set deployment annotation err:%v", ns, ccName, err)
				return err
			}
			if err = pm.updateDeploy(cc, &deploy); err != nil {
				log.Errorf("ns:%s,ccName:%s,update deployment err:%v", ns, ccName, err)
				return err
			}
			log.Errorf("ns:%s,ccName:%s,deployment change,update succ", ns, ccName)
		}
	}
	return nil
}

func (pm *proxyManager) syncCodisProxyHPA(cc *v1alpha1.CodisCluster) error {
	ns := cc.GetNamespace()
	ccName := cc.GetName()

	newCodisProxyHPA := pm.getNewCodisProxyHPA(cc)
	oldCodisProxyHPA := &as.HorizontalPodAutoscaler{}
	if err := pm.client.Get(context.TODO(), types.NamespacedName{Name: pm.getHPAName(ccName), Namespace: ns}, oldCodisProxyHPA); err != nil {
		if errors.IsNotFound(err) {
			err = utils.SetHPALastAppliedConfig(newCodisProxyHPA)
			if err != nil {
				return nil
			}
			return pm.createHPA(cc, newCodisProxyHPA)
		} else {
			log.Infof("ns:%s,ccName:%s,get proxy hpa err:%s", ns, ccName, err)
			return err
		}
	}
	log.Infof("ns:%s,ccName:%s,get proxy hpa info succ", ns, ccName)
	if equal, err := utils.HPAEqual(newCodisProxyHPA, oldCodisProxyHPA); err != nil {
		log.Errorf("ns:%s,ccName:%s,hpa equal err:%v", ns, ccName, err)
		return err
	} else {
		if !equal {
			hpa := *oldCodisProxyHPA
			hpa.Spec = newCodisProxyHPA.Spec
			if err = utils.SetHPALastAppliedConfig(&hpa); err != nil {
				log.Errorf("ns:%s,ccName:%s,set hpa annotation err:%v", ns, ccName, err)
				return err
			}
			if err = pm.updateHPA(cc, &hpa); err != nil {
				log.Errorf("ns:%s,ccName:%s,update hpa err:%v", ns, ccName, err)
				return err
			}
			log.Errorf("ns:%s,ccName:%s,hpa change,update succ", ns, ccName)
		}
	}
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
				{
					Name:       "admin-port",
					Port:       11080,
					TargetPort: intstr.FromInt(11080),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: codisProxyLabels,
		},
	}
}

func (pm *proxyManager) getNewCodisProxyHPA(cc *v1alpha1.CodisCluster) *as.HorizontalPodAutoscaler {
	ns := cc.GetNamespace()
	ccName := cc.GetName()

	deploy := &as.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:            pm.getHPAName(ccName),
			Namespace:       ns,
			OwnerReferences: []metav1.OwnerReference{utils.GetOwnerRef(cc)},
		},
		Spec: as.HorizontalPodAutoscalerSpec{
			ScaleTargetRef:                 as.CrossVersionObjectReference{APIVersion: "apps/v1", Kind: "Deployment", Name: pm.getDeployName(ccName)},
			MinReplicas:                    &cc.Spec.CodisProxy.HpaSpec.MinReplicas,
			MaxReplicas:                    cc.Spec.CodisProxy.HpaSpec.MaxReplicas,
			TargetCPUUtilizationPercentage: &cc.Spec.CodisProxy.HpaSpec.CpuUsedThreshold,
		},
	}
	log.Infof("codis proxy hpa:%v", cc.Spec.CodisProxy.HpaSpec)
	return deploy
}
