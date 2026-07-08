package device_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lokey/client-container/pkg/device"
)

func TestWriter_Write(t *testing.T) {
	// Create temporary device file
	tmpDir := t.TempDir()
	devicePath := filepath.Join(tmpDir, "test-device")

	// Create a regular file to simulate device (we can't create real char devices in tests)
	file, err := os.Create(devicePath)
	if err != nil {
		t.Fatalf("Failed to create test device: %v", err)
	}
	file.Close()

	writer, err := device.NewWriter([]string{devicePath}, "INFO")
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}
	defer writer.Close()

	// Write test data
	testData := []byte("test random data")
	err = writer.Write(testData)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Verify data was written
	readData, err := os.ReadFile(devicePath)
	if err != nil {
		t.Fatalf("Failed to read device: %v", err)
	}

	if string(readData) != string(testData) {
		t.Fatalf("Data mismatch: got %q, expected %q", string(readData), string(testData))
	}
}

func TestWriter_MultipleDevices(t *testing.T) {
	// Create multiple temporary device files
	tmpDir := t.TempDir()
	devicePath1 := filepath.Join(tmpDir, "device1")
	devicePath2 := filepath.Join(tmpDir, "device2")

	// Create test files
	os.Create(devicePath1)
	os.Create(devicePath2)

	writer, err := device.NewWriter([]string{devicePath1, devicePath2}, "INFO")
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}
	defer writer.Close()

	// Write test data
	testData := []byte("test data")
	err = writer.Write(testData)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Verify both devices received data
	data1, _ := os.ReadFile(devicePath1)
	data2, _ := os.ReadFile(devicePath2)

	if string(data1) != string(testData) {
		t.Fatalf("Device1 data mismatch: got %q, expected %q", string(data1), string(testData))
	}

	if string(data2) != string(testData) {
		t.Fatalf("Device2 data mismatch: got %q, expected %q", string(data2), string(testData))
	}
}

func TestWriter_InvalidPath(t *testing.T) {
	// Try to create writer with invalid path
	_, err := device.NewWriter([]string{"/nonexistent/device"}, "INFO")
	if err == nil {
		t.Fatal("Expected error for invalid device path")
	}
}

func TestWriter_RelativePathRejected(t *testing.T) {
	_, err := device.NewWriter([]string{"dev/lokeyrng"}, "INFO")
	if err == nil {
		t.Fatal("Expected error for relative device path")
	}
}

func TestWriter_DeviceRecovery(t *testing.T) {
	tmpDir := t.TempDir()
	latePath := filepath.Join(tmpDir, "late-device")
	readyPath := filepath.Join(tmpDir, "ready-device")

	file, err := os.Create(readyPath)
	if err != nil {
		t.Fatalf("Failed to create test device: %v", err)
	}
	file.Close()

	// One of two devices can't be opened yet: the writer must keep serving
	// the available one and keep the path/device mapping aligned
	writer, err := device.NewWriter([]string{latePath, readyPath}, "INFO")
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}
	defer writer.Close()

	first := []byte("first")
	if err := writer.Write(first); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	readyData, err := os.ReadFile(readyPath)
	if err != nil {
		t.Fatalf("Failed to read device: %v", err)
	}
	if string(readyData) != string(first) {
		t.Fatalf("Ready device data mismatch: got %q, expected %q", readyData, first)
	}

	// The missing device appears later (e.g. created by the init container):
	// the next write must pick it up instead of dropping it forever
	file, err = os.Create(latePath)
	if err != nil {
		t.Fatalf("Failed to create late device: %v", err)
	}
	file.Close()

	second := []byte("second")
	if err := writer.Write(second); err != nil {
		t.Fatalf("Write after device appeared failed: %v", err)
	}

	lateData, err := os.ReadFile(latePath)
	if err != nil {
		t.Fatalf("Failed to read late device: %v", err)
	}
	if string(lateData) != string(second) {
		t.Fatalf("Late device data mismatch: got %q, expected %q", lateData, second)
	}

	readyData, err = os.ReadFile(readyPath)
	if err != nil {
		t.Fatalf("Failed to read device: %v", err)
	}
	if string(readyData) != string(first)+string(second) {
		t.Fatalf("Ready device data mismatch: got %q, expected %q", readyData, string(first)+string(second))
	}
}
