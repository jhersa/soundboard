// Processor de Soundboard.
//
//	processor [serve]    levanta el servidor (PORT, PROCESSOR_SECRET, API_BASE_URL, YTDLP_COOKIES)
//	processor healthcheck  sale con 0 si responde (HEALTHCHECK de Docker)
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"soundboard/processor/internal/pipeline"
	"soundboard/processor/internal/server"
)

func main() {
	loadDotEnv(".env")
	logger := loggerFromEnv()
	command := "serve"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	var err error
	switch command {
	case "serve":
		err = serve(logger)
	case "healthcheck":
		err = healthcheck()
	default:
		err = fmt.Errorf("comando desconocido %q (serve, healthcheck)", command)
	}
	if err != nil {
		logger.Error(command+" falló", "err", err)
		os.Exit(1)
	}
}

func port() string {
	if value := os.Getenv("PORT"); value != "" {
		return value
	}
	return "8787"
}

func serve(logger *slog.Logger) error {
	secret := os.Getenv("PROCESSOR_SECRET")
	if secret == "" {
		return errors.New("Falta PROCESSOR_SECRET")
	}
	client := &http.Client{Timeout: 5 * time.Minute}
	processor := server.New(server.Config{
		Secret:       secret,
		APIBaseURL:   os.Getenv("API_BASE_URL"),
		YtDlpCookies: os.Getenv("YTDLP_COOKIES"),
	}, &pipeline.Pipeline{Runner: pipeline.ExecRunner{}, HTTP: client}, client, logger)

	signals, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	jobCtx, cancelJobs := context.WithCancel(context.Background())
	defer cancelJobs()
	workerDone := make(chan struct{})
	go func() { processor.Work(jobCtx); close(workerDone) }()

	httpServer := &http.Server{
		Addr:              ":" + port(),
		Handler:           processor.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
	}
	errs := make(chan error, 1)
	go func() { errs <- httpServer.ListenAndServe() }()
	logger.Info("processor escuchando", "addr", httpServer.Addr)

	select {
	case err := <-errs:
		return err
	case <-signals.Done():
	}

	// Apagado limpio: se cierra el HTTP, se deja terminar el job en curso (hasta 30 s) y los que
	// seguían en cola se reportan como fallidos para que no queden "en cola" para siempre.
	logger.Info("apagando", "pendientes", processor.Pending())
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	httpServer.Shutdown(shutdown)
	processor.Stop()
	select {
	case <-workerDone:
	case <-time.After(30 * time.Second):
		logger.Warn("el job en curso no terminó a tiempo; se cancela")
		cancelJobs()
		<-workerDone
	}
	return nil
}

func healthcheck() error {
	client := http.Client{Timeout: 3 * time.Second}
	response, err := client.Get("http://127.0.0.1:" + port() + "/health")
	if err != nil {
		return err
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("estado %d", response.StatusCode)
	}
	return nil
}

// loadDotEnv carga un archivo KEY=VALUE (desarrollo local) sin pisar variables ya definidas.
func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.HasPrefix(line, "#") {
			continue
		}
		if key = strings.TrimSpace(key); os.Getenv(key) == "" {
			os.Setenv(key, strings.Trim(strings.TrimSpace(value), `"`))
		}
	}
}
