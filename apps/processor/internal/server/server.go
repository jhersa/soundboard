// Package server recibe jobs firmados de la API, los procesa uno a la vez (ffmpeg y yt-dlp son
// pesados para una VM chica) y reporta el resultado a la API.
package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"path"
	"strings"
	"sync/atomic"
	"time"

	"soundboard/gosig"
	"soundboard/processor/internal/pipeline"
)

const (
	maxBodyBytes = 64 * 1024
	queueSize    = 256
	uploadScheme = "upload:"
)

type Config struct {
	Secret string
	// Si no está vacío, reemplaza el apiBaseUrl del job. En Docker local la API corre en el host:
	// la URL que manda (localhost) no sirve dentro del contenedor (p. ej. http://host.docker.internal:8788).
	APIBaseURL   string
	YtDlpCookies string
}

// ProcessRequest es lo que manda la API (POST /process); ver ProcessRequest en @soundboard/shared.
type ProcessRequest struct {
	JobID      string  `json:"jobId"`
	URL        string  `json:"url"`
	Title      string  `json:"title"`
	ImageURL   *string `json:"imageUrl"`
	From       *string `json:"from"`
	To         *string `json:"to"`
	Stereo     bool    `json:"stereo"`
	APIBaseURL string  `json:"apiBaseUrl"`
}

// SoundProcessor es lo que el servidor necesita del pipeline (se sustituye en tests).
type SoundProcessor interface {
	Process(ctx context.Context, options pipeline.Options) (pipeline.Result, error)
}

type Server struct {
	Config   Config
	Pipeline SoundProcessor
	HTTP     *http.Client
	Log      *slog.Logger
	Now      func() time.Time

	queue    chan ProcessRequest
	pending  atomic.Int64
	stopping atomic.Bool
}

func New(config Config, processor SoundProcessor, client *http.Client, log *slog.Logger) *Server {
	config.APIBaseURL = strings.TrimRight(config.APIBaseURL, "/")
	return &Server{Config: config, Pipeline: processor, HTTP: client, Log: log, Now: time.Now, queue: make(chan ProcessRequest, queueSize)}
}

func (s *Server) Pending() int64 { return s.pending.Load() }

// ---- HTTP ----

func (s *Server) Handler() http.Handler { return http.HandlerFunc(s.serve) }

