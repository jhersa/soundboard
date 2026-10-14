package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestLoggerJSONYNivel(t *testing.T) {
	var out bytes.Buffer
	logger := newLogger(&out, "JSON", "warn")
	logger.Info("no debe salir")
	logger.Warn("sí sale", "job", "j1", "ms", 12)
	var line map[string]any
	if err := json.Unmarshal(out.Bytes(), &line); err != nil {
		t.Fatalf("no es JSON de una sola línea: %q (%v)", out.String(), err)
	}
	if line["level"] != "WARN" || line["msg"] != "sí sale" || line["job"] != "j1" || line["ms"] != float64(12) || line["time"] == nil {
		t.Fatalf("campos inesperados: %v", line)
	}
}

func TestLoggerTextoPorDefecto(t *testing.T) {
	for _, format := range []string{"", "text", "otra-cosa"} {
		var out bytes.Buffer
		newLogger(&out, format, "").Info("hola", "k", "v")
		if !strings.Contains(out.String(), "level=INFO") || !strings.Contains(out.String(), "k=v") {
			t.Errorf("formato %q: %q", format, out.String())
		}
	}
	var out bytes.Buffer
	newLogger(&out, "", "basura").Debug("oculto")
	newLogger(&out, "", "basura").Info("visible")
	if strings.Contains(out.String(), "oculto") || !strings.Contains(out.String(), "visible") {
		t.Errorf("un nivel inválido debe caer a info: %q", out.String())
	}
	out.Reset()
	newLogger(&out, "", "debug").Debug("detalle")
	if !strings.Contains(out.String(), "detalle") {
		t.Errorf("debug debía salir: %q", out.String())
	}
}
