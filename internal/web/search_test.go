package web

import "testing"

func TestSearchURL(t *testing.T) {
	for raw, want := range map[string]string{
		"https://user:password@example.test/users?token=secret#private": "example.test/users",
		"https://example.test/a%20b":                                    "example.test/a%20b",
		"https://{{host}}/users":                                        "",
		"%invalid":                                                      "",
	} {
		if got := searchURL(raw); got != want {
			t.Errorf("searchURL(%q) = %q; want %q", raw, got, want)
		}
	}
}
