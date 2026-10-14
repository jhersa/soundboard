// Package pipeline descarga un sonido (Epidemic Sound, YouTube u otro sitio que soporte yt-dlp),
// lo convierte a Ogg Vorbis con volumen normalizado y prepara la imagen. Es la lógica de
// scripts/download.mjs, sin escribir en disco: el resultado se manda a la API, que lo sube al
// almacén de medios.
package pipeline

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36"

// Ogg Vorbis se reproduce en todas las versiones de Android (minSdk 24).
// loudnorm iguala el volumen percibido entre audios de fuentes distintas (EBU R128, -16 LUFS).
const (
	loudnorm        = "loudnorm=I=-16:TP=-1.5:LRA=11"
	audioSampleRate = "44100"
	audioQuality    = "4"

	// Suficiente para un botón de ~120 dp en pantallas xxxhdpi. Solo se reduce, nunca se amplía.
	imageMaxSize = 384
	imageQuality = "80"

	uploadScheme = "upload:"
)

var (
	timePattern = regexp.MustCompile(`^\d+(:\d{1,2}){0,2}(\.\d+)?$`)
	trackID     = regexp.MustCompile(`(?i)tracks/([0-9a-f-]{36})`)
	lqMp3URL    = regexp.MustCompile(`"lqMp3Url":"([^"]+)"`)

	// Mismos tipos que IMAGE_CONTENT_TYPES de @soundboard/shared.
	imageExtensions = map[string]string{
		"image/png": "png", "image/jpeg": "jpg", "image/gif": "gif",
		"image/webp": "webp", "image/svg+xml": "svg", "image/avif": "avif",
	}
	resizableImageTypes = map[string]bool{"image/png": true, "image/jpeg": true}
)

type Downloaded struct {
	Data        []byte
	ContentType string
	Ext         string
}

// UploadFetcher resuelve las fuentes `upload:<key>` (archivos subidos al panel) contra la API.
type UploadFetcher func(ctx context.Context, url string) (Downloaded, error)

type Options struct {
	URL          string
	ImageURL     *string
	From, To     *string
	Stereo       bool
	FetchUpload  UploadFetcher
	YtDlpCookies string
}

type Image struct {
	Data        []byte
	ContentType string
}

type Result struct {
	Audio      []byte
	DurationMs int64
	Image      *Image
}

type Pipeline struct {
	Runner Runner
	HTTP   *http.Client
}

// ParseTime convierte "3.5", "1:23" o "1:02:03.5" a segundos; nil si no se indicó.
func ParseTime(value *string) (*float64, error) {
	if value == nil {
		return nil, nil
	}
	if !timePattern.MatchString(*value) {
		return nil, fmt.Errorf("Tiempo inválido: \"%s\"", *value)
	}
	var total float64
	for _, part := range strings.Split(*value, ":") {
		number, err := strconv.ParseFloat(part, 64)
		if err != nil {
			return nil, fmt.Errorf("Tiempo inválido: \"%s\"", *value)
		}
		total = total*60 + number
	}
	return &total, nil
}

func seconds(value float64) string { return strconv.FormatFloat(value, 'f', -1, 64) }

// AudioFilter pone atrim antes de loudnorm para que el volumen se mida solo sobre el fragmento recortado.
func AudioFilter(trimStart, trimEnd *float64) string {
	if trimStart == nil && trimEnd == nil {
		return loudnorm
	}
	var bounds []string
	if trimStart != nil {
		bounds = append(bounds, "start="+seconds(*trimStart))
	}
	if trimEnd != nil {
		bounds = append(bounds, "end="+seconds(*trimEnd))
	}
	return "atrim=" + strings.Join(bounds, ":") + ",asetpts=PTS-STARTPTS," + loudnorm
}

func (p *Pipeline) Process(ctx context.Context, options Options) (Result, error) {
	audio, durationMs, err := p.downloadAudio(ctx, options)
	if err != nil {
		return Result{}, err
	}
	result := Result{Audio: audio, DurationMs: durationMs}
	if options.ImageURL != nil {
		image, err := p.downloadImage(ctx, *options.ImageURL, options.FetchUpload)
		if err != nil {
			return Result{}, err
		}
		result.Image = &image
	}
	return result, nil
}

