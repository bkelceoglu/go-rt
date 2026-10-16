package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/caveman/go-rt/internal/domain"
)

// SaveEvalRun inserts or updates an assessment session run in SQLite.
func (s *SQLiteStore) SaveEvalRun(ctx context.Context, run domain.EvalRun) error {
	targetsJSON, _ := json.Marshal(run.TargetProviders)
	createdAt := run.CreatedAt.UTC().Format(time.RFC3339)
	query := `INSERT INTO eval_runs (
		eval_id, trace_id, description, purpose, target_providers_json, judge_provider,
		total_tests, passed_tests, failed_tests, total_tokens, duration_ms, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(eval_id) DO UPDATE SET
		total_tests = excluded.total_tests,
		passed_tests = excluded.passed_tests,
		failed_tests = excluded.failed_tests,
		total_tokens = excluded.total_tokens,
		duration_ms = excluded.duration_ms`
	_, err := s.db.ExecContext(ctx, query,
		run.EvalID, run.TraceID, run.Description, run.Purpose, string(targetsJSON), run.JudgeProvider,
		run.TotalTests, run.PassedTests, run.FailedTests, run.TotalTokens, run.DurationMs, createdAt)
	if err != nil {
		return fmt.Errorf("save eval run %s: %w", run.EvalID, err)
	}
	return nil
}

// SaveEvalResult persists a single test evaluation outcome with trace linkage.
func (s *SQLiteStore) SaveEvalResult(ctx context.Context, res domain.EvalResult) error {
	convJSON, _ := json.Marshal(res.ConversationHistory)
	otelJSON, _ := json.Marshal(res.OTelAttributes)
	passedInt := 0
	if res.Pass {
		passedInt = 1
	}
	cachedInt := 0
	if res.Cached {
		cachedInt = 1
	}
	createdAt := res.CreatedAt.UTC().Format(time.RFC3339)
	query := `INSERT INTO eval_results (
		result_id, eval_id, trace_id, span_id, plugin_id, category, attack_type, source,
		strategy_id, severity, inject_var, raw_prompt, rendered_prompt, target_output,
		pass, score, failure_reason, judge_reason, judge_rubric, latency_ms,
		tokens_prompt, tokens_completion, tokens_total, cached, conversation_json, otel_attributes_json, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, query,
		res.ResultID, res.EvalID, res.TraceID, res.SpanID, res.PluginID, res.Category, res.AttackType, res.Source,
		res.StrategyID, string(res.Severity), res.InjectVar, res.RawPrompt, res.RenderedPrompt, res.TargetOutput,
		passedInt, res.Score, res.FailureReason, res.JudgeReason, res.JudgeRubric, res.LatencyMs,
		res.TokenUsage.Prompt, res.TokenUsage.Completion, res.TokenUsage.Total, cachedInt,
		string(convJSON), string(otelJSON), createdAt)
	if err != nil {
		return fmt.Errorf("save eval result %s: %w", res.ResultID, err)
	}
	return nil
}

// SaveOTelSpan stores an OpenTelemetry GenAI span record in SQLite.
func (s *SQLiteStore) SaveOTelSpan(ctx context.Context, span domain.OTelSpanRecord) error {
	attrsJSON, _ := json.Marshal(span.Attributes)
	eventsJSON, _ := json.Marshal(span.Events)
	query := `INSERT INTO otel_spans (
		span_id, trace_id, parent_span_id, eval_id, result_id, name, kind,
		start_time_unix_nano, end_time_unix_nano, status_code, attributes_json, events_json
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, query,
		span.SpanID, span.TraceID, span.ParentSpanID, span.EvalID, span.ResultID, span.Name, string(span.Kind),
		span.StartTimeUnixNano, span.EndTimeUnixNano, span.StatusCode, string(attrsJSON), string(eventsJSON))
	if err != nil {
		return fmt.Errorf("save otel span %s: %w", span.SpanID, err)
	}
	return nil
}

