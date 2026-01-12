package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LokeyClientSpec defines the desired state of LokeyClient
type LokeyClientSpec struct {
	// VirtIO service URL
	VirtIOURL string `json:"virtioUrl"`

	// List of device paths (default: [/dev/lokeyrng])
	DevicePaths []string `json:"devicePaths,omitempty"`

	// Stream chunk size in bytes (default: 1024)
	ChunkSize int `json:"chunkSize,omitempty"`

	// Reconnection interval (default: 5s)
	ReconnectInterval string `json:"reconnectInterval,omitempty"`

	// Enable daemonset deployment for node-level randomness (default: false)
	EnableDaemonset *bool `json:"enableDaemonset,omitempty"`

	// Enable sidecar injection into pods (default: false)
	EnableSidecar *bool `json:"enableSidecar,omitempty"`

	// Sidecar injection mode: annotation, label, or namespace
	InjectionMode string `json:"injectionMode,omitempty"`

	// Namespace selector for injection
	NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty"`

	// Pod selector for injection
	PodSelector *metav1.LabelSelector `json:"podSelector,omitempty"`

	// Resource limits/requests
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Container image override
	Image string `json:"image,omitempty"`

	// Init container image override
	InitImage string `json:"initImage,omitempty"`

	// Enable seccomp profiles for containers (default: false)
	// Requires seccomp profiles to be installed on nodes at /var/lib/kubelet/seccomp/profiles/
	EnableSeccomp *bool `json:"enableSeccomp,omitempty"`
}

// LokeyClientStatus defines the observed state of LokeyClient
type LokeyClientStatus struct {
	// Count of pods with injected sidecars
	InjectedPods int `json:"injectedPods,omitempty"`

	// Daemonset deployment status
	DaemonsetStatus string `json:"daemonsetStatus,omitempty"`

	// Last reconciliation time
	LastSyncTime metav1.Time `json:"lastSyncTime,omitempty"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// LokeyClient is the Schema for the lokeyclients API
type LokeyClient struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LokeyClientSpec   `json:"spec,omitempty"`
	Status LokeyClientStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LokeyClientList contains a list of LokeyClient
type LokeyClientList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LokeyClient `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LokeyClient{}, &LokeyClientList{})
}
