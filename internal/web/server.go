package web

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"github.com/caveman/go-rt/internal/store"
)

//go:embed static
var staticAssets embed.FS

// Server hosts the embedded GORT Web Command Center and REST APIs.
type Server struct {
	db      *store.SQLiteStore
	workDir string
	mux     *http.ServeMux
}

// NewServer initializes the modular web dashboard server and registers routes.
func NewServer(db *store.SQLiteStore, workDir string) *Server {
	if workDir == "" {
		workDir = "."
	}
	s := &Server{
		db:      db,
		workDir: workDir,
		mux:     http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

// Handler returns the root HTTP handler for testing or custom listeners.
func (s *Server) Handler() http.Handler {
	return s.mux
}

// Start launches the HTTP server on the given address (e.g. ":8585") until ctx is canceled.
func (s *Server) Start(ctx context.Context, addr string) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           s.mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()
	return waitForServerShutdown(ctx, srv, errCh)
}

func waitForServerShutdown(ctx context.Context, srv *http.Server, errCh <-chan error) error {
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutCtx)
	}
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /api/dashboard/stats", s.handleGetDashboardStats)
	s.mux.HandleFunc("GET /api/runs", s.handleListRuns)
	s.mux.HandleFunc("GET /api/runs/{id}", s.handleGetRun)
	s.mux.HandleFunc("GET /api/runs/{id}/results", s.handleGetRunResults)
	s.mux.HandleFunc("GET /api/runs/{id}/results/{resultId}", s.handleGetResultDetail)

	s.mux.HandleFunc("GET /api/catalog", s.handleGetCatalog)
	s.mux.HandleFunc("GET /api/datasets/{name}/prompts", s.handleGetDatasetPrompts)
	s.mux.HandleFunc("GET /api/configs", s.handleListConfigs)
	s.mux.HandleFunc("GET /api/configs/load", s.handleLoadConfig)
	s.mux.HandleFunc("POST /api/configs/save", s.handleSaveConfig)
	s.mux.HandleFunc("POST /api/datasets/custom", s.handleSaveCustomDataset)
	s.registerStaticSubtree()
}

func (s *Server) registerStaticSubtree() {
	sub, err := fs.Sub(staticAssets, "static")
	if err != nil {
		return
	}
	s.mux.Handle("/", http.FileServer(http.FS(sub)))
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": fmt.Sprintf("%s", msg)})
}
