package storage

import (
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFS(t *testing.T) {
	ctx := context.Background()
	store := FS{Dir: t.TempDir()}
	if err := store.Put(ctx, "sounds/a.ogg", []byte("hola"), PutOptions{ContentType: "audio/ogg", CacheControl: "immutable"}); err != nil {
		t.Fatal(err)
	}
	object, err := store.Get(ctx, "sounds/a.ogg")
	if err != nil || object == nil {
		t.Fatalf("get: %v %v", object, err)
	}
	body, _ := io.ReadAll(object.Body)
	object.Body.Close()
	if string(body) != "hola" || object.ContentType != "audio/ogg" || object.CacheControl != "immutable" {
		t.Fatalf("objeto inesperado: %q %+v", body, object)
	}
	if err := store.Delete(ctx, "sounds/a.ogg", "sounds/no-existe.ogg"); err != nil {
		t.Fatal(err)
	}
	if object, _ := store.Get(ctx, "sounds/a.ogg"); object != nil {
		t.Fatal("debió borrarse")
	}
	if err := store.Put(ctx, "../fuera.txt", nil, PutOptions{}); err == nil {
		t.Fatal("debió rechazar una key con ..")
	}
}

func TestGCS(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.RequestURI())
		switch {
		case r.Method == http.MethodPost:
			_, params, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
			reader := multipart.NewReader(r.Body, params["boundary"])
			meta, _ := reader.NextPart()
			metaBody, _ := io.ReadAll(meta)
			data, _ := reader.NextPart()
			dataBody, _ := io.ReadAll(data)
			if !strings.Contains(string(metaBody), `"cacheControl":"immutable"`) || string(dataBody) != "hola" ||
				data.Header.Get("Content-Type") != "audio/ogg" {
				http.Error(w, "multipart inesperado", http.StatusBadRequest)
			}
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "falta"):
			http.NotFound(w, r)
		case r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "audio/mpeg")
			io.WriteString(w, "datos")
		case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "prohibido"):
			http.Error(w, "denegado", http.StatusForbidden)
		}
	}))
	defer server.Close()
	ctx := context.Background()
	store := GCS{Bucket: "media", Client: server.Client(), BaseURL: server.URL}

	if err := store.Put(ctx, "sounds/a.ogg", []byte("hola"), PutOptions{ContentType: "audio/ogg", CacheControl: "immutable"}); err != nil {
		t.Fatal(err)
	}
	if object, err := store.Get(ctx, "uploads/falta.mp3"); object != nil || err != nil {
		t.Fatalf("404 debió dar nil, nil: %v %v", object, err)
	}
	object, err := store.Get(ctx, "uploads/x.mp3")
	if err != nil || object.ContentType != "audio/mpeg" {
		t.Fatalf("get: %+v %v", object, err)
	}
	object.Body.Close()
	if err := store.Delete(ctx, "sounds/a.ogg"); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, "sounds/prohibido.ogg"); err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("debió fallar con 403: %v", err)
	}

	want := []string{
		"POST /upload/storage/v1/b/media/o?uploadType=multipart",
		"GET /storage/v1/b/media/o/uploads%2Ffalta.mp3?alt=media",
		"GET /storage/v1/b/media/o/uploads%2Fx.mp3?alt=media",
		"DELETE /storage/v1/b/media/o/sounds%2Fa.ogg",
		"DELETE /storage/v1/b/media/o/sounds%2Fprohibido.ogg",
	}
	if strings.Join(requests, "\n") != strings.Join(want, "\n") {
		t.Fatalf("peticiones:\n%s", strings.Join(requests, "\n"))
	}
}
