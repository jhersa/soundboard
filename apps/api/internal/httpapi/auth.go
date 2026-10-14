package httpapi

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"soundboard/api/internal/db"
	signature "soundboard/gosig"
)

const sessionTTL = 90 * 24 * time.Hour

type sessionUser struct {
	UserDTO
	SessionID string
}

func (s *Server) isAdminEmail(email string) bool {
	for _, admin := range s.Config.AdminEmails {
		if strings.EqualFold(admin, email) {
			return true
		}
	}
	return false
}

// createSession devuelve un token opaco; en la base solo se guarda su hash, así una fuga de la
// base no permite suplantar sesiones.
func (s *Server) createSession(ctx context.Context, userID string) (string, error) {
	raw := make([]byte, 32)
	rand.Read(raw)
	token := base64.RawURLEncoding.EncodeToString(raw)
	expiresAt := s.now().Add(sessionTTL).UTC().Format(db.TimeFormat)
	_, err := s.DB.ExecContext(ctx, "INSERT INTO sessions (id, user_id, token_hash, expires_at) VALUES (?, ?, ?, ?)",
		uuid.NewString(), userID, signature.SHA256Hex([]byte(token)), expiresAt)
	return token, err
}

// currentUser resuelve el Bearer token; devuelve nil si no hay sesión válida.
func (s *Server) currentUser(r *http.Request) (*sessionUser, error) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return nil, nil
	}
	token := strings.TrimSpace(header[len("Bearer "):])
	if token == "" {
		return nil, nil
	}
	var user sessionUser
	err := s.DB.QueryRowContext(r.Context(),
		`SELECT s.id, u.id, u.email, u.name, u.avatar_url
		 FROM sessions s JOIN users u ON u.id = s.user_id
		 WHERE s.token_hash = ? AND s.expires_at > ?`,
		signature.SHA256Hex([]byte(token)), s.now().UTC().Format(db.TimeFormat),
	).Scan(&user.SessionID, &user.ID, &user.Email, &user.Name, &user.AvatarURL)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	user.IsAdmin = s.isAdminEmail(user.Email)
	return &user, nil
}

func (s *Server) requireUser(r *http.Request) (*sessionUser, error) {
	user, err := s.currentUser(r)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fail(http.StatusUnauthorized, "Inicia sesión")
	}
	return user, nil
}

// admin envuelve un handler que exige sesión de administrador.
func (s *Server) admin(next handlerFunc) handlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		user, err := s.requireUser(r)
		if err != nil {
			return err
		}
		if !user.IsAdmin {
			return fail(http.StatusForbidden, "Solo administradores")
		}
		return next(w, r)
	}
}

func (s *Server) signIn(w http.ResponseWriter, r *http.Request) error {
	var request struct {
		IDToken string `json:"idToken"`
	}
	if err := decodeJSON(r, &request); err != nil {
		return err
	}
	if request.IDToken == "" {
		return fail(http.StatusBadRequest, "idToken: es obligatorio")
	}
	identity, err := s.Google.Verify(r.Context(), request.IDToken, s.Config.GoogleClientIDs)
	if err != nil {
		s.Log.Warn("Google ID token rechazado", "err", err)
		return fail(http.StatusUnauthorized, "Token de Google inválido")
	}

	// google_sub es el identificador estable; el correo, nombre y foto se refrescan en cada login.
	var user UserDTO
	err = s.DB.QueryRowContext(r.Context(),
		`INSERT INTO users (id, google_sub, email, name, avatar_url) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT (google_sub) DO UPDATE SET email = excluded.email, name = excluded.name, avatar_url = excluded.avatar_url
		 RETURNING id, email, name, avatar_url`,
		uuid.NewString(), identity.Sub, strings.ToLower(identity.Email), identity.Name, identity.Picture,
	).Scan(&user.ID, &user.Email, &user.Name, &user.AvatarURL)
	if err != nil {
		return err
	}
	user.IsAdmin = s.isAdminEmail(user.Email)
	token, err := s.createSession(r.Context(), user.ID)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, SignInResponse{Token: token, User: user})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) error {
	user, err := s.requireUser(r)
	if err != nil {
		return err
	}
	if _, err := s.DB.ExecContext(r.Context(), "DELETE FROM sessions WHERE id = ?", user.SessionID); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) error {
	user, err := s.requireUser(r)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, user.UserDTO)
}
