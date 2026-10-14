package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// FS guarda cada objeto como archivo y sus metadatos HTTP en un .meta.json al lado.
type FS struct{ Dir string }

func (s FS) path(key string) (string, error) {
	if !ValidKey(key) {
		return "", fmt.Errorf("key inválida: %q", key)
	}
	return filepath.Join(s.Dir, filepath.FromSlash(key)), nil
}

func (s FS) Get(_ context.Context, key string) (*Object, error) {
	path, err := s.path(key)
	if err != nil {
		return nil, nil
	}
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var meta PutOptions
	if raw, err := os.ReadFile(path + ".meta.json"); err == nil {
		json.Unmarshal(raw, &meta)
	}
	return &Object{Body: file, ContentType: meta.ContentType, CacheControl: meta.CacheControl}, nil
}

func (s FS) Put(_ context.Context, key string, body []byte, options PutOptions) error {
	path, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	meta, _ := json.Marshal(options)
	if err := writeAtomic(path+".meta.json", meta); err != nil {
		return err
	}
	return writeAtomic(path, body)
}

func (s FS) Delete(_ context.Context, keys ...string) error {
	for _, key := range keys {
		path, err := s.path(key)
		if err != nil {
			continue
		}
		for _, file := range []string{path, path + ".meta.json"} {
			if err := os.Remove(file); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
	}
	return nil
}

func writeAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
