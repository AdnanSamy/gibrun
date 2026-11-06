package web

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"solarclient/internal/client"
)

func TestResponseFormatting(t *testing.T) {
	for _, raw := range []string{`{"n":900719925474099312345,"d":1.234567890123456789,"x":"</pre><script>alert(1)</script>"}`, `{"bad":`, "plain <b>text</b>", "", "{\n  \"n\": 1\n}"} {
		response := &client.Response{Body: []byte(raw), Status: "200 OK", Duration: 25 * time.Millisecond, BodySize: int64(len(raw))}
		body := formatResponse(response)
		if body.Raw != raw || !bytes.Equal(response.Body, []byte(raw)) {
			t.Fatal("raw body changed")
		}
		if strings.Contains(raw, "900719925474099312345") {
			if !strings.Contains(body.Formatted, "900719925474099312345") || !strings.Contains(body.Formatted, "1.234567890123456789") {
				t.Fatal("precision lost")
			}
		} else if body.Formatted != "" {
			t.Fatalf("unnecessary formatted copy: %q", body.Formatted)
		}
		s := &Server{}
		w := httptest.NewRecorder()
		s.render(w, "", "", response)
		html := w.Body.String()
		if strings.Contains(html, "<script>alert") || strings.Contains(html, "<b>text</b>") {
			t.Fatal("unescaped response")
		}
		if !strings.Contains(html, "25ms ·") || !strings.Contains(html, "bytes</span>") {
			t.Fatal("missing metrics")
		}
		if body.Formatted != "" && (!strings.Contains(html, `for="response-raw"`) || !strings.Contains(html, `name="response-view"`)) {
			t.Fatal("missing labelled native toggle")
		}
	}
}
