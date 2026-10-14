package pipeline

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func str(value string) *string { return &value }

// ---- Parte pura (los tests originales del processor en TS) ----

func TestParseTimeAceptaSegundosYFormatoDeReloj(t *testing.T) {
	cases := map[string]float64{"3.5": 3.5, "1:23": 83, "1:02:03.5": 3723.5}
	for input, want := range cases {
		got, err := ParseTime(&input)
		if err != nil || *got != want {
			t.Errorf("ParseTime(%q) = %v, %v; se esperaba %v", input, got, err, want)
		}
	}
	if got, err := ParseTime(nil); got != nil || err != nil {
		t.Errorf("ParseTime(nil) = %v, %v", got, err)
	}
	for _, bad := range []string{"abc", "", "1:2:3:4", "-1", "1m"} {
		if _, err := ParseTime(&bad); err == nil || err.Error() != fmt.Sprintf(`Tiempo inválido: "%s"`, bad) {
			t.Errorf("ParseTime(%q) debía fallar: %v", bad, err)
		}
	}
}

func TestAudioFilterRecortaAntesDeNormalizar(t *testing.T) {
	num := func(value float64) *float64 { return &value }
	const loud = "loudnorm=I=-16:TP=-1.5:LRA=11"
	cases := []struct {
		start, end *float64
		want       string
	}{
		{nil, nil, loud},
		{num(1), num(2.5), "atrim=start=1:end=2.5,asetpts=PTS-STARTPTS," + loud},
		{nil, num(4), "atrim=end=4,asetpts=PTS-STARTPTS," + loud},
		{num(3723.5), nil, "atrim=start=3723.5,asetpts=PTS-STARTPTS," + loud},
	}
	for _, tc := range cases {
		if got := AudioFilter(tc.start, tc.end); got != tc.want {
			t.Errorf("AudioFilter = %q; se esperaba %q", got, tc.want)
		}
	}
}

// ---- Con un Runner falso ----

type call struct {
	name string
	args []string
}

type fakeRunner struct {
	calls   []call
	handler func(name string, args []string) (string, error)
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) (string, error) {
	f.calls = append(f.calls, call{name, args})
	return f.handler(name, args)
}

func (f *fakeRunner) find(name string) []string {
	for _, c := range f.calls {
		if c.name == name {
			return c.args
		}
	}
	return nil
}

// ffmpegRunner simula ffprobe (10 s la fuente, 1.5 s el resultado) y ffmpeg (escribe el archivo de salida).
func ffmpegRunner(sourceDuration string) *fakeRunner {
	return &fakeRunner{handler: func(name string, args []string) (string, error) {
		switch name {
		case "ffprobe":
			if strings.HasSuffix(args[len(args)-1], "output.ogg") {
				return "1.5\n", nil
			}
			return sourceDuration, nil
		case "ffmpeg":
			return "", os.WriteFile(args[len(args)-1], []byte("OggS"), 0o600)
		}
		return "", fmt.Errorf("herramienta inesperada: %s", name)
	}}
}

func uploadOf(ext string) UploadFetcher {
	return func(context.Context, string) (Downloaded, error) {
		return Downloaded{Data: []byte("audio"), ContentType: "audio/mpeg", Ext: ext}, nil
	}
}

func next(args []string, flag string) string {
	if i := slices.Index(args, flag); i >= 0 && i+1 < len(args) {
		return args[i+1]
	}
	return ""
}

