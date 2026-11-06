package postman

import (
	"strings"
	"testing"
)

func TestAPIKeyInheritance(t *testing.T) {
	for _, attributes := range []string{
		`[{"key":"key","value":"{{name}}"},{"key":"value","value":"{{secret}}"},{"key":"in","value":"header"}]`,
		`{"key":"{{name}}","value":"{{secret}}","in":"query"}`,
		`{"key":"","value":"secret","in":"header"}`,
		`{"key":"X","value":"secret","in":"cookie"}`,
		`{"key":"X","value":42,"in":"header"}`,
		`null`,
	} {
		input := `{"info":{"name":"test","schema":"https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},"auth":{"type":"apikey","apikey":` + attributes + `},"item":[{"name":"folder","item":[{"name":"one","request":{"url":"https://example.com"}}]}]}`
		c, err := ImportCollection(strings.NewReader(input))
		if err != nil {
			t.Fatal(err)
		}
		a := c.Requests[0].Auth
		valid := strings.Contains(attributes, "{{name}}")
		if a == nil || (a.Type == "apikey") != valid {
			t.Fatalf("wrong inherited auth: %#v", a)
		}
		if valid && len(a.Values) != 3 {
			t.Fatal("attributes lost")
		}
		if !valid && len(c.Warnings) == 0 {
			t.Fatal("missing malformed auth warning")
		}
		for _, warning := range c.Warnings {
			if strings.Contains(warning, "secret") {
				t.Fatal("secret in warning")
			}
		}
	}
}
