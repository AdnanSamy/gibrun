// Package postman imports Postman collections and environments.
package postman

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Variable is a Postman variable. Disabled variables are retained but not applied.
type Variable struct {
	Key      string `json:"key"`
	Value    any    `json:"value"`
	Type     string `json:"type,omitempty"`
	Disabled bool   `json:"disabled,omitempty"`
}

// Auth describes supported authentication. Values retain Postman's attribute format.
type Auth struct {
	Type   string     `json:"type"`
	Values []Variable `json:"values,omitempty"`
}

// Folder and Request use IDs and ParentID to represent the collection tree as flat lists.
type Folder struct {
	ID, ParentID, Name string
}
type Request struct {
	ID, ParentID, Name, Method, URL, Description string
	Headers, Query                               []Variable
	Body                                         json.RawMessage
	Auth                                         *Auth
}
type Collection struct {
	ID, Name, Description string
	Variables             []Variable
	Folders               []Folder
	Requests              []Request
	Warnings              []string
}
type Environment struct {
	ID, Name  string
	Variables []Variable
	Warnings  []string
}

type rawAuth struct {
	Type   string                     `json:"type"`
	Fields map[string]json.RawMessage `json:"-"`
}

func (a *rawAuth) UnmarshalJSON(b []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(b, &fields); err != nil {
		return err
	}
	if v := fields["type"]; v != nil {
		if err := json.Unmarshal(v, &a.Type); err != nil {
			return err
		}
	}
	a.Fields = fields
	return nil
}

type rawItem struct {
	ID      string          `json:"id"`
	Name    string          `json:"name"`
	Item    []rawItem       `json:"item"`
	Request json.RawMessage `json:"request"`
	Auth    *rawAuth        `json:"auth"`
	Event   json.RawMessage `json:"event"`
}
type rawCollection struct {
	Info struct {
		ID          string          `json:"_postman_id"`
		Name        string          `json:"name"`
		Schema      string          `json:"schema"`
		Description json.RawMessage `json:"description"`
	} `json:"info"`
	Item     []rawItem       `json:"item"`
	Variable []Variable      `json:"variable"`
	Auth     *rawAuth        `json:"auth"`
	Event    json.RawMessage `json:"event"`
}

func description(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var obj struct {
		Content string `json:"content"`
	}
	_ = json.Unmarshal(raw, &obj)
	return obj.Content
}
func hasEvents(raw json.RawMessage) bool {
	var events []json.RawMessage
	return json.Unmarshal(raw, &events) == nil && len(events) > 0
}
func auth(a *rawAuth, inherited *Auth, warnings *[]string, where string) *Auth {
	if a == nil {
		return inherited
	}
	if a.Type == "noauth" {
		return nil
	}
	switch a.Type {
	case "basic", "bearer", "apikey":
		result := &Auth{Type: a.Type}
		if v := a.Fields[a.Type]; len(v) > 0 {
			if err := json.Unmarshal(v, &result.Values); err != nil {
				result.Values = nil
				// Postman v2.0 also allows an object of auth attributes.
				var fields map[string]any
				if json.Unmarshal(v, &fields) == nil {
					for k, value := range fields {
						result.Values = append(result.Values, Variable{Key: k, Value: value})
					}
				} else {
					result.Values = nil
					*warnings = append(*warnings, where+": invalid auth attributes")
				}
			}
		}
		if a.Type == "apikey" {
			fields := map[string]string{}
			valid := true
			for _, field := range result.Values {
				value, ok := field.Value.(string)
				if _, duplicate := fields[field.Key]; duplicate || !ok {
					valid = false
				}
				fields[field.Key] = value
			}
			_, hasValue := fields["value"]
			if !valid || !hasValue || strings.TrimSpace(fields["key"]) == "" || (fields["in"] != "header" && fields["in"] != "query") {
				result.Type = "invalid-apikey"
				*warnings = append(*warnings, where+": invalid API-key authentication; execution blocked")
			}
		}
		return result
	default:
		*warnings = append(*warnings, where+": unsupported auth type "+a.Type)
		return &Auth{Type: a.Type}
	}
}

