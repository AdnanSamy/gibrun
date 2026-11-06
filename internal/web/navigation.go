package web

import (
	"net/url"

	"solarclient/internal/store"
)

// Only literal URL host/path are searchable; never expose credentials or query values.
func searchURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return u.Host + u.EscapedPath()
}

// Navigation remains flat: indices connect records, never heap-allocated tree nodes.
type navRow struct {
	ID, Name, Method, CollectionID, URL string
	Depth                               int
	Folder                              bool
}

func navigation(data store.Data) []navRow {
	type parent struct{ collection, folder string }
	folders := make(map[parent][]int)
	requests := make(map[parent][]int)
	for i, f := range data.Folders {
		k := parent{f.CollectionID, f.ParentID}
		folders[k] = append(folders[k], i)
	}
	for i, r := range data.Requests {
		k := parent{r.CollectionID, r.FolderID}
		requests[k] = append(requests[k], i)
	}
	rows := make([]navRow, 0, len(data.Folders)+len(data.Requests))
	type entry struct {
		index, depth int
		folder       bool
	}
	visited := make([]bool, len(data.Folders))
	for _, c := range data.Collections {
		var stack []entry
		push := func(key parent, depth int) {
			rs := requests[key]
			for i := len(rs) - 1; i >= 0; i-- {
				stack = append(stack, entry{rs[i], depth, false})
			}
			fs := folders[key]
			for i := len(fs) - 1; i >= 0; i-- {
				stack = append(stack, entry{fs[i], depth, true})
			}
		}
		push(parent{c.ID, ""}, 0)
		for len(stack) > 0 {
			e := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if e.folder {
				if visited[e.index] {
					continue
				}
				visited[e.index] = true
				f := data.Folders[e.index]
				rows = append(rows, navRow{ID: f.ID, Name: f.Name, CollectionID: c.ID, Depth: e.depth, Folder: true})
				push(parent{c.ID, f.ID}, e.depth+1)
			} else {
				r := data.Requests[e.index]
				rows = append(rows, navRow{ID: r.ID, Name: r.Name, Method: r.Method, URL: r.URL, CollectionID: c.ID, Depth: e.depth})
			}
		}
	}
	return rows
}
