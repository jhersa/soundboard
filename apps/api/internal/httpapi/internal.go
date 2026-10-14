package httpapi

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"

	"soundboard/api/internal/storage"
	signature "soundboard/gosig"
)

// Rutas que solo llama el processor; se autentican con HMAC sobre el cuerpo (o la key en GET).

func (s *Server) verifiedBody(w http.ResponseWriter, r *http.Request, signedContent []byte) ([]byte, error) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxProcessorResult))
	if err != nil {
		return nil, err
	}
	content := body
	if signedContent != nil {
		content = signedContent
	}
	if !signature.Verify(s.Config.ProcessorSecret, content,
		r.Header.Get(signature.HeaderTimestamp), r.Header.Get(signature.HeaderSignature), s.now()) {
		return nil, fail(http.StatusUnauthorized, "Firma inválida")
	}
	return body, nil
}

func (s *Server) findJob(r *http.Request) (jobRow, error) {
	job, err := scanJob(s.DB.QueryRowContext(r.Context(), "SELECT "+jobColumns+" FROM jobs WHERE id = ?", r.PathValue("id")))
	if errors.Is(err, sql.ErrNoRows) {
		return job, fail(http.StatusNotFound, "Job no encontrado")
	}
	return job, err
}

func uploadKey(url *string) (string, bool) {
	if url == nil || !strings.HasPrefix(*url, uploadScheme) {
		return "", false
	}
	return strings.TrimPrefix(*url, uploadScheme), true
}

func (s *Server) internalUpload(w http.ResponseWriter, r *http.Request) error {
	key := r.PathValue("key")
	if _, err := s.verifiedBody(w, r, []byte(key)); err != nil {
		return err
	}
	if !strings.HasPrefix(key, "uploads/") {
		return fail(http.StatusBadRequest, "Key inválida")
	}
	object, err := s.Store.Get(r.Context(), key)
	if err != nil {
		return err
	}
	if object == nil {
		return fail(http.StatusNotFound, "Archivo no encontrado")
	}
	defer object.Body.Close()
	w.Header().Set("Content-Type", orDefault(object.ContentType, "application/octet-stream"))
	_, err = io.Copy(w, object.Body)
	return err
}

func (s *Server) jobStatus(w http.ResponseWriter, r *http.Request) error {
	body, err := s.verifiedBody(w, r, nil)
	if err != nil {
		return err
	}
	job, err := s.findJob(r)
	if err != nil {
		return err
	}
	var update jobStatusUpdate
	if err := json.Unmarshal(body, &update); err != nil {
		return fail(http.StatusBadRequest, "JSON inválido")
	}
	if err := update.validate(); err != nil {
		return err
	}
	var message *string
	if update.Status == "failed" {
		message = update.Error
	}
	if err := s.setJobStatus(r.Context(), job.ID, update.Status, message, nil); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// jobResult recibe multipart con `audio` (.ogg), `image` (opcional) y `meta` ({durationMs}).
// El id del sonido es el SHA-256 del audio; lo calcula la API, nunca el cliente.
func (s *Server) jobResult(w http.ResponseWriter, r *http.Request) error {
	body, err := s.verifiedBody(w, r, nil)
	if err != nil {
		return err
	}
	job, err := s.findJob(r)
	if err != nil {
		return err
	}
	form, err := parseMultipart(r.Header.Get("Content-Type"), body)
	if err != nil {
		return fail(http.StatusBadRequest, "Faltan audio o meta")
	}
	defer form.RemoveAll()

	var meta struct {
		DurationMs float64 `json:"durationMs"`
	}
	metaValues := form.Value["meta"]
	audioFiles := form.File["audio"]
	if len(audioFiles) == 0 || len(metaValues) == 0 || json.Unmarshal([]byte(metaValues[0]), &meta) != nil ||
		!validDuration(meta.DurationMs) {
		return fail(http.StatusBadRequest, "Faltan audio o meta")
	}
	audio, err := readPart(audioFiles[0])
	if err != nil {
		return err
	}

	ctx := r.Context()
	soundID := signature.SHA256Hex(audio)
	audioKey := "sounds/" + soundID + ".ogg"
	if err := s.Store.Put(ctx, audioKey, audio, storage.PutOptions{ContentType: audioContentType, CacheControl: immutableCache}); err != nil {
		return err
	}

	var imageKey, imageType *string
	if images := form.File["image"]; len(images) > 0 {
		contentType := images[0].Header.Get("Content-Type")
		ext, ok := imageContentTypes[contentType]
		if !ok {
			return fail(http.StatusBadRequest, fmt.Sprintf("Tipo de imagen no soportado: %s", contentType))
		}
		data, err := readPart(images[0])
		if err != nil {
			return err
		}
		key := "images/" + soundID + "." + ext
		if err := s.Store.Put(ctx, key, data, storage.PutOptions{ContentType: contentType, CacheControl: immutableCache}); err != nil {
			return err
		}
		imageKey, imageType = &key, &contentType
	}

	// Reprocesar el mismo audio actualiza título y duración; sin imagen nueva se conserva la anterior.
	var previousImage *string
	err = s.DB.QueryRowContext(ctx, "SELECT image_key FROM sounds WHERE id = ?", soundID).Scan(&previousImage)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if _, err := s.DB.ExecContext(ctx,
		`INSERT INTO sounds (id, title, duration_ms, audio_key, image_key, image_type) VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT (id) DO UPDATE SET title = excluded.title, duration_ms = excluded.duration_ms,
		   audio_key = excluded.audio_key,
		   image_key = COALESCE(excluded.image_key, image_key),
		   image_type = COALESCE(excluded.image_type, image_type)`,
		soundID, job.Title, int64(meta.DurationMs), audioKey, imageKey, imageType); err != nil {
		return err
	}
	if imageKey != nil && previousImage != nil && *previousImage != *imageKey {
		if err := s.Store.Delete(ctx, *previousImage); err != nil {
			return err
		}
	}

	if err := s.setJobStatus(ctx, job.ID, "done", nil, &soundID); err != nil {
		return err
	}
	var uploads []string
	for _, url := range []*string{&job.SourceURL, job.ImageURL} {
		if key, ok := uploadKey(url); ok {
			uploads = append(uploads, key)
		}
	}
	if err := s.Store.Delete(ctx, uploads...); err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, map[string]string{"soundId": soundID})
}

func parseMultipart(contentType string, body []byte) (*multipart.Form, error) {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType != "multipart/form-data" || params["boundary"] == "" {
		return nil, errors.New("no es multipart/form-data")
	}
	// Todo queda en memoria: el cuerpo ya está leído y limitado por maxProcessorResult.
	return multipart.NewReader(bytes.NewReader(body), params["boundary"]).ReadForm(int64(len(body)) + (1 << 20))
}

func readPart(header *multipart.FileHeader) ([]byte, error) {
	file, err := header.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(file)
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
