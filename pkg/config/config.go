package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config holds the application configuration
type Config struct {
	VirtIOURL         string
	StreamEndpoint    string
	DevicePaths       []string
	ChunkSize         int
	ReconnectInterval time.Duration
	LogLevel          string
}

// Default values
const (
	DefaultVirtIOURL         = "http://localhost:8083"
	DefaultStreamEndpoint    = "/stream"
	DefaultDevicePath        = "/dev/lokeyrng"
	DefaultChunkSize         = 1024
	DefaultReconnectInterval = 5 * time.Second
	DefaultLogLevel          = "INFO"
)

// LoadConfig loads configuration from environment variables with defaults
func LoadConfig() (*Config, error) {
	cfg := &Config{}

	// VirtIO URL
	cfg.VirtIOURL = getEnv("LOKEY_VIRTIO_URL", DefaultVirtIOURL)
	if cfg.VirtIOURL == "" {
		return nil, fmt.Errorf("LOKEY_VIRTIO_URL cannot be empty")
	}

	// Stream endpoint
	cfg.StreamEndpoint = getEnv("LOKEY_STREAM_ENDPOINT", DefaultStreamEndpoint)
	if !strings.HasPrefix(cfg.StreamEndpoint, "/") {
		cfg.StreamEndpoint = "/" + cfg.StreamEndpoint
	}

	// Device paths (comma-separated or single value)
	devicePathStr := getEnv("DEVICE_PATH", DefaultDevicePath)
	if devicePathStr != "" {
		cfg.DevicePaths = strings.Split(devicePathStr, ",")
		// Trim whitespace from each path
		for i, path := range cfg.DevicePaths {
			cfg.DevicePaths[i] = strings.TrimSpace(path)
		}
	} else {
		cfg.DevicePaths = []string{DefaultDevicePath}
	}

	// Validate device paths
	if len(cfg.DevicePaths) == 0 {
		return nil, fmt.Errorf("at least one device path must be specified")
	}
	for _, path := range cfg.DevicePaths {
		if path == "" {
			return nil, fmt.Errorf("device path cannot be empty")
		}
	}

	// Chunk size
	chunkSizeStr := getEnv("CHUNK_SIZE", strconv.Itoa(DefaultChunkSize))
	chunkSize, err := strconv.Atoi(chunkSizeStr)
	if err != nil || chunkSize < 1 {
		return nil, fmt.Errorf("invalid CHUNK_SIZE: %s (must be positive integer)", chunkSizeStr)
	}
	cfg.ChunkSize = chunkSize

	// Reconnect interval
	reconnectStr := getEnv("RECONNECT_INTERVAL", DefaultReconnectInterval.String())
	reconnectInterval, err := time.ParseDuration(reconnectStr)
	if err != nil || reconnectInterval < time.Second {
		return nil, fmt.Errorf("invalid RECONNECT_INTERVAL: %s (must be >= 1s)", reconnectStr)
	}
	cfg.ReconnectInterval = reconnectInterval

	// Log level
	cfg.LogLevel = getEnv("LOG_LEVEL", DefaultLogLevel)
	validLogLevels := map[string]bool{
		"DEBUG": true,
		"INFO":  true,
		"WARN":  true,
		"ERROR": true,
	}
	if !validLogLevels[strings.ToUpper(cfg.LogLevel)] {
		return nil, fmt.Errorf("invalid LOG_LEVEL: %s (must be DEBUG, INFO, WARN, or ERROR)", cfg.LogLevel)
	}
	cfg.LogLevel = strings.ToUpper(cfg.LogLevel)

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// getEnv gets an environment variable or returns the default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Validate performs additional validation on the configuration
func (c *Config) Validate() error {
	if c.VirtIOURL == "" {
		return fmt.Errorf("virtio URL cannot be empty")
	}
	if u, err := url.ParseRequestURI(c.VirtIOURL); err != nil {
		return fmt.Errorf("invalid virtio URL %q: %w", c.VirtIOURL, err)
	} else if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("invalid virtio URL %q: scheme must be http or https", c.VirtIOURL)
	}
	if len(c.DevicePaths) == 0 {
		return fmt.Errorf("at least one device path must be specified")
	}
	for _, path := range c.DevicePaths {
		if !filepath.IsAbs(path) || filepath.Clean(path) != path {
			return fmt.Errorf("invalid device path %q: must be an absolute, clean path", path)
		}
	}
	if c.ChunkSize < 1 {
		return fmt.Errorf("chunk size must be positive")
	}
	if c.ReconnectInterval < time.Second {
		return fmt.Errorf("reconnect interval must be at least 1 second")
	}
	return nil
}
