package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"soundboard/gosig"
	"soundboard/processor/internal/pipeline"
)

const secret = "test-secret"

// fakeAPI hace de API: verifica la firma de cada llamada del processor y registra lo recibido.
type fakeAPI struct {
	*httptest.Server
	mu       sync.Mutex
	statuses []map[string]any
	results  []receivedResult
	failures map[string]int // ruta → código a devolver
	uploads  map[string]string
}

type receivedResult struct {
	jobID, contentType string
	body               []byte
	meta               string
	audio, image       []byte
	audioType          string
	imageType          string
	audioFile          string
}

func newFakeAPI(t *testing.T) *fakeAPI {
	api := &fakeAPI{failures: map[string]int{}, uploads: map[string]string{"uploads/a.mp3": "datos"}}
	api.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api.mu.Lock()
		defer api.mu.Unlock()
		body, _ := io.ReadAll(r.Body)
		signed := body
		key, isUpload := strings.CutPrefix(r.URL.Path, "/v1/internal/uploads/")
		if isUpload {
			signed = []byte(key)
		}
		if !gosig.Verify(secret, signed, r.Header.Get(gosig.HeaderTimestamp), r.Header.Get(gosig.HeaderSignature), time.Now()) {
			t.Errorf("firma inválida en %s", r.URL.Path)
			http.Error(w, "Firma inválida", http.StatusUnauthorized)
			return
		}
		if isUpload {
			content, ok := api.uploads[key]
			if !ok {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "audio/mpeg; charset=x")
			io.WriteString(w, content)
			return
		}
		route := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		if code := api.failures[route]; code != 0 {
			http.Error(w, "boom", code)
			return
		}
		jobID := strings.Split(r.URL.Path, "/")[4]
		switch route {
		case "status":
			var update map[string]any
			json.Unmarshal(body, &update)
			update["jobID"] = jobID
			api.statuses = append(api.statuses, update)
			w.WriteHeader(http.StatusNoContent)
		case "result":
			api.results = append(api.results, parseResult(t, jobID, r.Header.Get("Content-Type"), body))
			io.WriteString(w, `{"soundId":"x"}`)
		}
	}))
	t.Cleanup(api.Close)
	return api
}

func parseResult(t *testing.T, jobID, contentType string, body []byte) receivedResult {
	result := receivedResult{jobID: jobID, contentType: contentType, body: body}
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		t.Fatalf("content-type: %v", err)
	}
	reader := multipart.NewReader(bytes.NewReader(body), params["boundary"])
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			return result
		}
		if err != nil {
			t.Fatal(err)
		}
		data, _ := io.ReadAll(part)
		switch part.FormName() {
		case "meta":
			result.meta = string(data)
			if part.FileName() != "" {
				t.Errorf("meta no debe ser un archivo")
			}
		case "audio":
			result.audio, result.audioType, result.audioFile = data, part.Header.Get("Content-Type"), part.FileName()
		case "image":
			result.image, result.imageType = data, part.Header.Get("Content-Type")
		}
	}
}

func (a *fakeAPI) snapshot() ([]map[string]any, []receivedResult) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]map[string]any(nil), a.statuses...), append([]receivedResult(nil), a.results...)
}

type processorFunc func(ctx context.Context, options pipeline.Options) (pipeline.Result, error)

func (f processorFunc) Process(ctx context.Context, options pipeline.Options) (pipeline.Result, error) {
	return f(ctx, options)
}

func okProcessor(context.Context, pipeline.Options) (pipeline.Result, error) {
	return pipeline.Result{Audio: []byte("ogg"), DurationMs: 1500}, nil
}

type harness struct {
	t      *testing.T
	api    *fakeAPI
	server *Server
	http   *httptest.Server
	done   chan struct{}
}