func (p *Pipeline) fetch(ctx context.Context, target string) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", userAgent)
	response, err := p.HTTP.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		response.Body.Close()
		return nil, fmt.Errorf("HTTP %d al descargar %s", response.StatusCode, target)
	}
	return response, nil
}

func (p *Pipeline) fetchBytes(ctx context.Context, target string) ([]byte, string, error) {
	response, err := p.fetch(ctx, target)
	if err != nil {
		return nil, "", err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	return data, response.Header.Get("Content-Type"), err
}

func (p *Pipeline) downloadAudio(ctx context.Context, options Options) ([]byte, int64, error) {
	dir, err := os.MkdirTemp("", "soundboard-")
	if err != nil {
		return nil, 0, err
	}
	defer os.RemoveAll(dir)

	var source string
	switch {
	case strings.HasPrefix(options.URL, uploadScheme):
		upload, err := options.FetchUpload(ctx, options.URL)
		if err != nil {
			return nil, 0, err
		}
		source = filepath.Join(dir, "source"+upload.Ext)
		err = os.WriteFile(source, upload.Data, 0o600)
		if err != nil {
			return nil, 0, err
		}
	case isEpidemic(options.URL):
		source, err = p.downloadFromEpidemic(ctx, options.URL, dir)
	default:
		source, err = p.downloadWithYtDlp(ctx, options.URL, dir, options.YtDlpCookies)
	}
	if err != nil {
		return nil, 0, err
	}
	return p.convertToOgg(ctx, source, dir, options)
}

func isEpidemic(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	return err == nil && strings.HasSuffix(parsed.Hostname(), "epidemicsound.com")
}

func (p *Pipeline) downloadFromEpidemic(ctx context.Context, trackURL, dir string) (string, error) {
	match := trackID.FindStringSubmatch(trackURL)
	if match == nil {
		return "", fmt.Errorf("No se encontró el id del track en la URL: %s", trackURL)
	}
	id := match[1]

	// La página embebe los datos del track en el payload de Next.js, a veces con comillas escapadas.
	page, _, err := p.fetchBytes(ctx, trackURL)
	if err != nil {
		return "", err
	}
	html := strings.ReplaceAll(string(page), `\"`, `"`)
	start := strings.Index(html, fmt.Sprintf(`"kosmosId":"%s"`, id))
	if start == -1 {
		return "", errors.New("No se encontraron los datos del track en la página")
	}
	data := html[start:min(start+5000, len(html))]
	audio := lqMp3URL.FindStringSubmatch(data)
	if audio == nil {
		return "", errors.New("No se encontró la URL del audio")
	}
	audioURL, err := url.Parse(audio[1])
	if err != nil {
		return "", err
	}
	ext := path.Ext(audioURL.Path)
	if ext == "" {
		ext = ".mp3"
	}
	body, _, err := p.fetchBytes(ctx, audio[1])
	if err != nil {
		return "", err
	}
	source := filepath.Join(dir, "source"+ext)
	return source, os.WriteFile(source, body, 0o600)
}

// Se baja el mejor audio original sin convertir; la única conversión la hace ffmpeg después.
func (p *Pipeline) downloadWithYtDlp(ctx context.Context, videoURL, dir, cookies string) (string, error) {
	args := []string{"--format", "bestaudio/best", "--no-playlist", "--quiet", "--no-warnings"}
	if cookies != "" {
		args = append(args, "--cookies", cookies)
	}
	args = append(args, "--output", filepath.Join(dir, "source.%(ext)s"), videoURL)
	if _, err := p.Runner.Run(ctx, "yt-dlp", args...); err != nil {
		return "", err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "source.") {
			return filepath.Join(dir, entry.Name()), nil
		}
	}
	return "", errors.New("yt-dlp no generó el archivo de audio")
}

