// Package store persists client data as a private, atomically replaced JSON file.
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Data stores flat records linked by their IDs.
type Data struct {
	Collections  []Collection  `json:"collections"`
	Folders      []Folder      `json:"folders"`
	Requests     []Request     `json:"requests"`
	Environments []Environment `json:"environments"`
}

type Collection struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Variables []Field `json:"variables,omitempty"`
}

type Folder struct {
	ID           string `json:"id"`
	CollectionID string `json:"collectionId"`
	ParentID     string `json:"parentId,omitempty"`
	Name         string `json:"name"`
}

// Field preserves ordering, duplicate names, and disabled entries.
type Field struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
}

type BodyField struct {
	Field
	File string `json:"file,omitempty"`
}

type Body struct {
	Type   string      `json:"type"`
	Text   string      `json:"text"`
	Fields []BodyField `json:"fields,omitempty"`
}

type Auth struct {
	Type        string `json:"type"`
	Username    string `json:"username,omitempty"`
	Password    string `json:"password,omitempty"`
	Token       string `json:"token,omitempty"`
	KeyName     string `json:"keyName,omitempty"`
	KeyValue    string `json:"keyValue,omitempty"`
	KeyLocation string `json:"keyLocation,omitempty"`
}

type TLS struct {
	InsecureSkipVerify bool   `json:"insecureSkipVerify"`
	CACertFile         string `json:"caCertFile,omitempty"`
	ClientCertFile     string `json:"clientCertFile,omitempty"`
	ClientKeyFile      string `json:"clientKeyFile,omitempty"`
}

type Request struct {
	TimeoutSeconds int     `json:"timeoutSeconds,omitempty"`
	ID             string  `json:"id"`
	CollectionID   string  `json:"collectionId"`
	FolderID       string  `json:"folderId,omitempty"`
	Name           string  `json:"name"`
	Method         string  `json:"method"`
	URL            string  `json:"url"`
	Headers        []Field `json:"headers"`
	Query          []Field `json:"query"`
	Body           Body    `json:"body"`
	Auth           Auth    `json:"auth"`
	TLS            TLS     `json:"tls"`
}

type Environment struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Variables []Field `json:"variables"`
}

// Load reads a store. A missing file is returned as an error matching os.ErrNotExist.
func Load(path string) (Data, error) {
	var data Data
	b, err := os.ReadFile(path)
	if err != nil {
		return data, fmt.Errorf("read store: %w", err)
	}
	if err := json.Unmarshal(b, &data); err != nil {
		return Data{}, fmt.Errorf("decode store: %w", err)
	}
	return data, nil
}

// Save atomically replaces path with private JSON. Its parent is created if needed.
// Errors are returned only before replacement; callers can safely roll back memory.
// ponytail: directory sync is best effort; strict crash durability needs a journal.
func Save(path string, data Data) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("encode store: %w", err)
	}
	b = append(b, '\n')
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create store directory: %w", err)
	}
	f, err := os.CreateTemp(dir, ".store-*")
	if err != nil {
		return fmt.Errorf("create store temp file: %w", err)
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(b); err != nil {
		f.Close()
		return fmt.Errorf("write store: %w", err)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return fmt.Errorf("sync store: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close store: %w", err)
	}
	if err := os.Rename(f.Name(), path); err != nil {
		return fmt.Errorf("replace store: %w", err)
	}
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}
