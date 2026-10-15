package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"soundboard/api/internal/db"

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

func TestAdminBorraJobs(t *testing.T) {
	api := newTestAPI(t)
	token := api.adminToken()
	setUpdatedAt := func(id, value string) {
		if _, err := api.server.DB.Exec("UPDATE jobs SET updated_at = ? WHERE id = ?", value, id); err != nil {
			t.Fatal(err)
		}
	}
	create := func(body any) JobDTO {
		var job JobDTO
		api.call("POST", "/v1/admin/jobs", body, token).decode(t, &job)
		return job
	}
	remaining := func() []string {
		var ids []string
		for _, job := range api.jobs(token) {
			ids = append(ids, job.ID)
		}
		return ids
	}

	// Fallido (el processor no responde): se puede borrar y desaparece de la lista.
	api.processor.status = 0
	failed := create(map[string]string{"url": "https://a.test/x", "title": "Fallido"})
	if res := api.call("DELETE", "/v1/admin/jobs/"+failed.ID, nil, token); res.Code != http.StatusNoContent {
		t.Fatalf("borrar fallido: %d %s", res.Code, res.Body.String())
	}
	if len(remaining()) != 0 {
		t.Fatalf("el job borrado sigue en la lista: %v", remaining())
	}
	if res := api.call("DELETE", "/v1/admin/jobs/"+failed.ID, nil, token); res.Code != http.StatusNotFound {
		t.Fatalf("borrar dos veces: %d", res.Code)
	}

	// Terminado: se borra el registro, pero el sonido publicado se queda.
	api.processor.status = http.StatusAccepted
	done := create(map[string]string{"url": "https://a.test/x", "title": "Listo"})
	body, contentType := multipartBody(t, map[string]string{"meta": `{"durationMs":10}`}, filePart{"audio", "a.ogg", "audio/ogg", []byte("ogg")})
	if res := api.signedCall("/v1/internal/jobs/"+done.ID+"/result", body, contentType); res.Code != http.StatusOK {
		t.Fatalf("result: %d", res.Code)
	}
	if res := api.call("DELETE", "/v1/admin/jobs/"+done.ID, nil, token); res.Code != http.StatusNoContent {
		t.Fatalf("borrar terminado: %d", res.Code)
	}
	var catalog CatalogPage
	api.call("GET", "/v1/catalog", nil, "").decode(t, &catalog)
	if len(catalog.Items) != 1 || catalog.Items[0].Title != "Listo" {
		t.Fatalf("el sonido debía conservarse: %+v", catalog.Items)
	}

	// En cola o procesando: se rechaza con un motivo, salvo que lleve más de 30 minutos sin cambios.
	queued := create(map[string]string{"url": "https://a.test/y", "title": "En cola"})
	running := create(map[string]string{"url": "https://a.test/z", "title": "Procesando"})
	api.signedCall("/v1/internal/jobs/"+running.ID+"/status", []byte(`{"status":"running"}`), "application/json")
	for _, id := range []string{queued.ID, running.ID} {
		res := api.call("DELETE", "/v1/admin/jobs/"+id, nil, token)
		if res.Code != http.StatusConflict || !strings.Contains(res.Body.String(), "sigue en proceso") {
			t.Fatalf("job activo: %d %s", res.Code, res.Body.String())
		}
	}
	setUpdatedAt(queued.ID, db.Now()) // recién actualizado: sigue protegido
	setUpdatedAt(running.ID, api.server.now().Add(-31*time.Minute).UTC().Format(db.TimeFormat))
	if res := api.call("DELETE", "/v1/admin/jobs/"+running.ID, nil, token); res.Code != http.StatusNoContent {
		t.Fatalf("job atascado: %d %s", res.Code, res.Body.String())
	}
	if res := api.call("DELETE", "/v1/admin/jobs/"+queued.ID, nil, token); res.Code != http.StatusConflict {
		t.Fatalf("job en cola reciente: %d", res.Code)
	}

	// Solo admins.
	user := api.signIn("user", "user@example.com").Token
	if res := api.call("DELETE", "/v1/admin/jobs/"+queued.ID, nil, user); res.Code != http.StatusForbidden {
		t.Fatalf("usuario normal: %d", res.Code)
	}
	if res := api.call("DELETE", "/v1/admin/jobs/"+queued.ID, nil, ""); res.Code != http.StatusUnauthorized {
		t.Fatalf("sin sesión: %d", res.Code)
	}
}

