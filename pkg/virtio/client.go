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

// StreamReader provides a reader interface for the VirtIO stream.
// It is not safe for concurrent use: Read must only be called from one
// goroutine at a time (the usual io.Reader convention). Close may be
// called concurrently with Read.
type StreamReader struct {
	client  *Client
	ctx     context.Context
	cancel  context.CancelFunc
	dataCh  chan []byte
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
		mu:     make(chan struct{}, 1),
	}
	sr.mu <- struct{}{} // Initialize mutex

	go sr.readLoop()
	return sr
}

// Read reads data from the stream. After Close (or cancellation of the
// parent context) it returns the context error, though already-buffered
// chunks may still be delivered first.
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
	case data := <-sr.dataCh:
		n = copy(p, data)
		// Buffer any remainder for the next read
		sr.pending = data[n:]
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
			// Request construction only fails for malformed configuration
			// (e.g. a bad URL). Log loudly and keep retrying like every other
			// error path, so a consumer blocked in Read is never stranded
			// without a producer.
			log.Printf("[ERROR] Failed to create stream request: %v, retrying in %v", err, backoff)
			sleepCtx(sr.ctx, backoff)
			backoff = min(backoff*2, maxBackoff)
			continue
		}

		// Make request
		resp, err := sr.client.httpClient.Do(req)
		if err != nil {
			if sr.client.logLevel == "DEBUG" || sr.client.logLevel == "INFO" {
				log.Printf("[WARN] Failed to connect to VirtIO service: %v, retrying in %v", err, backoff)
			}
			sleepCtx(sr.ctx, backoff)
			backoff = min(backoff*2, maxBackoff)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			sr.client.closeBody(resp.Body)
			err := fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			if sr.client.logLevel == "DEBUG" || sr.client.logLevel == "INFO" {
				log.Printf("[WARN] %v, retrying in %v", err, backoff)
			}
			sleepCtx(sr.ctx, backoff)
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
				sr.client.closeBody(resp.Body)
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
					sr.client.closeBody(resp.Body)
					return
				}
			}

			if err != nil {
				if err == io.EOF {
					// Stream ended, reconnect
					sr.client.closeBody(resp.Body)
					if sr.client.logLevel == "DEBUG" || sr.client.logLevel == "INFO" {
						log.Printf("[INFO] Stream ended, reconnecting...")
					}
					sleepCtx(sr.ctx, sr.client.reconnectInterval)
					break
				}
				// Other error, reconnect
				sr.client.closeBody(resp.Body)
				if sr.client.logLevel == "DEBUG" || sr.client.logLevel == "INFO" {
					log.Printf("[WARN] Stream read error: %v, reconnecting...", err)
				}
				sleepCtx(sr.ctx, sr.client.reconnectInterval)
				break
			}
		}
	}
}

// closeBody closes an HTTP response body, logging any close error
func (c *Client) closeBody(body io.Closer) {
	if err := body.Close(); err != nil {
		if c.logLevel == "DEBUG" || c.logLevel == "INFO" {
			log.Printf("[WARN] Error closing response body: %v", err)
		}
	}
}

// sleepCtx waits for the given duration or until ctx is cancelled,
// whichever comes first
func sleepCtx(ctx context.Context, d time.Duration) {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-ctx.Done():
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
	defer c.closeBody(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned non-200 status: %d", resp.StatusCode)
	}

	return nil
}
