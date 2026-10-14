package httpapi

import (
	"net/http"
	"net/url"
	"slices"
	"testing"
)

func titles[T any](items []T, title func(T) string) []string {
	var result []string
	for _, item := range items {
		result = append(result, title(item))
	}
	return result
}

func catalogTitles(page CatalogPage) []string {
	return titles(page.Items, func(item CatalogSoundDTO) string { return item.Title })
}

func TestCatalogoPublicadosBusquedaYPaginacion(t *testing.T) {
	api := newTestAPI(t)
	api.seedSound(soundID("a"), "uwu", "", "2026-01-01T00:00:00.000Z")
	api.seedSound(soundID("b"), "money money", "", "2026-01-02T00:00:00.000Z")
	api.seedSound(soundID("c"), "oculto", "hidden", "")

	var page1 CatalogPage
	api.call("GET", "/v1/catalog?limit=1", nil, "").decode(t, &page1)
	if !slices.Equal(catalogTitles(page1), []string{"money money"}) {
		t.Fatalf("página 1: %v", catalogTitles(page1))
	}
	item := page1.Items[0]
	if item.AudioURL != "https://cdn.test/sounds/"+soundID("b")+".ogg" ||
		*item.ImageURL != "https://cdn.test/images/"+soundID("b")+".gif" || !item.IsGif || item.InLibrary {
		t.Fatalf("item: %+v", item)
	}
	var page2 CatalogPage
	api.call("GET", "/v1/catalog?limit=1&cursor="+url.QueryEscape(*page1.NextCursor), nil, "").decode(t, &page2)
	if !slices.Equal(catalogTitles(page2), []string{"uwu"}) || page2.NextCursor != nil {
		t.Fatalf("página 2: %v %v", catalogTitles(page2), page2.NextCursor)
	}
	var search CatalogPage
	api.call("GET", "/v1/catalog?q=UWU", nil, "").decode(t, &search)
	if len(search.Items) != 1 || search.Items[0].ID != soundID("a") {
		t.Fatalf("búsqueda: %v", catalogTitles(search))
	}
	if res := api.call("GET", "/v1/catalog?cursor=basura", nil, ""); res.Code != http.StatusBadRequest {
		t.Fatalf("cursor inválido: %d", res.Code)
	}
}

// Los cursores que emitía la API en TS (btoa de JSON) siguen funcionando.
func TestCatalogoCursorCompatible(t *testing.T) {
	api := newTestAPI(t)
	api.seedSound(soundID("a"), "viejo", "", "2026-01-01T00:00:00.000Z")
	api.seedSound(soundID("b"), "nuevo", "", "2026-01-02T00:00:00.000Z")
	tsCursor := "eyJjcmVhdGVkQXQiOiIyMDI2LTAxLTAyVDAwOjAwOjAwLjAwMFoiLCJpZCI6ImJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmIifQ=="
	var page CatalogPage
	api.call("GET", "/v1/catalog?cursor="+tsCursor, nil, "").decode(t, &page)
	if !slices.Equal(catalogTitles(page), []string{"viejo"}) {
		t.Fatalf("cursor de TS: %v", catalogTitles(page))
	}
	if encodeCursor(cursor{CreatedAt: "2026-01-02T00:00:00.000Z", ID: soundID("b")}) != tsCursor {
		t.Fatal("el cursor de Go no coincide con el de TS")
	}
}

func TestBibliotecaAgregaOrdenaYQuita(t *testing.T) {
	api := newTestAPI(t)
	for _, char := range []string{"a", "b", "c"} {
		api.seedSound(soundID(char), char, "", "")
	}
	token := api.signIn("sub-lib", "lib@example.com").Token
	if res := api.call("GET", "/v1/me/library", nil, token); res.Code != http.StatusOK || res.Body.String() != "{\"items\":[]}\n" {
		t.Fatalf("biblioteca vacía: %d %s", res.Code, res.Body.String())
	}
	for _, char := range []string{"a", "b", "c"} {
		if res := api.call("PUT", "/v1/me/library/"+soundID(char), nil, token); res.Code != http.StatusOK {
			t.Fatalf("agregar %s: %d", char, res.Code)
		}
	}
	api.call("PUT", "/v1/me/library/"+soundID("a"), nil, token) // Idempotente.

	var reordered LibraryDTO
	api.call("PUT", "/v1/me/library/order", map[string][]string{"ids": {soundID("c"), soundID("a")}}, token).decode(t, &reordered)
	soundTitle := func(item SoundDTO) string { return item.Title }
	if got := titles(reordered.Items, soundTitle); !slices.Equal(got, []string{"c", "a", "b"}) {
		t.Fatalf("orden: %v", got)
	}
	var catalog CatalogPage
	api.call("GET", "/v1/catalog", nil, token).decode(t, &catalog)
	for _, item := range catalog.Items {
		if !item.InLibrary {
			t.Fatalf("%s debería estar en la biblioteca", item.Title)
		}
	}
	var removed LibraryDTO
	api.call("DELETE", "/v1/me/library/"+soundID("a"), nil, token).decode(t, &removed)
	if got := titles(removed.Items, soundTitle); !slices.Equal(got, []string{"c", "b"}) {
		t.Fatalf("tras quitar: %v", got)
	}
}

func TestBibliotecaValidaSesionIdsYExistencia(t *testing.T) {
	api := newTestAPI(t)
	if res := api.call("GET", "/v1/me/library", nil, ""); res.Code != http.StatusUnauthorized {
		t.Fatalf("sin sesión: %d", res.Code)
	}
	token := api.signIn("sub-val", "val@example.com").Token
	if res := api.call("PUT", "/v1/me/library/nope", nil, token); res.Code != http.StatusBadRequest {
		t.Fatalf("id inválido: %d", res.Code)
	}
	if res := api.call("PUT", "/v1/me/library/"+soundID("f"), nil, token); res.Code != http.StatusNotFound {
		t.Fatalf("inexistente: %d", res.Code)
	}
}
