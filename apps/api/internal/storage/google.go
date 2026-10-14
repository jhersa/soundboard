package storage

import (
	"context"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// NewGoogleClient usa las Application Default Credentials: en la VM, la cuenta de servicio
// (metadata server); en local, `gcloud auth application-default login`.
func NewGoogleClient(ctx context.Context) (*http.Client, error) {
	source, err := google.DefaultTokenSource(ctx, "https://www.googleapis.com/auth/devstorage.read_write")
	if err != nil {
		return nil, err
	}
	return oauth2.NewClient(ctx, source), nil
}
