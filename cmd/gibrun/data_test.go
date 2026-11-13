package main

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

func TestMigrateData(t *testing.T) {
	for _, tc := range []struct {
		name, old, current string
		wantError          bool
	}{
		{"missing", "", "", false},
		{"legacy", `{"requests":[{"id":"saved-1","name":"private"}]}`, "", false},
		{"both", `{"requests":[]}`, `{"collections":[]}`, false},
		{"current only", "", `{}`, false},
		{"corrupt", `{broken`, "", true},
		{"wrong shape", `[]`, "", true},
		{"current wins over corrupt", `{broken`, `{}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			old, path := filepath.Join(dir, "legacy.json"), filepath.Join(dir, "gibrun", "data.json")
			if tc.old != "" {
				if err := os.WriteFile(old, []byte(tc.old), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if tc.current != "" {
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(tc.current), 0600); err != nil {
					t.Fatal(err)
				}
			}
			for i := 0; i < 2; i++ {
				err := migrateData(old, path)
				if (err != nil) != tc.wantError {
					t.Fatalf("migration: %v", err)
				}
			}
			want := tc.current
			if want == "" && !tc.wantError {
				want = tc.old
			}
			got, err := os.ReadFile(path)
			if want == "" {
				if !os.IsNotExist(err) {
					t.Fatalf("unexpected destination: %q, %v", got, err)
				}
			} else {
				if err != nil || string(got) != want {
					t.Fatalf("destination: %q, %v", got, err)
				}
				if runtime.GOOS != "windows" {
					info, _ := os.Stat(path)
					if info.Mode().Perm() != 0600 {
						t.Fatalf("permissions: %v", info.Mode())
					}
				}
			}
			if tc.old != "" {
				got, err := os.ReadFile(old)
				if err != nil || string(got) != tc.old {
					t.Fatalf("legacy changed: %q, %v", got, err)
				}
			}
		})
	}
}

func TestDataPath(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)
	t.Setenv("APPDATA", root)
	dir, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	// macOS uses HOME/Library/Application Support, not XDG_CONFIG_HOME.
	if runtime.GOOS == "darwin" {
		t.Setenv("HOME", root)
		dir, err = os.UserConfigDir()
		if err != nil {
			t.Fatal(err)
		}
	}
	old := filepath.Join(dir, legacyApp, "data.json")
	if err := os.MkdirAll(filepath.Dir(old), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(old, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, explicit := range []string{"custom.json", "", filepath.Join(dir, "gibrun", "data.json")} {
		got, err := dataPath(explicit, true)
		if err != nil || got != explicit {
			t.Fatalf("explicit override: %q, %v", got, err)
		}
	}
	if _, err := dataPath("", false); err == nil {
		t.Fatal("corrupt legacy must stop startup")
	}
	if err := os.WriteFile(old, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := dataPath("", false)
	if err != nil || got != filepath.Join(dir, "gibrun", "data.json") {
		t.Fatalf("retry: %q, %v", got, err)
	}
}

func TestConcurrentMigration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gibrun", "data.json")
	var wg sync.WaitGroup
	for i, content := range []string{`{"requests":[{"id":"a"}]}`, `{"requests":[{"id":"b"}]}`} {
		old := filepath.Join(dir, []string{"a.json", "b.json"}[i])
		if err := os.WriteFile(old, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := migrateData(old, path); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	got, err := os.ReadFile(path)
	if err != nil || (string(got) != `{"requests":[{"id":"a"}]}` && string(got) != `{"requests":[{"id":"b"}]}`) {
		t.Fatalf("incomplete snapshot: %q, %v", got, err)
	}
	files, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".migration-*"))
	if err != nil || len(files) != 0 {
		t.Fatalf("leftover snapshots: %v, %v", files, err)
	}
}

func TestMigrationDestinationFailure(t *testing.T) {
	dir := t.TempDir()
	old, blocker := filepath.Join(dir, "old.json"), filepath.Join(dir, "blocker")
	if err := os.WriteFile(old, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(blocker, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := migrateData(old, filepath.Join(blocker, "data.json")); err == nil {
		t.Fatal("expected destination failure")
	}
	got, err := os.ReadFile(blocker)
	if err != nil || string(got) != "keep" {
		t.Fatalf("blocker changed: %q, %v", got, err)
	}
}
