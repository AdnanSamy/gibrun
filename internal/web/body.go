package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gibrun/internal/store"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
)

func bodyFieldsJSON(fields []store.BodyField) string {
	if fields == nil {
		return "[]"
	}
	b, _ := json.MarshalIndent(fields, "", "  ")
	return string(b)
}

func requestBody(r *http.Request, b store.Body, h http.Header) ([]byte, error) {
	switch b.Type {
	case "", "raw":
		return []byte(b.Text), nil
	case "json":
		if h.Get("Content-Type") == "" {
			h.Set("Content-Type", "application/json")
		}
		return []byte(b.Text), nil
	case "urlencoded":
		values := url.Values{}
		for _, f := range b.Fields {
			if f.Enabled {
				if f.File != "" {
					return nil, fmt.Errorf("files require multipart body")
				}
				values.Add(f.Key, f.Value)
			}
		}
		h.Set("Content-Type", "application/x-www-form-urlencoded")
		return []byte(values.Encode()), nil
	case "formdata":
		var out bytes.Buffer
		writer := multipart.NewWriter(&out)
		for i, f := range b.Fields {
			if !f.Enabled {
				continue
			}
			if f.File == "" {
				if err := writer.WriteField(f.Key, f.Value); err != nil {
					return nil, err
				}
				continue
			}
			file, header, err := r.FormFile(fmt.Sprintf("upload_%d", i))
			if err != nil {
				return nil, fmt.Errorf("reselect file for %s (%s)", f.Key, f.File)
			}
			part, err := writer.CreateFormFile(f.Key, header.Filename)
			if err == nil {
				_, err = io.Copy(part, io.LimitReader(file, (8<<20)+1))
			}
			file.Close()
			if err != nil {
				return nil, err
			}
			if out.Len() > 8<<20 {
				return nil, fmt.Errorf("multipart body exceeds 8 MiB")
			}
		}
		if err := writer.Close(); err != nil {
			return nil, err
		}
		if out.Len() > 8<<20 {
			return nil, fmt.Errorf("multipart body exceeds 8 MiB")
		}
		h.Set("Content-Type", writer.FormDataContentType())
		return out.Bytes(), nil
	default:
		return nil, fmt.Errorf("unsupported body mode: %s", b.Type)
	}
}

func importedBody(raw json.RawMessage) (store.Body, []string) {
	var body struct {
		Mode, Raw            string
		Options              struct{ Raw struct{ Language string } }
		URLEncoded, FormData []struct {
			Key, Value, Type string
			Disabled         bool
			Src              json.RawMessage
		}
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return store.Body{}, nil
	}
	result := store.Body{Type: body.Mode, Text: body.Raw}
	var warnings []string
	if body.Mode == "raw" && body.Options.Raw.Language == "json" {
		result.Type = "json"
	}
	fields := body.URLEncoded
	if body.Mode == "formdata" {
		fields = body.FormData
	}
	for _, f := range fields {
		field := store.BodyField{Field: store.Field{Key: f.Key, Value: f.Value, Enabled: !f.Disabled}}
		if f.Type == "file" {
			var refs []string
			var ref string
			if json.Unmarshal(f.Src, &ref) == nil {
				refs = []string{ref}
			} else {
				_ = json.Unmarshal(f.Src, &refs)
			}
			if len(refs) == 0 {
				refs = []string{"Select file"}
			}
			for _, ref := range refs {
				if ref == "" {
					ref = "Select file"
				}
				field.File = ref
				result.Fields = append(result.Fields, field)
			}
			warnings = append(warnings, "File references retained; reselect files before sending. Server paths are never read.")
		} else {
			result.Fields = append(result.Fields, field)
		}
	}
	switch result.Type {
	case "", "raw", "json", "urlencoded", "formdata":
	default:
		warnings = append(warnings, "unsupported body mode "+result.Type)
	}
	return result, warnings
}