func TestConversionUsaLosArgumentosDeFfmpegEsperados(t *testing.T) {
	runner := ffmpegRunner("10.0\n")
	p := &Pipeline{Runner: runner}
	result, err := p.Process(context.Background(), Options{
		URL: "upload:uploads/a.mp3", From: str("1"), To: str("2.5"), Stereo: true, FetchUpload: uploadOf(".mp3"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(result.Audio) != "OggS" || result.DurationMs != 1500 || result.Image != nil {
		t.Fatalf("resultado: %+v", result)
	}
	args := runner.find("ffmpeg")
	want := map[string]string{
		"-loglevel": "error", "-map_metadata": "-1", "-ac": "2", "-ar": "44100", "-c:a": "libvorbis", "-q:a": "4",
		"-af": "atrim=start=1:end=2.5,asetpts=PTS-STARTPTS,loudnorm=I=-16:TP=-1.5:LRA=11",
	}
	for flag, value := range want {
		if got := next(args, flag); got != value {
			t.Errorf("%s = %q; se esperaba %q", flag, got, value)
		}
	}
	if !slices.Contains(args, "-vn") || !strings.HasSuffix(next(args, "-i"), "source.mp3") || !strings.HasSuffix(args[len(args)-1], "output.ogg") {
		t.Errorf("argumentos: %v", args)
	}

	// Mono por defecto y sin recorte.
	runner = ffmpegRunner("10.0\n")
	p = &Pipeline{Runner: runner}
	if _, err := p.Process(context.Background(), Options{URL: "upload:uploads/a.wav", FetchUpload: uploadOf(".wav")}); err != nil {
		t.Fatal(err)
	}
	args = runner.find("ffmpeg")
	if next(args, "-ac") != "1" || next(args, "-af") != "loudnorm=I=-16:TP=-1.5:LRA=11" {
		t.Errorf("mono sin recorte: %v", args)
	}
}

func TestValidaElRecorte(t *testing.T) {
	cases := []struct {
		name     string
		from, to *string
		duration string
		want     string
	}{
		{"to menor que from", str("5"), str("3"), "10\n", "'to' debe ser mayor que 'from'"},
		{"to igual a from", str("5"), str("5"), "10\n", "'to' debe ser mayor que 'from'"},
		{"from después del final", str("20"), nil, "10.0\n", "'from' (20 s) está después del final del audio (10.00 s)"},
		{"tiempo inválido", str("abc"), nil, "10\n", `Tiempo inválido: "abc"`},
	}
	for _, tc := range cases {
		p := &Pipeline{Runner: ffmpegRunner(tc.duration)}
		_, err := p.Process(context.Background(), Options{URL: "upload:uploads/a.mp3", From: tc.from, To: tc.to, FetchUpload: uploadOf(".mp3")})
		if err == nil || err.Error() != tc.want {
			t.Errorf("%s: %v; se esperaba %q", tc.name, err, tc.want)
		}
	}
}

// Si ffprobe no entrega una duración numérica ("N/A") no se valida `from`: ffmpeg decide.
func TestDuracionDesconocidaDeLaFuenteNoBloquea(t *testing.T) {
	p := &Pipeline{Runner: ffmpegRunner("N/A\n")}
	if _, err := p.Process(context.Background(), Options{URL: "upload:uploads/a.webm", From: str("20"), FetchUpload: uploadOf(".webm")}); err != nil {
		t.Fatal(err)
	}
}

func TestYtDlp(t *testing.T) {
	runner := ffmpegRunner("10\n")
	inner := runner.handler
	runner.handler = func(name string, args []string) (string, error) {
		if name == "yt-dlp" {
			output := next(args, "--output")
			return "", os.WriteFile(strings.Replace(output, "%(ext)s", "webm", 1), []byte("x"), 0o600)
		}
		return inner(name, args)
	}
	p := &Pipeline{Runner: runner}
	if _, err := p.Process(context.Background(), Options{URL: "https://youtu.be/x", YtDlpCookies: "/secrets/cookies.txt"}); err != nil {
		t.Fatal(err)
	}
	args := runner.find("yt-dlp")
	if next(args, "--format") != "bestaudio/best" || next(args, "--cookies") != "/secrets/cookies.txt" ||
		!slices.Contains(args, "--no-playlist") || args[len(args)-1] != "https://youtu.be/x" {
		t.Errorf("argumentos de yt-dlp: %v", args)
	}
	if !strings.HasSuffix(next(runner.find("ffmpeg"), "-i"), "source.webm") {
		t.Errorf("ffmpeg debe convertir el archivo que dejó yt-dlp: %v", runner.find("ffmpeg"))
	}

	// Sin cookies no se pasa --cookies, y sin archivo de salida falla con un mensaje claro.
	runner = &fakeRunner{handler: func(string, []string) (string, error) { return "", nil }}
	_, err := (&Pipeline{Runner: runner}).Process(context.Background(), Options{URL: "https://youtu.be/x"})
	if err == nil || err.Error() != "yt-dlp no generó el archivo de audio" || slices.Contains(runner.find("yt-dlp"), "--cookies") {
		t.Errorf("sin archivo: %v %v", err, runner.find("yt-dlp"))
	}
}

func TestImagenes(t *testing.T) {
	images := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		types := map[string]string{
			"/a.png": "IMAGE/PNG; charset=binary", "/a.gif": "image/gif", "/a.bmp": "image/bmp", "/a.jpg": "image/jpeg", "/none": "",
		}
		if contentType, ok := types[r.URL.Path]; ok {
			if contentType != "" {
				w.Header().Set("Content-Type", contentType)
			} else {
				w.Header()["Content-Type"] = nil // sin esto Go detecta text/plain
			}
			if got := r.Header.Get("User-Agent"); !strings.Contains(got, "Chrome/128") {
				t.Errorf("User-Agent: %q", got)
			}
			fmt.Fprint(w, "bytes")
			return
		}
		http.NotFound(w, r)
	}))
	defer images.Close()

	process := func(image, dimensions string) (*fakeRunner, Result, error) {
		runner := ffmpegRunner("10\n")
		inner := runner.handler
		runner.handler = func(name string, args []string) (string, error) {
			switch name {
			case "ffprobe":
				if slices.Contains(args, "stream=width,height") {
					return dimensions + "\n", nil
				}
			case "cwebp":
				return "", os.WriteFile(args[len(args)-1], []byte("RIFFxxxxWEBP"), 0o600)
			}
			return inner(name, args)
		}
		result, err := (&Pipeline{Runner: runner, HTTP: http.DefaultClient}).Process(context.Background(), Options{
			URL: "upload:uploads/a.mp3", ImageURL: &image, FetchUpload: uploadOf(".mp3"),
		})
		return runner, result, err
	}

	for name, tc := range map[string]struct {
		dimensions string
		resize     []string
	}{
		"horizontal grande": {"1000x500", []string{"-resize", "384", "0"}},
		"vertical grande":   {"500x1000", []string{"-resize", "0", "384"}},
		"cuadrada grande":   {"800x800", []string{"-resize", "384", "0"}},
		"chica":             {"200x100", nil},
	} {
		runner, result, err := process(images.URL+"/a.png", tc.dimensions)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		args := runner.find("cwebp")
		wantPrefix := []string{"-quiet", "-q", "80", "-metadata", "none"}
		if !slices.Equal(args[:5], wantPrefix) || !slices.Equal(args[5:len(args)-3], tc.resize) {
			t.Errorf("%s: cwebp %v", name, args)
		}
		if result.Image == nil || result.Image.ContentType != "image/webp" || string(result.Image.Data) != "RIFFxxxxWEBP" {
			t.Errorf("%s: imagen %+v", name, result.Image)
		}
	}

	// JPEG también se convierte; GIF pasa sin tocar y sin invocar cwebp.
	if _, result, err := process(images.URL+"/a.jpg", "100x100"); err != nil || result.Image.ContentType != "image/webp" {
		t.Errorf("jpeg: %v %+v", err, result.Image)
	}
	runner, result, err := process(images.URL+"/a.gif", "100x100")
	if err != nil || runner.find("cwebp") != nil || result.Image.ContentType != "image/gif" || string(result.Image.Data) != "bytes" {
		t.Errorf("gif: %v %+v", err, result.Image)
	}

	for url, want := range map[string]string{
		"/a.bmp": "La imagen no es de un tipo soportado (content-type: image/bmp)",
		"/none":  "La imagen no es de un tipo soportado (content-type: desconocido)",
		"/falta": "HTTP 404 al descargar " + images.URL + "/falta",
	} {
		if _, _, err := process(images.URL+url, "1x1"); err == nil || err.Error() != want {
			t.Errorf("%s: %v; se esperaba %q", url, err, want)
		}
	}

	// Las imágenes subidas usan el content-type que devolvió la API.
	runner = ffmpegRunner("10\n")
	image := "upload:uploads/i.gif"
	result, err = (&Pipeline{Runner: runner}).Process(context.Background(), Options{
		URL: "upload:uploads/a.mp3", ImageURL: &image,
		FetchUpload: func(_ context.Context, source string) (Downloaded, error) {
			if source == image {
				return Downloaded{Data: []byte("gif"), ContentType: "image/gif", Ext: ".gif"}, nil
			}
			return Downloaded{Data: []byte("a"), Ext: ".mp3"}, nil
		},
	})
	if err != nil || result.Image.ContentType != "image/gif" {
		t.Errorf("upload de imagen: %v %+v", err, result.Image)
	}
}

