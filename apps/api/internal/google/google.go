// Package google verifica el ID token de Google con el que inician sesión la app y el panel.
package google

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/coreos/go-oidc/v3/oidc"
)

type Identity struct {
	Sub     string
	Email   string
	Name    *string
	Picture *string
}

type Verifier interface {
	Verify(ctx context.Context, idToken string, audiences []string) (Identity, error)
}

var issuers = []string{"https://accounts.google.com", "accounts.google.com"}

// TokenVerifier valida firma (JWKS de Google, en caché), expiración, emisor, audiencia y
// email_verified.
type TokenVerifier struct{ verifier *oidc.IDTokenVerifier }

func NewVerifier(ctx context.Context) *TokenVerifier {
	keys := oidc.NewRemoteKeySet(ctx, "https://www.googleapis.com/oauth2/v3/certs")
	// El emisor y la audiencia se validan a mano: Google usa dos emisores y aceptamos varios client IDs.
	config := &oidc.Config{SkipClientIDCheck: true, SkipIssuerCheck: true}
	return &TokenVerifier{verifier: oidc.NewVerifier(issuers[0], keys, config)}
}

func (v *TokenVerifier) Verify(ctx context.Context, idToken string, audiences []string) (Identity, error) {
	if len(audiences) == 0 {
		return Identity{}, errors.New("GOOGLE_CLIENT_IDS no está configurado")
	}
	token, err := v.verifier.Verify(ctx, idToken)
	if err != nil {
		return Identity{}, err
	}
	if !slices.Contains(issuers, token.Issuer) {
		return Identity{}, fmt.Errorf("emisor inesperado: %s", token.Issuer)
	}
	if !slices.ContainsFunc(token.Audience, func(audience string) bool { return slices.Contains(audiences, audience) }) {
		return Identity{}, fmt.Errorf("audiencia inesperada: %v", token.Audience)
	}
	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
	}
	if err := token.Claims(&claims); err != nil {
		return Identity{}, err
	}
	if token.Subject == "" || claims.Email == "" {
		return Identity{}, errors.New("token sin sub/email")
	}
	if !claims.EmailVerified {
		return Identity{}, errors.New("el correo de Google no está verificado")
	}
	return Identity{Sub: token.Subject, Email: claims.Email, Name: optional(claims.Name), Picture: optional(claims.Picture)}, nil
}

func optional(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
