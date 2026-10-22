package web

import (
	"context"
	"net/http"
	"strings"

	"github.com/caveman/go-rt/internal/domain"
)

// ExecutiveReportData represents the comprehensive audit report package.
type ExecutiveReportData struct {
	Scope            string                `json:"scope"`
	Run              *domain.EvalRun       `json:"run,omitempty"`
	Stats            domain.DashboardStats `json:"stats"`
	CriticalFindings []domain.EvalResult   `json:"criticalFindings"`
	RecentRuns       []domain.EvalRun      `json:"recentRuns"`
}

func (s *Server) handleGetExecutiveReport(w http.ResponseWriter, r *http.Request) {
	evalID := strings.TrimSpace(r.URL.Query().Get("evalId"))
	report, err := s.buildExecutiveReportData(r.Context(), evalID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to build report: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) buildExecutiveReportData(ctx context.Context, evalID string) (*ExecutiveReportData, error) {
	stats, err := s.db.GetDashboardStats(ctx, evalID)
	if err != nil {
		return nil, err
	}
	recentRuns, _ := s.db.ListEvalRuns(ctx, 20)
	targetRun := s.resolveTargetRun(ctx, evalID, stats.SelectedEvalID, recentRuns)
	findings := s.loadReportCriticalFindings(ctx, targetRun, evalID)

	return &ExecutiveReportData{
		Scope:            stats.Scope,
		Run:              targetRun,
		Stats:            *stats,
		CriticalFindings: findings,
		RecentRuns:       recentRuns,
	}, nil
}

func (s *Server) resolveTargetRun(ctx context.Context, evalID, selectedID string, recent []domain.EvalRun) *domain.EvalRun {
	if isExplicitRunID(evalID) {
		if run, err := s.db.GetEvalRun(ctx, evalID); err == nil {
			return run
		}
	}
	if selectedID != "" {
		if run, err := s.db.GetEvalRun(ctx, selectedID); err == nil {
			return run
		}
	}
	if len(recent) > 0 {
		return &recent[0]
	}
	return nil
}

func isExplicitRunID(evalID string) bool {
	return evalID != "" && evalID != "all" && evalID != "latest"
}

func (s *Server) loadReportCriticalFindings(ctx context.Context, run *domain.EvalRun, evalID string) []domain.EvalResult {
	queryEvalID := ""
	if evalID != "all" && run != nil {
		queryEvalID = run.EvalID
	}
	filter := domain.ResultFilter{
		EvalID:     queryEvalID,
		FailedOnly: true,
		Limit:      25,
	}
	results, err := s.db.QueryResults(ctx, filter)
	if err != nil || results == nil {
		return []domain.EvalResult{}
	}
	return results
}
