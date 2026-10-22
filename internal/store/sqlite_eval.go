package store

import (
	"context"
	"fmt"

	"github.com/caveman/go-rt/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SaveEvalRun inserts or updates an assessment session run in SQLite via GORM.
func (s *SQLiteStore) SaveEvalRun(ctx context.Context, run domain.EvalRun) error {
	model := toEvalRunModel(run)
	err := s.db.WithContext(ctx).
		Clauses(clause.OnConflict{UpdateAll: true}).
		Create(&model).Error
	if err != nil {
		return fmt.Errorf("save eval run %s: %w", run.EvalID, err)
	}
	return nil
}

// SaveEvalResult persists a single test evaluation outcome with trace linkage via GORM.
func (s *SQLiteStore) SaveEvalResult(ctx context.Context, res domain.EvalResult) error {
	model := toEvalResultModel(res)
	err := s.db.WithContext(ctx).
		Clauses(clause.OnConflict{UpdateAll: true}).
		Create(&model).Error
	if err != nil {
		return fmt.Errorf("save eval result %s: %w", res.ResultID, err)
	}
	return nil
}

// SaveOTelSpan stores an OpenTelemetry GenAI span record in SQLite via GORM.
func (s *SQLiteStore) SaveOTelSpan(ctx context.Context, span domain.OTelSpanRecord) error {
	model := toOTelSpanModel(span)
	err := s.db.WithContext(ctx).
		Clauses(clause.OnConflict{UpdateAll: true}).
		Create(&model).Error
	if err != nil {
		return fmt.Errorf("save otel span %s: %w", span.SpanID, err)
	}
	return nil
}

// QueryResults retrieves evaluation results filtered by criteria via GORM.
func (s *SQLiteStore) QueryResults(ctx context.Context, f domain.ResultFilter) ([]domain.EvalResult, error) {
	var models []EvalResultModel
	tx := applyResultFilter(s.db.WithContext(ctx).Model(&EvalResultModel{}), f)
	if err := tx.Order("created_at ASC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("query eval results: %w", err)
	}
	return mapModelsToEvalResults(models), nil
}

func applyResultFilter(tx *gorm.DB, f domain.ResultFilter) *gorm.DB {
	if f.EvalID != "" {
		tx = tx.Where("eval_id = ?", f.EvalID)
	}
	if f.PluginID != "" {
		tx = tx.Where("plugin_id = ?", f.PluginID)
	}
	if f.Category != "" {
		tx = tx.Where("category = ?", f.Category)
	}
	if f.StrategyID != "" {
		tx = tx.Where("strategy_id = ?", f.StrategyID)
	}
	if f.FailedOnly {
		tx = tx.Where("pass = ?", false)
	}
	if f.Limit > 0 {
		tx = tx.Limit(f.Limit)
	}
	return tx
}

// GetEvalRun retrieves a single assessment run by evalID via GORM.
func (s *SQLiteStore) GetEvalRun(ctx context.Context, evalID string) (*domain.EvalRun, error) {
	var model EvalRunModel
	if err := s.db.WithContext(ctx).Where("eval_id = ?", evalID).First(&model).Error; err != nil {
		return nil, fmt.Errorf("get eval run %s: %w", evalID, err)
	}
	runs := mapModelsToEvalRuns([]EvalRunModel{model})
	return &runs[0], nil
}

// GetEvalResult retrieves a single evaluation test result by resultID via GORM.
func (s *SQLiteStore) GetEvalResult(ctx context.Context, resultID string) (*domain.EvalResult, error) {
	var model EvalResultModel
	if err := s.db.WithContext(ctx).Where("result_id = ?", resultID).First(&model).Error; err != nil {
		return nil, fmt.Errorf("get eval result %s: %w", resultID, err)
	}
	results := mapModelsToEvalResults([]EvalResultModel{model})
	return &results[0], nil
}

// GetSpansByTrace retrieves all OTel span records for a trace ID via GORM.
func (s *SQLiteStore) GetSpansByTrace(ctx context.Context, traceID string) ([]domain.OTelSpanRecord, error) {
	var models []OTelSpanModel
	if err := s.db.WithContext(ctx).Where("trace_id = ?", traceID).Order("start_time_unix_nano ASC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("get spans by trace %s: %w", traceID, err)
	}
	out := make([]domain.OTelSpanRecord, 0, len(models))
	for _, m := range models {
		out = append(out, domain.OTelSpanRecord{
			SpanID:            m.SpanID,
			TraceID:           m.TraceID,
			ParentSpanID:      m.ParentSpanID,
			EvalID:            m.EvalID,
			ResultID:          m.ResultID,
			Name:              m.Name,
			Kind:              m.Kind,
			StartTimeUnixNano: m.StartTimeUnixNano,
			EndTimeUnixNano:   m.EndTimeUnixNano,
			StatusCode:        m.StatusCode,
			Attributes:        m.Attributes,
			Events:            m.Events,
		})
	}
	return out, nil
}

// GetDashboardStats aggregates executive posture KPIs across all or specific stored assessments.
func (s *SQLiteStore) GetDashboardStats(ctx context.Context, evalID string) (*domain.DashboardStats, error) {
	runs, err := s.ListEvalRuns(ctx, 100)
	if err != nil {
		return nil, err
	}
	targetEvalID, runsToCompute, scope := resolveDashboardTarget(runs, evalID)
	stats := computeRunTotals(runsToCompute)
	stats.Scope = scope
	stats.SelectedEvalID = targetEvalID
	stats.TotalRuns = int64(len(runs))
	if err := s.populateResultBreakdowns(ctx, stats, targetEvalID); err != nil {
		return nil, err
	}
	stats.RecentRuns = runs
	return stats, nil
}

func resolveDashboardTarget(runs []domain.EvalRun, evalID string) (string, []domain.EvalRun, string) {
	if len(runs) == 0 {
		return "", nil, "empty"
	}
	if evalID == "all" {
		return "", runs, "all"
	}
	if evalID == "" || evalID == "latest" {
		return runs[0].EvalID, []domain.EvalRun{runs[0]}, "latest"
	}
	for _, r := range runs {
		if r.EvalID == evalID {
			return r.EvalID, []domain.EvalRun{r}, evalID
		}
	}
	return runs[0].EvalID, []domain.EvalRun{runs[0]}, "latest"
}

func computeRunTotals(runs []domain.EvalRun) *domain.DashboardStats {
	stats := &domain.DashboardStats{
		TotalRuns:     int64(len(runs)),
		StrategyStats: make(map[string]domain.StrategyBypassStat),
		SeverityStats: make(map[string]int64),
	}
	var totalDuration int64
	for _, r := range runs {
		stats.TotalTests += int64(r.TotalTests)
		stats.PassedTests += int64(r.PassedTests)
		stats.FailedTests += int64(r.FailedTests)
		stats.TotalTokens += int64(r.TotalTokens)
		totalDuration += r.DurationMs
	}
	if stats.TotalTests > 0 {
		stats.PassRate = float64(stats.PassedTests) / float64(stats.TotalTests) * 100.0
	}
	if len(runs) > 0 {
		stats.AvgDurationMs = totalDuration / int64(len(runs))
	}
	return stats
}

type strategyRow struct {
	StrategyID string `gorm:"column:strategy_id"`
	Total      int64  `gorm:"column:total"`
	Failed     int64  `gorm:"column:failed"`
}

type severityRow struct {
	Severity string `gorm:"column:severity"`
	Count    int64  `gorm:"column:count"`
}

func (s *SQLiteStore) populateResultBreakdowns(ctx context.Context, stats *domain.DashboardStats, targetEvalID string) error {
	if err := s.queryStrategyBreakdown(ctx, stats, targetEvalID); err != nil {
		return err
	}
	if err := s.querySeverityBreakdown(ctx, stats, targetEvalID); err != nil {
		return err
	}
	return s.queryTopFailingPlugins(ctx, stats, targetEvalID)
}

func (s *SQLiteStore) queryStrategyBreakdown(ctx context.Context, stats *domain.DashboardStats, targetEvalID string) error {
	var stratRows []strategyRow
	tx := s.db.WithContext(ctx).Model(&EvalResultModel{}).
		Select("strategy_id, count(*) as total, sum(case when pass = 0 then 1 else 0 end) as failed")
	if targetEvalID != "" {
		tx = tx.Where("eval_id = ?", targetEvalID)
	}
	if err := tx.Group("strategy_id").Scan(&stratRows).Error; err != nil {
		return err
	}
	for _, sr := range stratRows {
		bypassRate := 0.0
		if sr.Total > 0 {
			bypassRate = float64(sr.Failed) / float64(sr.Total) * 100.0
		}
		stats.StrategyStats[sr.StrategyID] = domain.StrategyBypassStat{
			Total:      sr.Total,
			Passed:     sr.Total - sr.Failed,
			Failed:     sr.Failed,
			BypassRate: bypassRate,
		}
	}
	return nil
}

func (s *SQLiteStore) querySeverityBreakdown(ctx context.Context, stats *domain.DashboardStats, targetEvalID string) error {
	var sevRows []severityRow
	tx := s.db.WithContext(ctx).Model(&EvalResultModel{}).
		Select("severity, count(*) as count").
		Where("pass = ?", false)
	if targetEvalID != "" {
		tx = tx.Where("eval_id = ?", targetEvalID)
	}
	if err := tx.Group("severity").Scan(&sevRows).Error; err != nil {
		return err
	}
	for _, svr := range sevRows {
		stats.SeverityStats[svr.Severity] = svr.Count
	}
	return nil
}

func (s *SQLiteStore) queryTopFailingPlugins(ctx context.Context, stats *domain.DashboardStats, targetEvalID string) error {
	var topPlugins []domain.PluginFailureStat
	tx := s.db.WithContext(ctx).Model(&EvalResultModel{}).
		Select("category, plugin_id, count(*) as failures").
		Where("pass = ?", false)
	if targetEvalID != "" {
		tx = tx.Where("eval_id = ?", targetEvalID)
	}
	if err := tx.Group("category, plugin_id").Order("failures DESC").Limit(10).Scan(&topPlugins).Error; err != nil {
		return err
	}
	stats.TopFailingPlugins = topPlugins
	return nil
}

// ListEvalRuns returns recent evaluation run summaries via GORM.
func (s *SQLiteStore) ListEvalRuns(ctx context.Context, limit int) ([]domain.EvalRun, error) {
	if limit <= 0 {
		limit = 50
	}
	var models []EvalRunModel
	err := s.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(limit).
		Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("list eval runs: %w", err)
	}
	return mapModelsToEvalRuns(models), nil
}

