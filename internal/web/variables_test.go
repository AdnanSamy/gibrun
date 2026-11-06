package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"solarclient/internal/store"
)

func TestVariableCRUD(t *testing.T) {
	s, err := New(filepath.Join(t.TempDir(), "data.json"), "localhost")
	if err != nil {
		t.Fatal(err)
	}
	s.data.Collections = []store.Collection{{ID: "c", Name: "Collection"}}
	post := func(form url.Values) *httptest.ResponseRecorder {
		t.Helper()
		form.Set("token", s.token)
		r := httptest.NewRequest("POST", "http://localhost/variables", strings.NewReader(form.Encode()))
		r.Header.Set("Origin", "http://localhost")
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.AddCookie(&http.Cookie{Name: "solar_session", Value: s.token})
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		return w
	}
	form := url.Values{"kind": {"environment"}, "action": {"create"}, "name": {"Dev"}, "variables": {`[{"key":"host","value":"secret\nvalue","enabled":false}]`}}
	post(form)
	if len(s.data.Environments) != 1 {
		t.Fatal(s.data)
	}
	target := s.data.Environments[0].ID
	form.Set("target", target)
	form.Set("action", "save")
	form.Set("name", "Updated")
	post(form)
	loaded, err := store.Load(s.path)
	if err != nil || !reflect.DeepEqual(loaded, s.data) || loaded.Environments[0].Variables[0].Enabled || loaded.Environments[0].Name != "Updated" {
		t.Fatal(loaded, err)
	}
	before := variableJSON(s.data.Environments[0].Variables)
	for _, target := range []string{"", "missing", "../bad"} {
		form.Set("target", target)
		if !strings.Contains(post(form).Body.String(), "variable scope not found") {
			t.Fatal("invalid ID accepted")
		}
	}
	form.Set("target", target)
	form.Set("variables", `[{"key":"x","value":"secret","enabled":true,"unknown":1}]`)
	if !strings.Contains(post(form).Body.String(), "variables must be") || variableJSON(s.data.Environments[0].Variables) != before {
		t.Fatal("invalid JSON changed variables")
	}
	form.Set("kind", "collection")
	form.Set("target", "c")
	form.Set("variables", `[{"key":"host","value":"collection","enabled":true}]`)
	post(form)
	if len(s.data.Collections[0].Variables) != 1 {
		t.Fatal("collection variables not saved")
	}
	form.Set("target", "missing")
	if !strings.Contains(post(form).Body.String(), "variable scope not found") {
		t.Fatal("invalid collection accepted")
	}
	form.Set("kind", "environment")
	form.Set("target", target)
	form.Set("action", "delete")
	post(form)
	if len(s.data.Environments) != 1 {
		t.Fatal("unconfirmed delete")
	}
	originalPath := s.path
	s.path = t.TempDir() // Atomic rename onto a directory must fail.
	form.Set("confirm", "yes")
	if !strings.Contains(post(form).Body.String(), "Save failed") || len(s.data.Environments) != 1 {
		t.Fatal("rollback failed")
	}
	s.path = originalPath
	post(form)
	loaded, err = store.Load(s.path)
	if err != nil || len(loaded.Environments) != 0 {
		t.Fatal("delete not persisted", err)
	}
}

func TestVariablePrecedenceAndSelection(t *testing.T) {
	data := store.Data{Collections: []store.Collection{{ID: "c", Variables: []store.Field{{Key: "x", Value: "collection", Enabled: true}, {Key: "y", Value: "fallback", Enabled: true}}}}, Environments: []store.Environment{{ID: "e", Variables: []store.Field{{Key: "x", Value: "environment", Enabled: true}, {Key: "x", Value: "last", Enabled: true}, {Key: "y", Value: "disabled", Enabled: false}}}}}
	vars := variablesFor(data, "c", "e")
	if vars["x"] != "last" || vars["y"] != "fallback" {
		t.Fatal(vars)
	}
	q, err := resolve(store.Request{URL: "http://{{x}}/{{y}}"}, vars)
	if err != nil || q.URL != "http://last/fallback" {
		t.Fatal(q, err)
	}
	s := &Server{data: data}
	for _, method := range []string{"GET", "POST"} {
		r := httptest.NewRequest(method, "http://localhost/", nil)
		r.AddCookie(&http.Cookie{Name: "solar_environment", Value: "e"})
		if method == "POST" {
			r.Form = url.Values{"environment": {"e"}}
		}
		w := httptest.NewRecorder()
		s.renderFor(w, r, "", "", nil)
		if !strings.Contains(w.Body.String(), `value="e" selected`) {
			t.Fatal("selection lost", method)
		}
	}
	r := httptest.NewRequest("GET", "http://localhost/", nil)
	r.AddCookie(&http.Cookie{Name: "solar_environment", Value: "deleted"})
	w := httptest.NewRecorder()
	s.renderFor(w, r, "", "", nil)
	if w.Result().Cookies()[0].Value != "" {
		t.Fatal("stale selection retained")
	}
}
