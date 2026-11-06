package web

import (
	"reflect"
	"solarclient/internal/store"
	"strings"
	"testing"
)

func TestImportSummary(t *testing.T) {
	requests := []store.Request{{URL: "{{host}}/{{path}}", Auth: store.Auth{Token: "{{token}}"}, Headers: []store.Field{{Key: "X", Value: "{{ignored}}", Enabled: false}}, Query: []store.Field{{Key: "q", Value: "{{path}}", Enabled: true}}}}
	missing := missingCollectionVariables(requests, []store.Field{{Key: "host", Value: "secret", Enabled: true}})
	if !reflect.DeepEqual(missing, []string{"path", "token"}) {
		t.Fatalf("missing: %v", missing)
	}
	summary := importSummary("collection", "Example", "2 requests.", []string{"Scripts unsupported", "Scripts unsupported"})
	if !strings.Contains(summary, "2 requests.") || strings.Count(summary, "Scripts unsupported") != 1 {
		t.Fatal(summary)
	}
}
