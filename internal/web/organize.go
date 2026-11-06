package web

import (
	"fmt"
	"net/url"
	"slices"
	"strings"

	"solarclient/internal/store"
)

func descendants(d store.Data, kind, id string) map[string]bool {
	ids := map[string]bool{}
	if kind == "folder" {
		ids[id] = true
	}
	for changed := true; changed; {
		changed = false
		for _, f := range d.Folders {
			if !ids[f.ID] && ((kind == "collection" && f.CollectionID == id) || ids[f.ParentID]) {
				ids[f.ID] = true
				changed = true
			}
		}
	}
	return ids
}

func deletionCounts(d store.Data, kind, id string) string {
	fs := descendants(d, kind, id)
	n := 0
	for _, r := range d.Requests {
		if (kind == "request" && r.ID == id) || (kind == "collection" && r.CollectionID == id) || fs[r.FolderID] {
			n++
		}
	}
	folders := len(fs)
	if kind == "folder" {
		folders--
	}
	return fmt.Sprintf("%d descendant folders, %d requests", folders, n)
}

func destination(d store.Data, collection, folder string, unfiled bool) error {
	if collection == "" && folder == "" && unfiled {
		return nil
	}
	if !slices.ContainsFunc(d.Collections, func(c store.Collection) bool { return c.ID == collection }) {
		return fmt.Errorf("invalid collection")
	}
	if folder != "" && !slices.ContainsFunc(d.Folders, func(f store.Folder) bool { return f.ID == folder && f.CollectionID == collection }) {
		return fmt.Errorf("invalid parent folder")
	}
	return nil
}

func organize(d *store.Data, v url.Values) error {
	kind, action, key := v.Get("kind"), v.Get("action"), v.Get("id")
	name := strings.TrimSpace(v.Get("name"))
	ci := slices.IndexFunc(d.Collections, func(c store.Collection) bool { return c.ID == key })
	fi := slices.IndexFunc(d.Folders, func(f store.Folder) bool { return f.ID == key })
	ri := slices.IndexFunc(d.Requests, func(r store.Request) bool { return r.ID == key })
	if kind != "collection" && kind != "folder" && kind != "request" {
		return fmt.Errorf("invalid kind")
	}
	if action != "create" && ((kind == "collection" && ci < 0) || (kind == "folder" && fi < 0) || (kind == "request" && ri < 0)) {
		return fmt.Errorf("item no longer exists")
	}
	switch action {
	case "create", "rename":
		if name == "" {
			return fmt.Errorf("name is required")
		}
		if action == "rename" {
			switch kind {
			case "collection":
				d.Collections[ci].Name = name
			case "folder":
				d.Folders[fi].Name = name
			case "request":
				d.Requests[ri].Name = name
			}
			return nil
		}
		if kind == "collection" {
			d.Collections = append(d.Collections, store.Collection{ID: id(), Name: name})
			return nil
		}
		if kind != "folder" {
			return fmt.Errorf("use the request editor to create requests")
		}
		c, p := v.Get("collection"), v.Get("parent")
		if err := destination(*d, c, p, false); err != nil {
			return err
		}
		d.Folders = append(d.Folders, store.Folder{ID: id(), Name: name, CollectionID: c, ParentID: p})
	case "duplicate":
		if kind != "request" {
			return fmt.Errorf("only saved requests can be duplicated")
		}
		copy := d.Requests[ri]
		copy.ID = id()
		copy.Name += " (copy)"
		copy.Headers = slices.Clone(copy.Headers)
		copy.Query = slices.Clone(copy.Query)
		copy.Body.Fields = slices.Clone(copy.Body.Fields)
		d.Requests = append(d.Requests, copy)
	case "move":
		if kind == "collection" {
			return fmt.Errorf("collections are already top-level")
		}
		c, p := v.Get("collection"), v.Get("parent")
		if err := destination(*d, c, p, kind == "request"); err != nil {
			return err
		}
		if kind == "request" {
			d.Requests[ri].CollectionID = c
			d.Requests[ri].FolderID = p
			return nil
		}
		fs := descendants(*d, kind, key)
		if fs[p] {
			return fmt.Errorf("cannot move a folder into itself or its descendants")
		}
		d.Folders[fi].ParentID = p
		for i := range d.Folders {
			if fs[d.Folders[i].ID] {
				d.Folders[i].CollectionID = c
			}
		}
		for i := range d.Requests {
			if fs[d.Requests[i].FolderID] {
				d.Requests[i].CollectionID = c
			}
		}
	case "delete":
		if v.Get("confirm") != "yes" || v.Get("counts") != deletionCounts(*d, kind, key) {
			return fmt.Errorf("confirm deletion with current descendant counts")
		}
		fs := descendants(*d, kind, key)
		d.Requests = slices.DeleteFunc(d.Requests, func(r store.Request) bool {
			return (kind == "request" && r.ID == key) || (kind == "collection" && r.CollectionID == key) || fs[r.FolderID]
		})
		d.Folders = slices.DeleteFunc(d.Folders, func(f store.Folder) bool { return fs[f.ID] })
		if kind == "collection" {
			d.Collections = slices.DeleteFunc(d.Collections, func(c store.Collection) bool { return c.ID == key })
		}
	default:
		return fmt.Errorf("invalid action")
	}
	return nil
}