func newHarness(t *testing.T, config Config, processor processorFunc) *harness {
	t.Helper()
	api := newFakeAPI(t)
	config.Secret = secret
	server := New(config, processor, http.DefaultClient, slog.New(slog.NewTextHandler(io.Discard, nil)))
	front := httptest.NewServer(server.Handler())
	t.Cleanup(front.Close)
	h := &harness{t: t, api: api, server: server, http: front, done: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { server.Work(ctx); close(h.done) }()
	return h
}

// stop espera a que se procese todo lo encolado y luego cierra la cola. (Stop con jobs sin empezar
// los descarta: ese caso lo cubre TestApagadoReportaFallidosLosJobsQueNoEmpezaron.)
func (h *harness) stop() {
	h.t.Helper()
	for deadline := time.Now().Add(5 * time.Second); h.server.Pending() > 0; time.Sleep(5 * time.Millisecond) {
		if time.Now().After(deadline) {
			h.t.Fatal("la cola no se vació")
		}
	}
	h.server.Stop()
	select {
	case <-h.done:
	case <-time.After(5 * time.Second):
		h.t.Fatal("el worker no terminó")
	}
}

func (h *harness) post(path string, body []byte, headers map[string]string) *http.Response {
	h.t.Helper()
	request, _ := http.NewRequest(http.MethodPost, h.http.URL+path, bytes.NewReader(body))
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		h.t.Fatal(err)
	}
	return response
}

func (h *harness) submit(job ProcessRequest) *http.Response {
	h.t.Helper()
	if job.APIBaseURL == "" {
		job.APIBaseURL = h.api.URL
	}
	body, _ := json.Marshal(job)
	return h.post("/process", body, gosig.Headers(secret, body, time.Now()))
}

func text(response *http.Response) string {
	defer response.Body.Close()
	data, _ := io.ReadAll(response.Body)
	return strings.TrimSpace(string(data))
}

func TestHealthYRutasDesconocidas(t *testing.T) {
	h := newHarness(t, Config{}, okProcessor)
	response, _ := http.Get(h.http.URL + "/health")
	if body := text(response); response.StatusCode != 200 || body != `{"ok":true,"pending":0}` {
		t.Fatalf("health: %d %s", response.StatusCode, body)
	}
	for _, path := range []string{"/", "/otra", "/process?x=1"} {
		response, _ := http.Get(h.http.URL + path)
		if response.StatusCode != 404 {
			t.Errorf("GET %s: %d", path, response.StatusCode)
		}
	}
	if response := h.post("/health", nil, nil); response.StatusCode != 404 {
		t.Errorf("POST /health: %d", response.StatusCode)
	}
}

func TestProcessRechazaPeticionesInvalidas(t *testing.T) {
	h := newHarness(t, Config{}, okProcessor)
	good := []byte(`{"jobId":"j","url":"https://a.test/x","apiBaseUrl":"http://x"}`)
	signed := gosig.Headers(secret, good, time.Now())
	old := gosig.Headers(secret, good, time.Now().Add(-10*time.Minute))
	big := bytes.Repeat([]byte("a"), maxBodyBytes+1)
	notJSON := []byte("no es json")

	cases := []struct {
		name    string
		body    []byte
		headers map[string]string
		code    int
		message string
	}{
		{"sin firma", good, nil, 401, "Firma inválida"},
		{"firma errónea", good, map[string]string{gosig.HeaderTimestamp: signed[gosig.HeaderTimestamp], gosig.HeaderSignature: strings.Repeat("0", 64)}, 401, "Firma inválida"},
		{"firma de otro cuerpo", []byte(`{"jobId":"otro"}`), signed, 401, "Firma inválida"},
		{"firma vencida", good, old, 401, "Firma inválida"},
		{"cuerpo demasiado grande", big, gosig.Headers(secret, big, time.Now()), 400, "Cuerpo demasiado grande"},
		{"JSON roto", notJSON, gosig.Headers(secret, notJSON, time.Now()), 400, "JSON inválido"},
		{"faltan campos", []byte(`{"jobId":"j"}`), gosig.Headers(secret, []byte(`{"jobId":"j"}`), time.Now()), 400, "Faltan jobId, url o apiBaseUrl"},
	}
	for _, tc := range cases {
		response := h.post("/process", tc.body, tc.headers)
		if got := text(response); response.StatusCode != tc.code || got != tc.message {
			t.Errorf("%s: %d %q", tc.name, response.StatusCode, got)
		}
	}
	if h.server.Pending() != 0 {
		t.Errorf("no debió encolar nada: %d", h.server.Pending())
	}
}

