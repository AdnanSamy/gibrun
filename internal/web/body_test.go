package web

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gibrun/internal/store"
)

func TestFormBodies(t *testing.T) {
	for _, mode := range []string{"urlencoded", "formdata"} {
		t.Run(mode, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if mode == "formdata" {
					if err := r.ParseMultipartForm(1024); err != nil {
						t.Error(err)
					}
					defer r.MultipartForm.RemoveAll()
				} else {
					_ = r.ParseForm()
				}
				if got := r.Form["a &"]; len(got) != 2 || got[0] != "x=+" || got[1] != "two" {
					t.Errorf("fields: %v", r.Form)
				}
				if r.Form.Has("disabled") {
					t.Error("disabled sent")
				}
				if mode == "formdata" {
					f, h, err := r.FormFile("attachment")
					if err != nil {
						t.Error(err)
						return
					}
					defer f.Close()
					b, _ := io.ReadAll(f)
					if string(b) != "contents" || h.Filename != "safe.txt" {
						t.Error("file mismatch")
					}
				}
			}))
			defer server.Close()
			var upload bytes.Buffer
			writer := multipart.NewWriter(&upload)
			part, _ := writer.CreateFormFile("upload_3", "safe.txt")
			_, _ = io.WriteString(part, "contents")
			_ = writer.Close()
			r := httptest.NewRequest("POST", "/", &upload)
			r.Header.Set("Content-Type", writer.FormDataContentType())
			_ = r.ParseMultipartForm(1024)
			defer r.MultipartForm.RemoveAll()
			fields := []store.BodyField{{Field: store.Field{Key: "a &", Value: "x=+", Enabled: true}}, {Field: store.Field{Key: "a &", Value: "two", Enabled: true}}, {Field: store.Field{Key: "disabled", Value: "no"}}}
			if mode == "formdata" {
				fields = append(fields, store.BodyField{Field: store.Field{Key: "attachment", Enabled: true}, File: "/etc/passwd"})
			}
			_, err := send(r, store.Request{Method: "POST", URL: server.URL, Body: store.Body{Type: mode, Fields: fields}, Headers: []store.Field{{Key: "Content-Type", Value: "wrong", Enabled: true}}})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestImportedBodyFiles(t *testing.T) {
	body, warnings := importedBody(json.RawMessage(`{"mode":"formdata","formdata":[{"key":"a","value":"1","disabled":true},{"key":"f","type":"file","src":["/etc/passwd","other"]}]}`))
	if len(body.Fields) != 3 || body.Fields[0].Enabled || body.Fields[1].File != "/etc/passwd" || len(warnings) == 0 {
		t.Fatalf("%+v %v", body, warnings)
	}
	_, err := requestBody(httptest.NewRequest("POST", "/", nil), body, http.Header{})
	if err == nil || !strings.Contains(err.Error(), "reselect") {
		t.Fatalf("expected reselection: %v", err)
	}
	encoded, _ := importedBody(json.RawMessage(`{"mode":"urlencoded","urlencoded":[{"key":"x","value":"a+b"},{"key":"x","value":"2"}]}`))
	b, err := requestBody(nil, encoded, http.Header{})
	if err != nil || string(b) != "x=a%2Bb&x=2" {
		t.Fatalf("%s %v", b, err)
	}
}
