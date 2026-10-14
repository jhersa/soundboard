// Comando de la API de Soundboard.
//
//	api serve            levanta la API (aplica migraciones pendientes al arrancar)
//	api migrate          solo aplica las migraciones
//	api backup <destino> copia consistente de la base (VACUUM INTO); con "-" la escribe en stdout
//	api sql <archivo>    ejecuta un .sql en una transacción (p. ej. la siembra v1)
//	api healthcheck      sale con 0 si la API responde (HEALTHCHECK de Docker)
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"soundboard/api/internal/config"
	"soundboard/api/internal/db"
	"soundboard/api/internal/google"
	"soundboard/api/internal/httpapi"
	"soundboard/api/internal/storage"
	"soundboard/api/migrations"
)

func main() {
	// En desarrollo se leen las variables de .env; en Docker vienen del entorno.
	config.LoadDotEnv(".env")
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	command := "serve"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	var err error
	switch command {
	case "serve":
		err = serve(logger)
	case "migrate":
		err = withDB(func(database *sql.DB) error { return migrate(logger, database) })
	case "backup":
		err = withArg("backup <destino|->", func(dest string) error {
			return withDB(func(database *sql.DB) error { return backup(database, dest) })
		})
	case "sql":
		err = withArg("sql <archivo>", func(path string) error {
			script, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return withDB(func(database *sql.DB) error { return execScript(database, string(script)) })
		})
	case "healthcheck":
		err = healthcheck()
	default:
		err = fmt.Errorf("comando desconocido %q (serve, migrate, backup, sql, healthcheck)", command)
	}
	if err != nil {
		logger.Error(command+" falló", "err", err)
		os.Exit(1)
	}
}

func serve(logger *slog.Logger) error {
	cfg := config.Load()
	if cfg.ProcessorSecret == "" {
		return errors.New("falta PROCESSOR_SECRET")
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	database, err := db.Open(cfg.DatabasePath)
	if err != nil {
		return err
	}
	defer database.Close()
	if err := migrate(logger, database); err != nil {
		return err
	}

	var store storage.Store = storage.FS{Dir: cfg.MediaDir}
	if cfg.GCSBucket != "" {
		client, err := storage.NewGoogleClient(ctx)
		if err != nil {
			return fmt.Errorf("credenciales de Google Cloud: %w", err)
		}
		store = storage.GCS{Bucket: cfg.GCSBucket, Client: client}
		logger.Info("medios en Cloud Storage", "bucket", cfg.GCSBucket)
	} else {
		logger.Info("medios en disco", "dir", cfg.MediaDir)
	}

	server := &httpapi.Server{
		Config: cfg,
		DB:     database,
		Store:  store,
		Google: google.NewVerifier(ctx),
		HTTP:   &http.Client{Timeout: 30 * time.Second},
		Log:    logger,
	}
	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       5 * time.Minute,
		WriteTimeout:      5 * time.Minute,
		IdleTimeout:       2 * time.Minute,
	}
	errs := make(chan error, 1)
	go func() { errs <- httpServer.ListenAndServe() }()
	logger.Info("API escuchando", "addr", cfg.Addr)

	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
	}
	logger.Info("apagando")
	shutdown, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	err = httpServer.Shutdown(shutdown)
	server.Wait()
	return err
}

func migrate(logger *slog.Logger, database *sql.DB) error {
	applied, err := db.Migrate(context.Background(), database, migrations.FS)
	for _, name := range applied {
		logger.Info("migración aplicada", "name", name)
	}
	return err
}

func withDB(run func(*sql.DB) error) error {
	database, err := db.Open(config.Load().DatabasePath)
	if err != nil {
		return err
	}
	defer database.Close()
	return run(database)
}

func withArg(usage string, run func(string) error) error {
	if len(os.Args) < 3 {
		return fmt.Errorf("uso: api %s", usage)
	}
	return run(os.Args[2])
}

// backup usa VACUUM INTO, que da una copia consistente aunque la API esté escribiendo.
func backup(database *sql.DB, dest string) error {
	if dest != "-" {
		_, err := database.Exec("VACUUM INTO ?", dest)
		return err
	}
	dir, err := os.MkdirTemp("", "backup")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "soundboard.db")
	if _, err := database.Exec("VACUUM INTO ?", path); err != nil {
		return err
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(os.Stdout, file)
	return err
}

func execScript(database *sql.DB, script string) error {
	tx, err := database.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(script); err != nil {
		return err
	}
	return tx.Commit()
}

func healthcheck() error {
	_, port, err := net.SplitHostPort(config.Load().Addr)
	if err != nil {
		return err
	}
	client := http.Client{Timeout: 3 * time.Second}
	response, err := client.Get("http://127.0.0.1:" + port + "/")
	if err != nil {
		return err
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("estado %d", response.StatusCode)
	}
	return nil
}