func TestFlujoCompletoReportaRunningYResultado(t *testing.T) {
	var got pipeline.Options
	h := newHarness(t, Config{YtDlpCookies: "/c.txt"}, func(ctx context.Context, options pipeline.Options) (pipeline.Result, error) {
		got = options
		return pipeline.Result{
			Audio: []byte("ogg-bytes"), DurationMs: 1500,
			Image: &pipeline.Image{Data: []byte("webp-bytes"), ContentType: "image/webp"},
		}, nil
	})
	response := h.submit(ProcessRequest{
		JobID: "job-1", URL: "https://youtu.be/x", Title: "T", ImageURL: ptr("https://a.test/i.png"),
		From: ptr("1:02"), Stereo: true,
	})
	if body := text(response); response.StatusCode != 202 || body != `{"queued":1}` {
		t.Fatalf("encolar: %d %s", response.StatusCode, body)
	}
	h.stop()

	statuses, results := h.api.snapshot()
	if len(statuses) != 1 || statuses[0]["status"] != "running" || statuses[0]["jobID"] != "job-1" {
		t.Fatalf("estados: %v", statuses)
	}
	if len(results) != 1 {
		t.Fatalf("resultados: %d", len(results))
	}
	r := results[0]
	if r.jobID != "job-1" || r.meta != `{"durationMs":1500}` || string(r.audio) != "ogg-bytes" ||
		r.audioType != "audio/ogg" || r.audioFile != "audio.ogg" || string(r.image) != "webp-bytes" || r.imageType != "image/webp" {
		t.Fatalf("multipart recibido: %+v", r)
	}
	if got.URL != "https://youtu.be/x" || *got.From != "1:02" || got.To != nil || !got.Stereo ||
		*got.ImageURL != "https://a.test/i.png" || got.YtDlpCookies != "/c.txt" {
		t.Fatalf("opciones del pipeline: %+v", got)
	}
}

func TestResultadoSinImagenNoLlevaParteImage(t *testing.T) {
	h := newHarness(t, Config{}, okProcessor)
	h.submit(ProcessRequest{JobID: "j", URL: "https://a.test/x"})
	h.stop()
	_, results := h.api.snapshot()
	if len(results) != 1 || results[0].image != nil || results[0].imageType != "" || bytes.Contains(results[0].body, []byte(`name="image"`)) {
		t.Fatalf("no debía haber imagen: %+v", results)
	}
}

func TestFalloDelPipelineSeReportaConElMensaje(t *testing.T) {
	long := strings.Repeat("é", 2500)
	h := newHarness(t, Config{}, func(context.Context, pipeline.Options) (pipeline.Result, error) {
		return pipeline.Result{}, errors.New("yt-dlp: " + long)
	})
	h.submit(ProcessRequest{JobID: "j", URL: "https://a.test/x"})
	h.stop()
	statuses, results := h.api.snapshot()
	if len(results) != 0 || len(statuses) != 2 || statuses[1]["status"] != "failed" {
		t.Fatalf("estados: %v", statuses)
	}
	message := statuses[1]["error"].(string)
	if len([]rune(message)) != 2000 || !strings.HasPrefix(message, "yt-dlp: ééé") {
		t.Fatalf("el error debe recortarse a 2000 caracteres: %d", len([]rune(message)))
	}
}

func TestErrorDeLaAPIAlEnviarElResultadoSeReportaComoFallo(t *testing.T) {
	h := newHarness(t, Config{}, okProcessor)
	h.api.failures["result"] = 500
	h.submit(ProcessRequest{JobID: "j", URL: "https://a.test/x"})
	h.stop()
	statuses, _ := h.api.snapshot()
	message, _ := statuses[len(statuses)-1]["error"].(string)
	if statuses[len(statuses)-1]["status"] != "failed" || message != "La API respondió 500 en result: boom" {
		t.Fatalf("estados: %v", statuses)
	}
}

