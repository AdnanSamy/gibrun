package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gibrun/internal/store"
)

func TestTimeoutSaveValidation(t *testing.T) {
	for _, text := range []string{"", "1", "30", "300", "0", "-1", "301", "1.5", "abc", "99999999999999999999"} {
		t.Run(text, func(t *testing.T) {
			s, err := New(filepath.Join(t.TempDir(), "data.json"), "localhost")
			if err != nil {
				t.Fatal(err)
			}
			values := url.Values{"action": {"save"}, "url": {"https://example.com"}, "timeout_seconds": {text}}
			req := httptest.NewRequest("POST", "/request", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()
			s.action(w, req)
			want := map[string]int{"": 30, "1": 1, "30": 30, "300": 300}[text]
			if want == 0 {
				if w.Code != 422 || len(s.data.Requests) != 0 {
					t.Fatalf("invalid timeout accepted: %d", w.Code)
				}
				return
			}
			data, err := store.Load(s.path)
			if err != nil {
				t.Fatal(err)
			}
			if len(data.Requests) != 1 || data.Requests[0].TimeoutSeconds != want {
				t.Fatalf("timeout lost: %#v", data.Requests)
			}
			if !strings.Contains(w.Body.String(), `id="timeout-seconds"`) {
				t.Fatal("timeout control missing")
			}
		})
	}
}

func TestSendTimeoutAndLegacy(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/slow" {
			<-r.Context().Done()
			return
		}
		w.WriteHeader(204)
	}))
	defer target.Close()
	req := httptest.NewRequest("POST", "/request", nil)
	var legacy store.Request
	if err := json.Unmarshal([]byte(`{"method":"GET","url":"`+target.URL+`"}`), &legacy); err != nil {
		t.Fatal(err)
	}
	if legacy.TimeoutSeconds != 0 {
		t.Fatal("legacy timeout should use default")
	}
	if _, err := send(req, legacy); err != nil {
		t.Fatalf("legacy default: %v", err)
	}
	for _, seconds := range []int{-1, 301} {
		if _, err := send(req, store.Request{URL: target.URL, TimeoutSeconds: seconds}); err == nil {
			t.Fatal("invalid stored timeout accepted")
		}
	}
	start := time.Now()
	_, err := send(req, store.Request{URL: target.URL + "/slow", TimeoutSeconds: 1})
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > 5*time.Second {
		t.Fatalf("timeout: %v (%v)", err, time.Since(start))
	}
}

// Disconnecting the browser-side HTTP request must cancel the outbound exchange.
func TestBrowserDisconnectCancelsOutbound(t *testing.T) {
	started, stopped := make(chan struct{}), make(chan struct{})
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-r.Context().Done():
			close(stopped)
		case <-time.After(5 * time.Second):
		}
	}))
	defer target.Close()
	s, err := New(filepath.Join(t.TempDir(), "data.json"), "localhost")
	if err != nil {
		t.Fatal(err)
	}
	frontend := httptest.NewServer(http.HandlerFunc(s.action))
	defer frontend.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	values := url.Values{"action": {"send"}, "url": {target.URL}, "timeout_seconds": {"300"}}
	req, _ := http.NewRequestWithContext(ctx, "POST", frontend.URL+"/request", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	done := make(chan error, 1)
	go func() {
		resp, err := http.DefaultClient.Do(req)
		if resp != nil {
			resp.Body.Close()
		}
		done <- err
	}()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("outbound request not started")
	}
	cancel()
	select {
	case <-stopped:
	case <-time.After(3 * time.Second):
		t.Fatal("outbound context not cancelled")
	}
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("browser cancellation: %v", err)
	}
}
