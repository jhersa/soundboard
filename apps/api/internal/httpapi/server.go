// Package httpapi implementa la API /v1. Las rutas, el JSON y los errores ({"error": "..."}) son
// los mismos que los de @soundboard/shared: la app Android, el panel y el processor dependen de ellos.
package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"soundboard/api/internal/config"
	"soundboard/api/internal/google"
	"soundboard/api/internal/storage"
)

type Server struct {
	Config config.Config
	DB     *sql.DB
	Store  storage.Store
	Google google.Verifier
	// Cliente para llamar al processor.
	HTTP *http.Client
	Log  *slog.Logger
	// Now permite fijar el reloj en tests.
	Now func() time.Time

	background sync.WaitGroup
}

// Wait espera los envíos al processor que siguen en segundo plano (apagado y tests).
func (s *Server) Wait() { s.background.Wait() }

func (s *Server) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, "soundboard-api") })
	mux.HandleFunc("GET /cdn/{key...}", s.handle(s.cdn))

	mux.HandleFunc("POST /v1/auth/google", s.handle(s.signIn))
	mux.HandleFunc("POST /v1/auth/logout", s.handle(s.logout))
	mux.HandleFunc("GET /v1/me", s.handle(s.me))

	mux.HandleFunc("GET /v1/catalog", s.handle(s.catalog))

	mux.HandleFunc("GET /v1/me/library", s.handle(s.getLibrary))
	mux.HandleFunc("PUT /v1/me/library/order", s.handle(s.orderLibrary))
	mux.HandleFunc("PUT /v1/me/library/{soundId}", s.handle(s.addToLibrary))
	mux.HandleFunc("DELETE /v1/me/library/{soundId}", s.handle(s.removeFromLibrary))

	mux.HandleFunc("POST /v1/admin/uploads", s.handle(s.admin(s.upload)))
	mux.HandleFunc("POST /v1/admin/jobs", s.handle(s.admin(s.createJob)))
	mux.HandleFunc("GET /v1/admin/jobs", s.handle(s.admin(s.listJobs)))
	mux.HandleFunc("POST /v1/admin/jobs/{id}/retry", s.handle(s.admin(s.retryJob)))
	mux.HandleFunc("DELETE /v1/admin/jobs/{id}", s.handle(s.admin(s.deleteJob)))
	mux.HandleFunc("GET /v1/admin/sounds", s.handle(s.admin(s.listSounds)))
	mux.HandleFunc("PATCH /v1/admin/sounds/{id}", s.handle(s.admin(s.updateSound)))
	mux.HandleFunc("DELETE /v1/admin/sounds/{id}", s.handle(s.admin(s.deleteSound)))

	mux.HandleFunc("GET /v1/internal/uploads/{key...}", s.handle(s.internalUpload))
	mux.HandleFunc("POST /v1/internal/jobs/{id}/status", s.handle(s.jobStatus))
	mux.HandleFunc("POST /v1/internal/jobs/{id}/result", s.handle(s.jobResult))

	return s.logRequests(s.cors(mux))
}

// cors replica hono/cors con un único origen: solo en /v1/* y respondiendo el preflight con 204.
func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/v1/") {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", s.Config.AdminOrigin)
		w.Header().Add("Vary", "Origin")
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Max-Age", "600")
			w.Header().Set("Access-Control-Allow-Methods", "GET,HEAD,PUT,POST,DELETE,PATCH")
			w.Header().Set("Access-Control-Allow-Headers", "authorization,content-type")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requestLevel: los errores del servidor son Error y los del cliente Warn. El healthcheck de Docker
// (GET / cada 30 s) baja a Debug: era más de la mitad del log y gastaba la rotación sin aportar nada.
func requestLevel(r *http.Request, status int) slog.Level {
	switch {
	case status >= 500:
		return slog.LevelError
	case status >= 400:
		return slog.LevelWarn
	case r.Method == http.MethodGet && r.URL.Path == "/":
		return slog.LevelDebug
	}
	return slog.LevelInfo
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		s.Log.Log(r.Context(), requestLevel(r, recorder.status), "petición", "method", r.Method, "path", r.URL.Path,
			"status", recorder.status, "ms", time.Since(start).Milliseconds())
	})
}

// ---- Errores y respuestas ----

type apiError struct {
	status  int
	message string
}

func (e *apiError) Error() string { return e.message }

func fail(status int, message string) error { return &apiError{status: status, message: message} }

type handlerFunc func(w http.ResponseWriter, r *http.Request) error

func (s *Server) handle(handler handlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := handler(w, r)
		if err == nil {
			return
		}
		var known *apiError
		var tooLarge *http.MaxBytesError
		switch {
		case errors.As(err, &known):
			writeJSON(w, known.status, map[string]string{"error": known.message})
		case errors.As(err, &tooLarge):
			writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "Archivo demasiado grande"})
		default:
			s.Log.Error("error interno", "method", r.Method, "path", r.URL.Path, "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Error interno"})
		}
	}
}

// writeJSON no escapa <, > ni & (igual que JSON.stringify).
func writeJSON(w http.ResponseWriter, status int, value any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}

// ---- URLs ----

// cdnBase: sin CDN_BASE_URL (desarrollo) los medios se sirven desde /cdn del mismo host de la
// petición, así el emulador (10.0.2.2) y el navegador (localhost) reciben URLs que pueden abrir.
func (s *Server) cdnBase(r *http.Request) string {
	if s.Config.CDNBaseURL != "" {
		return s.Config.CDNBaseURL
	}
	return requestOrigin(r) + "/cdn"
}

func requestOrigin(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := r.Header.Get("X-Forwarded-Proto"); forwarded != "" {
		scheme = strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	return scheme + "://" + r.Host
}

var publicMediaKey = regexp.MustCompile(`^(sounds|images)/`)

// Solo para desarrollo local (SERVE_MEDIA=true): en producción los medios salen directo del bucket.
func (s *Server) cdn(w http.ResponseWriter, r *http.Request) error {
	key := r.PathValue("key")
	if !s.Config.ServeMedia || !publicMediaKey.MatchString(key) {
		http.NotFound(w, r)
		return nil
	}
	object, err := s.Store.Get(r.Context(), key)
	if err != nil {
		return err
	}
	if object == nil {
		http.NotFound(w, r)
		return nil
	}
	defer object.Body.Close()
	for header, value := range map[string]string{
		"Content-Type": object.ContentType, "Cache-Control": object.CacheControl, "ETag": object.ETag,
	} {
		if value != "" {
			w.Header().Set(header, value)
		}
	}
	_, err = io.Copy(w, object.Body)
	return err
}
