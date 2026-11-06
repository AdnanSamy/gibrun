// Package client executes HTTP requests.
package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Request describes one HTTP exchange. A zero MaxResponseBytes uses DefaultMaxResponseBytes.
type Request struct {
	Method             string
	URL                string
	Headers            http.Header
	Body               []byte
	Timeout            time.Duration
	InsecureSkipVerify bool
	MaxResponseBytes   int64
}

// Response contains the complete response body and metadata.
type Response struct {
	StatusCode int
	Status     string
	Headers    http.Header
	Body       []byte
	// Duration covers the request through reading the body, including redirects.
	Duration time.Duration
	// BodySize counts bytes read after automatic HTTP decompression, not wire bytes.
	BodySize int64
}

const DefaultMaxResponseBytes int64 = 10 << 20

var insecureTransport = func() *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{}
	} else {
		transport.TLSClientConfig = transport.TLSClientConfig.Clone()
	}
	// Explicit opt-in for connecting to servers with untrusted certificates.
	transport.TLSClientConfig.InsecureSkipVerify = true //nolint:gosec
	return transport
}()

// Execute performs a request. Context cancellation and Timeout stop the exchange.
func Execute(ctx context.Context, input Request) (Response, error) {
	if input.MaxResponseBytes < 0 {
		return Response{}, fmt.Errorf("max response bytes must not be negative")
	}
	limit := input.MaxResponseBytes
	if limit == 0 {
		limit = DefaultMaxResponseBytes
	}
	req, err := http.NewRequestWithContext(ctx, input.Method, input.URL, bytes.NewReader(input.Body))
	if err != nil {
		return Response{}, fmt.Errorf("create request: %w", err)
	}
	req.Header = input.Headers.Clone()
	var transport http.RoundTripper = http.DefaultTransport
	if input.InsecureSkipVerify {
		transport = insecureTransport
	}
	c := &http.Client{Transport: transport, Timeout: input.Timeout}
	start := time.Now()
	resp, err := c.Do(req)
	if err != nil {
		return Response{}, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()
	// Read at most limit bytes, then probe for another byte without overflowing limit.
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit))
	if err != nil {
		return Response{}, fmt.Errorf("read response: %w", err)
	}
	if int64(len(body)) == limit {
		var extra [1]byte
		n, err := io.ReadFull(resp.Body, extra[:])
		if n != 0 {
			return Response{}, fmt.Errorf("response exceeds %d bytes", limit)
		}
		if err != nil && err != io.EOF {
			return Response{}, fmt.Errorf("read response: %w", err)
		}
	}
	return Response{StatusCode: resp.StatusCode, Status: resp.Status, Duration: time.Since(start), BodySize: int64(len(body)), Headers: resp.Header.Clone(), Body: body}, nil
}