// Los uploads de un job fallido se borran junto con él (los de uno exitoso ya se borran solos).
func TestAdminBorrarJobFallidoLimpiaSusUploads(t *testing.T) {
	api := newTestAPI(t)
	token := api.adminToken()
	api.processor.status = 0
	upload := func(name, contentType string) string {
		body, header := multipartBody(t, nil, filePart{"file", name, contentType, []byte("x")})
		var uploaded struct{ URL string }
		api.call("POST", "/v1/admin/uploads", body, token, "Content-Type", header).decode(t, &uploaded)
		return uploaded.URL
	}
	audio, image := upload("a.mp3", "audio/mpeg"), upload("i.png", "image/png")
	var job JobDTO
	api.call("POST", "/v1/admin/jobs", map[string]string{"url": audio, "imageUrl": image, "title": "X"}, token).decode(t, &job)
	audioKey, imageKey := strings.TrimPrefix(audio, "upload:"), strings.TrimPrefix(image, "upload:")
	if !api.exists(audioKey) || !api.exists(imageKey) {
		t.Fatal("los uploads debían existir antes de borrar")
	}
	if res := api.call("DELETE", "/v1/admin/jobs/"+job.ID, nil, token); res.Code != http.StatusNoContent {
		t.Fatalf("borrar: %d", res.Code)
	}
	if api.exists(audioKey) || api.exists(imageKey) {
		t.Fatal("los uploads debían borrarse con el job")
	}
}

// replaceJob crea un job que reemplaza a `oldID` y entrega su resultado como lo haría el processor.
func (a *testAPI) replaceJob(token, oldID, title string, audio []byte, image *filePart) (JobDTO, string) {
	a.t.Helper()
	var job JobDTO
	created := a.call("POST", "/v1/admin/jobs", map[string]string{"url": "https://a.test/x", "title": title, "replacesSoundId": oldID}, token)
	if created.Code != http.StatusAccepted {
		a.t.Fatalf("crear job: %d %s", created.Code, created.Body.String())
	}
	created.decode(a.t, &job)
	files := []filePart{{"audio", "a.ogg", "audio/ogg", audio}}
	if image != nil {
		files = append(files, *image)
	}
	body, contentType := multipartBody(a.t, map[string]string{"meta": `{"durationMs":8800}`}, files...)
	if res := a.signedCall("/v1/internal/jobs/"+job.ID+"/result", body, contentType); res.Code != http.StatusOK {
		a.t.Fatalf("result: %d %s", res.Code, res.Body.String())
	}
	return job, signature.SHA256Hex(audio)
}

func (a *testAPI) putFiles(keys ...string) {
	a.t.Helper()
	for _, key := range keys {
		if err := a.store.Put(context.Background(), key, []byte(key), storage.PutOptions{}); err != nil {
			a.t.Fatal(err)
		}
	}
}

func TestReemplazarAudioConservaBibliotecasYCatalogo(t *testing.T) {
	api := newTestAPI(t)
	token := api.adminToken()
	oldID, otherID := soundID("a"), soundID("b")
	api.seedSound(oldID, "1500", "", "2020-01-01T00:00:00.000Z")
	api.seedSound(otherID, "otro", "", "")
	api.putFiles("sounds/"+oldID+".ogg", "images/"+oldID+".gif")

	first := api.signIn("uno", "uno@example.com").Token
	second := api.signIn("dos", "dos@example.com").Token
	for _, id := range []string{otherID, oldID} {
		api.call("PUT", "/v1/me/library/"+id, nil, first)
	}
	api.call("PUT", "/v1/me/library/"+oldID, nil, second)

	job, newID := api.replaceJob(token, oldID, "1500 recortado", []byte("audio nuevo"), nil)
	if job.ReplacesSoundID == nil || *job.ReplacesSoundID != oldID {
		t.Fatalf("job: %+v", job)
	}

	// El nuevo ocupa el mismo lugar en cada biblioteca.
	soundIDs := func(token string) []string {
		var library LibraryDTO
		api.call("GET", "/v1/me/library", nil, token).decode(t, &library)
		return titles(library.Items, func(item SoundDTO) string { return item.ID })
	}
	if got := soundIDs(first); len(got) != 2 || got[0] != otherID || got[1] != newID {
		t.Fatalf("biblioteca uno: %v", got)
	}
	if got := soundIDs(second); len(got) != 1 || got[0] != newID {
		t.Fatalf("biblioteca dos: %v", got)
	}

	// Hereda fecha e imagen; el viejo desaparece con su audio, pero no con la imagen que heredó.
	var sounds struct{ Items []AdminSoundDTO }
	api.call("GET", "/v1/admin/sounds", nil, token).decode(t, &sounds)
	var replaced *AdminSoundDTO
	for i, sound := range sounds.Items {
		if sound.ID == oldID {
			t.Fatal("el sonido viejo debió borrarse")
		}
		if sound.ID == newID {
			replaced = &sounds.Items[i]
		}
	}
	if replaced == nil || replaced.Title != "1500 recortado" || replaced.Status != "published" ||
		replaced.CreatedAt != "2020-01-01T00:00:00.000Z" || replaced.DurationMs != 8800 ||
		replaced.ImageURL == nil || *replaced.ImageURL != "https://cdn.test/images/"+oldID+".gif" || !replaced.IsGif {
		t.Fatalf("sonido nuevo: %+v", replaced)
	}
	if api.exists("sounds/"+oldID+".ogg") || !api.exists("images/"+oldID+".gif") || !api.exists("sounds/"+newID+".ogg") {
		t.Fatal("archivos tras reemplazar")
	}

	// Reintentar cuando el viejo ya no existe solo vuelve a guardar el nuevo.
	api.replaceJobResult(job.ID, []byte("audio nuevo"))
	if got := soundIDs(first); len(got) != 2 || got[1] != newID {
		t.Fatalf("tras reintento: %v", got)
	}
}

