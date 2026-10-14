package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Validaciones equivalentes a los esquemas zod de @soundboard/shared.

var (
	soundIDPattern  = regexp.MustCompile(`^[0-9a-f]{64}$`)
	trimTimePattern = regexp.MustCompile(`^\d+(:\d{1,2}){0,2}(\.\d+)?$`)
)

const maxJSONBody = 1 << 20

// decodeJSON lee el cuerpo como JSON. Los campos desconocidos se ignoran, como hace zod.
func decodeJSON(r *http.Request, dst any) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxJSONBody))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, dst); err != nil {
		var typeErr *json.UnmarshalTypeError
		if errors.As(err, &typeErr) && typeErr.Field != "" {
			return fail(http.StatusBadRequest, fmt.Sprintf("Campo %s inválido", typeErr.Field))
		}
		return fail(http.StatusBadRequest, "JSON inválido")
	}
	return nil
}

type issues []string

func (i *issues) add(message string) { *i = append(*i, message) }

func (i issues) err() error {
	if len(i) == 0 {
		return nil
	}
	return fail(http.StatusBadRequest, strings.Join(i, "; "))
}

func validTitle(title string) bool {
	length := utf8.RuneCountInString(title)
	return length >= 1 && length <= 80
}

// Solo http(s) o archivos subidos al panel: evita que yt-dlp lea file:// u otros esquemas locales.
func validSourceURL(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return parsed.Host != ""
	case "upload":
		return parsed.Opaque != "" || parsed.Path != ""
	}
	return false
}

func soundIDParam(r *http.Request, name string) (string, error) {
	id := r.PathValue(name)
	if !soundIDPattern.MatchString(id) {
		return "", fail(http.StatusBadRequest, "Id de sonido inválido")
	}
	return id, nil
}

type createJobRequest struct {
	URL      *string `json:"url"`
	Title    *string `json:"title"`
	ImageURL *string `json:"imageUrl"`
	From     *string `json:"from"`
	To       *string `json:"to"`
	Stereo   *bool   `json:"stereo"`
}

func (req *createJobRequest) validate() error {
	var problems issues
	if req.URL == nil || !validSourceURL(*req.URL) {
		problems.add("url: usa un enlace http(s) o un archivo subido")
	}
	if req.Title != nil {
		trimmed := strings.TrimSpace(*req.Title)
		req.Title = &trimmed
	}
	if req.Title == nil || !validTitle(*req.Title) {
		problems.add("title: debe tener entre 1 y 80 caracteres")
	}
	if req.ImageURL != nil && !validSourceURL(*req.ImageURL) {
		problems.add("imageUrl: usa un enlace http(s) o un archivo subido")
	}
	for _, field := range []struct {
		name  string
		value *string
	}{{"from", req.From}, {"to", req.To}} {
		if field.value != nil && !trimTimePattern.MatchString(*field.value) {
			problems.add(field.name + ": usa segundos (3.5) o mm:ss (1:23)")
		}
	}
	return problems.err()
}

type updateSoundRequest struct {
	Title  *string `json:"title"`
	Status *string `json:"status"`
}

func (req *updateSoundRequest) validate() error {
	var problems issues
	if req.Title != nil {
		trimmed := strings.TrimSpace(*req.Title)
		req.Title = &trimmed
		if !validTitle(trimmed) {
			problems.add("title: debe tener entre 1 y 80 caracteres")
		}
	}
	if req.Status != nil && *req.Status != "published" && *req.Status != "hidden" {
		problems.add("status: debe ser published o hidden")
	}
	if req.Title == nil && req.Status == nil {
		problems.add("Nada que actualizar")
	}
	return problems.err()
}

type libraryOrderRequest struct {
	IDs *[]string `json:"ids"`
}

func (req libraryOrderRequest) validate() error {
	if req.IDs == nil {
		return fail(http.StatusBadRequest, "ids: es obligatorio")
	}
	if len(*req.IDs) > 500 {
		return fail(http.StatusBadRequest, "ids: máximo 500")
	}
	for _, id := range *req.IDs {
		if !soundIDPattern.MatchString(id) {
			return fail(http.StatusBadRequest, "Debe ser un SHA-256 en hexadecimal")
		}
	}
	return nil
}

type jobStatusUpdate struct {
	Status string  `json:"status"`
	Error  *string `json:"error"`
}

func (req jobStatusUpdate) validate() error {
	switch {
	case req.Status == "running":
		return nil
	case req.Status == "failed" && req.Error != nil && utf8.RuneCountInString(*req.Error) <= 2000:
		return nil
	}
	return fail(http.StatusBadRequest, "status: se espera running, o failed con error")
}

// durationMs debe ser un entero positivo.
func validDuration(value float64) bool {
	return value > 0 && value == math.Trunc(value) && value <= math.MaxInt32
}
