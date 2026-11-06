// Package web serves the local SolarClient interface.
package web

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"solarclient/internal/client"
	"solarclient/internal/postman"
	"solarclient/internal/store"
)

//go:embed static/* page.html
var assets embed.FS

var page = template.Must(template.New("page.html").Funcs(template.FuncMap{"deletionCounts": deletionCounts, "variableJSON": variableJSON, "bodyFieldsJSON": bodyFieldsJSON}).ParseFS(assets, "page.html"))

type Server struct {
	mu                sync.Mutex
	data              store.Data
	path, host, token string
}

func id() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}
func New(path, host string) (*Server, error) {
	d, err := store.Load(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return &Server{data: d, path: path, host: host, token: id()}, nil
}
func (s *Server) URL() string { return "http://" + s.host + "/?token=" + s.token }
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Host != s.host {
		http.Error(w, "invalid host", http.StatusForbidden)
		return
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self'; base-uri 'none'; frame-ancestors 'none'")
	if (r.URL.Path == "/static/htmx.min.js" || r.URL.Path == "/static/app.js" || r.URL.Path == "/static/request.js" || r.URL.Path == "/static/tabs.js") && r.Method == "GET" {
		b, _ := assets.ReadFile(strings.TrimPrefix(r.URL.Path, "/"))
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Write(b)
		return
	}
	if r.Method == "GET" && r.URL.Query().Get("token") == s.token {
		http.SetCookie(w, &http.Cookie{Name: "solar_session", Value: s.token, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode})
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	cookie, err := r.Cookie("solar_session")
	if err != nil || cookie.Value != s.token {
		http.Error(w, "unauthorized", http.StatusForbidden)
		return
	}
	if r.Method == "POST" {
		if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" {
			http.Error(w, "invalid fetch site", 403)
			return
		}
		if r.Header.Get("Origin") != "http://"+s.host {
			http.Error(w, "invalid origin", http.StatusForbidden)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 8<<20)
		defer func() {
			if r.MultipartForm != nil {
				_ = r.MultipartForm.RemoveAll()
			}
		}()
		if err := r.ParseMultipartForm(1 << 20); err != nil && !errors.Is(err, http.ErrNotMultipart) {
			http.Error(w, err.Error(), 400)
			return
		}
		if r.FormValue("token") != s.token {
			http.Error(w, "invalid token", 403)
			return
		}
		s.action(w, r)
		return
	}
	if r.Method != "GET" || r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if collection := r.URL.Query().Get("collection"); collection != "" && r.URL.Query().Get("id") == "" {
		s.mu.Lock()
		valid := false
		for _, c := range s.data.Collections {
			if c.ID == collection {
				valid = true
				break
			}
		}
		s.mu.Unlock()
		if !valid {
			http.Error(w, "collection not found", http.StatusNotFound)
			return
		}
		s.renderFor(w, r, "", "", nil, store.Request{Method: "GET", CollectionID: collection})
		return
	}
	s.renderFor(w, r, r.URL.Query().Get("id"), "", nil)
}
func (s *Server) renderFor(w http.ResponseWriter, r *http.Request, selected, message string, response *client.Response, draft ...store.Request) {
	environment := ""
	if cookie, err := r.Cookie("solar_environment"); err == nil {
		environment = cookie.Value
	}
	if r.Method == "POST" && r.Form.Has("environment") {
		environment = r.FormValue("environment")
	}
	s.renderSelected(w, environment, selected, message, response, draft...)
}
func (s *Server) render(w http.ResponseWriter, selected, message string, response *client.Response, draft ...store.Request) {
	s.renderSelected(w, "", selected, message, response, draft...)
}
func (s *Server) renderSelected(w http.ResponseWriter, environment, selected, message string, response *client.Response, draft ...store.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	validEnvironment := false
	for _, e := range s.data.Environments {
		if e.ID == environment {
			validEnvironment = true
		}
	}
	if !validEnvironment {
		environment = ""
	}
	http.SetCookie(w, &http.Cookie{Name: "solar_environment", Value: environment, Path: "/", SameSite: http.SameSiteStrictMode})
	var req store.Request
	req.Method = "GET"
	for _, v := range s.data.Requests {
		if v.ID == selected {
			req = v
			break
		}
	}
	if len(draft) > 0 {
		req = draft[0]
	}
	saveState := "Not saved"
	for _, saved := range s.data.Requests {
		if saved.ID == req.ID {
			a, _ := json.Marshal(saved)
			b, _ := json.Marshal(req)
			if string(a) == string(b) {
				saveState = "Saved"
			} else {
				saveState = "Unsaved changes"
			}
			break
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = page.Execute(w, struct {
		Data                      store.Data
		Request                   store.Request
		Token, Message, SaveState string
		Environment               string
		Response                  *client.Response
		ResponseBody              responseBody
		Navigation                []navRow
	}{s.data, req, s.token, message, saveState, environment, response, formatResponse(response), navigation(s.data)})
}
func fields(text string) []store.Field {
	var out []store.Field
	for _, line := range strings.Split(text, "\n") {
		k, v, ok := strings.Cut(strings.TrimSpace(line), ":")
		if ok && strings.TrimSpace(k) != "" {
			out = append(out, store.Field{Key: strings.TrimSpace(k), Value: strings.TrimSpace(v), Enabled: true})
		}
	}
	return out
}

// editorFields keeps disabled entries explicit rather than dropping them on save.
func editorFields(r *http.Request, key string) []store.Field {
	out := fields(r.FormValue(key))
	for _, field := range fields(r.FormValue(key + "_disabled")) {
		field.Enabled = false
		out = append(out, field)
	}
	return out
}

func (s *Server) action(w http.ResponseWriter, r *http.Request) {
	op := r.URL.Path
	selected := r.FormValue("id")
	message := ""
	var response *client.Response
	var draft []store.Request
	s.mu.Lock()
	snapshot, _ := json.Marshal(s.data)
	switch op {
	case "/request":
		if environment := r.FormValue("environment"); environment != "" {
			valid := false
			for _, e := range s.data.Environments {
				if e.ID == environment {
					valid = true
				}
			}
			if !valid {
				s.mu.Unlock()
				s.renderFor(w, r, selected, "Environment no longer exists. Select an environment again.", nil)
				return
			}
		}
		q := store.Request{ID: selected, Name: r.FormValue("name"), CollectionID: r.FormValue("collection"), Method: strings.ToUpper(strings.TrimSpace(r.FormValue("method"))), URL: strings.TrimSpace(r.FormValue("url")), Headers: editorFields(r, "headers"), Query: editorFields(r, "query"), Body: store.Body{Type: r.FormValue("body_type"), Text: r.FormValue("body")}, Auth: store.Auth{Type: r.FormValue("auth_type"), Username: r.FormValue("username"), Password: r.FormValue("password"), Token: r.FormValue("auth_token"), KeyName: r.FormValue("auth_key_name"), KeyValue: r.FormValue("auth_key_value"), KeyLocation: r.FormValue("auth_key_location")}, TLS: store.TLS{InsecureSkipVerify: r.FormValue("insecure") == "on"}}
		timeout := r.FormValue("timeout_seconds")
		q.TimeoutSeconds = 30
		if timeout != "" {
			seconds, err := strconv.Atoi(timeout)
			if err != nil || seconds < 1 || seconds > 300 {
				s.mu.Unlock()
				http.Error(w, "Timeout must be a whole number from 1 to 300 seconds.", http.StatusUnprocessableEntity)
				return
			}
			q.TimeoutSeconds = seconds
		}
		if q.Body.Type == "" {
			q.Body.Type = "raw"
		}
		if text := r.FormValue("body_fields"); text != "" {
			if err := json.Unmarshal([]byte(text), &q.Body.Fields); err != nil {
				s.mu.Unlock()
				// Keep the submitted editor intact on validation failure.
				http.Error(w, "Invalid body fields JSON; return to the editor to correct it: "+err.Error(), 422)
				return
			}
		}
		collectionName := "Unfiled"
		validCollection := q.CollectionID == ""
		for _, c := range s.data.Collections {
			if c.ID == q.CollectionID {
				validCollection = true
				collectionName = c.Name
				break
			}
		}
		if !validCollection {
			s.mu.Unlock()
			s.renderFor(w, r, selected, "Collection no longer exists. Choose a save destination.", nil, q)
			return
		}
		if q.Method == "" {
			q.Method = "GET"
		}
		if q.Name == "" {
			q.Name = q.URL
		}
		if q.ID == "" {
			q.ID = id()
		}
		selected = q.ID
		if r.FormValue("action") == "send" {
			vars := variablesFor(s.data, q.CollectionID, r.FormValue("environment"))
			s.mu.Unlock()
			resolved, errSend := resolve(q, vars)
			if errSend == nil {
				response, errSend = send(r, resolved)
			}
			if errSend != nil {
				message = errSend.Error()
			}
			s.renderFor(w, r, selected, message, response, q)
			return
		}
		draft = []store.Request{q}
		found := false
		for i := range s.data.Requests {
			if s.data.Requests[i].ID == q.ID {
				if q.CollectionID == s.data.Requests[i].CollectionID {
					q.FolderID = s.data.Requests[i].FolderID
				}
				q.TLS.CACertFile = s.data.Requests[i].TLS.CACertFile
				q.TLS.ClientCertFile = s.data.Requests[i].TLS.ClientCertFile
				q.TLS.ClientKeyFile = s.data.Requests[i].TLS.ClientKeyFile
				s.data.Requests[i] = q
				found = true
				break
			}
		}
		if !found {
			if r.FormValue("id") != "" {
				s.mu.Unlock()
				s.renderFor(w, r, selected, "Request no longer exists. Keep this draft and explicitly save as new.", nil, q)
				return
			}
			s.data.Requests = append(s.data.Requests, q)
		}
		message = "Saved to " + collectionName + "."
	case "/organize", "/delete":
		if op == "/delete" {
			r.Form.Set("kind", "request")
			r.Form.Set("action", "delete")
		}
		if err := organize(&s.data, r.Form); err != nil {
			s.mu.Unlock()
			s.renderFor(w, r, selected, err.Error(), nil)
			return
		}
		message = "Organization updated."
		if r.FormValue("action") == "delete" {
			selected = ""
		}
	case "/variables":
		if err := editVariables(&s.data, r.Form); err != nil {
			s.mu.Unlock()
			s.renderFor(w, r, selected, err.Error(), nil)
			return
		}
		message = "Variables updated."
	case "/collection":
		if n := strings.TrimSpace(r.FormValue("name")); n != "" {
			s.data.Collections = append(s.data.Collections, store.Collection{ID: id(), Name: n})
		}
	case "/import":
		f, _, err := r.FormFile("file")
		if err != nil {
			message = err.Error()
			break
		}
		b, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			message = err.Error()
			break
		}
		if r.FormValue("kind") == "environment" {
			e, err := postman.ImportEnvironment(strings.NewReader(string(b)))
			if err != nil {
				message = err.Error()
				break
			}
			env := store.Environment{ID: id(), Name: e.Name}
			for _, v := range e.Variables {
				env.Variables = append(env.Variables, store.Field{Key: v.Key, Value: fmt.Sprint(v.Value), Enabled: !v.Disabled})
			}
			s.data.Environments = append(s.data.Environments, env)
			message = importSummary("environment", e.Name, fmt.Sprintf("%d variables. Select this environment before sending requests.", len(e.Variables)), e.Warnings)
		} else {
			c, err := postman.ImportCollection(strings.NewReader(string(b)))
			if err != nil {
				message = err.Error()
				break
			}
			cid := id()
			collection := store.Collection{ID: cid, Name: c.Name}
			for _, v := range c.Variables {
				collection.Variables = append(collection.Variables, store.Field{Key: v.Key, Value: fmt.Sprint(v.Value), Enabled: !v.Disabled})
			}
			s.data.Collections = append(s.data.Collections, collection)
			ids := map[string]string{}
			for _, f := range c.Folders {
				ids[f.ID] = id()
			}
			for _, f := range c.Folders {
				s.data.Folders = append(s.data.Folders, store.Folder{ID: ids[f.ID], CollectionID: cid, ParentID: ids[f.ParentID], Name: f.Name})
			}
			requestStart := len(s.data.Requests)
			for _, v := range c.Requests {
				q := store.Request{ID: id(), CollectionID: cid, FolderID: ids[v.ParentID], Name: v.Name, Method: v.Method, URL: v.URL}
				for _, h := range v.Headers {
					q.Headers = append(q.Headers, store.Field{Key: h.Key, Value: fmt.Sprint(h.Value), Enabled: !h.Disabled})
				}
				if parsed, err := url.Parse(q.URL); err == nil && parsed.RawQuery != "" && len(v.Query) > 0 {
					parsed.RawQuery = ""
					q.URL = parsed.String()
				}
				for _, h := range v.Query {
					q.Query = append(q.Query, store.Field{Key: h.Key, Value: fmt.Sprint(h.Value), Enabled: !h.Disabled})
				}
				var warnings []string
				q.Body, warnings = importedBody(v.Body)
				for _, warning := range warnings {
					c.Warnings = append(c.Warnings, v.Name+": "+warning)
				}
				if v.Auth != nil {
					if v.Auth.Type != "basic" && v.Auth.Type != "bearer" && v.Auth.Type != "apikey" && v.Auth.Type != "noauth" {
						c.Warnings = append(c.Warnings, v.Name+": unsupported authentication "+v.Auth.Type)
					}
					q.Auth.Type = v.Auth.Type
					for _, a := range v.Auth.Values {
						switch a.Key {
						case "username":
							q.Auth.Username = fmt.Sprint(a.Value)
						case "password":
							q.Auth.Password = fmt.Sprint(a.Value)
						case "token":
							q.Auth.Token = fmt.Sprint(a.Value)
						case "key":
							q.Auth.KeyName, _ = a.Value.(string)
						case "value":
							q.Auth.KeyValue, _ = a.Value.(string)
						case "in":
							q.Auth.KeyLocation, _ = a.Value.(string)
						}
					}
				}
				s.data.Requests = append(s.data.Requests, q)
			}
			if missing := missingCollectionVariables(s.data.Requests[requestStart:], collection.Variables); len(missing) > 0 {
				c.Warnings = append(c.Warnings, "Not defined in collection: "+strings.Join(missing, ", ")+". Import/select an environment supplying these variables before sending.")
			}
			message = importSummary("collection", c.Name, fmt.Sprintf("%d folders, %d requests, %d variables. Saved as a new collection; existing collections unchanged.", len(c.Folders), len(c.Requests), len(c.Variables)), c.Warnings)
		}
	default:
		s.mu.Unlock()
		http.NotFound(w, r)
		return
	}
	if err := store.Save(s.path, s.data); err != nil {
		_ = json.Unmarshal(snapshot, &s.data)
		message = "Save failed: " + err.Error()
		s.mu.Unlock()
		s.renderFor(w, r, selected, message, response, draft...)
		return
	}
	s.mu.Unlock()
	s.renderFor(w, r, selected, message, response)
}
func resolve(q store.Request, vars map[string]string) (store.Request, error) {
	var failure error
	replace := func(s string) string {
		for start := strings.Index(s, "{{"); start >= 0; start = strings.Index(s, "{{") {
			end := strings.Index(s[start+2:], "}}")
			if end < 0 {
				failure = fmt.Errorf("unclosed variable")
				return s
			}
			end += start + 2
			key := strings.TrimSpace(s[start+2 : end])
			v, ok := vars[key]
			if !ok {
				failure = fmt.Errorf("missing variable: %s", key)
				return s
			}
			if strings.Contains(v, "{{") {
				failure = fmt.Errorf("nested variables are not supported: %s", key)
				return s
			}
			s = s[:start] + v + s[end+2:]
		}
		return s
	}
	q.URL = replace(q.URL)
	q.Body.Text = replace(q.Body.Text)
	q.Body.Fields = append([]store.BodyField(nil), q.Body.Fields...)
	for i := range q.Body.Fields {
		if q.Body.Fields[i].Enabled {
			q.Body.Fields[i].Key = replace(q.Body.Fields[i].Key)
			q.Body.Fields[i].Value = replace(q.Body.Fields[i].Value)
		}
	}
	q.Auth.Username = replace(q.Auth.Username)
	q.Auth.Password = replace(q.Auth.Password)
	q.Auth.Token = replace(q.Auth.Token)
	if q.Auth.Type == "apikey" {
		q.Auth.KeyName = replace(q.Auth.KeyName)
		q.Auth.KeyValue = replace(q.Auth.KeyValue)
		if failure != nil {
			return q, fmt.Errorf("API-key variable resolution failed")
		}
	}
	for i := range q.Headers {
		if !q.Headers[i].Enabled {
			continue
		}
		q.Headers[i].Key = replace(q.Headers[i].Key)
		q.Headers[i].Value = replace(q.Headers[i].Value)
	}
	for i := range q.Query {
		if !q.Query[i].Enabled {
			continue
		}
		q.Query[i].Key = replace(q.Query[i].Key)
		q.Query[i].Value = replace(q.Query[i].Value)
	}
	return q, failure
}
func send(r *http.Request, q store.Request) (*client.Response, error) {
	if q.TimeoutSeconds == 0 {
		q.TimeoutSeconds = 30 // Legacy stores and imported requests.
	}
	if q.TimeoutSeconds < 1 || q.TimeoutSeconds > 300 {
		return nil, fmt.Errorf("timeout must be from 1 to 300 seconds")
	}
	if q.Auth.Type != "" && q.Auth.Type != "noauth" && q.Auth.Type != "basic" && q.Auth.Type != "bearer" && q.Auth.Type != "apikey" {
		return nil, fmt.Errorf("unsupported authentication: %s", q.Auth.Type)
	}
	if q.Auth.Type == "apikey" {
		if strings.TrimSpace(q.Auth.KeyName) == "" {
			return nil, fmt.Errorf("API-key name must not be empty")
		}
		if q.Auth.KeyLocation != "header" && q.Auth.KeyLocation != "query" {
			return nil, fmt.Errorf("API-key location must be header or query")
		}
	}
	u, err := url.Parse(q.URL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, fmt.Errorf("URL must be HTTP or HTTPS")
	}
	values := u.Query()
	for _, v := range q.Query {
		if v.Enabled {
			values.Add(v.Key, v.Value)
		}
	}
	u.RawQuery = values.Encode()
	h := http.Header{}
	for _, v := range q.Headers {
		if v.Enabled {
			h.Add(v.Key, v.Value)
		}
	}
	if q.Auth.Type == "bearer" {
		h.Set("Authorization", "Bearer "+q.Auth.Token)
	}
	if q.Auth.Type == "basic" {
		req := &http.Request{Header: h}
		req.SetBasicAuth(q.Auth.Username, q.Auth.Password)
	}
	body, err := requestBody(r, q.Body, h)
	if err != nil {
		return nil, err
	}
	// Authentication overrides configured credentials, including URL query duplicates.
	if q.Auth.Type == "apikey" {
		if q.Auth.KeyLocation == "header" {
			h.Set(q.Auth.KeyName, q.Auth.KeyValue)
		} else {
			values.Set(q.Auth.KeyName, q.Auth.KeyValue)
			u.RawQuery = values.Encode()
		}
	}
	result, err := client.Execute(r.Context(), client.Request{Method: q.Method, URL: u.String(), Headers: h, Body: body, Timeout: time.Duration(q.TimeoutSeconds) * time.Second, InsecureSkipVerify: q.TLS.InsecureSkipVerify})
	if err != nil && q.Auth.Type == "apikey" {
		// net/http errors can include the URL or invalid header name (both may be secrets).
		return nil, fmt.Errorf("API-key request failed; check request configuration and connection")
	}
	return &result, err
}
