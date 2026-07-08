package reconciler

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// ShouldInjectSidecar determines if a pod should have the sidecar injected.
// Selectors fail closed: an unparsable selector, or missing namespace labels,
// never widens the match.
func ShouldInjectSidecar(pod *corev1.Pod, namespaceLabels map[string]string, namespaceSelector, podSelector *metav1.LabelSelector) bool {
	// Check namespace labels
	if namespaceSelector != nil {
		nsSelector, err := metav1.LabelSelectorAsSelector(namespaceSelector)
		if err != nil {
			return false
		}
		if !nsSelector.Matches(labels.Set(namespaceLabels)) {
			return false
		}
	}

	// Check pod labels
	if podSelector != nil {
		podSel, err := metav1.LabelSelectorAsSelector(podSelector)
		if err != nil {
			return false
		}
		if !podSel.Matches(labels.Set(pod.Labels)) {
			return false
		}
	}

	// Check annotation
	if pod.Annotations != nil {
		if val, ok := pod.Annotations["lokey.io/inject"]; ok && val == "true" {
			return true
		}
	}

	return namespaceSelector != nil || podSelector != nil
}

// InjectSidecar injects init and sidecar containers into a pod
func InjectSidecar(pod *corev1.Pod, spec *SidecarSpec) error {
	// Check if already injected
	if isInjected(pod) {
		return nil
	}

	// Inject init container
	initContainer := createInitContainer(spec)
	pod.Spec.InitContainers = append(pod.Spec.InitContainers, initContainer)

	// Inject sidecar container
	sidecarContainer := createSidecarContainer(spec)
	pod.Spec.Containers = append(pod.Spec.Containers, sidecarContainer)

	// Add volume mount
	devVolume := corev1.Volume{
		Name: "dev",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	pod.Spec.Volumes = append(pod.Spec.Volumes, devVolume)

	// Add annotation to mark as injected
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}
	pod.Annotations["lokey.io/injected"] = "true"

	return nil
}

// SidecarSpec contains the specification for sidecar injection
type SidecarSpec struct {
	VirtIOURL         string
	DevicePaths       []string
	ChunkSize         int
	ReconnectInterval string
	Image             string
	InitImage         string
	Resources         corev1.ResourceRequirements
	EnableSeccomp     bool
}

func isInjected(pod *corev1.Pod) bool {
	if pod.Annotations == nil {
		return false
	}
	return pod.Annotations["lokey.io/injected"] == "true"
}

func createInitContainer(spec *SidecarSpec) corev1.Container {
	devicePaths := spec.DevicePaths
	if len(devicePaths) == 0 {
		devicePaths = []string{"/dev/lokeyrng"}
	}

	initImage := spec.InitImage
	if initImage == "" {
		initImage = "ghcr.io/lokey/client-container-init:latest"
	}

	args := []string{
		"-devices", devicePaths[0],
		"-major", "10",
		"-minor", "240",
		"-perms", "0666",
	}

	securityContext := &corev1.SecurityContext{
		Capabilities: &corev1.Capabilities{
			Add: []corev1.Capability{"MKNOD"},
		},
		RunAsUser: func() *int64 { u := int64(0); return &u }(),
	}

	// Only add seccomp if enabled
	if spec.EnableSeccomp {
		securityContext.SeccompProfile = &corev1.SeccompProfile{
			Type:             corev1.SeccompProfileTypeLocalhost,
			LocalhostProfile: func() *string { p := "lokey-client-init-container.json"; return &p }(),
		}
	}

	return corev1.Container{
		Name:            "lokey-device-init",
		Image:           initImage,
		Command:         []string{"/app/init"},
		Args:            args,
		SecurityContext: securityContext,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "dev",
				MountPath: "/dev",
			},
		},
	}
}

func createSidecarContainer(spec *SidecarSpec) corev1.Container {
	devicePaths := spec.DevicePaths
	if len(devicePaths) == 0 {
		devicePaths = []string{"/dev/lokeyrng"}
	}

	image := spec.Image
	if image == "" {
		image = "ghcr.io/lokey/client-container:latest"
	}

	chunkSize := spec.ChunkSize
	if chunkSize == 0 {
		chunkSize = 1024
	}

	reconnectInterval := spec.ReconnectInterval
	if reconnectInterval == "" {
		reconnectInterval = "5s"
	}

	env := []corev1.EnvVar{
		{
			Name:  "LOKEY_VIRTIO_URL",
			Value: spec.VirtIOURL,
		},
		{
			Name:  "LOKEY_STREAM_ENDPOINT",
			Value: "/stream",
		},
		{
			Name:  "DEVICE_PATH",
			Value: joinDevicePaths(devicePaths),
		},
		{
			Name:  "CHUNK_SIZE",
			Value: fmt.Sprintf("%d", chunkSize),
		},
		{
			Name:  "RECONNECT_INTERVAL",
			Value: reconnectInterval,
		},
		{
			Name:  "LOG_LEVEL",
			Value: "INFO",
		},
	}

	container := corev1.Container{
		Name:  "lokey-client",
		Image: image,
		Env:   env,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "dev",
				MountPath: "/dev",
			},
		},
	}

	// Only add seccomp if enabled
	if spec.EnableSeccomp {
		container.SecurityContext = &corev1.SecurityContext{
			SeccompProfile: &corev1.SeccompProfile{
				Type:             corev1.SeccompProfileTypeLocalhost,
				LocalhostProfile: func() *string { p := "lokey-client-container.json"; return &p }(),
			},
		}
	}

	if spec.Resources.Requests != nil || spec.Resources.Limits != nil {
		container.Resources = spec.Resources
	} else {
		// Default resources
		container.Resources = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("32Mi"),
				corev1.ResourceCPU:    resource.MustParse("50m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("128Mi"),
				corev1.ResourceCPU:    resource.MustParse("200m"),
			},
		}
	}

	return container
}

// joinDevicePaths joins device paths with comma separator
func joinDevicePaths(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	if len(paths) == 1 {
		return paths[0]
	}
	result := paths[0]
	for _, p := range paths[1:] {
		result += "," + p
	}
	return result
}
