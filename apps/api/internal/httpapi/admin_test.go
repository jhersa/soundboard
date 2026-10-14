package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"soundboard/api/internal/storage"
	signature "soundboard/gosig"
)

func TestAdminSoloAdmins(t *testing.T) {
	api := newTestAPI(t)
	token := api.signIn("user", "user@example.com").Token
	if res := api.call("GET", "/v1/admin/jobs", nil, token); res.Code != http.StatusForbidden {
		t.Fatalf("usuario normal: %d", res.Code)
	}
	if res := api.call("GET", "/v1/admin/jobs", nil, ""); res.Code != http.StatusUnauthorized {
		t.Fatalf("sin sesión: %d", res.Code)
	}
}

func (a *testAPI) jobs(token string) []JobDTO {
	var list struct{ Items []JobDTO }
	a.call("GET", "/v1/admin/jobs", nil, token).decode(a.t, &list)
	return list.Items
}

func TestAdminJobCompleto(t *testing.T) {
	api := newTestAPI(t)
	token := api.adminToken()

	created := api.call("POST", "/v1/admin/jobs", `{"url":"https://youtu.be/x","title":"  Nuevo  ","from":"1:02"}`, token)
	if created.Code != http.StatusAccepted {
		t.Fatalf("crear job: %d %s", created.Code, created.Body.String())
	}
	var job JobDTO
	created.decode(t, &job)
	if job.Status != "queued" || *job.From != "1:02" || job.Stereo || job.Title != "Nuevo" || job.To != nil {
		t.Fatalf("job: %+v", job)
	}

	// El job llega firmado al processor, sin los opcionales vacíos.
	request, body := api.processor.requests[0], api.processor.bodies[0]
	if request.URL.String() != "https://processor.test/process" {
		t.Fatalf("URL del processor: %s", request.URL)
	}
	var sent map[string]any
	json.Unmarshal(body, &sent)
	if sent["jobId"] != job.ID || sent["apiBaseUrl"] != "https://api.test" || sent["from"] != "1:02" {
		t.Fatalf("cuerpo enviado: %s", body)
	}
	if _, hasTo := sent["to"]; hasTo {
		t.Fatalf("`to` vacío no debe enviarse: %s", body)
	}
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(request.Header.Get(signature.HeaderSignature)) ||
		!signature.Verify("test-secret", body, request.Header.Get(signature.HeaderTimestamp),
			request.Header.Get(signature.HeaderSignature), api.server.now()) {
		t.Fatal("firma inválida hacia el processor")
	}

	if res := api.signedCall("/v1/internal/jobs/"+job.ID+"/status", []byte(`{"status":"running"}`), "application/json"); res.Code != http.StatusNoContent {
		t.Fatalf("status: %d %s", res.Code, res.Body.String())
	}

	audio := []byte{1, 2, 3, 4}
	body, contentType := multipartBody(t, map[string]string{"meta": `{"durationMs":1500}`},
		filePart{"audio", "audio.ogg", "audio/ogg", audio}, filePart{"image", "image.gif", "image/gif", []byte{9}})
	result := api.signedCall("/v1/internal/jobs/"+job.ID+"/result", body, contentType)
	if result.Code != http.StatusOK {
		t.Fatalf("result: %d %s", result.Code, result.Body.String())
	}
	var done struct{ SoundID string }
	result.decode(t, &done)
	if done.SoundID != signature.SHA256Hex(audio) {
		t.Fatalf("soundId: %s", done.SoundID)
	}

	stored, _ := api.store.Get(context.Background(), "sounds/"+done.SoundID+".ogg")
	stored.Body.Close()
	if !strings.Contains(stored.CacheControl, "immutable") || stored.ContentType != "audio/ogg" {
		t.Fatalf("audio guardado: %+v", stored)
	}
	if !api.exists("images/" + done.SoundID + ".gif") {
		t.Fatal("falta la imagen")
	}
	if jobs := api.jobs(token); jobs[0].Status != "done" || *jobs[0].SoundID != done.SoundID || jobs[0].Error != nil {
		t.Fatalf("job terminado: %+v", jobs[0])
	}
	var catalog CatalogPage
	api.call("GET", "/v1/catalog", nil, "").decode(t, &catalog)
	if len(catalog.Items) != 1 || catalog.Items[0].Title != "Nuevo" || catalog.Items[0].DurationMs != 1500 {
		t.Fatalf("catálogo: %+v", catalog.Items)
	}
}

