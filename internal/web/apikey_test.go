package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"solarclient/internal/store"
	"strings"
	"testing"
)

func TestAPIKeyEditorSave(t *testing.T) {
	s, err := New(filepath.Join(t.TempDir(), "data.json"), "localhost")
	if err != nil {
		t.Fatal(err)
	}
	values := url.Values{"action": {"save"}, "name": {"key request"}, "method": {"GET"}, "url": {"https://example.com"}, "auth_type": {"apikey"}, "auth_key_name": {"{{name}}"}, "auth_key_value": {"{{secret}}"}, "auth_key_location": {"query"}}
	req := httptest.NewRequest("POST", "/request", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	s.action(httptest.NewRecorder(), req)
	data, err := store.Load(s.path)
	if err != nil {
		t.Fatal(err)
	}
	want := store.Auth{Type: "apikey", KeyName: "{{name}}", KeyValue: "{{secret}}", KeyLocation: "query"}
	if len(data.Requests) != 1 || data.Requests[0].Auth != want {
		t.Fatal("API-key editor fields not persisted")
	}
}

func TestAPIKeyExecution(t *testing.T) {
	for _, location := range []string{"header", "query"} {
		t.Run(location, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got := r.Header.Values("X-Key")
				if location == "query" {
					got = r.URL.Query()["X-Key"]
				}
				if len(got) != 1 || got[0] != "secret & value" {
					t.Errorf("wrong credentials count/value")
				}
			}))
			defer server.Close()
			q := store.Request{Method: "GET", URL: server.URL + "?X-Key=old&X-Key=older", Auth: store.Auth{Type: "apikey", KeyName: "{{name}}", KeyValue: "{{value}}", KeyLocation: location}, Headers: []store.Field{{Key: "x-key", Value: "old", Enabled: true}}, Query: []store.Field{{Key: "X-Key", Value: "old", Enabled: true}}}
			q, err := resolve(q, map[string]string{"name": "X-Key", "value": "secret & value"})
			if err != nil {
				t.Fatal(err)
			}
			if _, err = send(httptest.NewRequest("GET", "/", nil), q); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAPIKeyInvalidAndRedaction(t *testing.T) {
	for _, auth := range []store.Auth{
		{Type: "apikey", KeyName: "", KeyValue: "secret", KeyLocation: "header"},
		{Type: "apikey", KeyName: "X-Key", KeyValue: "secret", KeyLocation: "secret"},
		{Type: "apikey", KeyName: "secret\n", KeyValue: "secret", KeyLocation: "header"},
		{Type: "apikey", KeyName: "X-Key", KeyValue: "secret\n", KeyLocation: "header"},
		{Type: "apikey", KeyName: "X-Key", KeyValue: "secret", KeyLocation: "query"},
	} {
		_, err := send(httptest.NewRequest("GET", "/", nil), store.Request{Method: "GET", URL: "http://127.0.0.1:1", Auth: auth})
		if err == nil || strings.Contains(err.Error(), "secret") {
			t.Fatalf("unsafe error: %v", err)
		}
	}
	_, err := resolve(store.Request{Auth: store.Auth{Type: "apikey", KeyValue: "{{secret}}"}}, nil)
	if err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatalf("unsafe resolution error: %v", err)
	}
}