func TestEpidemicSound(t *testing.T) {
	const id = "0123abcd-0123-abcd-0123-0123456789ab"
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tracks/" + id:
			// El payload de Next.js trae las comillas escapadas.
			fmt.Fprintf(w, `<script>{\"kosmosId\":\"%s\",\"title\":\"x\",\"lqMp3Url\":\"%s/media/song.mp3\"}</script>`, id, server.URL)
		case "/media/song.mp3":
			fmt.Fprint(w, "mp3")
		case "/tracks/sin-datos":
			fmt.Fprint(w, "<html></html>")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	p := &Pipeline{HTTP: http.DefaultClient}
	dir := t.TempDir()

	source, err := p.downloadFromEpidemic(context.Background(), server.URL+"/tracks/"+id, dir)
	if err != nil || filepath.Base(source) != "source.mp3" {
		t.Fatalf("descarga: %v %v", source, err)
	}
	if data, _ := os.ReadFile(source); string(data) != "mp3" {
		t.Fatalf("contenido: %q", data)
	}
	for url, want := range map[string]string{
		server.URL + "/sin-id":                            "No se encontró el id del track en la URL: " + server.URL + "/sin-id",
		server.URL + "/tracks/" + strings.Repeat("a", 36): "HTTP 404 al descargar " + server.URL + "/tracks/" + strings.Repeat("a", 36),
	} {
		if _, err := p.downloadFromEpidemic(context.Background(), url, dir); err == nil || err.Error() != want {
			t.Errorf("%s: %v; se esperaba %q", url, err, want)
		}
	}
}

