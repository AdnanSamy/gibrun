package web

import (
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestTabstripRendering(t *testing.T) {
	s, err := New(filepath.Join(t.TempDir(), "data.json"), "127.0.0.1:1234")
	if err != nil {
		t.Fatal(err)
	}
	login := httptest.NewRecorder()
	s.ServeHTTP(login, httptest.NewRequest("GET", "http://127.0.0.1:1234/?token="+s.token, nil))
	req := httptest.NewRequest("GET", "http://127.0.0.1:1234/", nil)
	req.AddCookie(login.Result().Cookies()[0])
	result := httptest.NewRecorder()
	s.ServeHTTP(result, req)
	if result.Code != 200 {
		t.Fatal(result.Code)
	}
	for _, want := range []string{`src="/static/tabs.js"`, `.workspace-tabs [role=tablist]{flex:1 1 0`, `scrollbar-width:none`, `.workspace-tabs .request-tab.active`, `.workspace-tabs .new-request-tab`, `min-width:0;max-width:100%`} {
		if !strings.Contains(result.Body.String(), want) {
			t.Errorf("missing tabstrip markup/style %q", want)
		}
	}
}