// QueryResults retrieves evaluation results filtered by criteria.
func (s *SQLiteStore) QueryResults(ctx context.Context, f domain.ResultFilter) ([]domain.EvalResult, error) {
	query, args := buildResultQuery(f)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query eval results: %w", err)
	}
	defer rows.Close()
	return scanResultRows(rows)
}

func buildResultQuery(f domain.ResultFilter) (string, []any) {
	sb := strings.Builder{}
	sb.WriteString(`SELECT
		result_id, eval_id, trace_id, span_id, plugin_id, category, attack_type, source,
		strategy_id, severity, inject_var, raw_prompt, rendered_prompt, target_output,
		pass, score, failure_reason, judge_reason, judge_rubric, latency_ms,
		tokens_prompt, tokens_completion, tokens_total, cached, conversation_json, otel_attributes_json, created_at
	FROM eval_results WHERE 1=1`)
	var args []any
	if f.EvalID != "" {
		sb.WriteString(" AND eval_id = ?")
		args = append(args, f.EvalID)
	}
	if f.PluginID != "" {
		sb.WriteString(" AND plugin_id = ?")
		args = append(args, f.PluginID)
	}
	if f.FailedOnly {
		sb.WriteString(" AND pass = 0")
	}
	sb.WriteString(" ORDER BY created_at ASC")
	if f.Limit > 0 {
		sb.WriteString(fmt.Sprintf(" LIMIT %d", f.Limit))
	}
	return sb.String(), args
}

func scanResultRows(rows *sql.Rows) ([]domain.EvalResult, error) {
	var list []domain.EvalResult
	for rows.Next() {
		res, err := scanSingleResultRow(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, res)
	}
	return list, rows.Err()
}

func scanSingleResultRow(rows *sql.Rows) (domain.EvalResult, error) {
	var res domain.EvalResult
	var passedInt, cachedInt int
	var sevStr, convJSON, otelJSON, createdStr string
	err := rows.Scan(
		&res.ResultID, &res.EvalID, &res.TraceID, &res.SpanID, &res.PluginID, &res.Category, &res.AttackType, &res.Source,
		&res.StrategyID, &sevStr, &res.InjectVar, &res.RawPrompt, &res.RenderedPrompt, &res.TargetOutput,
		&passedInt, &res.Score, &res.FailureReason, &res.JudgeReason, &res.JudgeRubric, &res.LatencyMs,
		&res.TokenUsage.Prompt, &res.TokenUsage.Completion, &res.TokenUsage.Total, &cachedInt,
		&convJSON, &otelJSON, &createdStr)
	if err != nil {
		return res, err
	}
	res.Pass = (passedInt == 1)
	res.Cached = (cachedInt == 1)
	res.Severity = domain.Severity(sevStr)
	_ = json.Unmarshal([]byte(convJSON), &res.ConversationHistory)
	_ = json.Unmarshal([]byte(otelJSON), &res.OTelAttributes)
	res.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	return res, nil
}

// ListEvalRuns returns recent evaluation run summaries.
func (s *SQLiteStore) ListEvalRuns(ctx context.Context, limit int) ([]domain.EvalRun, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `SELECT eval_id, trace_id, description, purpose, target_providers_json, judge_provider,
		total_tests, passed_tests, failed_tests, total_tokens, duration_ms, created_at
	FROM eval_runs ORDER BY created_at DESC LIMIT ?`
	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("list eval runs: %w", err)
	}
	defer rows.Close()
	return scanEvalRunRows(rows)
}

func scanEvalRunRows(rows *sql.Rows) ([]domain.EvalRun, error) {
	var list []domain.EvalRun
	for rows.Next() {
		var r domain.EvalRun
		var targetsJSON, createdStr string
		err := rows.Scan(&r.EvalID, &r.TraceID, &r.Description, &r.Purpose, &targetsJSON, &r.JudgeProvider,
			&r.TotalTests, &r.PassedTests, &r.FailedTests, &r.TotalTokens, &r.DurationMs, &createdStr)
		if err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(targetsJSON), &r.TargetProviders)
		r.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		list = append(list, r)
	}
	return list, rows.Err()
}
