package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"soundboard/api/internal/config"
	"soundboard/api/internal/db"
	"soundboard/api/internal/google"
	"soundboard/api/internal/storage"
	"soundboard/api/migrations"
	signature "soundboard/gosig"
)

// fakeGoogle acepta tokens "<sub>|<email>"; evita depender de las llaves reales de Google.
type fakeGoogle struct{}

func (fakeGoogle) Verify(_ context.Context, idToken string, audiences []string) (google.Identity, error) {
	if len(audiences) != 1 || audiences[0] != "test-client-id" {
		return google.Identity{}, errors.New("audiencia")
	}
	sub, email, ok := strings.Cut(idToken, "|")
	if !ok || sub == "" || email == "" {
		return google.Identity{}, errors.New("token inválido")
	}
	name := "Usuario " + sub
	return google.Identity{Sub: sub, Email: email, Name: &name}, nil
}

// fakeProcessor registra lo que recibe y responde con `status` (0 = conexión rechazada).
type fakeProcessor struct {
	mu       sync.Mutex
	status   int
	requests []*http.Request
	bodies   [][]byte
}

func (p *fakeProcessor) RoundTrip(r *http.Request) (*http.Response, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	body, _ := io.ReadAll(r.Body)
	p.requests = append(p.requests, r)
	p.bodies = append(p.bodies, body)
	if p.status == 0 {
		return nil, errors.New("down")
	}
	return &http.Response{StatusCode: p.status, Body: io.NopCloser(strings.NewReader("")), Header: http.Header{}}, nil
}

type testAPI struct {
	t         *testing.T
	server    *Server
	handler   http.Handler
	store     storage.FS
	processor *fakeProcessor
}

func newTestAPI(t *testing.T) *testAPI {
	t.Helper()
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	if _, err := db.Migrate(context.Background(), database, migrations.FS); err != nil {
		t.Fatal(err)
	}
	processor := &fakeProcessor{status: http.StatusAccepted}
	store := storage.FS{Dir: t.TempDir()}
	server := &Server{
		Config: config.Config{
			GoogleClientIDs: []string{"test-client-id"},
			AdminEmails:     []string{"admin@example.com"},
			CDNBaseURL:      "https://cdn.test",
			AdminOrigin:     "https://admin.test",
			ProcessorURL:    "https://processor.test",
			ProcessorSecret: "test-secret",
		},
		DB:     database,
		Store:  store,
		Google: fakeGoogle{},
		HTTP:   &http.Client{Transport: processor},
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	return &testAPI{t: t, server: server, handler: server.Handler(), store: store, processor: processor}
}

type response struct {
	*httptest.ResponseRecorder
}

func (r response) decode(t *testing.T, dst any) {
	t.Helper()
	if err := json.Unmarshal(r.Body.Bytes(), dst); err != nil {
		t.Fatalf("JSON inválido (%d): %s", r.Code, r.Body.String())
	}
}

// call hace la petición y espera los envíos al processor en segundo plano.
func (a *testAPI) call(method, path string, body any, token string, headers ...string) response {
	a.t.Helper()
	var reader io.Reader
	contentType := ""
	switch value := body.(type) {
	case nil:
	case []byte:
		reader = bytes.NewReader(value)
	case string:
		reader = strings.NewReader(value)
		contentType = "application/json"
	default:
		raw, _ := json.Marshal(value)
		reader = bytes.NewReader(raw)
		contentType = "application/json"
	}
	request := httptest.NewRequest(method, "https://api.test"+path, reader)
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	for i := 0; i+1 < len(headers); i += 2 {
		request.Header.Set(headers[i], headers[i+1])
	}
	recorder := httptest.NewRecorder()
	a.handler.ServeHTTP(recorder, request)
	a.server.Wait()
	return response{recorder}
}

func (a *testAPI) signIn(sub, email string) SignInResponse {
	a.t.Helper()
	res := a.call("POST", "/v1/auth/google", map[string]string{"idToken": sub + "|" + email}, "")
	if res.Code != http.StatusOK {
		a.t.Fatalf("signIn %d: %s", res.Code, res.Body.String())
	}
	var signIn SignInResponse
	res.decode(a.t, &signIn)
	return signIn
}

func (a *testAPI) adminToken() string { return a.signIn("admin", "admin@example.com").Token }

// signedCall firma el cuerpo como lo hace el processor.
func (a *testAPI) signedCall(path string, body []byte, contentType string) response {
	headers := []string{"Content-Type", contentType}
	for header, value := range signature.Headers("test-secret", body, a.server.now()) {
		headers = append(headers, header, value)
	}
	return a.call("POST", path, body, "", headers...)
}

type filePart struct {
	field, name, contentType string
	data                     []byte
}

func multipartBody(t *testing.T, fields map[string]string, files ...filePart) ([]byte, string) {
	t.Helper()
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	for name, value := range fields {
		writer.WriteField(name, value)
	}
	for _, file := range files {
		header := make(map[string][]string)
		header["Content-Disposition"] = []string{`form-data; name="` + file.field + `"; filename="` + file.name + `"`}
		header["Content-Type"] = []string{file.contentType}
		part, _ := writer.CreatePart(header)
		part.Write(file.data)
	}
	writer.Close()
	return buffer.Bytes(), writer.FormDataContentType()
}

func (a *testAPI) seedSound(id, title, status, createdAt string) {
	a.t.Helper()
	if status == "" {
		status = "published"
	}
	if createdAt == "" {
		createdAt = db.Now()
	}
	_, err := a.server.DB.Exec(
		`INSERT INTO sounds (id, title, duration_ms, audio_key, image_key, image_type, status, created_at)
		 VALUES (?, ?, 1000, ?, ?, 'image/gif', ?, ?)`,
		id, title, "sounds/"+id+".ogg", "images/"+id+".gif", status, createdAt)
	if err != nil {
		a.t.Fatal(err)
	}
}

func soundID(char string) string { return strings.Repeat(char, 64) }

func (a *testAPI) exists(key string) bool {
	object, err := a.store.Get(context.Background(), key)
	if object != nil {
		object.Body.Close()
	}
	return err == nil && object != nil
}