func toEvalRunModel(run domain.EvalRun) EvalRunModel {
	return EvalRunModel{
		EvalID:          run.EvalID,
		TraceID:         run.TraceID,
		Description:     run.Description,
		Purpose:         run.Purpose,
		TargetProviders: run.TargetProviders,
		JudgeProvider:   run.JudgeProvider,
		TotalTests:      run.TotalTests,
		PassedTests:     run.PassedTests,
		FailedTests:     run.FailedTests,
		TotalTokens:     run.TotalTokens,
		DurationMs:      run.DurationMs,
		CreatedAt:       run.CreatedAt.UTC(),
	}
}

func mapModelsToEvalRuns(models []EvalRunModel) []domain.EvalRun {
	out := make([]domain.EvalRun, 0, len(models))
	for _, m := range models {
		out = append(out, domain.EvalRun{
			EvalID:          m.EvalID,
			TraceID:         m.TraceID,
			Description:     m.Description,
			Purpose:         m.Purpose,
			TargetProviders: m.TargetProviders,
			JudgeProvider:   m.JudgeProvider,
			TotalTests:      m.TotalTests,
			PassedTests:     m.PassedTests,
			FailedTests:     m.FailedTests,
			TotalTokens:     m.TotalTokens,
			DurationMs:      m.DurationMs,
			CreatedAt:       m.CreatedAt,
		})
	}
	return out
}