func TestAdminProcessorCaidoYReintento(t *testing.T) {
	api := newTestAPI(t)
	api.processor.status = 0
	token := api.adminToken()
	var job JobDTO
	api.call("POST", "/v1/admin/jobs", `{"url":"https://a.test/x","title":"X"}`, token).decode(t, &job)
	jobs := api.jobs(token)
	if jobs[0].ID != job.ID || jobs[0].Status != "failed" || !strings.HasPrefix(*jobs[0].Error, "No se pudo contactar al processor") {
		t.Fatalf("job fallido: %+v", jobs[0])
	}

	api.processor.status = http.StatusInternalServerError
	api.call("POST", "/v1/admin/jobs/"+job.ID+"/retry", nil, token)
	if jobs := api.jobs(token); jobs[0].Status != "failed" {
		t.Fatalf("un 500 del processor también falla el job: %+v", jobs[0])
	}

	api.processor.status = http.StatusAccepted
	if res := api.call("POST", "/v1/admin/jobs/"+job.ID+"/retry", nil, token); res.Code != http.StatusAccepted {
		t.Fatalf("reintento: %d", res.Code)
	}
	if res := api.call("POST", "/v1/admin/jobs/"+job.ID+"/retry", nil, token); res.Code != http.StatusConflict {
		t.Fatalf("reintentar uno en cola: %d", res.Code)
	}
	if res := api.call("POST", "/v1/admin/jobs/no-existe/retry", nil, token); res.Code != http.StatusNotFound {
		t.Fatalf("job inexistente: %d", res.Code)
	}
}

func TestAdminValidaJobs(t *testing.T) {
	api := newTestAPI(t)
	token := api.adminToken()
	for name, body := range map[string]string{
		"file://":      `{"url":"file:///etc/passwd","title":"X"}`,
		"sin título":   `{"url":"https://a.test/x","title":"   "}`,
		"título largo": `{"url":"https://a.test/x","title":"` + strings.Repeat("a", 81) + `"}`,
		"tiempo raro":  `{"url":"https://a.test/x","title":"X","to":"1m"}`,
		"tipo erróneo": `{"url":"https://a.test/x","title":5}`,
		"imagen local": `{"url":"https://a.test/x","title":"X","imageUrl":"file:///x.png"}`,
	} {
		if res := api.call("POST", "/v1/admin/jobs", body, token); res.Code != http.StatusBadRequest {
			t.Errorf("%s: %d %s", name, res.Code, res.Body.String())
		}
	}
	if res := api.call("POST", "/v1/admin/jobs", `{"url":"upload:uploads/a.mp3","title":"X","stereo":true}`, token); res.Code != http.StatusAccepted {
		t.Fatalf("upload: debe aceptarse: %d %s", res.Code, res.Body.String())
	}
}

func TestInternalRechazaFirmasInvalidas(t *testing.T) {
	api := newTestAPI(t)
	res := api.call("POST", "/v1/internal/jobs/x/status", `{"status":"running"}`, "",
		signature.HeaderTimestamp, strconv.FormatInt(api.server.now().Unix(), 10), signature.HeaderSignature, strings.Repeat("0", 64))
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("firma inválida: %d", res.Code)
	}
}

func TestAdminOcultaRenombraYBorra(t *testing.T) {
	api := newTestAPI(t)
	token := api.adminToken()
	id := soundID("d")
	api.seedSound(id, "viejo", "", "")
	api.store.Put(context.Background(), "sounds/"+id+".ogg", []byte("x"), storage.PutOptions{})

	var patched AdminSoundDTO
	api.call("PATCH", "/v1/admin/sounds/"+id, `{"title":"nuevo","status":"hidden"}`, token).decode(t, &patched)
	if patched.Title != "nuevo" || patched.Status != "hidden" {
		t.Fatalf("patch: %+v", patched)
	}
	if res := api.call("PATCH", "/v1/admin/sounds/"+id, `{}`, token); res.Code != http.StatusBadRequest {
		t.Fatalf("patch vacío: %d", res.Code)
	}
	var catalog CatalogPage
	api.call("GET", "/v1/catalog", nil, "").decode(t, &catalog)
	if len(catalog.Items) != 0 {
		t.Fatalf("oculto visible en el catálogo: %v", catalog.Items)
	}
	var sounds struct{ Items []AdminSoundDTO }
	api.call("GET", "/v1/admin/sounds", nil, token).decode(t, &sounds)
	if len(sounds.Items) != 1 || sounds.Items[0].Status != "hidden" {
		t.Fatalf("admin ve los ocultos: %+v", sounds.Items)
	}

	if res := api.call("DELETE", "/v1/admin/sounds/"+id, nil, token); res.Code != http.StatusNoContent {
		t.Fatalf("delete: %d", res.Code)
	}
	if api.exists("sounds/" + id + ".ogg") {
		t.Fatal("el audio debió borrarse")
	}
	if res := api.call("DELETE", "/v1/admin/sounds/"+id, nil, token); res.Code != http.StatusNotFound {
		t.Fatalf("delete repetido: %d", res.Code)
	}
}

