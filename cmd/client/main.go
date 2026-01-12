package main

import (
	"context"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lokey/client-container/pkg/config"
	"github.com/lokey/client-container/pkg/device"
	"github.com/lokey/client-container/pkg/virtio"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("[ERROR] Failed to load configuration: %v", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("[ERROR] Invalid configuration: %v", err)
	}

	// Set up logging based on log level
	if cfg.LogLevel == "DEBUG" || cfg.LogLevel == "INFO" {
		log.Printf("[INFO] Starting Lokey client container")
		log.Printf("[INFO] Configuration:")
		log.Printf("[INFO]   VirtIO URL: %s", cfg.VirtIOURL)
		log.Printf("[INFO]   Stream Endpoint: %s", cfg.StreamEndpoint)
		log.Printf("[INFO]   Device Paths: %v", cfg.DevicePaths)
		log.Printf("[INFO]   Chunk Size: %d bytes", cfg.ChunkSize)
		log.Printf("[INFO]   Reconnect Interval: %v", cfg.ReconnectInterval)
		log.Printf("[INFO]   Log Level: %s", cfg.LogLevel)
	}

	// Create VirtIO client
	virtioClient := virtio.NewClient(
		cfg.VirtIOURL,
		cfg.StreamEndpoint,
		cfg.ChunkSize,
		cfg.ReconnectInterval,
		cfg.LogLevel,
	)

	// Perform initial health check
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := virtioClient.HealthCheck(ctx); err != nil {
		if cfg.LogLevel == "DEBUG" || cfg.LogLevel == "INFO" {
			log.Printf("[WARN] Initial health check failed: %v (will continue anyway)", err)
		}
	}
	cancel()

	// Create device writer
	deviceWriter, err := device.NewWriter(cfg.DevicePaths, cfg.LogLevel)
	if err != nil {
		log.Fatalf("[ERROR] Failed to create device writer: %v", err)
	}
	defer deviceWriter.Close()

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Create main context
	mainCtx, mainCancel := context.WithCancel(context.Background())
	defer mainCancel()

	// Start streaming
	streamReader := virtioClient.NewStreamReader(mainCtx)
	defer streamReader.Close()

	if cfg.LogLevel == "DEBUG" || cfg.LogLevel == "INFO" {
		log.Printf("[INFO] Started streaming from VirtIO service")
	}

	// Main loop: read from stream and write to devices
	buffer := make([]byte, cfg.ChunkSize)
	bytesWritten := int64(0)
	lastLogTime := time.Now()

	go func() {
		<-sigCh
		if cfg.LogLevel == "DEBUG" || cfg.LogLevel == "INFO" {
			log.Printf("[INFO] Received shutdown signal, gracefully shutting down...")
		}
		mainCancel()
	}()

	for {
		select {
		case <-mainCtx.Done():
			if cfg.LogLevel == "DEBUG" || cfg.LogLevel == "INFO" {
				log.Printf("[INFO] Shutting down. Total bytes written: %d", bytesWritten)
			}
			return
		default:
		}

		// Read from stream
		n, err := streamReader.Read(buffer)
		if err != nil {
			if err == io.EOF || err == context.Canceled {
				if cfg.LogLevel == "DEBUG" || cfg.LogLevel == "INFO" {
					log.Printf("[INFO] Stream closed")
				}
				return
			}
			if cfg.LogLevel == "DEBUG" || cfg.LogLevel == "INFO" {
				log.Printf("[ERROR] Failed to read from stream: %v", err)
			}
			// Wait a bit before retrying
			time.Sleep(cfg.ReconnectInterval)
			continue
		}

		if n == 0 {
			continue
		}

		// Write to devices
		data := buffer[:n]
		if err := deviceWriter.Write(data); err != nil {
			if cfg.LogLevel == "DEBUG" || cfg.LogLevel == "INFO" {
				log.Printf("[ERROR] Failed to write to devices: %v", err)
			}
			// Continue anyway - device writer handles individual device errors
		} else {
			bytesWritten += int64(n)
		}

		// Periodic logging
		if cfg.LogLevel == "DEBUG" {
			now := time.Now()
			if now.Sub(lastLogTime) > 10*time.Second {
				log.Printf("[DEBUG] Bytes written: %d", bytesWritten)
				lastLogTime = now
			}
		}
	}
}