// ImportCollection reads a Postman collection v2.0 or v2.1 JSON document.
// Missing IDs are generated deterministically from item positions.
func ImportCollection(r io.Reader) (*Collection, error) {
	var raw rawCollection
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode collection: %w", err)
	}
	if !strings.Contains(raw.Info.Schema, "/v2.0.0/") && !strings.Contains(raw.Info.Schema, "/v2.1.0/") {
		return nil, fmt.Errorf("unsupported collection schema %q", raw.Info.Schema)
	}
	c := &Collection{ID: raw.Info.ID, Name: raw.Info.Name, Description: description(raw.Info.Description), Variables: raw.Variable}
	if hasEvents(raw.Event) {
		c.Warnings = append(c.Warnings, "collection: scripts are not supported")
	}
	rootAuth := auth(raw.Auth, nil, &c.Warnings, "collection")
	var walk func([]rawItem, string, string, *Auth) error
	walk = func(items []rawItem, parent, path string, inherited *Auth) error {
		for i, item := range items {
			pos := fmt.Sprintf("%s%d", path, i+1)
			id := item.ID
			if id == "" {
				id = "postman-" + pos
			}
			if hasEvents(item.Event) {
				c.Warnings = append(c.Warnings, item.Name+": scripts are not supported")
			}
			itemAuth := auth(item.Auth, inherited, &c.Warnings, item.Name)
			if item.Item != nil {
				c.Folders = append(c.Folders, Folder{ID: id, ParentID: parent, Name: item.Name})
				if err := walk(item.Item, id, pos+"-", itemAuth); err != nil {
					return err
				}
				continue
			}
			var req struct {
				Method      string          `json:"method"`
				URL         json.RawMessage `json:"url"`
				Description json.RawMessage `json:"description"`
				Header      []Variable      `json:"header"`
				Body        json.RawMessage `json:"body"`
				Auth        *rawAuth        `json:"auth"`
			}
			if len(item.Request) == 0 {
				return fmt.Errorf("item %q has no request or folder", item.Name)
			}
			var url string
			var query []Variable
			if json.Unmarshal(item.Request, &url) == nil {
				req.Method = "GET"
			} else {
				if err := json.Unmarshal(item.Request, &req); err != nil {
					return fmt.Errorf("request %q: %w", item.Name, err)
				}
				if json.Unmarshal(req.URL, &url) != nil {
					var u struct {
						Raw   string     `json:"raw"`
						Query []Variable `json:"query"`
					}
					if err := json.Unmarshal(req.URL, &u); err != nil {
						return fmt.Errorf("request %q URL: %w", item.Name, err)
					}
					url, query = u.Raw, u.Query
				}
			}
			c.Requests = append(c.Requests, Request{ID: id, ParentID: parent, Name: item.Name, Method: req.Method, URL: url, Description: description(req.Description), Headers: req.Header, Query: query, Body: req.Body, Auth: auth(req.Auth, itemAuth, &c.Warnings, item.Name)})
		}
		return nil
	}
	if err := walk(raw.Item, "", "", rootAuth); err != nil {
		return nil, err
	}
	return c, nil
}

// ImportEnvironment reads a Postman environment JSON document.
func ImportEnvironment(r io.Reader) (*Environment, error) {
	var raw struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Values []struct {
			Key      string `json:"key"`
			Value    any    `json:"value"`
			Type     string `json:"type"`
			Enabled  *bool  `json:"enabled"`
			Disabled bool   `json:"disabled"`
		} `json:"values"`
	}
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode environment: %w", err)
	}
	if raw.Name == "" && raw.Values == nil {
		return nil, fmt.Errorf("invalid environment: missing name and values")
	}
	e := &Environment{ID: raw.ID, Name: raw.Name}
	for _, v := range raw.Values {
		e.Variables = append(e.Variables, Variable{Key: v.Key, Value: v.Value, Type: v.Type, Disabled: v.Disabled || (v.Enabled != nil && !*v.Enabled)})
	}
	return e, nil
}