func TestAPIBaseURLDelEntornoReemplazaLaDelJob(t *testing.T) {
	api := newFakeAPI(t)
	h := newHarness(t, Config{APIBaseURL: api.URL + "/"}, okProcessor)
	h.submit(ProcessRequest{JobID: "j", URL: "https://a.test/x", APIBaseURL: "http://localhost:1"})
	h.stop()
	if statuses, results := api.snapshot(); len(statuses) != 1 || len(results) != 1 {
		t.Fatalf("la API del entorno debió recibir el job: %v %d", statuses, len(results))
	}
	if statuses, _ := h.api.snapshot(); len(statuses) != 0 {
		t.Fatal("la API del job no debía usarse")
	}
}

func TestUploadsSeBajanFirmandoLaKey(t *testing.T) {
	var data pipeline.Downloaded
	var missing error
	h := newHarness(t, Config{}, nil)
	h.server.Pipeline = processorFunc(func(ctx context.Context, options pipeline.Options) (pipeline.Result, error) {
		var err error
		data, err = options.FetchUpload(ctx, "upload:uploads/a.mp3")
		if err != nil {
			return pipeline.Result{}, err
		}
		_, missing = options.FetchUpload(ctx, "upload:uploads/no-existe.mp3")
		return pipeline.Result{Audio: []byte("x"), DurationMs: 1}, nil
	})
	h.submit(ProcessRequest{JobID: "j", URL: "upload:uploads/a.mp3"})
	h.stop()
	if string(data.Data) != "datos" || data.ContentType != "audio/mpeg" || data.Ext != ".mp3" {
		t.Fatalf("upload: %+v", data)
	}
	if missing == nil || missing.Error() != "No se pudo bajar uploads/no-existe.mp3: HTTP 404" {
		t.Fatalf("upload inexistente: %v", missing)
	}
}

func TestLosJobsSeProcesanDeUnoEnUno(t *testing.T) {
	var mu sync.Mutex
	running, maxRunning := 0, 0
	var order []string
	h := newHarness(t, Config{}, func(ctx context.Context, options pipeline.Options) (pipeline.Result, error) {
		mu.Lock()
		running++
		maxRunning = max(maxRunning, running)
		order = append(order, options.URL)
		mu.Unlock()
		time.Sleep(20 * time.Millisecond)
		mu.Lock()
		running--
		mu.Unlock()
		return pipeline.Result{Audio: []byte("x"), DurationMs: 1}, nil
	})
	for _, id := range []string{"a", "b", "c"} {
		h.submit(ProcessRequest{JobID: id, URL: "https://a.test/" + id})
	}
	h.stop()
	if maxRunning != 1 || strings.Join(order, ",") != "https://a.test/a,https://a.test/b,https://a.test/c" {
		t.Fatalf("en paralelo: %d, orden %v", maxRunning, order)
	}
	if _, results := h.api.snapshot(); len(results) != 3 || h.server.Pending() != 0 {
		t.Fatalf("resultados: %d, pendientes %d", len(results), h.server.Pending())
	}
}

func TestApagadoReportaFallidosLosJobsQueNoEmpezaron(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	h := newHarness(t, Config{}, func(ctx context.Context, options pipeline.Options) (pipeline.Result, error) {
		close(started)
		<-release
		return pipeline.Result{Audio: []byte("x"), DurationMs: 1}, nil
	})
	h.submit(ProcessRequest{JobID: "en-curso", URL: "https://a.test/1"})
	<-started
	h.submit(ProcessRequest{JobID: "en-cola", URL: "https://a.test/2"})
	if h.server.Pending() != 2 {
		t.Fatalf("pendientes: %d", h.server.Pending())
	}
	h.server.Stop()
	close(release)
	<-h.done

	statuses, results := h.api.snapshot()
	if len(results) != 1 || results[0].jobID != "en-curso" {
		t.Fatalf("el job en curso debía terminar: %+v", results)
	}
	last := statuses[len(statuses)-1]
	if last["jobID"] != "en-cola" || last["status"] != "failed" || !strings.Contains(last["error"].(string), "reintenta") {
		t.Fatalf("el job en cola debía reportarse como fallido: %v", statuses)
	}
	if h.server.Pending() != 0 {
		t.Fatalf("pendientes: %d", h.server.Pending())
	}
}

func TestStopEsIdempotente(t *testing.T) {
	h := newHarness(t, Config{}, okProcessor)
	h.server.Stop()
	h.server.Stop()
	<-h.done
}
