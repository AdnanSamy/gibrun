package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gibrun/internal/store"
)

const legacyApp = "solarclient"

// dataPath leaves explicit -data paths alone, including the default path itself.
func dataPath(explicit string, overridden bool) (string, error) {
	if overridden {
		return explicit, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "gibrun", "data.json")
	return path, migrateData(filepath.Join(dir, legacyApp, "data.json"), path)
}

// migrateData publishes a validated private snapshot exclusively. The legacy
// store remains untouched; failed attempts leave no partial destination.
func migrateData(old, path string) error {
	if _, err := os.Lstat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect gibrun store: %w", err)
	}
	if _, err := store.Load(old); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("validate legacy store %s: %w", old, err)
	}
	source, err := os.Open(old)
	if err != nil {
		return fmt.Errorf("open legacy store: %w", err)
	}
	defer source.Close()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create gibrun directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".migration-*")
	if err != nil {
		return fmt.Errorf("create migration snapshot: %w", err)
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	if _, err := io.Copy(tmp, source); err != nil {
		return fmt.Errorf("copy legacy store: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sync migration snapshot: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close migration snapshot: %w", err)
	}
	// Validate the copied bytes too, in case the source changed during copying.
	if _, err := store.Load(tmp.Name()); err != nil {
		return fmt.Errorf("validate migration snapshot: %w", err)
	}
	// Unlike Rename, Link cannot overwrite a concurrently created destination.
	// ponytail: requires hard links; unsupported filesystems fail safely, use -data.
	if err := os.Link(tmp.Name(), path); err != nil && !errors.Is(err, os.ErrExist) {
		return fmt.Errorf("publish migrated store: %w", err)
	}
	return nil
}
