// Package storage guarda los medios (sounds/, images/, uploads/). En producción es un bucket de
// Cloud Storage; en desarrollo, una carpeta local que la API sirve en /cdn.
package storage

import (
	"context"
	"io"
	"regexp"
)

type Object struct {
	Body         io.ReadCloser
	ContentType  string
	CacheControl string
	ETag         string
}

type PutOptions struct {
	ContentType  string `json:"contentType,omitempty"`
	CacheControl string `json:"cacheControl,omitempty"`
}

type Store interface {
	// Get devuelve (nil, nil) si la key no existe.
	Get(ctx context.Context, key string) (*Object, error)
	Put(ctx context.Context, key string, body []byte, options PutOptions) error
	// Delete ignora las keys que no existen.
	Delete(ctx context.Context, keys ...string) error
}

// Las keys siempre son <carpeta>/<nombre> sin rutas relativas: evita escapar del bucket o del directorio.
var validKey = regexp.MustCompile(`^(sounds|images|uploads)/[A-Za-z0-9][A-Za-z0-9._-]*$`)

func ValidKey(key string) bool { return validKey.MatchString(key) }
