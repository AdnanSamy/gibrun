package web

import (
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeletedSnapshotSave(t *testing.T) {
	s, err := New(filepath.Join(t.TempDir(), "data.json"), "localhost")
	if err != nil {
		t.Fatal(err)
	}
	values := url.Values{"id": {"deleted"}, "action": {"save"}, "name": {"My draft"}, "body": {"keep me"}}
	submit := func() string {
		r := httptest.NewRequest("POST", "/request", strings.NewReader(values.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		s.action(w, r)
		return w.Body.String()
	}
	html := submit()
	if len(s.data.Requests) != 0 || !strings.Contains(html, "Request no longer exists") || !strings.Contains(html, "keep me") {
		t.Fatal("stale save recreated request or lost draft", html)
	}
	values.Set("id", "")
	submit()
	if len(s.data.Requests) != 1 || s.data.Requests[0].ID == "deleted" || s.data.Requests[0].Body.Text != "keep me" {
		t.Fatal("explicit new save failed")
	}
}
