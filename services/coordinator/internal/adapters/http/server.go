package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"traffic-coordinator/internal/config"
	"traffic-coordinator/internal/core/domain"
)

var (
	ErrRunHTTPServer = errors.New("failed to run HTTP server")
)

type Service interface {
	Sources(zoneID string) []domain.Source
	Assignments(zoneID, clusterID, instanceID, dataClass string) []domain.Source
	UpsertWorkerStatus(status domain.WorkerStatusSnapshot)
	ListWorkerStatuses() []domain.WorkerStatusSnapshot
	IngestionInstances(zoneID string) []domain.IngestionInstance
}

type Server struct {
	service Service
}

func New(service Service) *Server {
	return &Server{service: service}
}

func (s *Server) Run(ctx context.Context, cfg config.ServerConfig) error {
	srv := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: s.routes(),
	}

	go func() {
		<-ctx.Done()
		shCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shCtx)
	}()

	slog.Info("coordinator starting", "listen", cfg.ListenAddr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("%w: %w", ErrRunHTTPServer, err)
	}

	return nil
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc(healthPath, s.handleHealth)
	mux.HandleFunc(sourcesPath, s.handleSources)
	mux.HandleFunc(assignmentsPath, s.handleAssignments)
	mux.HandleFunc(workerStatusUpsertPath, s.handleWorkerStatusUpsert)
	mux.HandleFunc(workersPath, s.handleWorkers)
	mux.HandleFunc(ingestionInstancesPath, s.handleIngestionInstances)

	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
