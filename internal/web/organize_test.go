package web

import (
	"gibrun/internal/store"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

func TestOrganize(t *testing.T) {
	d := store.Data{Collections: []store.Collection{{ID: "a"}, {ID: "b"}}, Folders: []store.Folder{{ID: "f", CollectionID: "a"}, {ID: "g", CollectionID: "a", ParentID: "f"}}, Requests: []store.Request{{ID: "r", CollectionID: "a", FolderID: "g"}}}
	v := url.Values{"kind": {"folder"}, "action": {"move"}, "id": {"f"}, "collection": {"a"}, "parent": {"g"}}
	if organize(&d, v) == nil {
		t.Fatal("accepted cycle")
	}
	v.Set("collection", "b")
	if organize(&d, v) == nil {
		t.Fatal("accepted cross-collection parent")
	}
	v.Set("parent", "missing")
	if organize(&d, v) == nil {
		t.Fatal("accepted invalid parent")
	}
	v.Set("parent", "")
	if err := organize(&d, v); err != nil {
		t.Fatal(err)
	}
	if d.Folders[1].CollectionID != "b" || d.Requests[0].CollectionID != "b" {
		t.Fatal("subtree not moved")
	}
	v.Set("action", "rename")
	v.Set("name", "renamed")
	if err := organize(&d, v); err != nil || d.Folders[0].Name != "renamed" {
		t.Fatal(err)
	}
	v.Set("action", "delete")
	if organize(&d, v) == nil {
		t.Fatal("accepted unconfirmed deletion")
	}
	v.Set("confirm", "yes")
	v.Set("counts", deletionCounts(d, "folder", "f"))
	if v.Get("counts") != "1 descendant folders, 1 requests" {
		t.Fatal(v)
	}
	if err := organize(&d, v); err != nil {
		t.Fatal(err)
	}
	if len(d.Folders) != 0 || len(d.Requests) != 0 || len(d.Collections) != 2 {
		t.Fatal("incorrect cascade")
	}
	v = url.Values{"kind": {"folder"}, "action": {"create"}, "collection": {"a"}, "name": {"new"}}
	if err := organize(&d, v); err != nil || len(d.Folders) != 1 {
		t.Fatal(err)
	}
	v = url.Values{"kind": {"collection"}, "action": {"delete"}, "id": {"a"}, "confirm": {"yes"}, "counts": {deletionCounts(d, "collection", "a")}}
	if err := organize(&d, v); err != nil || len(d.Folders) != 0 || len(d.Collections) != 1 {
		t.Fatal(err)
	}
}

func TestOrganizeRollback(t *testing.T) {
	s := &Server{path: filepath.Join(t.TempDir(), "missing", ".."), data: store.Data{Collections: []store.Collection{{ID: "a", Name: "original"}}}}
	v := url.Values{"kind": {"collection"}, "action": {"rename"}, "id": {"a"}, "name": {"changed"}}
	r := httptest.NewRequest("POST", "/organize", strings.NewReader(v.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	s.action(w, r)
	if s.data.Collections[0].Name != "original" || !strings.Contains(w.Body.String(), "Save failed") {
		t.Fatal("failed save did not roll back", w.Body.String())
	}
}
