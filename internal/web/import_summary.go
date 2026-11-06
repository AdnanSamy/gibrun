package web

import (
	"fmt"
	"sort"
	"strings"

	"solarclient/internal/store"
)

func importSummary(kind, name string, counts string, warnings []string) string {
	message := fmt.Sprintf("Imported %s %q. %s", kind, name, counts)
	seen := make(map[string]bool)
	for _, warning := range warnings {
		if !seen[warning] {
			message += "\n• " + warning
			seen[warning] = true
		}
	}
	return message
}

// Report names only: never expose variable values or request credentials in summaries.
func missingCollectionVariables(requests []store.Request, variables []store.Field) []string {
	known := make(map[string]bool)
	for _, v := range variables {
		if v.Enabled {
			known[v.Key] = true
		}
	}
	missing := make(map[string]bool)
	scan := func(text string) {
		for {
			start := strings.Index(text, "{{")
			if start < 0 {
				return
			}
			text = text[start+2:]
			end := strings.Index(text, "}}")
			if end < 0 {
				return
			}
			key := strings.TrimSpace(text[:end])
			if !known[key] {
				missing[key] = true
			}
			text = text[end+2:]
		}
	}
	for _, r := range requests {
		scan(r.URL)
		scan(r.Body.Text)
		for _, field := range r.Body.Fields {
			if field.Enabled {
				scan(field.Key)
				scan(field.Value)
			}
		}
		scan(r.Auth.Username)
		scan(r.Auth.Password)
		scan(r.Auth.Token)
		scan(r.Auth.KeyName)
		scan(r.Auth.KeyValue)
		for _, fields := range [][]store.Field{r.Headers, r.Query} {
			for _, f := range fields {
				if f.Enabled {
					scan(f.Key)
					scan(f.Value)
				}
			}
		}
	}
	names := make([]string, 0, len(missing))
	for name := range missing {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
