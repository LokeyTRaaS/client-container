package reconciler

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateDaemonset creates or updates a daemonset for node-level randomness
func CreateDaemonset(name, namespace string, spec *DaemonsetSpec) *appsv1.DaemonSet {
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
			Value: joinStrings(devicePaths, ","),
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

	daemonset := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": "lokey-client",
			},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "lokey-client",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "lokey-client",
					},
				},
				Spec: corev1.PodSpec{
					HostNetwork: true,
					HostPID:     true,
					Containers: []corev1.Container{
						{
							Name:  "lokey-client",
							Image: image,
							Env:   env,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "dev",
									MountPath: "/dev",
								},
							},
							SecurityContext: func() *corev1.SecurityContext {
								sc := &corev1.SecurityContext{
									Privileged: func() *bool { p := true; return &p }(),
									RunAsUser:  func() *int64 { u := int64(0); return &u }(),
								}
								// Only add seccomp if enabled
								if spec.EnableSeccomp {
									sc.SeccompProfile = &corev1.SeccompProfile{
										Type:             corev1.SeccompProfileTypeLocalhost,
										LocalhostProfile: func() *string { p := "lokey-client-container.json"; return &p }(),
									}
								}
								return sc
							}(),
							Resources: spec.Resources,
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "dev",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/dev",
									Type: func() *corev1.HostPathType { t := corev1.HostPathDirectory; return &t }(),
								},
							},
						},
					},
					Tolerations: []corev1.Toleration{
						{
							Effect:   corev1.TaintEffectNoSchedule,
							Operator: corev1.TolerationOpExists,
						},
						{
							Effect:   corev1.TaintEffectNoExecute,
							Operator: corev1.TolerationOpExists,
						},
					},
				},
			},
		},
	}

	return daemonset
}

// DaemonsetSpec contains the specification for daemonset creation
type DaemonsetSpec struct {
	VirtIOURL         string
	DevicePaths       []string
	ChunkSize         int
	ReconnectInterval string
	Image             string
	Resources         corev1.ResourceRequirements
	EnableSeccomp     bool
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}
	result := strs[0]
	for _, s := range strs[1:] {
		result += sep + s
	}
	return result
}
