package device

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
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
	for _, path := range devicePaths {
		if !filepath.IsAbs(path) || filepath.Clean(path) != path {
			return nil, fmt.Errorf("invalid device path %q: must be an absolute, clean path", path)
		}
	}

	w := &Writer{
		devicePaths: devicePaths,
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
	if err := w.closeDevices(); err != nil {
		if w.logLevel == "DEBUG" || w.logLevel == "INFO" {
			log.Printf("[WARN] Error closing devices before reopen: %v", err)
		}
	}

	// Keep w.devices index-aligned with w.devicePaths: entries stay nil for
	// paths that fail to open so they can be retried on later writes
	w.devices = make([]*os.File, len(w.devicePaths))
	opened := 0
	for i, path := range w.devicePaths {
		// Paths are validated in NewWriter (absolute, clean)
		file, err := os.OpenFile(path, os.O_WRONLY, 0) // #nosec G304
		if err != nil {
			// Log error but continue with other devices
			if w.logLevel == "DEBUG" || w.logLevel == "INFO" {
				log.Printf("[WARN] Failed to open device %s: %v", path, err)
			}
			continue
		}

		w.devices[i] = file
		opened++
		if w.logLevel == "DEBUG" || w.logLevel == "INFO" {
			log.Printf("[INFO] Opened device: %s", path)
		}
	}

	if opened == 0 {
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
		if device == nil {
			// Device failed to open earlier; try to bring it back
			if reopenErr := w.reopenDevice(i); reopenErr != nil {
				lastErr = fmt.Errorf("device %s unavailable: %w", w.devicePaths[i], reopenErr)
				continue
			}
			device = w.devices[i]
		}

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
			// A short write means the device received a truncated chunk;
			// count it as a failure so the caller doesn't treat these bytes
			// as delivered
			lastErr = fmt.Errorf("partial write to device %s: wrote %d of %d bytes", w.devicePaths[i], n, len(data))
			if w.logLevel == "DEBUG" || w.logLevel == "INFO" {
				log.Printf("[WARN] %v", lastErr)
			}
			continue
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

// reopenDevice attempts to reopen the device at the given index.
// The caller must hold w.mu.
func (w *Writer) reopenDevice(index int) error {
	if index < 0 || index >= len(w.devices) {
		return fmt.Errorf("invalid device index: %d", index)
	}

	// Close existing device if open
	if w.devices[index] != nil {
		if err := w.devices[index].Close(); err != nil {
			if w.logLevel == "DEBUG" || w.logLevel == "INFO" {
				log.Printf("[WARN] Error closing device %s before reopen: %v", w.devicePaths[index], err)
			}
		}
		w.devices[index] = nil
	}

	// Try to reopen; paths are validated in NewWriter (absolute, clean)
	file, err := os.OpenFile(w.devicePaths[index], os.O_WRONLY, 0) // #nosec G304
	if err != nil {
		return err
	}

	w.devices[index] = file
	return nil
}

// closeDevices closes all open devices and returns the combined close
// errors. The caller must hold w.mu.
func (w *Writer) closeDevices() error {
	var errs []error
	for i, device := range w.devices {
		if device != nil {
			if err := device.Close(); err != nil {
				errs = append(errs, fmt.Errorf("close %s: %w", w.devicePaths[i], err))
			}
		}
	}
	w.devices = w.devices[:0]
	return errors.Join(errs...)
}

// Close closes all device files
func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.closeDevices()
}

// DevicePaths returns the list of device paths
func (w *Writer) DevicePaths() []string {
	return w.devicePaths
}