func TestIsEpidemic(t *testing.T) {
	for url, want := range map[string]bool{
		"https://www.epidemicsound.com/track/abc/": true, "https://epidemicsound.com/x": true,
		"https://youtu.be/x": false, "https://evil.test/epidemicsound.com": false, "%%": false,
	} {
		if got := isEpidemic(url); got != want {
			t.Errorf("isEpidemic(%q) = %v", url, got)
		}
	}
}

// ---- ExecRunner con herramientas reales del sistema ----

func TestExecRunner(t *testing.T) {
	runner := ExecRunner{}
	ctx := context.Background()
	if out, err := runner.Run(ctx, "sh", "-c", "echo hola"); err != nil || out != "hola\n" {
		t.Errorf("éxito: %q %v", out, err)
	}
	if _, err := runner.Run(ctx, "herramienta-que-no-existe"); err == nil || err.Error() != "No se encontró herramienta-que-no-existe en el contenedor" {
		t.Errorf("no encontrada: %v", err)
	}
	if _, err := runner.Run(ctx, "sh", "-c", "echo mala cosa >&2; exit 3"); err == nil || err.Error() != "sh: mala cosa" {
		t.Errorf("stderr: %v", err)
	}
	if _, err := runner.Run(ctx, "sh", "-c", "exit 3"); err == nil || err.Error() != "exit status 3" {
		t.Errorf("sin stderr: %v", err)
	}
	// Solo se conservan los últimos 1500 caracteres del stderr.
	_, err := runner.Run(ctx, "sh", "-c", "printf 'A%.0s' $(seq 1 2000) >&2; printf FIN >&2; exit 1")
	if err == nil || !strings.HasSuffix(err.Error(), "FIN") || len([]rune(err.Error())) != len("sh: ")+1500 {
		t.Errorf("stderr largo: %d caracteres", len([]rune(fmt.Sprint(err))))
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := runner.Run(cancelled, "sleep", "5"); err == nil || !strings.Contains(err.Error(), "cancelado") {
		t.Errorf("cancelado: %v", err)
	}
}