func (s *Server) serve(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet && r.URL.RequestURI() == "/health" {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"ok":true,"pending":%d}`, s.pending.Load())
		return
	}
	if r.Method != http.MethodPost || r.URL.RequestURI() != "/process" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes+1))
	if err != nil || len(body) > maxBodyBytes {
		http.Error(w, "Cuerpo demasiado grande", http.StatusBadRequest)
		return
	}
	if !gosig.Verify(s.Config.Secret, body, r.Header.Get(gosig.HeaderTimestamp), r.Header.Get(gosig.HeaderSignature), s.Now()) {
		http.Error(w, "Firma inválida", http.StatusUnauthorized)
		return
	}
	var job ProcessRequest
	if err := json.Unmarshal(body, &job); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}
	if s.Config.APIBaseURL != "" {
		job.APIBaseURL = s.Config.APIBaseURL
	}
	if job.JobID == "" || job.URL == "" || job.APIBaseURL == "" {
		http.Error(w, "Faltan jobId, url o apiBaseUrl", http.StatusBadRequest)
		return
	}

	s.pending.Add(1)
	select {
	case s.queue <- job:
	default:
		s.pending.Add(-1)
		http.Error(w, "Cola llena", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintf(w, `{"queued":%d}`, s.pending.Load())
}

// ---- Cola ----

// Work procesa la cola de a un job hasta que se cierra con Stop. jobCtx cancela el job en curso
// (mata ffmpeg o yt-dlp); los jobs que no alcanzaron a empezar se reportan como fallidos.
func (s *Server) Work(jobCtx context.Context) {
	for job := range s.queue {
		if s.stopping.Load() {
			s.reportDropped(job)
		} else {
			s.runJob(jobCtx, job)
		}
		s.pending.Add(-1)
	}
}

// Stop deja de aceptar jobs. Hay que llamarlo después de cerrar el servidor HTTP.
func (s *Server) Stop() {
	if s.stopping.CompareAndSwap(false, true) {
		close(s.queue)
	}
}

func (s *Server) reportDropped(job ProcessRequest) {
	message := "El processor se reinició antes de procesar este job; reintenta"
	s.reportFailure(job, message)
}

func (s *Server) reportFailure(job ProcessRequest, message string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.reportStatus(ctx, job, statusUpdate{Status: "failed", Error: ptr(truncate(message, 2000))}); err != nil {
		s.Log.Error("no se pudo reportar el fallo", "job", job.JobID, "err", err)
	}
}

type statusUpdate struct {
	Status string  `json:"status"`
	Error  *string `json:"error,omitempty"`
}

func (s *Server) runJob(ctx context.Context, job ProcessRequest) {
	s.Log.Info("procesando", "job", job.JobID, "title", job.Title, "url", job.URL)
	if err := s.process(ctx, job); err != nil {
		s.Log.Error("falló", "job", job.JobID, "err", err)
		s.reportFailure(job, err.Error())
	}
}

func (s *Server) process(ctx context.Context, job ProcessRequest) error {
	if err := s.reportStatus(ctx, job, statusUpdate{Status: "running"}); err != nil {
		return err
	}
	result, err := s.Pipeline.Process(ctx, pipeline.Options{
		URL:          job.URL,
		ImageURL:     job.ImageURL,
		From:         job.From,
		To:           job.To,
		Stereo:       job.Stereo,
		FetchUpload:  s.uploadFetcher(job),
		YtDlpCookies: s.Config.YtDlpCookies,
	})
	if err != nil {
		return err
	}
	body, contentType, err := resultBody(result)
	if err != nil {
		return err
	}
	if err := s.post(ctx, job, "result", body, contentType); err != nil {
		return err
	}
	s.Log.Info("listo", "job", job.JobID, "segundos", float64(result.DurationMs)/1000)
	return nil
}

// resultBody arma el multipart (meta, audio, image) que espera /v1/internal/jobs/:id/result. Se
// serializa completo para firmar exactamente los bytes que se envían.
func resultBody(result pipeline.Result) ([]byte, string, error) {
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	meta, _ := json.Marshal(map[string]int64{"durationMs": result.DurationMs})
	if err := writer.WriteField("meta", string(meta)); err != nil {
		return nil, "", err
	}
	// CreateFormFile fijaría application/octet-stream y la API necesita el tipo real de la imagen.
	files := []struct {
		field, name, contentType string
		data                     []byte
	}{{"audio", "audio.ogg", "audio/ogg", result.Audio}}
	if result.Image != nil {
		files = append(files, struct {
			field, name, contentType string
			data                     []byte
		}{"image", "image", result.Image.ContentType, result.Image.Data})
	}
	for _, file := range files {
		part, err := writer.CreatePart(textproto.MIMEHeader{
			"Content-Disposition": {fmt.Sprintf(`form-data; name=%q; filename=%q`, file.field, file.name)},
			"Content-Type":        {file.contentType},
		})
		if err != nil {
			return nil, "", err
		}
		if _, err := part.Write(file.data); err != nil {
			return nil, "", err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return buffer.Bytes(), writer.FormDataContentType(), nil
}

// ---- Llamadas a la API ----

func (s *Server) jobURL(job ProcessRequest, route string) string {
	return strings.TrimRight(job.APIBaseURL, "/") + "/v1/internal/jobs/" + url.PathEscape(job.JobID) + "/" + route
}

func (s *Server) reportStatus(ctx context.Context, job ProcessRequest, update statusUpdate) error {
	body, _ := json.Marshal(update)
	return s.post(ctx, job, "status", body, "application/json")
}

func (s *Server) post(ctx context.Context, job ProcessRequest, route string, body []byte, contentType string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, s.jobURL(job, route), bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", contentType)
	for header, value := range gosig.Headers(s.Config.Secret, body, s.Now()) {
		request.Header.Set(header, value)
	}
	response, err := s.HTTP.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return fmt.Errorf("La API respondió %d en %s: %s", response.StatusCode, route, strings.TrimSpace(string(detail)))
	}
	return nil
}

// uploadFetcher baja los uploads del panel por la API; se firma la key en lugar del cuerpo (GET).
func (s *Server) uploadFetcher(job ProcessRequest) pipeline.UploadFetcher {
	return func(ctx context.Context, source string) (pipeline.Downloaded, error) {
		key := strings.TrimPrefix(source, uploadScheme)
		request, err := http.NewRequestWithContext(ctx, http.MethodGet,
			strings.TrimRight(job.APIBaseURL, "/")+"/v1/internal/uploads/"+key, nil)
		if err != nil {
			return pipeline.Downloaded{}, err
		}
		for header, value := range gosig.Headers(s.Config.Secret, []byte(key), s.Now()) {
			request.Header.Set(header, value)
		}
		response, err := s.HTTP.Do(request)
		if err != nil {
			return pipeline.Downloaded{}, err
		}
		defer response.Body.Close()
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			return pipeline.Downloaded{}, fmt.Errorf("No se pudo bajar %s: HTTP %d", key, response.StatusCode)
		}
		data, err := io.ReadAll(response.Body)
		if err != nil {
			return pipeline.Downloaded{}, err
		}
		ext := path.Ext(key)
		if ext == "" {
			ext = ".bin"
		}
		contentType := strings.TrimSpace(strings.SplitN(response.Header.Get("Content-Type"), ";", 2)[0])
		return pipeline.Downloaded{Data: data, ContentType: contentType, Ext: ext}, nil
	}
}

func truncate(text string, max int) string {
	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	return string(runes[:max])
}

func ptr[T any](value T) *T { return &value }
