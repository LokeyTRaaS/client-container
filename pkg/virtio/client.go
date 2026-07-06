package virtio

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// Client handles connection to Lokey VirtIO service
type Client struct {
	baseURL           string
	streamEndpoint    string
	chunkSize         int
	reconnectInterval time.Duration
	httpClient        *http.Client
	logLevel          string
}

// NewClient creates a new VirtIO client
func NewClient(baseURL, streamEndpoint string, chunkSize int, reconnectInterval time.Duration, logLevel string) *Client {
	return &Client{
		baseURL:           baseURL,
		streamEndpoint:    streamEndpoint,
		chunkSize:         chunkSize,
		reconnectInterval: reconnectInterval,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logLevel: logLevel,
	}
}

// StreamReader provides a reader interface for the VirtIO stream
type StreamReader struct {
	client  *Client
	ctx     context.Context
	cancel  context.CancelFunc
	dataCh  chan []byte
	errCh   chan error
	pending []byte // Remainder of a chunk that didn't fit into the caller's buffer
	closed  bool
	mu      chan struct{} // Simple mutex using channel
}

// NewStreamReader creates a new stream reader
func (c *Client) NewStreamReader(ctx context.Context) *StreamReader {
	streamCtx, cancel := context.WithCancel(ctx)
	sr := &StreamReader{
		client: c,
		ctx:    streamCtx,
		cancel: cancel,
		dataCh: make(chan []byte, 10),
		errCh:  make(chan error, 1),
		mu:     make(chan struct{}, 1),
	}
	sr.mu <- struct{}{} // Initialize mutex

	go sr.readLoop()
	return sr
}

// Read reads data from the stream
func (sr *StreamReader) Read(p []byte) (n int, err error) {
	// Serve buffered remainder from a previous read first
	if len(sr.pending) > 0 {
		n = copy(p, sr.pending)
		sr.pending = sr.pending[n:]
		return n, nil
	}

	select {
	case <-sr.ctx.Done():
		return 0, sr.ctx.Err()
	case err := <-sr.errCh:
		return 0, err
	case data := <-sr.dataCh:
		n = copy(p, data)
		if n < len(data) {
			// Buffer the remainder for the next read
			sr.pending = data[n:]
		}
		return n, nil
	}
}

// Close closes the stream reader
func (sr *StreamReader) Close() error {
	<-sr.mu                                // Lock
	defer func() { sr.mu <- struct{}{} }() // Unlock

	if sr.closed {
		return nil
	}
	sr.closed = true
	// Only cancel the context; closing the channels here would race with
	// readLoop, which may still be sending on them
	sr.cancel()
	return nil
}

// readLoop continuously reads from the HTTP stream with reconnection logic
func (sr *StreamReader) readLoop() {
	backoff := time.Second
	maxBackoff := 30 * time.Second

	for {
		select {
		case <-sr.ctx.Done():
			return
		default:
		}

		// Build stream URL
		streamURL := fmt.Sprintf("%s%s?chunk_size=%d", sr.client.baseURL, sr.client.streamEndpoint, sr.client.chunkSize)

		if sr.client.logLevel == "DEBUG" || sr.client.logLevel == "INFO" {
			log.Printf("[INFO] Connecting to VirtIO stream: %s", streamURL)
		}

		// Create request with context
		req, err := http.NewRequestWithContext(sr.ctx, "GET", streamURL, nil)
		if err != nil {
			sr.sendError(fmt.Errorf("failed to create request: %w", err))
			return
		}

		// Make request
		resp, err := sr.client.httpClient.Do(req)
		if err != nil {
			if sr.client.logLevel == "DEBUG" || sr.client.logLevel == "INFO" {
				log.Printf("[WARN] Failed to connect to VirtIO service: %v, retrying in %v", err, backoff)
			}
			time.Sleep(backoff)
			backoff = min(backoff*2, maxBackoff)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			err := fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			if sr.client.logLevel == "DEBUG" || sr.client.logLevel == "INFO" {
				log.Printf("[WARN] %v, retrying in %v", err, backoff)
			}
			time.Sleep(backoff)
			backoff = min(backoff*2, maxBackoff)
			continue
		}

		// Reset backoff on successful connection
		backoff = time.Second

		if sr.client.logLevel == "DEBUG" || sr.client.logLevel == "INFO" {
			log.Printf("[INFO] Connected to VirtIO stream")
		}

		// Read from stream
		buffer := make([]byte, sr.client.chunkSize)
		for {
			select {
			case <-sr.ctx.Done():
				resp.Body.Close()
				return
			default:
			}

			n, err := resp.Body.Read(buffer)
			if n > 0 {
				data := make([]byte, n)
				copy(data, buffer[:n])
				select {
				case sr.dataCh <- data:
				case <-sr.ctx.Done():
					resp.Body.Close()
					return
				}
			}

			if err != nil {
				if err == io.EOF {
					// Stream ended, reconnect
					resp.Body.Close()
					if sr.client.logLevel == "DEBUG" || sr.client.logLevel == "INFO" {
						log.Printf("[INFO] Stream ended, reconnecting...")
					}
					time.Sleep(sr.client.reconnectInterval)
					break
				}
				// Other error, reconnect
				resp.Body.Close()
				if sr.client.logLevel == "DEBUG" || sr.client.logLevel == "INFO" {
					log.Printf("[WARN] Stream read error: %v, reconnecting...", err)
				}
				time.Sleep(sr.client.reconnectInterval)
				break
			}
		}
	}
}

// sendError sends an error to the error channel
func (sr *StreamReader) sendError(err error) {
	select {
	case sr.errCh <- err:
	case <-sr.ctx.Done():
	}
}

// HealthCheck checks if the VirtIO service is healthy
func (c *Client) HealthCheck(ctx context.Context) error {
	healthURL := fmt.Sprintf("%s/health", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned non-200 status: %d", resp.StatusCode)
	}

	return nil
}
