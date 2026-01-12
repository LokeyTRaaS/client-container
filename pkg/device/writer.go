package device

import (
	"fmt"
	"log"
	"os"
	"sync"
)

// Writer manages writing to character devices
type Writer struct {
	devicePaths []string
	devices     []*os.File
	mu          sync.Mutex
	logLevel    string
}

// NewWriter creates a new device writer
func NewWriter(devicePaths []string, logLevel string) (*Writer, error) {
	if len(devicePaths) == 0 {
		return nil, fmt.Errorf("no device paths provided")
	}

	w := &Writer{
		devicePaths: devicePaths,
		devices:     make([]*os.File, 0, len(devicePaths)),
		logLevel:    logLevel,
	}

	// Open all devices
	if err := w.Open(); err != nil {
		return nil, fmt.Errorf("failed to open devices: %w", err)
	}

	return w, nil
}

// Open opens all device files for writing
func (w *Writer) Open() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Close existing devices if any
	w.closeDevices()

	for _, path := range w.devicePaths {
		file, err := os.OpenFile(path, os.O_WRONLY, 0)
		if err != nil {
			// Log error but continue with other devices
			if w.logLevel == "DEBUG" || w.logLevel == "INFO" {
				log.Printf("[WARN] Failed to open device %s: %v", path, err)
			}
			continue
		}

		w.devices = append(w.devices, file)
		if w.logLevel == "DEBUG" || w.logLevel == "INFO" {
			log.Printf("[INFO] Opened device: %s", path)
		}
	}

	if len(w.devices) == 0 {
		return fmt.Errorf("failed to open any devices")
	}

	return nil
}

// Write writes data to all open devices
func (w *Writer) Write(data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(w.devices) == 0 {
		return fmt.Errorf("no devices open")
	}

	var lastErr error
	successCount := 0

	for i, device := range w.devices {
		n, err := device.Write(data)
		if err != nil {
			lastErr = fmt.Errorf("failed to write to device %s: %w", w.devicePaths[i], err)
			if w.logLevel == "DEBUG" || w.logLevel == "INFO" {
				log.Printf("[WARN] %v", lastErr)
			}
			// Try to reopen the device
			if reopenErr := w.reopenDevice(i); reopenErr != nil {
				if w.logLevel == "DEBUG" || w.logLevel == "INFO" {
					log.Printf("[WARN] Failed to reopen device %s: %v", w.devicePaths[i], reopenErr)
				}
			}
			continue
		}

		if n != len(data) {
			lastErr = fmt.Errorf("partial write to device %s: wrote %d of %d bytes", w.devicePaths[i], n, len(data))
			if w.logLevel == "DEBUG" || w.logLevel == "INFO" {
				log.Printf("[WARN] %v", lastErr)
			}
		}

		successCount++
	}

	// If at least one device succeeded, don't return error
	if successCount > 0 {
		return nil
	}

	// All devices failed
	if lastErr != nil {
		return lastErr
	}

	return fmt.Errorf("no devices available for writing")
}

// reopenDevice attempts to reopen a device at the given index
func (w *Writer) reopenDevice(index int) error {
	if index < 0 || index >= len(w.devicePaths) {
		return fmt.Errorf("invalid device index: %d", index)
	}

	// Close existing device if open
	if index < len(w.devices) && w.devices[index] != nil {
		w.devices[index].Close()
		w.devices[index] = nil
	}

	// Try to reopen
	file, err := os.OpenFile(w.devicePaths[index], os.O_WRONLY, 0)
	if err != nil {
		return err
	}

	// Update device in slice
	if index < len(w.devices) {
		w.devices[index] = file
	} else {
		// Extend slice if needed
		for len(w.devices) <= index {
			w.devices = append(w.devices, nil)
		}
		w.devices[index] = file
	}

	return nil
}

// closeDevices closes all open devices
func (w *Writer) closeDevices() {
	for i, device := range w.devices {
		if device != nil {
			if err := device.Close(); err != nil {
				if w.logLevel == "DEBUG" || w.logLevel == "INFO" {
					log.Printf("[WARN] Error closing device %s: %v", w.devicePaths[i], err)
				}
			}
		}
	}
	w.devices = w.devices[:0]
}

// Close closes all device files
func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.closeDevices()
	return nil
}

// DevicePaths returns the list of device paths
func (w *Writer) DevicePaths() []string {
	return w.devicePaths
}
