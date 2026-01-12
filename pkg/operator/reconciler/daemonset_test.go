package reconciler

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestCreateDaemonset_SeccompEnabled(t *testing.T) {
	spec := &DaemonsetSpec{
		VirtIOURL:         "http://test:8083",
		DevicePaths:       []string{"/dev/lokeyrng"},
		Image:             "test-client:latest",
		ChunkSize:         1024,
		ReconnectInterval: "5s",
		EnableSeccomp:     true,
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{},
			Limits:   corev1.ResourceList{},
		},
	}

	daemonset := CreateDaemonset("test-daemonset", "default", spec)

	if daemonset == nil {
		t.Fatal("Daemonset should not be nil")
	}

	containers := daemonset.Spec.Template.Spec.Containers
	if len(containers) != 1 {
		t.Fatalf("Expected 1 container, got %d", len(containers))
	}

	container := containers[0]
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

func TestCreateDaemonset_SeccompDisabled(t *testing.T) {
	spec := &DaemonsetSpec{
		VirtIOURL:         "http://test:8083",
		DevicePaths:       []string{"/dev/lokeyrng"},
		Image:             "test-client:latest",
		ChunkSize:         1024,
		ReconnectInterval: "5s",
		EnableSeccomp:     false,
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{},
			Limits:   corev1.ResourceList{},
		},
	}

	daemonset := CreateDaemonset("test-daemonset", "default", spec)

	if daemonset == nil {
		t.Fatal("Daemonset should not be nil")
	}

	containers := daemonset.Spec.Template.Spec.Containers
	if len(containers) != 1 {
		t.Fatalf("Expected 1 container, got %d", len(containers))
	}

	container := containers[0]
	if container.SecurityContext == nil {
		t.Fatal("SecurityContext should not be nil (still needs privileged)")
	}

	if container.SecurityContext.SeccompProfile != nil {
		t.Fatal("SeccompProfile should be nil when EnableSeccomp is false")
	}

	// Verify privileged is still set
	if container.SecurityContext.Privileged == nil || !*container.SecurityContext.Privileged {
		t.Error("Container should be privileged")
	}
}
