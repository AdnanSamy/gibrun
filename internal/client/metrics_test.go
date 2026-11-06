package client

import (
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestResponseMetrics(t *testing.T) {
	body := strings.Repeat("hello 世界", 100)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		time.Sleep(30 * time.Millisecond)
		gz := gzip.NewWriter(w)
		_, _ = gz.Write([]byte(body))
		_ = gz.Close()
	}))
	defer server.Close()
	response, err := Execute(context.Background(), Request{URL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if string(response.Body) != body || response.BodySize != int64(len(body)) {
		t.Fatalf("body size: %d, want %d", response.BodySize, len(body))
	}
	if response.Duration < 30*time.Millisecond {
		t.Fatalf("duration omitted body read: %s", response.Duration)
	}
}
