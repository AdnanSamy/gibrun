package web

import (
	"solarclient/internal/store"
	"testing"
)

func TestNestedNavigation(t *testing.T) {
	data := store.Data{
		Collections: []store.Collection{{ID: "c"}},
		Folders:     []store.Folder{{ID: "a", CollectionID: "c", Name: "Users"}, {ID: "b", CollectionID: "c", ParentID: "a", Name: "Admin"}},
		Requests:    []store.Request{{ID: "root", CollectionID: "c"}, {ID: "nested", CollectionID: "c", FolderID: "b"}},
	}
	rows := navigation(data)
	if len(rows) != 4 {
		t.Fatalf("rows: %#v", rows)
	}
	for i, want := range []struct {
		id    string
		depth int
	}{{"a", 0}, {"b", 1}, {"nested", 2}, {"root", 0}} {
		if rows[i].ID != want.id || rows[i].Depth != want.depth {
			t.Fatalf("row %d: %#v", i, rows[i])
		}
	}
	// Malformed duplicate IDs must not cause infinite traversal.
	data.Folders = append(data.Folders, store.Folder{ID: "a", CollectionID: "c", ParentID: "b"})
	if len(navigation(data)) > 8 {
		t.Fatal("unbounded traversal")
	}
}
