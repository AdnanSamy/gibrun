package web

import (
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"solarclient/internal/store"
)

func TestDuplicateRollback(t *testing.T) {
	s := &Server{path: filepath.Join(t.TempDir(), "missing", ".."), data: store.Data{Requests: []store.Request{{ID: "r", Name: "original"}}}}
	v := url.Values{"kind": {"request"}, "action": {"duplicate"}, "id": {"r"}}
	r := httptest.NewRequest("POST", "/organize", strings.NewReader(v.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	s.action(w, r)
	if len(s.data.Requests) != 1 || s.data.Requests[0].Name != "original" || !strings.Contains(w.Body.String(), "Save failed") {
		t.Fatal("duplicate persistence failure did not roll back")
	}
}

func TestDuplicateRequest(t *testing.T) {
	original := store.Request{ID: "r", Name: "Original", CollectionID: "c", FolderID: "f", Method: "POST", URL: "https://example.test", TimeoutSeconds: 42,
		Headers: []store.Field{{Key: "header", Value: "secret", Enabled: true}}, Query: []store.Field{{Key: "q", Value: "disabled"}},
		Body: store.Body{Type: "multipart", Text: "body", Fields: []store.BodyField{{Field: store.Field{Key: "file", Enabled: true}, File: "reference"}}},
		Auth: store.Auth{Type: "basic", Username: "u", Password: "secret"}, TLS: store.TLS{InsecureSkipVerify: true, CACertFile: "reference"}}
	d := store.Data{Requests: []store.Request{original}}
	v := url.Values{"kind": {"request"}, "action": {"duplicate"}, "id": {"r"}}
	if err := organize(&d, v); err != nil {
		t.Fatal(err)
	}
	copy := d.Requests[1]
	if copy.ID == "" || copy.ID == original.ID || copy.Name != "Original (copy)" {
		t.Fatal("invalid duplicate identity")
	}
	copy.ID, copy.Name = original.ID, original.Name
	if !reflect.DeepEqual(copy, original) {
		t.Fatal("duplicate lost saved fields")
	}
	d.Requests[1].Headers[0].Value = "changed"
	d.Requests[1].Query[0].Value = "changed"
	d.Requests[1].Body.Fields[0].File = "changed"
	if !reflect.DeepEqual(d.Requests[0], original) || original.Headers[0].Value != "secret" || original.Query[0].Value != "disabled" || original.Body.Fields[0].File != "reference" {
		t.Fatal("duplicate shares mutable fields")
	}
	v.Set("id", "deleted")
	if organize(&d, v) == nil {
		t.Fatal("duplicated missing request")
	}
	v.Set("kind", "folder")
	if organize(&d, v) == nil {
		t.Fatal("duplicated folder")
	}
}
