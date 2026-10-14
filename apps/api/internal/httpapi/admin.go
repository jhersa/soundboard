package httpapi

import (
	"database/sql"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"soundboard/api/internal/storage"
)

var extensionPattern = regexp.MustCompile(`(?i)\.([a-z0-9]{1,5})$`)

// upload es el respaldo para cuando yt-dlp no puede bajar un link: se sube el archivo y el job
// usa `upload:<key>`.
func (s *Server) upload(w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes+(1<<20))
	reader, err := r.MultipartReader()
	if err != nil {
		return fail(http.StatusBadRequest, "Se esperaba multipart/form-data")
	}
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			return fail(http.StatusBadRequest, "Falta el campo file")
		}
		if err != nil {
			return err
		}
		if part.FormName() != "file" || part.FileName() == "" {
			continue
		}
		data, err := io.ReadAll(io.LimitReader(part, maxUploadBytes+1))
		if err != nil {
			return err
		}
		if len(data) > maxUploadBytes {
			return fail(http.StatusRequestEntityTooLarge, "Archivo demasiado grande")
		}
		ext := "bin"
		if match := extensionPattern.FindStringSubmatch(part.FileName()); match != nil {
			ext = strings.ToLower(match[1])
		}
		key := "uploads/" + uuid.NewString() + "." + ext
		if err := s.Store.Put(r.Context(), key, data, storage.PutOptions{ContentType: part.Header.Get("Content-Type")}); err != nil {
			return err
		}
		return writeJSON(w, http.StatusCreated, map[string]string{"url": uploadScheme + key})
	}
}

func (s *Server) createJob(w http.ResponseWriter, r *http.Request) error {
	var request createJobRequest
	if err := decodeJSON(r, &request); err != nil {
		return err
	}
	if err := request.validate(); err != nil {
		return err
	}
	stereo := 0
	if request.Stereo != nil && *request.Stereo {
		stereo = 1
	}
	job, err := scanJob(s.DB.QueryRowContext(r.Context(),
		`INSERT INTO jobs (id, source_url, image_url, title, trim_from, trim_to, stereo)
		 VALUES (?, ?, ?, ?, ?, ?, ?) RETURNING `+jobColumns,
		uuid.NewString(), *request.URL, request.ImageURL, *request.Title, request.From, request.To, stereo))
	if err != nil {
		return err
	}
	s.dispatchJob(job, requestOrigin(r))
	return writeJSON(w, http.StatusAccepted, job.dto())
}

func (s *Server) listJobs(w http.ResponseWriter, r *http.Request) error {
	rows, err := s.DB.QueryContext(r.Context(), "SELECT "+jobColumns+" FROM jobs ORDER BY created_at DESC LIMIT 50")
	if err != nil {
		return err
	}
	defer rows.Close()
	items := []JobDTO{}
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return err
		}
		items = append(items, job.dto())
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) retryJob(w http.ResponseWriter, r *http.Request) error {
	var status string
	err := s.DB.QueryRowContext(r.Context(), "SELECT status FROM jobs WHERE id = ?", r.PathValue("id")).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return fail(http.StatusNotFound, "Job no encontrado")
	}
	if err != nil {
		return err
	}
	if status != "failed" {
		return fail(http.StatusConflict, "Solo se reintentan jobs fallidos")
	}
	job, err := scanJob(s.DB.QueryRowContext(r.Context(),
		`UPDATE jobs SET status = 'queued', error = NULL, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		 WHERE id = ? RETURNING `+jobColumns, r.PathValue("id")))
	if errors.Is(err, sql.ErrNoRows) {
		return fail(http.StatusNotFound, "Job no encontrado")
	}
	if err != nil {
		return err
	}
	s.dispatchJob(job, requestOrigin(r))
	return writeJSON(w, http.StatusAccepted, job.dto())
}

func (s *Server) listSounds(w http.ResponseWriter, r *http.Request) error {
	rows, err := s.DB.QueryContext(r.Context(),
		"SELECT "+prefixedSoundCols+" FROM sounds s ORDER BY s.created_at DESC, s.id DESC")
	if err != nil {
		return err
	}
	defer rows.Close()
	base := s.cdnBase(r)
	items := []AdminSoundDTO{}
	for rows.Next() {
		row, err := scanSound(rows)
		if err != nil {
			return err
		}
		items = append(items, row.adminDTO(base))
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) updateSound(w http.ResponseWriter, r *http.Request) error {
	id, err := soundIDParam(r, "id")
	if err != nil {
		return err
	}
	var request updateSoundRequest
	if err := decodeJSON(r, &request); err != nil {
		return err
	}
	if err := request.validate(); err != nil {
		return err
	}
	row, err := scanSound(s.DB.QueryRowContext(r.Context(),
		"UPDATE sounds SET title = COALESCE(?, title), status = COALESCE(?, status) WHERE id = ? RETURNING "+soundColumns,
		request.Title, request.Status, id))
	if errors.Is(err, sql.ErrNoRows) {
		return fail(http.StatusNotFound, "Sonido no encontrado")
	}
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, row.adminDTO(s.cdnBase(r)))
}

// deleteSound lo quita de todas las bibliotecas (ON DELETE CASCADE) y borra sus archivos.
func (s *Server) deleteSound(w http.ResponseWriter, r *http.Request) error {
	id, err := soundIDParam(r, "id")
	if err != nil {
		return err
	}
	var audioKey string
	var imageKey *string
	err = s.DB.QueryRowContext(r.Context(), "DELETE FROM sounds WHERE id = ? RETURNING audio_key, image_key", id).
		Scan(&audioKey, &imageKey)
	if errors.Is(err, sql.ErrNoRows) {
		return fail(http.StatusNotFound, "Sonido no encontrado")
	}
	if err != nil {
		return err
	}
	keys := []string{audioKey}
	if imageKey != nil {
		keys = append(keys, *imageKey)
	}
	if err := s.Store.Delete(r.Context(), keys...); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