// probeDuration devuelve ok=false si ffprobe no entrega una duración numérica (p. ej. "N/A").
func (p *Pipeline) probeDuration(ctx context.Context, file string) (float64, bool, error) {
	out, err := p.Runner.Run(ctx, "ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "csv=p=0", file)
	if err != nil {
		return 0, false, err
	}
	value, err := strconv.ParseFloat(strings.TrimSpace(out), 64)
	return value, err == nil && !math.IsNaN(value), nil
}

func (p *Pipeline) convertToOgg(ctx context.Context, source, dir string, options Options) ([]byte, int64, error) {
	trimStart, err := ParseTime(options.From)
	if err != nil {
		return nil, 0, err
	}
	trimEnd, err := ParseTime(options.To)
	if err != nil {
		return nil, 0, err
	}
	if trimStart != nil && trimEnd != nil && *trimEnd <= *trimStart {
		return nil, 0, errors.New("'to' debe ser mayor que 'from'")
	}

	sourceDuration, known, err := p.probeDuration(ctx, source)
	if err != nil {
		return nil, 0, err
	}
	if known && trimStart != nil && *trimStart >= sourceDuration {
		return nil, 0, fmt.Errorf("'from' (%s s) está después del final del audio (%.2f s)", seconds(*trimStart), sourceDuration)
	}

	channels := "1"
	if options.Stereo {
		channels = "2"
	}
	output := filepath.Join(dir, "output.ogg")
	if _, err := p.Runner.Run(ctx, "ffmpeg",
		"-hide_banner", "-loglevel", "error",
		"-i", source,
		"-vn",
		"-map_metadata", "-1",
		"-af", AudioFilter(trimStart, trimEnd),
		"-ac", channels,
		"-ar", audioSampleRate,
		"-c:a", "libvorbis",
		"-q:a", audioQuality,
		output,
	); err != nil {
		return nil, 0, err
	}
	audio, err := os.ReadFile(output)
	if err != nil {
		return nil, 0, err
	}
	duration, known, err := p.probeDuration(ctx, output)
	if err != nil {
		return nil, 0, err
	}
	if !known {
		return nil, 0, errors.New("ffprobe no devolvió la duración del audio convertido")
	}
	return audio, int64(math.Round(duration * 1000)), nil
}

func (p *Pipeline) downloadImage(ctx context.Context, target string, fetchUpload UploadFetcher) (Image, error) {
	var data []byte
	var contentType string
	if strings.HasPrefix(target, uploadScheme) {
		upload, err := fetchUpload(ctx, target)
		if err != nil {
			return Image{}, err
		}
		data, contentType = upload.Data, upload.ContentType
	} else {
		body, header, err := p.fetchBytes(ctx, target)
		if err != nil {
			return Image{}, err
		}
		data = body
		contentType = strings.ToLower(strings.TrimSpace(strings.SplitN(header, ";", 2)[0]))
	}
	ext, ok := imageExtensions[contentType]
	if !ok {
		shown := contentType
		if shown == "" {
			shown = "desconocido"
		}
		return Image{}, fmt.Errorf("La imagen no es de un tipo soportado (content-type: %s)", shown)
	}
	if !resizableImageTypes[contentType] {
		return Image{Data: data, ContentType: contentType}, nil
	}
	webp, err := p.convertToWebp(ctx, data, "."+ext)
	if err != nil {
		return Image{}, err
	}
	return Image{Data: webp, ContentType: "image/webp"}, nil
}

func (p *Pipeline) convertToWebp(ctx context.Context, data []byte, sourceExt string) ([]byte, error) {
	dir, err := os.MkdirTemp("", "soundboard-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	source := filepath.Join(dir, "source"+sourceExt)
	output := filepath.Join(dir, "output.webp")
	if err := os.WriteFile(source, data, 0o600); err != nil {
		return nil, err
	}

	out, err := p.Runner.Run(ctx, "ffprobe", "-v", "error", "-select_streams", "v:0",
		"-show_entries", "stream=width,height", "-of", "csv=s=x:p=0", source)
	if err != nil {
		return nil, err
	}
	var width, height float64
	if dims := strings.Split(strings.TrimSpace(out), "x"); len(dims) >= 2 {
		width, _ = strconv.ParseFloat(dims[0], 64)
		height, _ = strconv.ParseFloat(dims[1], 64)
	}
	// cwebp mantiene la proporción cuando una de las dos dimensiones es 0.
	var resize []string
	switch {
	case math.Max(width, height) <= imageMaxSize:
	case width >= height:
		resize = []string{"-resize", strconv.Itoa(imageMaxSize), "0"}
	default:
		resize = []string{"-resize", "0", strconv.Itoa(imageMaxSize)}
	}
	args := append([]string{"-quiet", "-q", imageQuality, "-metadata", "none"}, resize...)
	args = append(args, source, "-o", output)
	if _, err := p.Runner.Run(ctx, "cwebp", args...); err != nil {
		return nil, err
	}
	return os.ReadFile(output)
}
