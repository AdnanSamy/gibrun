package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"solarclient/internal/store"
)

func TestDisabledFieldsSaveAndExecute(t *testing.T) {
	s, err := New(filepath.Join(t.TempDir(), "data.json"), "localhost")
	if err != nil {
		t.Fatal(err)
	}
	values := url.Values{
		"action": {"save"}, "name": {"Example"}, "method": {"GET"}, "url": {"https://example.com"},
		"headers": {"X-Enabled: yes"}, "headers_disabled": {"X-Disabled: {{missing}}"},
		"query": {"active: yes"}, "query_disabled": {"disabled: {{missing}}"},
	}
	req := httptest.NewRequest("POST", "/request", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	s.action(httptest.NewRecorder(), req)
	data, err := store.Load(s.path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Requests) != 1 {
		t.Fatal("request not saved")
	}
	q := data.Requests[0]
	for _, fields := range [][]store.Field{q.Headers, q.Query} {
		if len(fields) != 2 || !fields[0].Enabled || fields[1].Enabled || fields[1].Value != "{{missing}}" {
			t.Fatalf("lost disabled entry: %#v", fields)
		}
	}
	if _, err := resolve(q, nil); err != nil {
		t.Fatalf("disabled variable blocks execution: %v", err)
	}
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Disabled") != "" || r.URL.Query().Has("disabled") {
			t.Error("disabled entries sent")
		}
		if r.Header.Get("X-Enabled") != "yes" || r.URL.Query().Get("active") != "yes" {
			t.Error("enabled entries missing")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer target.Close()
	q.URL = target.URL
	if _, err := send(httptest.NewRequest("GET", "/", nil), q); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	s.render(w, q.ID, "", nil)
	if !strings.Contains(w.Body.String(), "X-Disabled: {{missing}}") {
		t.Fatal("disabled header not rendered")
	}
}
