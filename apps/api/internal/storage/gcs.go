package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
)

// GCS usa la JSON API de Cloud Storage. El cliente HTTP ya viene autenticado (en la VM, con la
// cuenta de servicio vía metadata server; ver NewGoogleClient).
type GCS struct {
	Bucket  string
	Client  *http.Client
	BaseURL string // https://storage.googleapis.com; se cambia en tests
}

func (s GCS) base() string {
	if s.BaseURL != "" {
		return strings.TrimRight(s.BaseURL, "/")
	}
	return "https://storage.googleapis.com"
}

func (s GCS) objectURL(key string) string {
	return fmt.Sprintf("%s/storage/v1/b/%s/o/%s", s.base(), url.PathEscape(s.Bucket), url.PathEscape(key))
}

func (s GCS) Get(ctx context.Context, key string) (*Object, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, s.objectURL(key)+"?alt=media", nil)
	if err != nil {
		return nil, err
	}
	response, err := s.Client.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode == http.StatusNotFound {
		response.Body.Close()
		return nil, nil
	}
	if err := check(response, "GET", key); err != nil {
		return nil, err
	}
	return &Object{
		Body:         response.Body,
		ContentType:  response.Header.Get("Content-Type"),
		CacheControl: response.Header.Get("Cache-Control"),
		ETag:         response.Header.Get("ETag"),
	}, nil
}

// Put sube con uploadType=multipart: metadatos (contentType, cacheControl) y contenido en una
// petición. El cuerpo se arma con io.MultiReader para no copiar el archivo en memoria.
func (s GCS) Put(ctx context.Context, key string, body []byte, options PutOptions) error {
	metadata, err := json.Marshal(map[string]string{
		"name":         key,
		"contentType":  options.ContentType,
		"cacheControl": options.CacheControl,
	})
	if err != nil {
		return err
	}
	boundary := multipart.NewWriter(io.Discard).Boundary()
	head := fmt.Sprintf("--%s\r\nContent-Type: application/json; charset=UTF-8\r\n\r\n%s\r\n--%s\r\nContent-Type: %s\r\n\r\n",
		boundary, metadata, boundary, orDefault(options.ContentType, "application/octet-stream"))
	tail := fmt.Sprintf("\r\n--%s--\r\n", boundary)
	payload := io.MultiReader(strings.NewReader(head), bytes.NewReader(body), strings.NewReader(tail))

	endpoint := fmt.Sprintf("%s/upload/storage/v1/b/%s/o?uploadType=multipart", s.base(), url.PathEscape(s.Bucket))
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, payload)
	if err != nil {
		return err
	}
	request.ContentLength = int64(len(head) + len(body) + len(tail))
	request.Header.Set("Content-Type", "multipart/related; boundary="+boundary)
	response, err := s.Client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return check(response, "PUT", key)
}

func (s GCS) Delete(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		request, err := http.NewRequestWithContext(ctx, http.MethodDelete, s.objectURL(key), nil)
		if err != nil {
			return err
		}
		response, err := s.Client.Do(request)
		if err != nil {
			return err
		}
		response.Body.Close()
		if response.StatusCode != http.StatusNotFound {
			if err := check(response, "DELETE", key); err != nil {
				return err
			}
		}
	}
	return nil
}

func check(response *http.Response, action, key string) error {
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return nil
	}
	detail, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
	response.Body.Close()
	return fmt.Errorf("storage %s %s falló: %d %s", action, key, response.StatusCode, detail)
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
