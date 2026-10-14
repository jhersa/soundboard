package httpapi

import "strings"

// DTOs de @soundboard/shared. Los campos opcionales son punteros para que se serialicen como
// null (no se omiten): la app Android los espera presentes.

type UserDTO struct {
	ID        string  `json:"id"`
	Email     string  `json:"email"`
	Name      *string `json:"name"`
	AvatarURL *string `json:"avatarUrl"`
	IsAdmin   bool    `json:"isAdmin"`
}

type SignInResponse struct {
	Token string  `json:"token"`
	User  UserDTO `json:"user"`
}

type SoundDTO struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	DurationMs int64   `json:"durationMs"`
	AudioURL   string  `json:"audioUrl"`
	ImageURL   *string `json:"imageUrl"`
	IsGif      bool    `json:"isGif"`
}

type CatalogSoundDTO struct {
	SoundDTO
	InLibrary bool `json:"inLibrary"`
}

type CatalogPage struct {
	Items      []CatalogSoundDTO `json:"items"`
	NextCursor *string           `json:"nextCursor"`
}

type LibraryDTO struct {
	Items []SoundDTO `json:"items"`
}

type AdminSoundDTO struct {
	SoundDTO
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
}

type JobDTO struct {
	ID        string  `json:"id"`
	SourceURL string  `json:"sourceUrl"`
	ImageURL  *string `json:"imageUrl"`
	Title     string  `json:"title"`
	From      *string `json:"from"`
	To        *string `json:"to"`
	Stereo    bool    `json:"stereo"`
	Status    string  `json:"status"`
	Error     *string `json:"error"`
	SoundID   *string `json:"soundId"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt string  `json:"updatedAt"`
}

// ProcessRequest es lo que la API manda al processor (POST /process). Los opcionales se omiten,
// igual que los `undefined` en JSON.stringify.
type ProcessRequest struct {
	JobID      string  `json:"jobId"`
	URL        string  `json:"url"`
	Title      string  `json:"title"`
	ImageURL   *string `json:"imageUrl,omitempty"`
	From       *string `json:"from,omitempty"`
	To         *string `json:"to,omitempty"`
	Stereo     bool    `json:"stereo"`
	APIBaseURL string  `json:"apiBaseUrl"`
}

// ---- Filas de la base ----

const (
	soundColumns       = "id, title, duration_ms, audio_key, image_key, image_type, status, created_at"
	prefixedSoundCols  = "s.id, s.title, s.duration_ms, s.audio_key, s.image_key, s.image_type, s.status, s.created_at"
	jobColumns         = "id, source_url, image_url, title, trim_from, trim_to, stereo, status, error, sound_id, created_at, updated_at"
	immutableCache     = "public, max-age=31536000, immutable"
	audioContentType   = "audio/ogg"
	uploadScheme       = "upload:"
	maxUploadBytes     = 50 << 20
	maxProcessorResult = 100 << 20
)

var imageContentTypes = map[string]string{
	"image/png":     "png",
	"image/jpeg":    "jpg",
	"image/gif":     "gif",
	"image/webp":    "webp",
	"image/svg+xml": "svg",
	"image/avif":    "avif",
}

type scanner interface{ Scan(dest ...any) error }

type soundRow struct {
	ID         string
	Title      string
	DurationMs int64
	AudioKey   string
	ImageKey   *string
	ImageType  *string
	Status     string
	CreatedAt  string
}

func scanSound(row scanner, extra ...any) (soundRow, error) {
	var s soundRow
	dest := append([]any{&s.ID, &s.Title, &s.DurationMs, &s.AudioKey, &s.ImageKey, &s.ImageType, &s.Status, &s.CreatedAt}, extra...)
	return s, row.Scan(dest...)
}

func mediaURL(base, key string) string { return strings.TrimRight(base, "/") + "/" + key }

func (row soundRow) dto(cdnBase string) SoundDTO {
	dto := SoundDTO{
		ID:         row.ID,
		Title:      row.Title,
		DurationMs: row.DurationMs,
		AudioURL:   mediaURL(cdnBase, row.AudioKey),
		IsGif:      row.ImageType != nil && *row.ImageType == "image/gif",
	}
	if row.ImageKey != nil {
		url := mediaURL(cdnBase, *row.ImageKey)
		dto.ImageURL = &url
	}
	return dto
}

func (row soundRow) adminDTO(cdnBase string) AdminSoundDTO {
	return AdminSoundDTO{SoundDTO: row.dto(cdnBase), Status: row.Status, CreatedAt: row.CreatedAt}
}

type jobRow struct {
	ID        string
	SourceURL string
	ImageURL  *string
	Title     string
	TrimFrom  *string
	TrimTo    *string
	Stereo    int
	Status    string
	Error     *string
	SoundID   *string
	CreatedAt string
	UpdatedAt string
}

func scanJob(row scanner) (jobRow, error) {
	var j jobRow
	return j, row.Scan(&j.ID, &j.SourceURL, &j.ImageURL, &j.Title, &j.TrimFrom, &j.TrimTo, &j.Stereo, &j.Status,
		&j.Error, &j.SoundID, &j.CreatedAt, &j.UpdatedAt)
}

func (j jobRow) dto() JobDTO {
	return JobDTO{
		ID: j.ID, SourceURL: j.SourceURL, ImageURL: j.ImageURL, Title: j.Title, From: j.TrimFrom, To: j.TrimTo,
		Stereo: j.Stereo == 1, Status: j.Status, Error: j.Error, SoundID: j.SoundID,
		CreatedAt: j.CreatedAt, UpdatedAt: j.UpdatedAt,
	}
}
