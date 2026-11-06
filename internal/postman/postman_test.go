package postman

import (
	"strings"
	"testing"
)

func TestCollection(t *testing.T) {
	for _, version := range []string{"2.0.0", "2.1.0"} {
		input := `{"info":{"name":"test","schema":"https://schema.getpostman.com/json/collection/v` + version + `/collection.json"},"variable":[{"key":"host","value":"example.com"}],"auth":{"type":"bearer","bearer":[{"key":"token","value":"{{token}}"}]},"event":[{"listen":"prerequest"}],"item":[{"name":"folder","item":[{"name":"one","request":{"method":"GET","url":{"raw":"https://{{host}}?x=1","query":[{"key":"x","value":"1"}]}}},{"name":"two","request":{"method":"POST","url":"https://example.com","auth":{"type":"noauth"}}},{"name":"three","request":{"url":"x","auth":{"type":"oauth2"}}}]}]}`
		c, err := ImportCollection(strings.NewReader(input))
		if err != nil {
			t.Fatal(err)
		}
		if len(c.Folders) != 1 || len(c.Requests) != 3 || c.Requests[0].ParentID != c.Folders[0].ID || c.Requests[0].Auth == nil || c.Requests[0].Auth.Type != "bearer" || c.Requests[1].Auth != nil || c.Requests[2].Auth == nil || c.Requests[2].Auth.Type != "oauth2" || len(c.Warnings) != 2 || len(c.Requests[0].Query) != 1 || len(c.Variables) != 1 {
			t.Fatalf("unexpected collection: %#v", c)
		}
	}
}
func TestEnvironment(t *testing.T) {
	e, err := ImportEnvironment(strings.NewReader(`{"name":"dev","values":[{"key":"a","value":"b","enabled":false},{"key":"c","value":3}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(e.Variables) != 2 || !e.Variables[0].Disabled || e.Variables[1].Disabled {
		t.Fatalf("unexpected environment: %#v", e)
	}
}
func TestInvalid(t *testing.T) {
	for _, input := range []string{`{`, `{"info":{"schema":"unknown"}}`, `{"info":{"schema":"https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},"item":[{"name":"bad"}]}`} {
		if _, err := ImportCollection(strings.NewReader(input)); err == nil {
			t.Fatalf("accepted %s", input)
		}
	}
}
