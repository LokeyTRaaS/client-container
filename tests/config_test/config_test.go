package config_test

import (
	"os"
	"testing"
	"time"

	"github.com/lokey/client-container/pkg/config"
)

func TestLoadConfig_Defaults(t *testing.T) {
	// Clear environment variables
	os.Clearenv()

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.VirtIOURL != config.DefaultVirtIOURL {
		t.Fatalf("Unexpected VirtIOURL: got %q, expected %q", cfg.VirtIOURL, config.DefaultVirtIOURL)
	}

	if cfg.StreamEndpoint != config.DefaultStreamEndpoint {
		t.Fatalf("Unexpected StreamEndpoint: got %q, expected %q", cfg.StreamEndpoint, config.DefaultStreamEndpoint)
	}

	if len(cfg.DevicePaths) != 1 || cfg.DevicePaths[0] != config.DefaultDevicePath {
		t.Fatalf("Unexpected DevicePaths: got %v, expected [%q]", cfg.DevicePaths, config.DefaultDevicePath)
	}

	if cfg.ChunkSize != config.DefaultChunkSize {
		t.Fatalf("Unexpected ChunkSize: got %d, expected %d", cfg.ChunkSize, config.DefaultChunkSize)
	}

	if cfg.ReconnectInterval != config.DefaultReconnectInterval {
		t.Fatalf("Unexpected ReconnectInterval: got %v, expected %v", cfg.ReconnectInterval, config.DefaultReconnectInterval)
	}
}

func TestLoadConfig_EnvironmentVariables(t *testing.T) {
	os.Clearenv()
	os.Setenv("LOKEY_VIRTIO_URL", "http://test:8083")
	os.Setenv("LOKEY_STREAM_ENDPOINT", "/custom/stream")
	os.Setenv("DEVICE_PATH", "/dev/test1,/dev/test2")
	os.Setenv("CHUNK_SIZE", "2048")
	os.Setenv("RECONNECT_INTERVAL", "10s")
	os.Setenv("LOG_LEVEL", "DEBUG")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.VirtIOURL != "http://test:8083" {
		t.Fatalf("Unexpected VirtIOURL: got %q", cfg.VirtIOURL)
	}

	if cfg.StreamEndpoint != "/custom/stream" {
		t.Fatalf("Unexpected StreamEndpoint: got %q", cfg.StreamEndpoint)
	}

	if len(cfg.DevicePaths) != 2 {
		t.Fatalf("Unexpected DevicePaths count: got %d, expected 2", len(cfg.DevicePaths))
	}

	if cfg.ChunkSize != 2048 {
		t.Fatalf("Unexpected ChunkSize: got %d, expected 2048", cfg.ChunkSize)
	}

	if cfg.ReconnectInterval != 10*time.Second {
		t.Fatalf("Unexpected ReconnectInterval: got %v, expected 10s", cfg.ReconnectInterval)
	}

	if cfg.LogLevel != "DEBUG" {
		t.Fatalf("Unexpected LogLevel: got %q, expected DEBUG", cfg.LogLevel)
	}
}

func TestLoadConfig_InvalidValues(t *testing.T) {
	testCases := []struct {
		name   string
		env    map[string]string
		errMsg string
	}{
		{
			name:   "invalid chunk size",
			env:    map[string]string{"CHUNK_SIZE": "0"},
			errMsg: "invalid CHUNK_SIZE",
		},
		{
			name:   "invalid reconnect interval",
			env:    map[string]string{"RECONNECT_INTERVAL": "100ms"},
			errMsg: "invalid RECONNECT_INTERVAL",
		},
		{
			name:   "invalid log level",
			env:    map[string]string{"LOG_LEVEL": "INVALID"},
			errMsg: "invalid LOG_LEVEL",
		},
		{
			name:   "relative device path",
			env:    map[string]string{"DEVICE_PATH": "dev/lokeyrng"},
			errMsg: "invalid device path",
		},
		{
			name:   "malformed virtio URL",
			env:    map[string]string{"LOKEY_VIRTIO_URL": "http://bad host:8083"},
			errMsg: "invalid virtio URL",
		},
		{
			name:   "virtio URL without scheme",
			env:    map[string]string{"LOKEY_VIRTIO_URL": "localhost:8083"},
			errMsg: "invalid virtio URL",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			os.Clearenv()
			for k, v := range tc.env {
				os.Setenv(k, v)
			}

			_, err := config.LoadConfig()
			if err == nil {
				t.Fatal("Expected error but got none")
			}
		})
	}
}
