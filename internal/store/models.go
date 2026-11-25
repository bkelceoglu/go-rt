package store

import (
	"time"

	"github.com/bkelceoglu/go-rt/internal/domain"
)

// PluginRecord represents a row in the SQLite plugins table managed by GORM.
type PluginRecord struct {
	ID               string          `gorm:"column:id;primaryKey" json:"id"`
	Category         string          `gorm:"column:category;not null" json:"category"`
	Severity         domain.Severity `gorm:"column:severity;not null" json:"severity"`
	StrategyExempt   bool            `gorm:"column:strategy_exempt;not null;default:false" json:"strategyExempt"`
	Description      string          `gorm:"column:description;not null" json:"description"`
	AttackerTemplate string          `gorm:"column:attacker_template;not null;default:''" json:"-"`
	GraderRubric     string          `gorm:"column:grader_rubric;not null;default:''" json:"-"`
}

func (PluginRecord) TableName() string { return "plugins" }

// PluginCollectionRecord maps a preset collection ID to a plugin ID in SQLite.
type PluginCollectionRecord struct {
	CollectionID string `gorm:"column:collection_id;primaryKey"`
	PluginID     string `gorm:"column:plugin_id;primaryKey"`
}

func (PluginCollectionRecord) TableName() string { return "plugin_collections" }

// StrategyRecord represents a row in the SQLite strategies table managed by GORM.
type StrategyRecord struct {
	ID              string `gorm:"column:id;primaryKey" json:"id"`
	Kind            string `gorm:"column:kind;not null" json:"kind"`
	Label           string `gorm:"column:label;not null" json:"label"`
	Description     string `gorm:"column:description;not null" json:"description"`
	AttackMechanism string `gorm:"column:attack_mechanism;not null" json:"attackMechanism"`
	ExampleYAML     string `gorm:"column:example_yaml;not null" json:"exampleYaml"`
}

func (StrategyRecord) TableName() string { return "strategies" }

// DatasetSourceRecord represents a row in the SQLite dataset_sources table.
type DatasetSourceRecord struct {
	DatasetName     string `gorm:"column:dataset_name;primaryKey"`
	Kind            string `gorm:"column:kind;not null"`
	URLOrRepo       string `gorm:"column:url_or_repo;not null"`
	HFConfig        string `gorm:"column:hf_config;not null;default:'default'"`
	HFSplit         string `gorm:"column:hf_split;not null;default:'train'"`
	PromptCol       string `gorm:"column:prompt_col;not null;default:'prompt'"`
	CategoryCol     string `gorm:"column:category_col;not null;default:''"`
	DefaultCategory string `gorm:"column:default_category;not null"`
	AttackType      string `gorm:"column:attack_type;not null"`
	SourceLabel     string `gorm:"column:source_label;not null"`
	FilterRule      string `gorm:"column:filter_rule;not null;default:''"`
	Enabled         bool   `gorm:"column:enabled;not null;default:true"`
}

func (DatasetSourceRecord) TableName() string { return "dataset_sources" }

// PromptBankRecord represents a deduplicated adversarial prompt in prompt_bank.
type PromptBankRecord struct {
	DatasetName string `gorm:"column:dataset_name;primaryKey;index:idx_prompt_bank_dataset"`
	PromptHash  string `gorm:"column:prompt_hash;primaryKey"`
	Prompt      string `gorm:"column:prompt;not null"`
	Category    string `gorm:"column:category;not null;index:idx_prompt_bank_category"`
	AttackType  string `gorm:"column:attack_type;not null"`
	Source      string `gorm:"column:source;not null"`
	CreatedAt   string `gorm:"column:created_at;not null;default:(datetime('now'))"`
}

func (PromptBankRecord) TableName() string { return "prompt_bank" }

// DatasetSummary summarizes a named prompt set stored in prompt_bank.
type DatasetSummary struct {
	DatasetName string `gorm:"column:dataset_name" json:"datasetName"`
	PromptCount int    `gorm:"column:prompt_count" json:"promptCount"`
	Source      string `gorm:"column:source" json:"source"`
}

