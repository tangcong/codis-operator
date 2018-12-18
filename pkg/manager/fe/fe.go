package fe

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

type feManager struct {
	client   client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
}

func NewFeManager(client client.Client, scheme *runtime.Scheme, record record.EventRecorder) manager.Manager {
	return &feManager{client, scheme, record}
}

func (fm *feManager) Reconcile(cc *v1alpha1.CodisCluster) error {
	// Reconcile Codis Fe Service
	if err := fm.syncCodisFeService(cc); err != nil {
		return err
	}

	// Reconcile Codis Fe Deployment
	return fm.syncCodisFeDeployment(cc)
}

func (fm *feManager) getSvcName(ccName string) string {
	return ccName + "-fe"
}

func (fm *feManager) getDeployName(ccName string) string {
	return ccName + "-fe"
}

func (fm *feManager) recordServiceEvent(verb string, cc *v1alpha1.CodisCluster, svc *corev1.Service, err error) {
	ccName := cc.Name
	svcName := svc.Name
	if err == nil {
		reason := fmt.Sprintf("Successful %s", strings.Title(verb))
		msg := fmt.Sprintf("%s Service %s in CodisCluster %s successful",
			strings.ToLower(verb), svcName, ccName)
		fm.recorder.Event(cc, corev1.EventTypeNormal, reason, msg)
		log.Infof("%s,%s", reason, msg)
	} else {
		reason := fmt.Sprintf("Failed %s", strings.Title(verb))
		msg := fmt.Sprintf("%s Service %s in CodisCluster %s failed error: %s",
			strings.ToLower(verb), svcName, ccName, err)
		fm.recorder.Event(cc, corev1.EventTypeWarning, reason, msg)
		log.Infof("%s,%s", reason, msg)
	}
}

func (fm *feManager) recordDeployEvent(verb string, cc *v1alpha1.CodisCluster, svc *apps.Deployment, err error) {
	ccName := cc.Name
	svcName := svc.Name
	if err == nil {
		reason := fmt.Sprintf("Successful %s", strings.Title(verb))
		msg := fmt.Sprintf("%s Deploy %s in CodisCluster %s successful",
			strings.ToLower(verb), svcName, ccName)
		fm.recorder.Event(cc, corev1.EventTypeNormal, reason, msg)
		log.Infof("%s,%s", reason, msg)
	} else {
		reason := fmt.Sprintf("Failed %s", strings.Title(verb))
		msg := fmt.Sprintf("%s Deploy %s in CodisCluster %s failed error: %s",
			strings.ToLower(verb), svcName, ccName, err)
		fm.recorder.Event(cc, corev1.EventTypeWarning, reason, msg)
		log.Infof("%s,%s", reason, msg)
	}
}

func (fm *feManager) createService(cc *v1alpha1.CodisCluster, svc *corev1.Service) error {
	if err := fm.client.Create(context.TODO(), svc); err != nil {
		fm.recordServiceEvent("create", cc, svc, err)
		return err
	} else {
		fm.recordServiceEvent("create", cc, svc, err)
		return nil
	}
}

func (fm *feManager) createDeploy(cc *v1alpha1.CodisCluster, deploy *apps.Deployment) error {
	if err := fm.client.Create(context.TODO(), deploy); err != nil {
		fm.recordDeployEvent("create", cc, deploy, err)
		return err
	} else {
		fm.recordDeployEvent("create", cc, deploy, err)
		return nil
	}
}

func (fm *feManager) updateDeploy(cc *v1alpha1.CodisCluster, deploy *apps.Deployment) error {
	if err := fm.client.Update(context.TODO(), deploy); err != nil {
		fm.recordDeployEvent("update", cc, deploy, err)
		return err
	}
	fm.recordDeployEvent("update", cc, deploy, nil)
	return nil
}

func (fm *feManager) syncCodisFeService(cc *v1alpha1.CodisCluster) error {
	ns := cc.GetNamespace()
	ccName := cc.GetName()

	newSvc := fm.getNewCodisFeService(cc)

	oldSvc := &corev1.Service{}
	if err := fm.client.Get(context.TODO(), types.NamespacedName{Name: fm.getSvcName(ccName), Namespace: ns}, oldSvc); err != nil {
		if errors.IsNotFound(err) {
			return fm.createService(cc, newSvc)
		} else {
			log.Infof("ns:%s,ccName:%s,get fe svc err:%s", ns, ccName, err)
			return err
		}
	} else {
		log.Infof("ns:%s,ccName:%s,get fe svc ok", ns, ccName)
	}
	//to do
	return nil
}

func (fm *feManager) populateArgVar(cc *v1alpha1.CodisCluster) []string {
	var argList []string
	//[]string{"--assets-dir", "$(CODIS_PATH)/bin/assets", "--$(COORDINATOR_NAME)", "$(COORDINATOR_ADDR)", "--listen", "$(POD_IP):9090"},
	argList = append(argList, "--assets-dir")
	argList = append(argList, "$(CODIS_PATH)/bin/assets")
	if cc.Spec.CoordinatorName == "filesystem" {
		argList = append(argList, "--dashboard-list")
		argList = append(argList, "$(CODIS_PATH)/config/dashboard.txt")
	} else {
		argList = append(argList, "--$(COORDINATOR_NAME)")
		argList = append(argList, "$(COORDINATOR_ADDR)")
	}
	argList = append(argList, "--listen")
	argList = append(argList, "$(POD_IP):9090")
	return argList
}