func (a *testAPI) replaceJobResult(jobID string, audio []byte) {
	a.t.Helper()
	body, contentType := multipartBody(a.t, map[string]string{"meta": `{"durationMs":8800}`}, filePart{"audio", "a.ogg", "audio/ogg", audio})
	if res := a.signedCall("/v1/internal/jobs/"+jobID+"/result", body, contentType); res.Code != http.StatusOK {
		a.t.Fatalf("result: %d %s", res.Code, res.Body.String())
	}
}

func TestReemplazarAudioConImagenNuevaYConflictos(t *testing.T) {
	api := newTestAPI(t)
	token := api.adminToken()
	oldID := soundID("a")
	audio := []byte("audio nuevo")
	newID := signature.SHA256Hex(audio)
	api.seedSound(oldID, "viejo", "", "")
	api.seedSound(newID, "ya existía", "", "")
	api.putFiles("sounds/"+oldID+".ogg", "images/"+oldID+".gif")

	// Quien ya tenía los dos conserva el nuevo en su posición y solo pierde el viejo.
	user := api.signIn("uno", "uno@example.com").Token
	for _, id := range []string{oldID, newID} {
		api.call("PUT", "/v1/me/library/"+id, nil, user)
	}
	api.replaceJob(token, oldID, "nuevo", audio, &filePart{"image", "a.png", "image/png", []byte("png")})

	var library LibraryDTO
	api.call("GET", "/v1/me/library", nil, user).decode(t, &library)
	if len(library.Items) != 1 || library.Items[0].ID != newID || library.Items[0].Title != "nuevo" ||
		library.Items[0].ImageURL == nil || *library.Items[0].ImageURL != "https://cdn.test/images/"+newID+".png" {
		t.Fatalf("biblioteca: %+v", library.Items)
	}
	if api.exists("images/"+oldID+".gif") || api.exists("sounds/"+oldID+".ogg") {
		t.Fatal("los archivos del viejo debieron borrarse")
	}
}

func TestReemplazarAudioValidaElSonido(t *testing.T) {
	api := newTestAPI(t)
	token := api.adminToken()
	if res := api.call("POST", "/v1/admin/jobs", `{"url":"https://a.test/x","title":"X","replacesSoundId":"abc"}`, token); res.Code != http.StatusBadRequest {
		t.Fatalf("id inválido: %d %s", res.Code, res.Body.String())
	}
	res := api.call("POST", "/v1/admin/jobs", `{"url":"https://a.test/x","title":"X","replacesSoundId":"`+soundID("f")+`"}`, token)
	if res.Code != http.StatusNotFound || len(api.processor.requests) != 0 {
		t.Fatalf("sonido inexistente: %d %s", res.Code, res.Body.String())
	}
	var job JobDTO
	api.call("POST", "/v1/admin/jobs", `{"url":"https://a.test/x","title":"X"}`, token).decode(t, &job)
	if job.ReplacesSoundID != nil {
		t.Fatalf("un job normal no reemplaza: %+v", job)
	}
}

func TestReemplazarAudioConservaElEstadoOculto(t *testing.T) {
	api := newTestAPI(t)
	token := api.adminToken()
	api.seedSound(soundID("a"), "oculto", "hidden", "")
	_, newID := api.replaceJob(token, soundID("a"), "oculto", []byte("audio nuevo"), nil)
	var sounds struct{ Items []AdminSoundDTO }
	api.call("GET", "/v1/admin/sounds", nil, token).decode(t, &sounds)
	if len(sounds.Items) != 1 || sounds.Items[0].ID != newID || sounds.Items[0].Status != "hidden" {
		t.Fatalf("sonidos: %+v", sounds.Items)
	}
}
