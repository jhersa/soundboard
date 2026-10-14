package httpapi

import (
	"net/http"
	"testing"
)

func TestAuthCreaUsuarioYReconoceAdmins(t *testing.T) {
	api := newTestAPI(t)
	signIn := api.signIn("sub-1", "Admin@Example.com")
	if signIn.User.Email != "admin@example.com" || !signIn.User.IsAdmin || *signIn.User.Name != "Usuario sub-1" {
		t.Fatalf("usuario inesperado: %+v", signIn.User)
	}
	res := api.call("GET", "/v1/me", nil, signIn.Token)
	var me UserDTO
	res.decode(t, &me)
	if res.Code != http.StatusOK || me.ID != signIn.User.ID || !me.IsAdmin {
		t.Fatalf("/me: %d %+v", res.Code, me)
	}
}

func TestAuthReutilizaUsuarioPorGoogleSub(t *testing.T) {
	api := newTestAPI(t)
	first := api.signIn("sub-2", "a@example.com")
	second := api.signIn("sub-2", "nuevo@example.com")
	if second.User.ID != first.User.ID || second.User.Email != "nuevo@example.com" || second.User.IsAdmin {
		t.Fatalf("segundo login: %+v", second.User)
	}
}

func TestAuthRechazaTokensInvalidosYSesionesCerradas(t *testing.T) {
	api := newTestAPI(t)
	if res := api.call("POST", "/v1/auth/google", `{"idToken":"x"}`, ""); res.Code != http.StatusUnauthorized {
		t.Fatalf("token inválido: %d", res.Code)
	}
	if res := api.call("GET", "/v1/me", nil, ""); res.Code != http.StatusUnauthorized {
		t.Fatalf("sin sesión: %d", res.Code)
	}
	if res := api.call("POST", "/v1/auth/google", `{"idToken":`, ""); res.Code != http.StatusBadRequest ||
		res.Body.String() != "{\"error\":\"JSON inválido\"}\n" {
		t.Fatalf("JSON roto: %d %s", res.Code, res.Body.String())
	}
	token := api.signIn("sub-3", "b@example.com").Token
	if res := api.call("POST", "/v1/auth/logout", nil, token); res.Code != http.StatusNoContent {
		t.Fatalf("logout: %d", res.Code)
	}
	if res := api.call("GET", "/v1/me", nil, token); res.Code != http.StatusUnauthorized {
		t.Fatalf("sesión cerrada: %d", res.Code)
	}
}

func TestCORS(t *testing.T) {
	api := newTestAPI(t)
	res := api.call("OPTIONS", "/v1/admin/jobs", nil, "", "Origin", "https://admin.test", "Access-Control-Request-Method", "POST")
	if res.Code != http.StatusNoContent || res.Header().Get("Access-Control-Allow-Origin") != "https://admin.test" ||
		res.Header().Get("Access-Control-Allow-Headers") != "authorization,content-type" {
		t.Fatalf("preflight: %d %v", res.Code, res.Header())
	}
	if res := api.call("GET", "/", nil, ""); res.Body.String() != "soundboard-api" || res.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("raíz: %q %v", res.Body.String(), res.Header())
	}
}