func toEvalResultModel(res domain.EvalResult) EvalResultModel {
	return EvalResultModel{
		ResultID:            res.ResultID,
		EvalID:              res.EvalID,
		TraceID:             res.TraceID,
		SpanID:              res.SpanID,
		PluginID:            res.PluginID,
		Category:            res.Category,
		AttackType:          res.AttackType,
		Source:              res.Source,
		StrategyID:          res.StrategyID,
		Severity:            res.Severity,
		InjectVar:           res.InjectVar,
		RawPrompt:           res.RawPrompt,
		RenderedPrompt:      res.RenderedPrompt,
		TargetOutput:        res.TargetOutput,
		Pass:                res.Pass,
		Score:               res.Score,
		FailureReason:       res.FailureReason,
		JudgeReason:         res.JudgeReason,
		JudgeRubric:         res.JudgeRubric,
		LatencyMs:           res.LatencyMs,
		TokensPrompt:        res.TokenUsage.Prompt,
		TokensCompletion:    res.TokenUsage.Completion,
		TokensTotal:         res.TokenUsage.Total,
		Cached:              res.Cached,
		ConversationHistory: res.ConversationHistory,
		OTelAttributes:      res.OTelAttributes,
		CreatedAt:           res.CreatedAt.UTC(),
	}
}