// EvalRunModel maps domain.EvalRun to the eval_runs table via GORM.
type EvalRunModel struct {
	EvalID          string    `gorm:"column:eval_id;primaryKey"`
	TraceID         string    `gorm:"column:trace_id;not null;index:idx_eval_runs_trace"`
	Description     string    `gorm:"column:description;not null"`
	Purpose         string    `gorm:"column:purpose;not null"`
	TargetProviders []string  `gorm:"column:target_providers_json;serializer:json;not null"`
	JudgeProvider   string    `gorm:"column:judge_provider;not null"`
	TotalTests      int       `gorm:"column:total_tests;not null;default:0"`
	PassedTests     int       `gorm:"column:passed_tests;not null;default:0"`
	FailedTests     int       `gorm:"column:failed_tests;not null;default:0"`
	TotalTokens     int       `gorm:"column:total_tokens;not null;default:0"`
	DurationMs      int64     `gorm:"column:duration_ms;not null;default:0"`
	CreatedAt       time.Time `gorm:"column:created_at;not null"`
}

func (EvalRunModel) TableName() string { return "eval_runs" }

// EvalResultModel maps domain.EvalResult to the eval_results table via GORM.
type EvalResultModel struct {
	ResultID            string                    `gorm:"column:result_id;primaryKey"`
	EvalID              string                    `gorm:"column:eval_id;not null;index:idx_eval_results_eval"`
	TraceID             string                    `gorm:"column:trace_id;not null"`
	SpanID              string                    `gorm:"column:span_id;not null"`
	PluginID            string                    `gorm:"column:plugin_id;not null;index:idx_eval_results_plugin"`
	Category            string                    `gorm:"column:category;not null"`
	AttackType          string                    `gorm:"column:attack_type;not null"`
	Source              string                    `gorm:"column:source;not null"`
	StrategyID          string                    `gorm:"column:strategy_id;not null"`
	Severity            domain.Severity           `gorm:"column:severity;not null"`
	InjectVar           string                    `gorm:"column:inject_var;not null"`
	RawPrompt           string                    `gorm:"column:raw_prompt;not null"`
	RenderedPrompt      string                    `gorm:"column:rendered_prompt;not null"`
	TargetOutput        string                    `gorm:"column:target_output;not null"`
	Pass                bool                      `gorm:"column:pass;not null;index:idx_eval_results_pass"`
	Score               float64                   `gorm:"column:score;not null"`
	FailureReason       string                    `gorm:"column:failure_reason;not null;default:''"`
	JudgeReason         string                    `gorm:"column:judge_reason;not null"`
	JudgeRubric         string                    `gorm:"column:judge_rubric;not null;default:''"`
	LatencyMs           int64                     `gorm:"column:latency_ms;not null"`
	TokensPrompt        int                       `gorm:"column:tokens_prompt;not null;default:0"`
	TokensCompletion    int                       `gorm:"column:tokens_completion;not null;default:0"`
	TokensTotal         int                       `gorm:"column:tokens_total;not null;default:0"`
	Cached              bool                      `gorm:"column:cached;not null;default:false"`
	ConversationHistory []domain.ConversationTurn `gorm:"column:conversation_json;serializer:json"`
	OTelAttributes      map[string]any            `gorm:"column:otel_attributes_json;serializer:json"`
	CreatedAt           time.Time                 `gorm:"column:created_at;not null"`
}

func (EvalResultModel) TableName() string { return "eval_results" }

// OTelSpanModel maps domain.OTelSpanRecord to the otel_spans table via GORM.
type OTelSpanModel struct {
	SpanID            string              `gorm:"column:span_id;primaryKey"`
	TraceID           string              `gorm:"column:trace_id;not null;index:idx_otel_spans_trace"`
	ParentSpanID      string              `gorm:"column:parent_span_id;not null;default:''"`
	EvalID            string              `gorm:"column:eval_id;not null;default:'';index:idx_otel_spans_eval"`
	ResultID          string              `gorm:"column:result_id;not null;default:''"`
	Name              string              `gorm:"column:name;not null"`
	Kind              domain.OTelSpanKind `gorm:"column:kind;not null"`
	StartTimeUnixNano int64               `gorm:"column:start_time_unix_nano;not null"`
	EndTimeUnixNano   int64               `gorm:"column:end_time_unix_nano;not null"`
	StatusCode        string              `gorm:"column:status_code;not null"`
	Attributes        map[string]any      `gorm:"column:attributes_json;serializer:json"`
	Events            []map[string]any    `gorm:"column:events_json;serializer:json"`
}

func (OTelSpanModel) TableName() string { return "otel_spans" }
