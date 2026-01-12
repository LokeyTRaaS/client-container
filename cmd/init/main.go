package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"syscall"
)

func main() {
	var devicePaths = flag.String("devices", "/dev/lokeyrng", "Comma-separated list of device paths to create")
	var major = flag.Int("major", 10, "Major device number")
	var minor = flag.Int("minor", 240, "Minor device number (for /dev/lokeyrng, use 240; for /dev/hwrng, use 183)")
	var permissions = flag.String("perms", "0666", "Device permissions (octal)")
	flag.Parse()

	// Parse device paths
	paths := strings.Split(*devicePaths, ",")
	for i, path := range paths {
		paths[i] = strings.TrimSpace(path)
	}

	// Parse permissions
	perm, err := strconv.ParseUint(*permissions, 8, 32)
	if err != nil {
		log.Fatalf("[ERROR] Invalid permissions: %v", err)
	}

	// Create each device
	for _, path := range paths {
		if path == "" {
			continue
		}

		// Determine major/minor based on device path
		maj, min := *major, *minor
		if strings.Contains(path, "hwrng") {
			// Standard /dev/hwrng uses major 10, minor 183
			maj = 10
			min = 183
		} else if strings.Contains(path, "lokeyrng") {
			// Custom /dev/lokeyrng uses major 10, minor 240
			maj = 10
			min = 240
		}

		if err := createDevice(path, maj, min, os.FileMode(perm)); err != nil {
			log.Fatalf("[ERROR] Failed to create device %s: %v", path, err)
		}

		log.Printf("[INFO] Created device: %s (major: %d, minor: %d, perms: %s)", path, maj, min, *permissions)
	}

	log.Printf("[INFO] All devices created successfully")
}

func createDevice(path string, major, minor int, mode os.FileMode) error {
	// Create directory if it doesn't exist
	dir := path[:strings.LastIndex(path, "/")]
	if dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Remove existing file/device if it exists
	if _, err := os.Stat(path); err == nil {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("failed to remove existing device: %w", err)
		}
	}

	// Create character device
	dev := int((major << 8) | (minor & 0xff) | ((minor & 0xfff00) << 12))
	if err := syscall.Mknod(path, syscall.S_IFCHR|uint32(mode), dev); err != nil {
		return fmt.Errorf("mknod failed: %w", err)
	}

	// Set permissions
	if err := os.Chmod(path, mode); err != nil {
		return fmt.Errorf("chmod failed: %w", err)
	}

	return nil
}