func mapModelsToEvalResults(models []EvalResultModel) []domain.EvalResult {
	out := make([]domain.EvalResult, 0, len(models))
	for _, m := range models {
		out = append(out, domain.EvalResult{
			ResultID:            m.ResultID,
			EvalID:              m.EvalID,
			TraceID:             m.TraceID,
			SpanID:              m.SpanID,
			PluginID:            m.PluginID,
			Category:            m.Category,
			AttackType:          m.AttackType,
			Source:              m.Source,
			StrategyID:          m.StrategyID,
			Severity:            m.Severity,
			InjectVar:           m.InjectVar,
			RawPrompt:           m.RawPrompt,
			RenderedPrompt:      m.RenderedPrompt,
			TargetOutput:        m.TargetOutput,
			Pass:                m.Pass,
			Score:               m.Score,
			FailureReason:       m.FailureReason,
			JudgeReason:         m.JudgeReason,
			JudgeRubric:         m.JudgeRubric,
			LatencyMs:           m.LatencyMs,
			TokenUsage:          domain.TokenUsage{Prompt: m.TokensPrompt, Completion: m.TokensCompletion, Total: m.TokensTotal},
			Cached:              m.Cached,
			ConversationHistory: m.ConversationHistory,
			OTelAttributes:      m.OTelAttributes,
			CreatedAt:           m.CreatedAt,
		})
	}
	return out
}

func toOTelSpanModel(span domain.OTelSpanRecord) OTelSpanModel {
	return OTelSpanModel{
		SpanID:            span.SpanID,
		TraceID:           span.TraceID,
		ParentSpanID:      span.ParentSpanID,
		EvalID:            span.EvalID,
		ResultID:          span.ResultID,
		Name:              span.Name,
		Kind:              span.Kind,
		StartTimeUnixNano: span.StartTimeUnixNano,
		EndTimeUnixNano:   span.EndTimeUnixNano,
		StatusCode:        span.StatusCode,
		Attributes:        span.Attributes,
		Events:            span.Events,
	}
}
