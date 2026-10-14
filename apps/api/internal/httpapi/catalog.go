package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"strings"
)

const (
	defaultCatalogLimit = 50
	maxCatalogLimit     = 100
)

// El cursor es btoa(JSON.stringify({createdAt, id})), compatible con los que emitía la API en TS.
type cursor struct {
	CreatedAt string `json:"createdAt"`
	ID        string `json:"id"`
}

func encodeCursor(c cursor) string {
	raw, _ := json.Marshal(c)
	return base64.StdEncoding.EncodeToString(raw)
}

func decodeCursor(value string) (cursor, error) {
	var c cursor
	raw, err := base64.StdEncoding.DecodeString(value)
	if err == nil {
		err = json.Unmarshal(raw, &c)
	}
	if err != nil || c.CreatedAt == "" || c.ID == "" {
		return c, fail(http.StatusBadRequest, "Cursor inválido")
	}
	return c, nil
}

// catalogLimit replica Math.min(Math.max(Number(limit) || 50, 1), 100).
func catalogLimit(value string) int {
	number, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || math.IsNaN(number) || number == 0 {
		number = defaultCatalogLimit
	}
	return int(math.Min(math.Max(number, 1), maxCatalogLimit))
}

func (s *Server) catalog(w http.ResponseWriter, r *http.Request) error {
	user, err := s.currentUser(r)
	if err != nil {
		return err
	}
	var userID any
	if user != nil {
		userID = user.ID
	}
	query := r.URL.Query()
	limit := catalogLimit(query.Get("limit"))

	conditions := []string{"s.status = 'published'"}
	params := []any{userID}
	if search := strings.TrimSpace(query.Get("q")); search != "" {
		conditions = append(conditions, `s.title LIKE ? ESCAPE '\'`)
		escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(search)
		params = append(params, "%"+escaped+"%")
	}
	if value := query.Get("cursor"); value != "" {
		c, err := decodeCursor(value)
		if err != nil {
			return err
		}
		conditions = append(conditions, "(s.created_at < ? OR (s.created_at = ? AND s.id < ?))")
		params = append(params, c.CreatedAt, c.CreatedAt, c.ID)
	}
	params = append(params, limit+1)

	// Más nuevos primero; keyset pagination para que agregar sonidos no desplace páginas.
	rows, err := s.DB.QueryContext(r.Context(),
		`SELECT `+prefixedSoundCols+`, us.sound_id IS NOT NULL AS in_library
		 FROM sounds s LEFT JOIN user_sounds us ON us.sound_id = s.id AND us.user_id = ?
		 WHERE `+strings.Join(conditions, " AND ")+`
		 ORDER BY s.created_at DESC, s.id DESC
		 LIMIT ?`, params...)
	if err != nil {
		return err
	}
	defer rows.Close()

	base := s.cdnBase(r)
	page := CatalogPage{Items: []CatalogSoundDTO{}}
	var last soundRow
	hasMore := false
	for rows.Next() {
		var inLibrary bool
		row, err := scanSound(rows, &inLibrary)
		if err != nil {
			return err
		}
		if len(page.Items) == limit {
			hasMore = true
			break
		}
		page.Items = append(page.Items, CatalogSoundDTO{SoundDTO: row.dto(base), InLibrary: inLibrary})
		last = row
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if hasMore {
		next := encodeCursor(cursor{CreatedAt: last.CreatedAt, ID: last.ID})
		page.NextCursor = &next
	}
	return writeJSON(w, http.StatusOK, page)
}
