package store

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSaveLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private", "store.json")
	want := Data{
		Collections:  []Collection{{ID: "c", Name: "collection"}},
		Folders:      []Folder{{ID: "f", CollectionID: "c", ParentID: "parent", Name: "folder"}},
		Requests:     []Request{{ID: "r", CollectionID: "c", FolderID: "f", Name: "request", Method: "POST", URL: "https://example.com", Headers: []Field{{Key: "X", Value: "1", Enabled: true}, {Key: "X", Value: "2"}}, Query: []Field{{Key: "q", Value: "v", Enabled: true}}, Body: Body{Type: "json", Text: "{}"}, Auth: Auth{Type: "basic", Username: "u", Password: "secret", Token: "token", KeyName: "X-Key", KeyValue: "secret-key", KeyLocation: "header"}, TLS: TLS{InsecureSkipVerify: true, CACertFile: "ca", ClientCertFile: "cert", ClientKeyFile: "key"}}},
		Environments: []Environment{{ID: "e", Name: "env", Variables: []Field{{Key: "secret", Value: "value", Enabled: true}}}},
	}
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("permissions: %o", info.Mode().Perm())
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if err := Save(path, Data{}); err != nil {
		t.Fatal(err)
	}
	info, err = os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("replacement permissions: %o", info.Mode().Perm())
	}
}

func TestLoadErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.json")
	if _, err := Load(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing: %v", err)
	}
	if err := os.WriteFile(path, []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected invalid JSON error")
	}
}

func TestSaveFailurePreservesDestination(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "destination")
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if err := Save(path, Data{}); err == nil {
		t.Fatal("expected rename error")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "destination" {
		t.Fatalf("unexpected entries: %v", entries)
	}
}