// Subida desde el panel → el processor la baja firmada → al terminar el job se borra.
func TestUploadsDelPanel(t *testing.T) {
	api := newTestAPI(t)
	token := api.adminToken()
	body, contentType := multipartBody(t, nil, filePart{"file", "Clip.MP3", "audio/mpeg", []byte("mp3")})
	res := api.call("POST", "/v1/admin/uploads", body, token, "Content-Type", contentType)
	if res.Code != http.StatusCreated {
		t.Fatalf("upload: %d %s", res.Code, res.Body.String())
	}
	var uploaded struct{ URL string }
	res.decode(t, &uploaded)
	if !regexp.MustCompile(`^upload:uploads/[0-9a-f-]{36}\.mp3$`).MatchString(uploaded.URL) {
		t.Fatalf("url: %s", uploaded.URL)
	}
	key := strings.TrimPrefix(uploaded.URL, "upload:")

	headers := []string{}
	for header, value := range signature.Headers("test-secret", []byte(key), api.server.now()) {
		headers = append(headers, header, value)
	}
	download := api.call("GET", "/v1/internal/uploads/"+key, nil, "", headers...)
	if download.Code != http.StatusOK || download.Body.String() != "mp3" || download.Header().Get("Content-Type") != "audio/mpeg" {
		t.Fatalf("descarga firmada: %d %q", download.Code, download.Body.String())
	}
	if res := api.call("GET", "/v1/internal/uploads/"+key, nil, ""); res.Code != http.StatusUnauthorized {
		t.Fatalf("descarga sin firma: %d", res.Code)
	}

	var job JobDTO
	api.call("POST", "/v1/admin/jobs", map[string]string{"url": uploaded.URL, "title": "Subido"}, token).decode(t, &job)
	result, resultType := multipartBody(t, map[string]string{"meta": `{"durationMs":10}`}, filePart{"audio", "a.ogg", "audio/ogg", []byte("ogg")})
	if res := api.signedCall("/v1/internal/jobs/"+job.ID+"/result", result, resultType); res.Code != http.StatusOK {
		t.Fatalf("result: %d %s", res.Code, res.Body.String())
	}
	if api.exists(key) {
		t.Fatal("el upload debió borrarse al terminar el job")
	}

	if res := api.call("POST", "/v1/admin/uploads", `{}`, token); res.Code != http.StatusBadRequest {
		t.Fatalf("sin multipart: %d", res.Code)
	}
}

// Reprocesar el mismo audio actualiza título; sin imagen nueva conserva la anterior y con una
// imagen distinta borra la vieja.
func TestReprocesarMismoAudio(t *testing.T) {
	api := newTestAPI(t)
	token := api.adminToken()
	audio := []byte("mismo audio")
	process := func(title string, image *filePart) {
		var job JobDTO
		api.call("POST", "/v1/admin/jobs", map[string]string{"url": "https://a.test/x", "title": title}, token).decode(t, &job)
		files := []filePart{{"audio", "a.ogg", "audio/ogg", audio}}
		if image != nil {
			files = append(files, *image)
		}
		body, contentType := multipartBody(t, map[string]string{"meta": `{"durationMs":5}`}, files...)
		if res := api.signedCall("/v1/internal/jobs/"+job.ID+"/result", body, contentType); res.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", title, res.Code, res.Body.String())
		}
	}
	id := signature.SHA256Hex(audio)
	process("uno", &filePart{"image", "a.gif", "image/gif", []byte("gif")})
	process("dos", nil)
	var sounds struct{ Items []AdminSoundDTO }
	api.call("GET", "/v1/admin/sounds", nil, token).decode(t, &sounds)
	if len(sounds.Items) != 1 || sounds.Items[0].Title != "dos" || sounds.Items[0].ImageURL == nil || !sounds.Items[0].IsGif {
		t.Fatalf("tras reprocesar sin imagen: %+v", sounds.Items)
	}
	process("tres", &filePart{"image", "a.png", "image/png", []byte("png")})
	if api.exists("images/"+id+".gif") || !api.exists("images/"+id+".png") {
		t.Fatal("la imagen vieja debió reemplazarse")
	}

	var job JobDTO
	api.call("POST", "/v1/admin/jobs", map[string]string{"url": "https://a.test/x", "title": "x"}, token).decode(t, &job)
	body, contentType := multipartBody(t, map[string]string{"meta": `{"durationMs":5}`},
		filePart{"audio", "a.ogg", "audio/ogg", audio}, filePart{"image", "a.bmp", "image/bmp", []byte("bmp")})
	res := api.signedCall("/v1/internal/jobs/"+job.ID+"/result", body, contentType)
	if res.Code != http.StatusBadRequest || !strings.Contains(res.Body.String(), "image/bmp") {
		t.Fatalf("imagen no soportada: %d %s", res.Code, res.Body.String())
	}
}
