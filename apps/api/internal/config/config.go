// Package config lee la configuración de variables de entorno (mismos nombres que la API en TS).
package config

import (
	"bufio"
	"os"
	"strings"
)

type Config struct {
	Addr            string
	DatabasePath    string
	MediaDir        string
	GCSBucket       string
	GoogleClientIDs []string
	AdminEmails     []string
	CDNBaseURL      string
	AdminOrigin     string
	ProcessorURL    string
	ProcessorSecret string
	ServeMedia      bool
}

func Load() Config {
	return Config{
		Addr:            env("ADDR", ":8788"),
		DatabasePath:    env("DATABASE_PATH", ".data/soundboard.db"),
		MediaDir:        env("MEDIA_DIR", ".data/media"),
		GCSBucket:       os.Getenv("GCS_BUCKET"),
		GoogleClientIDs: CSV(os.Getenv("GOOGLE_CLIENT_IDS")),
		AdminEmails:     CSV(os.Getenv("ADMIN_EMAILS")),
		CDNBaseURL:      os.Getenv("CDN_BASE_URL"),
		AdminOrigin:     env("ADMIN_ORIGIN", "http://localhost:5173"),
		ProcessorURL:    env("PROCESSOR_URL", "http://localhost:8790"),
		ProcessorSecret: os.Getenv("PROCESSOR_SECRET"),
		ServeMedia:      os.Getenv("SERVE_MEDIA") == "true",
	}
}

// LoadDotEnv carga un archivo KEY=VALUE (para desarrollo local) sin pisar variables ya definidas.
func LoadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.HasPrefix(line, "#") {
			continue
		}
		key = strings.TrimSpace(key)
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, strings.Trim(strings.TrimSpace(value), `"`))
		}
	}
	return scanner.Err()
}

func CSV(value string) []string {
	var items []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}
	return items
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
