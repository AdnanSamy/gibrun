package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestExecute(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if r.Method != http.MethodPost || r.Header.Get("X-Test") != "value" || string(body) != "payload" {
			t.Errorf("unexpected request: %s %q %q", r.Method, r.Header.Get("X-Test"), body)
		}
		w.Header().Set("X-Reply", "yes")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("hello"))
	}))
	defer s.Close()
	request := Request{Method: http.MethodPost, URL: s.URL, Headers: http.Header{"X-Test": {"value"}}, Body: []byte("payload"), MaxResponseBytes: 5}
	resp, err := Execute(context.Background(), request)
	if err != nil || resp.StatusCode != http.StatusCreated || resp.Headers.Get("X-Reply") != "yes" || string(resp.Body) != "hello" {
		t.Fatalf("response: %#v, error: %v", resp, err)
	}
	request.MaxResponseBytes = 4
	if _, err := Execute(context.Background(), request); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected response limit error, got %v", err)
	}
	request.MaxResponseBytes = -1
	if _, err := Execute(context.Background(), request); err == nil {
		t.Fatal("expected negative limit error")
	}
}

func TestTLSAndCancellation(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer s.Close()
	if _, err := Execute(context.Background(), Request{URL: s.URL}); err == nil {
		t.Fatal("expected certificate error")
	}
	if _, err := Execute(context.Background(), Request{URL: s.URL, InsecureSkipVerify: true}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Execute(ctx, Request{URL: s.URL}); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer slow.Close()
	if _, err := Execute(context.Background(), Request{URL: slow.URL, Timeout: time.Millisecond}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
}
