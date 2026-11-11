package domain

import (
	"context"
	"maps"
	"slices"
	"time"
)

// DatasetRecord represents a standardized row from curated external datasets
// (e.g., HarmBench, AdvBench, BeaverTails, ToxicChat, OWASP, Enterprise-Security-Suite).
type DatasetRecord struct {
	Prompt     string `json:"prompt"`
	Category   string `json:"category"`
	AttackType string `json:"attack_type"`
	Source     string `json:"source"`
}

// ConversationTurn captures a single turn in a multi-turn strategy (e.g., Crescendo, GOAT, Hydra).
type ConversationTurn struct {
	Turn      int       `json:"turn"`
	Role      string    `json:"role"`
	Prompt    string    `json:"prompt"`
	Output    string    `json:"output"`
	Backtrack bool      `json:"backtrack,omitzero"`
	Timestamp time.Time `json:"timestamp"`
}

// OTelSpanKind defines the OpenTelemetry span role.
type OTelSpanKind string

const (
	SpanKindInternal OTelSpanKind = "INTERNAL"
	SpanKindClient   OTelSpanKind = "CLIENT"
)

// OTelSpanRecord represents an OpenTelemetry GenAI & Redteam compliant span persisted in SQLite/PostgreSQL.
type OTelSpanRecord struct {
	TraceID           string           `json:"traceId"`
	SpanID            string           `json:"spanId"`
	ParentSpanID      string           `json:"parentSpanId,omitempty"`
	EvalID            string           `json:"evalId"`
	ResultID          string           `json:"resultId,omitempty"`
	Name              string           `json:"name"`
	Kind              OTelSpanKind     `json:"kind"`
	StartTimeUnixNano int64            `json:"startTimeUnixNano"`
	EndTimeUnixNano   int64            `json:"endTimeUnixNano"`
	StatusCode        string           `json:"statusCode"`
	Attributes        map[string]any   `json:"attributes"`
	Events            []map[string]any `json:"events,omitempty"`
}

// Clone returns a deep copy of OTelSpanRecord.
func (s OTelSpanRecord) Clone() OTelSpanRecord {
	cloned := s
	if s.Attributes != nil {
		cloned.Attributes = maps.Clone(s.Attributes)
	}
	if s.Events != nil {
		cloned.Events = slices.Clone(s.Events)
	}
	return cloned
}

// EvalRun represents a top-level evaluation or redteam assessment session.
type EvalRun struct {
	EvalID          string    `json:"evalId"`
	TraceID         string    `json:"traceId"`
	Description     string    `json:"description"`
	Purpose         string    `json:"purpose"`
	TargetProviders []string  `json:"targetProviders"`
	JudgeProvider   string    `json:"judgeProvider"`
	TotalTests      int       `json:"totalTests"`
	PassedTests     int       `json:"passedTests"`
	FailedTests     int       `json:"failedTests"`
	TotalTokens     int       `json:"totalTokens"`
	DurationMs      int64     `json:"durationMs"`
	CreatedAt       time.Time `json:"createdAt"`
}

// EvalResult represents a single evaluated test case outcome with full OTel trace linkage.
type EvalResult struct {
	ResultID            string             `json:"resultId"`
	EvalID              string             `json:"evalId"`
	TraceID             string             `json:"traceId"`
	SpanID              string             `json:"spanId"`
	PluginID            string             `json:"pluginId"`
	Category            string             `json:"category"`
	AttackType          string             `json:"attackType"`
	Source              string             `json:"source"`
	StrategyID          string             `json:"strategyId"`
	Severity            Severity           `json:"severity"`
	InjectVar           string             `json:"injectVar"`
	RawPrompt           string             `json:"rawPrompt"`
	RenderedPrompt      string             `json:"renderedPrompt"`
	TargetOutput        string             `json:"targetOutput"`
	Pass                bool               `json:"pass"`
	Score               float64            `json:"score"`
	FailureReason       string             `json:"failureReason,omitempty"`
	JudgeReason         string             `json:"judgeReason"`
	JudgeRubric         string             `json:"judgeRubric,omitempty"`
	LatencyMs           int64              `json:"latencyMs"`
	TokenUsage          TokenUsage         `json:"tokenUsage"`
	Cached              bool               `json:"cached"`
	ConversationHistory []ConversationTurn `json:"conversationHistory,omitempty"`
	OTelAttributes      map[string]any     `json:"otelAttributes,omitempty"`
	CreatedAt           time.Time          `json:"createdAt"`
}

// ResultFilter defines query parameters for retrieving stored evaluation results.
type ResultFilter struct {
	EvalID     string
	PluginID   string
	Category   string
	Source     string
	StrategyID string
	FailedOnly bool
	Limit      int
}

// StrategyBypassStat holds aggregate bypass counts for an attack strategy.
type StrategyBypassStat struct {
	Total      int64   `json:"total"`
	Passed     int64   `json:"passed"`
	Failed     int64   `json:"failed"`
	BypassRate float64 `json:"bypassRate"`
}

// PluginFailureStat holds failure counts grouped by plugin and category.
type PluginFailureStat struct {
	Category string `json:"category"`
	PluginID string `json:"pluginId"`
	Failures int64  `json:"failures"`
}

// DashboardStats summarizes executive security posture across evaluation runs.
type DashboardStats struct {
	Scope             string                        `json:"scope"`
	SelectedEvalID    string                        `json:"selectedEvalId"`
	TotalRuns         int64                         `json:"totalRuns"`
	TotalTests        int64                         `json:"totalTests"`
	PassedTests       int64                         `json:"passedTests"`
	FailedTests       int64                         `json:"failedTests"`
	PassRate          float64                       `json:"passRate"`
	TotalTokens       int64                         `json:"totalTokens"`
	AvgDurationMs     int64                         `json:"avgDurationMs"`
	StrategyStats     map[string]StrategyBypassStat `json:"strategyStats"`
	SeverityStats     map[string]int64              `json:"severityStats"`
	TopFailingPlugins []PluginFailureStat           `json:"topFailingPlugins"`
	RecentRuns        []EvalRun                     `json:"recentRuns"`
}

// ResultStore defines the persistence contract implemented by SQLite, PostgreSQL, and MultiStore backends.
type ResultStore interface {
	SaveEvalRun(ctx context.Context, run EvalRun) error
	SaveEvalResult(ctx context.Context, res EvalResult) error
	SaveOTelSpan(ctx context.Context, span OTelSpanRecord) error
	QueryResults(ctx context.Context, filter ResultFilter) ([]EvalResult, error)
	Close() error
}
