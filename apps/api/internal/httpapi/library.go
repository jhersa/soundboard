package httpapi

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
)

// loadLibrary: los sonidos que el admin oculta desaparecen de las bibliotecas sin borrar la relación.
func (s *Server) loadLibrary(ctx context.Context, r *http.Request, userID string) (LibraryDTO, error) {
	library := LibraryDTO{Items: []SoundDTO{}}
	rows, err := s.DB.QueryContext(ctx,
		`SELECT `+prefixedSoundCols+` FROM user_sounds us JOIN sounds s ON s.id = us.sound_id
		 WHERE us.user_id = ? AND s.status = 'published'
		 ORDER BY us.position, us.added_at`, userID)
	if err != nil {
		return library, err
	}
	defer rows.Close()
	base := s.cdnBase(r)
	for rows.Next() {
		row, err := scanSound(rows)
		if err != nil {
			return library, err
		}
		library.Items = append(library.Items, row.dto(base))
	}
	return library, rows.Err()
}

func (s *Server) respondLibrary(w http.ResponseWriter, r *http.Request, userID string) error {
	library, err := s.loadLibrary(r.Context(), r, userID)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, library)
}

func (s *Server) getLibrary(w http.ResponseWriter, r *http.Request) error {
	user, err := s.requireUser(r)
	if err != nil {
		return err
	}
	return s.respondLibrary(w, r, user.ID)
}

// orderLibrary: los ids enviados van primero en ese orden; los que falten conservan su orden
// relativo al final.
func (s *Server) orderLibrary(w http.ResponseWriter, r *http.Request) error {
	user, err := s.requireUser(r)
	if err != nil {
		return err
	}
	var request libraryOrderRequest
	if err := decodeJSON(r, &request); err != nil {
		return err
	}
	if err := request.validate(); err != nil {
		return err
	}

	rows, err := s.DB.QueryContext(r.Context(),
		"SELECT sound_id FROM user_sounds WHERE user_id = ? ORDER BY position, added_at", user.ID)
	if err != nil {
		return err
	}
	var current []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		current = append(current, id)
	}
	rows.Close()

	owned := map[string]bool{}
	for _, id := range current {
		owned[id] = true
	}
	listed := map[string]bool{}
	var ordered []string
	for _, id := range *request.IDs {
		if owned[id] && !listed[id] {
			listed[id] = true
			ordered = append(ordered, id)
		}
	}
	for _, id := range current {
		if !listed[id] {
			ordered = append(ordered, id)
		}
	}

	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for position, id := range ordered {
		if _, err := tx.ExecContext(r.Context(),
			"UPDATE user_sounds SET position = ? WHERE user_id = ? AND sound_id = ?", position, user.ID, id); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return s.respondLibrary(w, r, user.ID)
}

func (s *Server) addToLibrary(w http.ResponseWriter, r *http.Request) error {
	user, err := s.requireUser(r)
	if err != nil {
		return err
	}
	soundID, err := soundIDParam(r, "soundId")
	if err != nil {
		return err
	}
	var found string
	err = s.DB.QueryRowContext(r.Context(), "SELECT id FROM sounds WHERE id = ? AND status = 'published'", soundID).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return fail(http.StatusNotFound, "Sonido no encontrado")
	}
	if err != nil {
		return err
	}
	// Idempotente: agregar dos veces no cambia la posición original.
	if _, err := s.DB.ExecContext(r.Context(),
		`INSERT INTO user_sounds (user_id, sound_id, position)
		 VALUES (?1, ?2, (SELECT COALESCE(MAX(position) + 1, 0) FROM user_sounds WHERE user_id = ?1))
		 ON CONFLICT (user_id, sound_id) DO NOTHING`, user.ID, soundID); err != nil {
		return err
	}
	return s.respondLibrary(w, r, user.ID)
}

func (s *Server) removeFromLibrary(w http.ResponseWriter, r *http.Request) error {
	user, err := s.requireUser(r)
	if err != nil {
		return err
	}
	soundID, err := soundIDParam(r, "soundId")
	if err != nil {
		return err
	}
	if _, err := s.DB.ExecContext(r.Context(),
		"DELETE FROM user_sounds WHERE user_id = ? AND sound_id = ?", user.ID, soundID); err != nil {
		return err
	}
	return s.respondLibrary(w, r, user.ID)
}
