package v1alpha1

import (
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/apimachinery/pkg/types"
)

// MemberPhase is the current state of member
type MemberPhase string

const (
	// NormalPhase represents normal state of Codis cluster.
	NormalPhase MemberPhase = "Normal"
	// UpgradePhase represents the upgrade state of Codis cluster.
	UpgradePhase MemberPhase = "Upgrade"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CodisCluster describes a database.
type CodisCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CodisClusterSpec   `json:"spec"`
	Status CodisClusterStatus `json:"status,omitempty"`
}

// CodisClusterSpec describes the attributes that a user creates on a codis cluster
type CodisClusterSpec struct {
	SchedulerName   string             `json:"schedulerName,omitempty"`
	CodisProxy      CodisProxySpec     `json:"codisProxy,omitempty"`
	CodisServer     CodisServerSpec    `json:"codisServer,omitempty"`
	CodisDashboard  CodisDashboardSpec `json:"codisDashboard,omitempty"`
	CodisFe         CodisFeSpec        `json:"codisFe,omitempty"`
	Sentinel        SentinelSpec       `json:"sentinel,omitempty"`
	CoordinatorName string             `json:"coordinatorName"`
	CoordinatorAddr string             `json:"coordinatorAddr"`
}

// CodisClusterStatus represents the current status of a codis cluster.
type CodisClusterStatus struct {
	CodisProxy     CodisProxyStatus     `json:"codisProxy,omitempty"`
	CodisServer    CodisServerStatus    `json:"codisServer,omitempty"`
	CodisDashboard CodisDashboardStatus `json:"codisDashboard,omitempty"`
	CodisFe        CodisFeStatus        `json:"codisFeStatus,omitempty"`
	Sentinel       SentinelStatus       `json:"sentinelStatus,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// CodisClusterList is a list of CodisCluster resources
type CodisClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []CodisCluster `json:"items"`
}

// ContainerSpec is the container spec of a pod
type ContainerSpec struct {
	Image           string               `json:"image"`
	ImagePullPolicy corev1.PullPolicy    `json:"imagePullPolicy,omitempty"`
	Requests        *ResourceRequirement `json:"requests,omitempty"`
	Limits          *ResourceRequirement `json:"limits,omitempty"`
}

// ResourceRequirement is resource requirements for a pod
type ResourceRequirement struct {
	// CPU is how many cores a pod requires
	CPU string `json:"cpu,omitempty"`
	// Memory is how much memory a pod requires
	Memory string `json:"memory,omitempty"`
	// Storage is storage size a pod requires
	Storage string `json:"storage,omitempty"`
}

type CodisProxyHPASpec struct {
	MinReplicas      int32 `json:"minReplicas"`
	MaxReplicas      int32 `json:"maxReplicas"`
	CpuUsedThreshold int32 `json:"cpuUsedThreshold"`
}

// CodisProxySpec contains details of CodisProxy member
type CodisProxySpec struct {
	ContainerSpec
	Replicas    int32             `json:"replicas"`
	SessionAuth string            `json:"sessionAuth"`
	HpaSpec     CodisProxyHPASpec `json:"hpaSpec,omitempty"`
	//how many pods we can add at a time
	MaxSurge int `json:"maxSurge,omitempty"`
	//MaxUnavailable define how many pods can be unavailable during the rolling update
	MaxUnavailable int `json:"maxUnavailable,omitempty"`
}

// CodisServerSpec contains details of CodisServer member
type CodisServerSpec struct {
	ContainerSpec
	Replicas      int32 `json:"replicas"`
	GroupReplicas int32 `json:"groupReplicas"`
	//When a partition is specified, all Pods with an ordinal that is greater than or equal to the partition will be updated when the StatefulSet’s .spec.template is updated. If a Pod that has an ordinal less than the partition is deleted or otherwise terminated, it will be restored to its original configuration.
	Partition        int32   `json:"partition,omitempty"`
	StorageClassName *string `json:"storageClassName,omitempty"`
}

// CodisDashboardSpec contains details of CodisDashboard
type CodisDashboardSpec struct {
	ContainerSpec
	Replicas    int32  `json:"replicas"`
	ProductAuth string `json:"productAuth"`
}

// CodisFeSpec contains details of CodisFe
type CodisFeSpec struct {
	ContainerSpec
	Replicas int32 `json:"replicas"`
}

// SentinelSpec contains details of Sentinel
type SentinelSpec struct {
	ContainerSpec
	Replicas         int32   `json:"replicas"`
	StorageClassName *string `json:"storageClassName,omitempty"`
}

type CodisProxyStatus struct {
	Phase      MemberPhase            `json:"phase,omitempty"`
	Deployment *apps.DeploymentStatus `json:"deployment,omitempty"`
}

type CodisServerStatus struct {
	Phase       MemberPhase             `json:"phase,omitempty"`
	StatefulSet *apps.StatefulSetStatus `json:"statefulSet,omitempty"`
}

type CodisDashboardStatus struct {
	Phase       MemberPhase             `json:"phase,omitempty"`
	StatefulSet *apps.StatefulSetStatus `json:"statefulSet,omitempty"`
}

type CodisFeStatus struct {
	Phase      MemberPhase            `json:"phase,omitempty"`
	Deployment *apps.DeploymentStatus `json:"deployment,omitempty"`
}

type SentinelStatus struct {
	Phase       MemberPhase             `json:"phase,omitempty"`
	StatefulSet *apps.StatefulSetStatus `json:"statefulSet,omitempty"`
}

func init() {
	SchemeBuilder.Register(&CodisCluster{}, &CodisClusterList{})
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
