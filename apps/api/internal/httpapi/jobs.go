package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	signature "soundboard/gosig"
)

func (s *Server) setJobStatus(ctx context.Context, jobID, status string, errMessage, soundID *string) error {
	_, err := s.DB.ExecContext(ctx,
		`UPDATE jobs SET status = ?, error = ?, sound_id = COALESCE(?, sound_id),
		   updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		 WHERE id = ?`, status, errMessage, soundID, jobID)
	return err
}

// dispatchJob envía el job al processor en segundo plano. El processor responde 202 y reporta el
// avance por /v1/internal; si no se le puede contactar, el job queda como fallido.
func (s *Server) dispatchJob(job jobRow, apiBaseURL string) {
	s.background.Add(1)
	go func() {
		defer s.background.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := s.sendToProcessor(ctx, job, apiBaseURL); err != nil {
			message := "No se pudo contactar al processor: " + err.Error()
			if err := s.setJobStatus(ctx, job.ID, "failed", &message, nil); err != nil {
				s.Log.Error("no se pudo marcar el job como fallido", "job", job.ID, "err", err)
			}
		}
	}()
}

func (s *Server) sendToProcessor(ctx context.Context, job jobRow, apiBaseURL string) error {
	body, err := json.Marshal(ProcessRequest{
		JobID: job.ID, URL: job.SourceURL, Title: job.Title, ImageURL: job.ImageURL,
		From: job.TrimFrom, To: job.TrimTo, Stereo: job.Stereo == 1, APIBaseURL: apiBaseURL,
	})
	if err != nil {
		return err
	}
	endpoint := strings.TrimRight(s.Config.ProcessorURL, "/") + "/process"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	for header, value := range signature.Headers(s.Config.ProcessorSecret, body, s.now()) {
		request.Header.Set(header, value)
	}
	response, err := s.HTTP.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return fmt.Errorf("el processor respondió %d: %s", response.StatusCode, detail)
	}
	return nil
}
