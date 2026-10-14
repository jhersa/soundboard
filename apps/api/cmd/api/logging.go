package main

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// newLogger arma el logger según LOG_FORMAT (text | json) y LOG_LEVEL (debug | info | warn | error).
// En desarrollo el default es texto legible; las imágenes Docker fijan LOG_FORMAT=json, que es lo que
// entienden `docker logs | jq`, Cloud Logging o Loki sin parsear. Va a stderr para no mezclarse con
// la salida de comandos como `api backup -`.
func newLogger(w io.Writer, format, level string) *slog.Logger {
	var minimum slog.Level
	if err := minimum.UnmarshalText([]byte(strings.TrimSpace(level))); err != nil || level == "" {
		minimum = slog.LevelInfo
	}
	options := &slog.HandlerOptions{Level: minimum}
	if strings.EqualFold(strings.TrimSpace(format), "json") {
		return slog.New(slog.NewJSONHandler(w, options))
	}
	return slog.New(slog.NewTextHandler(w, options))
}

func loggerFromEnv() *slog.Logger {
	return newLogger(os.Stderr, os.Getenv("LOG_FORMAT"), os.Getenv("LOG_LEVEL"))
}
