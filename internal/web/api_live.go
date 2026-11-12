package web

import (
	"bufio"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/caveman/go-rt/internal/config"
	"github.com/caveman/go-rt/internal/domain"
	"github.com/caveman/go-rt/internal/plugin"
	"github.com/caveman/go-rt/internal/telemetry"
)

type startAssessmentRequest struct {
	ConfigFile      string `json:"configFile"`
	Dataset         string `json:"dataset"`
	RawYAML         string `json:"rawYaml"`
	SkipHealthCheck bool   `json:"skipHealthCheck,omitzero"`
}

func (s *Server) handleStartAssessment(w http.ResponseWriter, r *http.Request) {
	req, cfg, err := s.extractStartAssessmentParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateTargetEndpoints(r.Context(), cfg, req.SkipHealthCheck); err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	if err := s.manager.Start(cfg, req.Dataset); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	s.respondStartedAssessment(w)
}

func (s *Server) extractStartAssessmentParams(r *http.Request) (startAssessmentRequest, *domain.Config, error) {
	req, err := decodeStartAssessmentRequest(r)
	if err != nil {
		return req, nil, err
	}
	cfg, err := s.resolveAndValidateConfig(r.Context(), req)
	if err != nil {
		return req, nil, err
	}
	return req, cfg, nil
}

func validateTargetEndpoints(ctx context.Context, cfg *domain.Config, skip bool) error {
	if skip || cfg == nil {
		return nil
	}
	results := config.CheckProvidersReachability(ctx, *cfg)
	if bad, hasOffline := config.FirstUnreachableProvider(results); hasOffline {
		return fmt.Errorf("%s '%s' (%s) unreachable: %s", bad.Role, bad.ID, bad.BaseURL, bad.Message)
	}
	return nil
}

func decodeStartAssessmentRequest(r *http.Request) (startAssessmentRequest, error) {
	var req startAssessmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		return req, fmt.Errorf("invalid request json: %w", err)
	}
	return req, nil
}

func (s *Server) respondStartedAssessment(w http.ResponseWriter) {
	writeJSON(w, http.StatusAccepted, map[string]any{
		"message": "assessment started",
		"status":  s.manager.Status(),
	})
}

func (s *Server) handleStopAssessment(w http.ResponseWriter, r *http.Request) {
	if err := s.manager.Stop(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "assessment stopped",
		"status":  s.manager.Status(),
	})
}

func (s *Server) handleGetAssessmentStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.manager.Status())
}

func (s *Server) resolveAndValidateConfig(ctx context.Context, req startAssessmentRequest) (*domain.Config, error) {
	raw, err := s.loadRawConfigBytes(req)
	if err != nil {
		return nil, err
	}
	cat := plugin.BuildCatalogSnapshot(ctx, s.db)
	rep, cfg := config.ValidateYAML(raw, cat)
	if !rep.Valid || rep.NeedsAttackerResolution {
		return nil, formatValidationFailureError(rep)
	}
	return &cfg, nil
}

func (s *Server) loadRawConfigBytes(req startAssessmentRequest) ([]byte, error) {
	if strings.TrimSpace(req.RawYAML) != "" {
		return []byte(req.RawYAML), nil
	}
	filename := cmp.Or(req.ConfigFile, "config.yaml")
	raw, err := readWorkspaceFile(s.workDir, sanitizeConfigFilename(filename))
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", filename, err)
	}
	return raw, nil
}

func formatValidationFailureError(rep config.ValidationReport) error {
	if rep.NeedsAttackerResolution {
		return fmt.Errorf("validation failed: active strategies %v require an Attacker LLM", rep.AttackerDependentStrategies)
	}
	for _, iss := range rep.Issues {
		if iss.Severity == config.IssueError {
			return fmt.Errorf("validation failed: [%s] %s (%s)", iss.Code, iss.Field, iss.Message)
		}
	}
	return errors.New("configuration validation failed: please resolve issues before running")
}

func (s *Server) handleGetLogsTail(w http.ResponseWriter, r *http.Request) {
	limit := parseLimitParam(r.URL.Query().Get("lines"), 100)
	logPath := filepath.Join(s.workDir, telemetry.DefaultLogPath)
	lines, err := readLastLines(logPath, limit)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"lines": []string{}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"lines": lines})
}

func (s *Server) handleStreamLogs(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	prepareSSEHeaders(w)
	flusher.Flush()
	logPath := filepath.Join(s.workDir, telemetry.DefaultLogPath)
	s.streamLogLoop(r.Context(), w, flusher, logPath)
}

func prepareSSEHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
}

func (s *Server) streamLogLoop(ctx context.Context, w io.Writer, flusher http.Flusher, logPath string) {
	f, err := openOrCreateLogFile(logPath)
	if err != nil {
		return
	}
	defer f.Close()

	sendInitialLogLines(w, flusher, f)
	pumpLogLinesUntilDone(ctx, w, flusher, bufio.NewReader(f))
}

func openOrCreateLogFile(logPath string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return nil, err
	}
	return os.OpenFile(logPath, os.O_RDONLY|os.O_CREATE, 0o644)
}

func pumpLogLinesUntilDone(ctx context.Context, w io.Writer, flusher http.Flusher, reader *bufio.Reader) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			readAndFlushNewLines(w, flusher, reader)
		}
	}
}

func sendInitialLogLines(w io.Writer, flusher http.Flusher, f *os.File) {
	lines, _ := readLastLinesFromFile(f, 40)
	for _, l := range lines {
		fmt.Fprintf(w, "data: %s\n\n", l)
	}
	flusher.Flush()
}

func readAndFlushNewLines(w io.Writer, flusher http.Flusher, reader *bufio.Reader) {
	var hasFlushed bool
	for {
		line, err := reader.ReadString('\n')
		if writeSSELogLine(w, line) {
			hasFlushed = true
		}
		if err != nil {
			break
		}
	}
	if hasFlushed {
		flusher.Flush()
	}
}

func writeSSELogLine(w io.Writer, line string) bool {
	if len(line) == 0 {
		return false
	}
	clean := strings.TrimRight(line, "\r\n")
	fmt.Fprintf(w, "data: %s\n\n", clean)
	return true
}

func readLastLines(path string, count int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return readLastLinesFromFile(f, count)
}

func readLastLinesFromFile(f *os.File, count int) ([]string, error) {
	lines := []string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if len(lines) > count {
		lines = lines[len(lines)-count:]
	}
	return lines, scanner.Err()
}
