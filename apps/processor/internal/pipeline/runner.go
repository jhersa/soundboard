package pipeline

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Runner ejecuta una herramienta externa (yt-dlp, ffmpeg, ffprobe, cwebp) y devuelve su stdout.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) (string, error)
}

// ExecRunner usa os/exec. Los errores salen con el mismo formato que daba el processor en TS:
// "No se encontró X en el contenedor" o "X: <últimos 1500 caracteres de stderr>".
type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	var stdout, stderr bytes.Buffer
	command := exec.CommandContext(ctx, name, args...)
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", fmt.Errorf("No se encontró %s en el contenedor", name)
		}
		if ctx.Err() != nil {
			return "", fmt.Errorf("%s: cancelado (%w)", name, ctx.Err())
		}
		if detail := strings.TrimSpace(stderr.String()); detail != "" {
			return "", fmt.Errorf("%s: %s", name, lastRunes(detail, 1500))
		}
		return "", err
	}
	return stdout.String(), nil
}

func lastRunes(text string, max int) string {
	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	return string(runes[len(runes)-max:])
}
