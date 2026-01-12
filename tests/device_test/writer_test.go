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