func (fm *feManager) populateEnvVar(cc *v1alpha1.CodisCluster) []corev1.EnvVar {
	ccName := cc.GetName()
	var envVarList []corev1.EnvVar
	envVarList = append(envVarList, corev1.EnvVar{Name: "CODIS_PATH", Value: "/gopath/src/github.com/CodisLabs/codis"})
	envVarList = append(envVarList, corev1.EnvVar{Name: "PRODUCT_NAME", Value: ccName})
	envVarList = append(envVarList, corev1.EnvVar{Name: "POD_IP", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "status.podIP"}}})
	envVarList = append(envVarList, corev1.EnvVar{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}})
	envVarList = append(envVarList, corev1.EnvVar{Name: "DASHBOARD", Value: utils.GetDashboardSvr(cc)})
	envVarList = append(envVarList, corev1.EnvVar{Name: "COORDINATOR_NAME", Value: cc.Spec.CoordinatorName})
	envVarList = append(envVarList, corev1.EnvVar{Name: "COORDINATOR_ADDR", Value: cc.Spec.CoordinatorAddr})
	return envVarList
}

func (fm *feManager) getNewCodisFeDeployment(cc *v1alpha1.CodisCluster) *apps.Deployment {
	ns := cc.GetNamespace()
	ccName := cc.GetName()

	codisFeLabels := map[string]string{
		"component":   "codis-fe",
		"clusterName": ccName,
	}

	envVarList := fm.populateEnvVar(cc)

	argList := fm.populateArgVar(cc)

	deploy := &apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:            fm.getDeployName(ccName),
			Namespace:       ns,
			OwnerReferences: []metav1.OwnerReference{utils.GetOwnerRef(cc)},
		},
		Spec: apps.DeploymentSpec{
			Replicas: &cc.Spec.CodisFe.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: codisFeLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: codisFeLabels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "codis-fe",
							Image:           cc.Spec.CodisFe.Image,
							ImagePullPolicy: "IfNotPresent",

							Command:   []string{"codis-fe"},
							Args:      argList,
							Env:       envVarList,
							Resources: utils.ResourceRequirement(cc.Spec.CodisFe.ContainerSpec),
							Ports: []corev1.ContainerPort{
								{Name: "fe-port", ContainerPort: 9090},
							},
						},
					},
				},
			},
		},
	}
	log.Infof("deploy codis fe image:%s", cc.Spec.CodisFe.Image)
	return deploy
}

func (fm *feManager) syncCodisFeDeployment(cc *v1alpha1.CodisCluster) error {
	ns := cc.GetNamespace()
	ccName := cc.GetName()

	newCodisFeDeploy := fm.getNewCodisFeDeployment(cc)
	oldCodisFeDeploy := &apps.Deployment{}
	if err := fm.client.Get(context.TODO(), types.NamespacedName{Name: fm.getDeployName(ccName), Namespace: ns}, oldCodisFeDeploy); err != nil {
		if errors.IsNotFound(err) {
			err = utils.SetDeploymentLastAppliedConfig(newCodisFeDeploy)
			if err != nil {
				return nil
			}
			return fm.createDeploy(cc, newCodisFeDeploy)
		} else {
			log.Infof("ns:%s,ccName:%s,get fe deployment err:%s", ns, ccName, err)
			return err
		}
	}
	log.Infof("ns:%s,ccName:%s,get fe deployment info succ", ns, ccName)

	if equal, err := utils.DeploymentEqual(newCodisFeDeploy, oldCodisFeDeploy); err != nil {
		log.Errorf("ns:%s,ccName:%s,deployment equal err:%v", ns, ccName, err)
		return err
	} else {
		if !equal {
			deploy := *oldCodisFeDeploy
			deploy.Spec = newCodisFeDeploy.Spec
			if err = utils.SetDeploymentLastAppliedConfig(&deploy); err != nil {
				log.Errorf("ns:%s,ccName:%s,set deployment annotation err:%v", ns, ccName, err)
				return err
			}
			if err = fm.updateDeploy(cc, &deploy); err != nil {
				log.Errorf("ns:%s,ccName:%s,update deployment err:%v", ns, ccName, err)
				return err
			}
			log.Errorf("ns:%s,ccName:%s,deployment change,update succ", ns, ccName)
		}
	}
	return nil
}

func (fm *feManager) getNewCodisFeService(cc *v1alpha1.CodisCluster) *corev1.Service {
	ns := cc.Namespace
	ccName := cc.Name

	codisFeLabels := map[string]string{
		"component":   "codis-fe",
		"clusterName": ccName,
	}
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            fm.getSvcName(ccName),
			Namespace:       ns,
			Labels:          codisFeLabels,
			OwnerReferences: []metav1.OwnerReference{utils.GetOwnerRef(cc)},
		},
		Spec: corev1.ServiceSpec{
			Type: "NodePort",
			Ports: []corev1.ServicePort{
				{
					Name:       "fe-port",
					Port:       9090,
					TargetPort: intstr.FromInt(9090),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: codisFeLabels,
		},
	}
}
