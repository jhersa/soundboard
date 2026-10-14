package pipeline

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Corre las herramientas reales: se salta si no están instaladas (en la imagen Docker sí lo están).
func requireTools(t *testing.T, tools ...string) {
	t.Helper()
	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("falta %s en el PATH", tool)
		}
	}
}

func generate(t *testing.T, args ...string) []byte {
	t.Helper()
	out := filepath.Join(t.TempDir(), "gen"+filepath.Ext(args[len(args)-1]))
	args[len(args)-1] = out
	if output, err := exec.Command("ffmpeg", append([]string{"-loglevel", "error"}, args...)...).CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg: %v %s", err, output)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

type probe struct {
	Streams []struct {
		CodecName  string `json:"codec_name"`
		Channels   int    `json:"channels"`
		SampleRate string `json:"sample_rate"`
	} `json:"streams"`
	Format struct {
		FormatName string `json:"format_name"`
		Duration   string `json:"duration"`
	} `json:"format"`
}

func probeBytes(t *testing.T, data []byte, name string) probe {
	t.Helper()
	file := filepath.Join(t.TempDir(), name)
	os.WriteFile(file, data, 0o600)
	out, err := exec.Command("ffprobe", "-v", "error", "-show_streams", "-show_format", "-of", "json", file).Output()
	if err != nil {
		t.Fatal(err)
	}
	var result probe
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestIntegracionAudioReal(t *testing.T) {
	requireTools(t, "ffmpeg", "ffprobe")
	// Tono de 4 s en estéreo.
	source := generate(t, "-f", "lavfi", "-i", "sine=frequency=440:duration=4", "-ac", "2", "x.wav")
	fetch := func(context.Context, string) (Downloaded, error) {
		return Downloaded{Data: source, Ext: ".wav"}, nil
	}
	p := &Pipeline{Runner: ExecRunner{}, HTTP: http.DefaultClient}

	cases := []struct {
		name     string
		options  Options
		channels int
		minMs    int64
		maxMs    int64
	}{
		{"mono sin recorte", Options{}, 1, 3800, 4300},
		{"estéreo", Options{Stereo: true}, 2, 3800, 4300},
		{"recortado 1 a 2.5 s", Options{From: str("1"), To: str("2.5")}, 1, 1300, 1700},
	}
	for _, tc := range cases {
		options := tc.options
		options.URL, options.FetchUpload = "upload:uploads/x.wav", fetch
		result, err := p.Process(context.Background(), options)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		info := probeBytes(t, result.Audio, "out.ogg")
		if len(info.Streams) != 1 || info.Streams[0].CodecName != "vorbis" || info.Streams[0].Channels != tc.channels ||
			info.Streams[0].SampleRate != "44100" || !strings.Contains(info.Format.FormatName, "ogg") {
			t.Errorf("%s: %+v", tc.name, info)
		}
		seconds, _ := strconv.ParseFloat(info.Format.Duration, 64)
		if result.DurationMs < tc.minMs || result.DurationMs > tc.maxMs || int64(seconds*1000+0.5) != result.DurationMs {
			t.Errorf("%s: duración %d ms (ffprobe %v)", tc.name, result.DurationMs, info.Format.Duration)
		}
	}

	if _, err := p.Process(context.Background(), Options{URL: "upload:uploads/x.wav", FetchUpload: fetch, From: str("30")}); err == nil ||
		!strings.Contains(err.Error(), "está después del final del audio") {
		t.Errorf("from fuera de rango: %v", err)
	}
	garbage := func(context.Context, string) (Downloaded, error) {
		return Downloaded{Data: []byte("no es audio"), Ext: ".mp3"}, nil
	}
	if _, err := p.Process(context.Background(), Options{URL: "upload:uploads/x.mp3", FetchUpload: garbage}); err == nil ||
		!strings.HasPrefix(err.Error(), "ffprobe: ") {
		t.Errorf("archivo corrupto: %v", err)
	}
}

func TestIntegracionImagenReal(t *testing.T) {
	requireTools(t, "ffmpeg", "ffprobe", "cwebp")
	big := generate(t, "-f", "lavfi", "-i", "color=c=red:s=1000x500", "-frames:v", "1", "x.png")
	small := generate(t, "-f", "lavfi", "-i", "color=c=blue:s=200x100", "-frames:v", "1", "x.png")
	tall := generate(t, "-f", "lavfi", "-i", "color=c=green:s=400x800", "-frames:v", "1", "x.jpg")
	audio := generate(t, "-f", "lavfi", "-i", "sine=duration=1", "x.wav")
	p := &Pipeline{Runner: ExecRunner{}, HTTP: http.DefaultClient}

	for name, tc := range map[string]struct {
		data        []byte
		contentType string
		ext         string
		width, high int
	}{
		"png grande": {big, "image/png", ".png", 384, 192},
		"png chica":  {small, "image/png", ".png", 200, 100},
		"jpg alta":   {tall, "image/jpeg", ".jpg", 192, 384},
	} {
		image := "upload:uploads/i" + tc.ext
		result, err := p.Process(context.Background(), Options{
			URL: "upload:uploads/a.wav", ImageURL: &image,
			FetchUpload: func(_ context.Context, source string) (Downloaded, error) {
				if source == image {
					return Downloaded{Data: tc.data, ContentType: tc.contentType, Ext: tc.ext}, nil
				}
				return Downloaded{Data: audio, Ext: ".wav"}, nil
			},
		})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if result.Image.ContentType != "image/webp" || string(result.Image.Data[:4]) != "RIFF" || string(result.Image.Data[8:12]) != "WEBP" {
			t.Fatalf("%s: no es WebP: %q", name, result.Image.Data[:12])
		}
		out, err := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=width,height", "-of", "csv=s=x:p=0",
			writeTemp(t, result.Image.Data, "o.webp")).Output()
		if err != nil || strings.TrimSpace(string(out)) != strconv.Itoa(tc.width)+"x"+strconv.Itoa(tc.high) {
			t.Errorf("%s: tamaño %q (%v); se esperaba %dx%d", name, out, err, tc.width, tc.high)
		}
	}
}

func writeTemp(t *testing.T, data []byte, name string) string {
	file := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(file, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return file
}
