package reconciler

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateInitContainer_SeccompEnabled(t *testing.T) {
	spec := &SidecarSpec{
		DevicePaths:   []string{"/dev/lokeyrng"},
		InitImage:     "test-init:latest",
		EnableSeccomp: true,
	}

	container := createInitContainer(spec)

	if container.SecurityContext == nil {
		t.Fatal("SecurityContext should not be nil")
	}

	if container.SecurityContext.SeccompProfile == nil {
		t.Fatal("SeccompProfile should be set when EnableSeccomp is true")
	}

	if container.SecurityContext.SeccompProfile.Type != corev1.SeccompProfileTypeLocalhost {
		t.Errorf("Expected SeccompProfile type Localhost, got %v", container.SecurityContext.SeccompProfile.Type)
	}

	if container.SecurityContext.SeccompProfile.LocalhostProfile == nil {
		t.Fatal("LocalhostProfile should not be nil")
	}

	if *container.SecurityContext.SeccompProfile.LocalhostProfile != "lokey-client-init-container.json" {
		t.Errorf("Expected profile name lokey-client-init-container.json, got %s", *container.SecurityContext.SeccompProfile.LocalhostProfile)
	}
}

func TestCreateInitContainer_SeccompDisabled(t *testing.T) {
	spec := &SidecarSpec{
		DevicePaths:   []string{"/dev/lokeyrng"},
		InitImage:     "test-init:latest",
		EnableSeccomp: false,
	}

	container := createInitContainer(spec)

	if container.SecurityContext == nil {
		t.Fatal("SecurityContext should not be nil (still needs capabilities)")
	}

	if container.SecurityContext.SeccompProfile != nil {
		t.Fatal("SeccompProfile should be nil when EnableSeccomp is false")
	}

	// Verify capabilities are still set
	if container.SecurityContext.Capabilities == nil {
		t.Fatal("Capabilities should be set")
	}
}

func TestCreateSidecarContainer_SeccompEnabled(t *testing.T) {
	spec := &SidecarSpec{
		DevicePaths:   []string{"/dev/lokeyrng"},
		Image:         "test-client:latest",
		VirtIOURL:     "http://test:8083",
		EnableSeccomp: true,
	}

	container := createSidecarContainer(spec)

	if container.SecurityContext == nil {
		t.Fatal("SecurityContext should not be nil")
	}

	if container.SecurityContext.SeccompProfile == nil {
		t.Fatal("SeccompProfile should be set when EnableSeccomp is true")
	}

	if container.SecurityContext.SeccompProfile.Type != corev1.SeccompProfileTypeLocalhost {
		t.Errorf("Expected SeccompProfile type Localhost, got %v", container.SecurityContext.SeccompProfile.Type)
	}

	if *container.SecurityContext.SeccompProfile.LocalhostProfile != "lokey-client-container.json" {
		t.Errorf("Expected profile name lokey-client-container.json, got %s", *container.SecurityContext.SeccompProfile.LocalhostProfile)
	}
}

func TestCreateSidecarContainer_SeccompDisabled(t *testing.T) {
	spec := &SidecarSpec{
		DevicePaths:   []string{"/dev/lokeyrng"},
		Image:         "test-client:latest",
		VirtIOURL:     "http://test:8083",
		EnableSeccomp: false,
	}

	container := createSidecarContainer(spec)

	if container.SecurityContext != nil {
		t.Fatal("SecurityContext should be nil when EnableSeccomp is false")
	}
}

func TestInjectSidecar(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "app:latest",
				},
			},
		},
	}

	spec := &SidecarSpec{
		DevicePaths:   []string{"/dev/lokeyrng"},
		Image:         "test-client:latest",
		InitImage:     "test-init:latest",
		VirtIOURL:     "http://test:8083",
		EnableSeccomp: true,
	}

	err := InjectSidecar(pod, spec)
	if err != nil {
		t.Fatalf("InjectSidecar failed: %v", err)
	}

	// Verify init container was added
	if len(pod.Spec.InitContainers) != 1 {
		t.Fatalf("Expected 1 init container, got %d", len(pod.Spec.InitContainers))
	}

	// Verify sidecar container was added
	if len(pod.Spec.Containers) != 2 {
		t.Fatalf("Expected 2 containers, got %d", len(pod.Spec.Containers))
	}

	// Verify annotation was added
	if pod.Annotations["lokey.io/injected"] != "true" {
		t.Error("Expected injection annotation to be set")
	}

	// Verify seccomp is set in init container
	initContainer := pod.Spec.InitContainers[0]
	if initContainer.SecurityContext.SeccompProfile == nil {
		t.Error("Init container should have seccomp profile when enabled")
	}

	// Verify seccomp is set in sidecar container
	sidecarContainer := pod.Spec.Containers[1]
	if sidecarContainer.SecurityContext == nil || sidecarContainer.SecurityContext.SeccompProfile == nil {
		t.Error("Sidecar container should have seccomp profile when enabled")
	}
}
