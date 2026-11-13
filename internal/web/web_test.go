package web

import (
	"gibrun/internal/store"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

func TestSecurityAndExecution(t *testing.T) {
	s, err := New(filepath.Join(t.TempDir(), "data.json"), "127.0.0.1:1234")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "http://127.0.0.1:1234/?token="+s.token, nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != 303 {
		t.Fatal(w.Code)
	}
	cookie := w.Result().Cookies()[0]
	form := url.Values{"token": {s.token}, "name": {"x"}}
	req = httptest.NewRequest("POST", "http://127.0.0.1:1234/collection", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Fatal("missing origin accepted")
	}
	req = httptest.NewRequest("POST", "http://127.0.0.1:1234/collection", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://127.0.0.1:1234")
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != 200 || len(s.data.Collections) != 1 {
		t.Fatal(w.Code, w.Body.String())
	}
	req = httptest.NewRequest("GET", "http://evil.test/", nil)
	req.AddCookie(cookie)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatal("invalid host accepted")
	}
	_, err = send(httptest.NewRequest("GET", "http://localhost/", nil), store.Request{URL: "http://localhost", Auth: store.Auth{Type: "apikey"}})
	if err == nil {
		t.Fatal("unsupported auth accepted")
	}
	_, err = resolve(store.Request{URL: "{{missing}}"}, nil)
	if err == nil {
		t.Fatal("missing variable accepted")
	}
}
