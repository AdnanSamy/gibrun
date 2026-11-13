package web

import (
	"bytes"
	"encoding/json"

	"gibrun/internal/client"
)

type responseBody struct {
	Raw, Formatted string
}

func formatResponse(response *client.Response) responseBody {
	if response == nil {
		return responseBody{}
	}
	body := responseBody{Raw: string(response.Body)}
	var formatted bytes.Buffer
	// Indent preserves number literals and leaves the original bytes untouched.
	if json.Indent(&formatted, response.Body, "", "  ") == nil && !bytes.Equal(formatted.Bytes(), response.Body) {
		body.Formatted = formatted.String()
	}
	return body
}
