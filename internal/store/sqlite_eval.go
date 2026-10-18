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
	if f.FailedOnly {
		tx = tx.Where("pass = ?", false)
	}
	if f.Limit > 0 {
		tx = tx.Limit(f.Limit)
	}
	return tx
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
