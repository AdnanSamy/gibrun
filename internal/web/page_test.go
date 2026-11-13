package web

import (
	"net/http/httptest"
	"strings"
	"testing"

	"gibrun/internal/store"
)

func TestWorkbenchRender(t *testing.T) {
	s := &Server{token: "test", data: store.Data{
		Collections: []store.Collection{{ID: "c", Name: "Example API"}},
		Requests:    []store.Request{{ID: "r", CollectionID: "c", Name: "<script>alert(1)</script>", Method: "GET", URL: "https://example.com"}},
	}}
	w := httptest.NewRecorder()
	s.render(w, "r", "", nil)
	html := w.Body.String()
	for _, text := range []string{"Example API", `aria-current="page"`, "Your response lands here", `name="token" value="test"`, "&lt;script&gt;", `name="auth_key_name"`, `name="auth_key_value" type="password"`, `name="auth_key_location"`, `<option value="apikey">API key</option>`} {
		if !strings.Contains(html, text) {
			t.Fatalf("missing rendered content %q", text)
		}
	}
	if strings.Contains(html, "<summary>+ Create collection</summary>") {
		t.Fatal("legacy collection form still displayed")
	}
	if !strings.Contains(html, `id="collection-menu"`) || !strings.Contains(html, `data-collection="c"`) {
		t.Fatal("context menu missing")
	}
	if strings.Contains(html, "<script>alert(1)</script>") {
		t.Fatal("request name not escaped")
	}
}
