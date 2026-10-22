package web

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/caveman/go-rt/internal/config"
	"github.com/caveman/go-rt/internal/domain"
	"github.com/caveman/go-rt/internal/telemetry"
)

type startAssessmentRequest struct {
	ConfigFile string `json:"configFile"`
	Dataset    string `json:"dataset"`
	RawYAML    string `json:"rawYaml"`
}

func (s *Server) handleStartAssessment(w http.ResponseWriter, r *http.Request) {
	req, err := decodeStartAssessmentRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	cfg, err := s.resolveAssessmentConfig(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "config error: "+err.Error())
		return
	}
	if err := s.manager.Start(cfg, req.Dataset); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	s.respondStartedAssessment(w)
}

func decodeStartAssessmentRequest(r *http.Request) (startAssessmentRequest, error) {
	var req startAssessmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
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

func (s *Server) resolveAssessmentConfig(req startAssessmentRequest) (*domain.Config, error) {
	if strings.TrimSpace(req.RawYAML) != "" {
		cfg, err := config.ParseBytes([]byte(req.RawYAML))
		return &cfg, err
	}
	return s.loadConfigFromFile(req.ConfigFile)
}

func (s *Server) loadConfigFromFile(filename string) (*domain.Config, error) {
	if filename == "" {
		filename = "config.yaml"
	}
	path := filepath.Join(s.workDir, sanitizeConfigFilename(filename))
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", filename, err)
	}
	cfg, err := config.ParseBytes(raw)
	return &cfg, err
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
	_ = os.MkdirAll(filepath.Dir(logPath), 0o755)
	f, err := os.OpenFile(logPath, os.O_RDONLY|os.O_CREATE, 0o644)
	if err != nil {
		return
	}
	defer f.Close()

	sendInitialLogLines(w, flusher, f)
	reader := bufio.NewReader(f)
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
	flushed := false
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			clean := strings.TrimRight(line, "\r\n")
			fmt.Fprintf(w, "data: %s\n\n", clean)
			flushed = true
		}
		if err != nil {
			break
		}
	}
	if flushed {
		flusher.Flush()
	}
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
	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if len(lines) > count {
		lines = lines[len(lines)-count:]
	}
	return lines, scanner.Err()
}
