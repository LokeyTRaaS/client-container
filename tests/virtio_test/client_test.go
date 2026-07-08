package virtio_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lokey/client-container/pkg/virtio"
)

func TestClient_HealthCheck(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"healthy"}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := virtio.NewClient(server.URL, "/stream", 1024, 5*time.Second, "INFO")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.HealthCheck(ctx)
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
}

func TestClient_HealthCheck_Failure(t *testing.T) {
	// Create mock server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := virtio.NewClient(server.URL, "/stream", 1024, 5*time.Second, "INFO")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.HealthCheck(ctx)
	if err == nil {
		t.Fatal("Expected health check to fail")
	}
}

func TestStreamReader_Read(t *testing.T) {
	// Create mock server that streams data
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/stream" {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)
			// Write some test data
			w.Write([]byte("test data 12345"))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := virtio.NewClient(server.URL, "/stream", 1024, 5*time.Second, "INFO")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	reader := client.NewStreamReader(ctx)
	defer reader.Close()

	// Read data
	buffer := make([]byte, 100)
	n, err := reader.Read(buffer)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if n == 0 {
		t.Fatal("Expected to read some data")
	}

	// Verify data
	data := string(buffer[:n])
	if data != "test data 12345" {
		t.Fatalf("Unexpected data: got %q, expected %q", data, "test data 12345")
	}
}

func TestStreamReader_SmallBufferReads(t *testing.T) {
	payload := []byte("0123456789abcdefghijklmnopqrstuv")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/stream" {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)
			w.Write(payload)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := virtio.NewClient(server.URL, "/stream", 1024, 5*time.Second, "INFO")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	reader := client.NewStreamReader(ctx)
	defer reader.Close()

	// Read with a buffer smaller than the streamed chunk; the remainder must
	// be preserved across reads and delivered in order
	got := make([]byte, 0, len(payload))
	buf := make([]byte, 7)
	for len(got) < len(payload) {
		n, err := reader.Read(buf)
		if err != nil {
			t.Fatalf("Read failed after %d bytes: %v", len(got), err)
		}
		got = append(got, buf[:n]...)
	}

	if string(got) != string(payload) {
		t.Fatalf("Data mismatch: got %q, expected %q", got, payload)
	}
}
