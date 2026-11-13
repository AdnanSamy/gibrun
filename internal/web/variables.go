package web

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"gibrun/internal/store"
)

// Variable JSON preserves multiline values, ordering, duplicates, and disabled entries.
func editVariables(data *store.Data, form url.Values) error {
	kind, action, target := form.Get("kind"), form.Get("action"), form.Get("target")
	index := -1
	switch kind {
	case "environment":
		for i, e := range data.Environments {
			if e.ID == target {
				index = i
				break
			}
		}
	case "collection":
		for i, c := range data.Collections {
			if c.ID == target {
				index = i
				break
			}
		}
	default:
		return fmt.Errorf("invalid variable scope")
	}
	if action == "create" {
		if kind != "environment" || target != "" {
			return fmt.Errorf("invalid create target")
		}
	} else if target == "" || index < 0 {
		return fmt.Errorf("variable scope not found")
	}
	if action == "delete" {
		if kind != "environment" || form.Get("confirm") != "yes" {
			return fmt.Errorf("confirm environment deletion")
		}
		data.Environments = append(data.Environments[:index], data.Environments[index+1:]...)
		return nil
	}
	if action != "create" && action != "save" {
		return fmt.Errorf("invalid variable action")
	}
	name := strings.TrimSpace(form.Get("name"))
	if kind == "environment" && name == "" {
		return fmt.Errorf("environment name is required")
	}
	var variables []store.Field
	decoder := json.NewDecoder(strings.NewReader(form.Get("variables")))
	decoder.DisallowUnknownFields()
	// Validate the complete document before decoding; errors never echo secret input.
	if !strings.HasPrefix(strings.TrimSpace(form.Get("variables")), "[") || !json.Valid([]byte(form.Get("variables"))) || decoder.Decode(&variables) != nil {
		return fmt.Errorf("variables must be a JSON array of key, value, enabled records")
	}
	for _, v := range variables {
		if strings.TrimSpace(v.Key) == "" {
			return fmt.Errorf("variable keys must not be blank")
		}
	}
	if kind == "collection" {
		data.Collections[index].Variables = variables
	} else if action == "create" {
		data.Environments = append(data.Environments, store.Environment{ID: id(), Name: name, Variables: variables})
	} else {
		data.Environments[index].Name = name
		data.Environments[index].Variables = variables
	}
	return nil
}

func variableJSON(fields []store.Field) string {
	if fields == nil {
		return "[]"
	}
	b, _ := json.MarshalIndent(fields, "", "  ")
	return string(b)
}

func variablesFor(data store.Data, collection, environment string) map[string]string {
	vars := map[string]string{}
	add := func(fields []store.Field) {
		for _, v := range fields {
			if v.Enabled {
				vars[v.Key] = v.Value
			}
		}
	}
	for _, c := range data.Collections {
		if c.ID == collection {
			add(c.Variables)
		}
	}
	for _, e := range data.Environments {
		if e.ID == environment {
			add(e.Variables)
		}
	}
	return vars
}
