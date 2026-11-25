package web

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/bkelceoglu/go-rt/internal/domain"
)

func (s *Server) handleGetDashboardStats(w http.ResponseWriter, r *http.Request) {
	evalID := strings.TrimSpace(r.URL.Query().Get("evalId"))
	stats, err := s.db.GetDashboardStats(r.Context(), evalID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get dashboard stats: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) handleListRuns(w http.ResponseWriter, r *http.Request) {
	limit := parseLimitParam(r.URL.Query().Get("limit"), 50)
	runs, err := s.db.ListEvalRuns(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list eval runs: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"runs": ensureNonNilSlice(runs)})
}

func (s *Server) handleGetRun(w http.ResponseWriter, r *http.Request) {
	evalID := strings.TrimSpace(r.PathValue("id"))
	if evalID == "" {
		writeError(w, http.StatusBadRequest, "eval id required")
		return
	}
	run, err := s.db.GetEvalRun(r.Context(), evalID)
	if err != nil {
		writeError(w, http.StatusNotFound, "eval run not found: "+evalID)
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (s *Server) handleGetRunResults(w http.ResponseWriter, r *http.Request) {
	evalID := strings.TrimSpace(r.PathValue("id"))
	filter := buildResultFilter(evalID, r)
	results, err := s.db.QueryResults(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query results: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"evalId": evalID, "results": ensureNonNilSlice(results)})
}

func (s *Server) handleGetResultDetail(w http.ResponseWriter, r *http.Request) {
	resultID := strings.TrimSpace(r.PathValue("resultId"))
	result, err := s.db.GetEvalResult(r.Context(), resultID)
	if err != nil {
		writeError(w, http.StatusNotFound, "result not found: "+resultID)
		return
	}
	spans, _ := s.db.GetSpansByTrace(r.Context(), result.TraceID)
	writeJSON(w, http.StatusOK, map[string]any{
		"result": result,
		"spans":  ensureNonNilSlice(spans),
	})
}

func buildResultFilter(evalID string, r *http.Request) domain.ResultFilter {
	q := r.URL.Query()
	return domain.ResultFilter{
		EvalID:     evalID,
		PluginID:   q.Get("plugin"),
		Category:   q.Get("category"),
		StrategyID: q.Get("strategy"),
		FailedOnly: q.Get("failedOnly") == "true" || q.Get("failed") == "true",
		Limit:      parseLimitParam(q.Get("limit"), 500),
	}
}

func parseLimitParam(raw string, defaultLimit int) int {
	if raw == "" {
		return defaultLimit
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return defaultLimit
	}
	return v
}
