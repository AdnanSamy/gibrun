package web

import (
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestSplitRendering(t *testing.T) {
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
	body := result.Body.String()
	for _, want := range []string{`src="/static/split.js"`, `class="request-editor"`, `role="separator"`, `aria-orientation="horizontal"`, `aria-valuenow="50"`, `class="card response"`} {
		if !strings.Contains(body, want) {
			t.Errorf("missing split markup %q", want)
		}
	}
	for _, want := range []string{
		`@media (min-width: 851px) and (min-height: 650px)`,
		`grid-template-rows: auto minmax(0, 1fr)`,
		`.shell > .workspace-tabs + .workspace { grid-row: 2; }`,
		`min-height: 0; padding: 12px`,
		`flex: 0 0 8px; cursor: row-resize`,
		`.request-editor { flex: 1; min-height: 0; overflow: auto; }`,
		`.response { flex: 1 1 0; min-height: 0; margin: 0; overflow: auto; }`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing split layout %q", want)
		}
	}
	if strings.Index(body, `id="save-request"`) > strings.Index(body, `class="request-editor"`) {
		t.Error("save must precede scrolling editor")
	}
}
