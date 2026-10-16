PRAGMA journal_mode = WAL;
PRAGMA busy_timeout = 10000;
PRAGMA synchronous = NORMAL;
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS plugins (
    id TEXT PRIMARY KEY,
    category TEXT NOT NULL,
    severity TEXT NOT NULL,
    strategy_exempt INTEGER NOT NULL DEFAULT 0,
    description TEXT NOT NULL,
    attacker_template TEXT NOT NULL DEFAULT '',
    grader_rubric TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS plugin_collections (
    collection_id TEXT NOT NULL,
    plugin_id TEXT NOT NULL,
    PRIMARY KEY (collection_id, plugin_id)
);

CREATE TABLE IF NOT EXISTS dataset_sources (
    dataset_name TEXT PRIMARY KEY,
    kind TEXT NOT NULL,
    url_or_repo TEXT NOT NULL,
    hf_config TEXT NOT NULL DEFAULT 'default',
    hf_split TEXT NOT NULL DEFAULT 'train',
    prompt_col TEXT NOT NULL DEFAULT 'prompt',
    category_col TEXT NOT NULL DEFAULT '',
    default_category TEXT NOT NULL,
    attack_type TEXT NOT NULL,
    source_label TEXT NOT NULL,
    filter_rule TEXT NOT NULL DEFAULT '',
    enabled INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS prompt_bank (
    dataset_name TEXT NOT NULL,
    prompt_hash TEXT NOT NULL,
    prompt TEXT NOT NULL,
    category TEXT NOT NULL,
    attack_type TEXT NOT NULL,
    source TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (dataset_name, prompt_hash)
);

CREATE INDEX IF NOT EXISTS idx_prompt_bank_dataset ON prompt_bank(dataset_name);
CREATE INDEX IF NOT EXISTS idx_prompt_bank_category ON prompt_bank(category);

CREATE TABLE IF NOT EXISTS eval_runs (
    eval_id TEXT PRIMARY KEY,
    trace_id TEXT NOT NULL,
    description TEXT NOT NULL,
    purpose TEXT NOT NULL,
    target_providers_json TEXT NOT NULL,
    judge_provider TEXT NOT NULL,
    total_tests INTEGER NOT NULL DEFAULT 0,
    passed_tests INTEGER NOT NULL DEFAULT 0,
    failed_tests INTEGER NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_eval_runs_trace ON eval_runs(trace_id);

CREATE TABLE IF NOT EXISTS eval_results (
    result_id TEXT PRIMARY KEY,
    eval_id TEXT NOT NULL,
    trace_id TEXT NOT NULL,
    span_id TEXT NOT NULL,
    plugin_id TEXT NOT NULL,
    category TEXT NOT NULL,
    attack_type TEXT NOT NULL,
    source TEXT NOT NULL,
    strategy_id TEXT NOT NULL,
    severity TEXT NOT NULL,
    inject_var TEXT NOT NULL,
    raw_prompt TEXT NOT NULL,
    rendered_prompt TEXT NOT NULL,
    target_output TEXT NOT NULL,
    pass INTEGER NOT NULL,
    score REAL NOT NULL,
    failure_reason TEXT NOT NULL DEFAULT '',
    judge_reason TEXT NOT NULL,
    judge_rubric TEXT NOT NULL DEFAULT '',
    latency_ms INTEGER NOT NULL,
    tokens_prompt INTEGER NOT NULL DEFAULT 0,
    tokens_completion INTEGER NOT NULL DEFAULT 0,
    tokens_total INTEGER NOT NULL DEFAULT 0,
    cached INTEGER NOT NULL DEFAULT 0,
    conversation_json TEXT NOT NULL DEFAULT '[]',
    otel_attributes_json TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_eval_results_eval ON eval_results(eval_id);
CREATE INDEX IF NOT EXISTS idx_eval_results_plugin ON eval_results(plugin_id);
CREATE INDEX IF NOT EXISTS idx_eval_results_pass ON eval_results(pass);

CREATE TABLE IF NOT EXISTS otel_spans (
    span_id TEXT PRIMARY KEY,
    trace_id TEXT NOT NULL,
    parent_span_id TEXT NOT NULL DEFAULT '',
    eval_id TEXT NOT NULL DEFAULT '',
    result_id TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL,
    kind TEXT NOT NULL,
    start_time_unix_nano INTEGER NOT NULL,
    end_time_unix_nano INTEGER NOT NULL,
    status_code TEXT NOT NULL,
    attributes_json TEXT NOT NULL,
    events_json TEXT NOT NULL DEFAULT '[]'
);

CREATE INDEX IF NOT EXISTS idx_otel_spans_trace ON otel_spans(trace_id);
CREATE INDEX IF NOT EXISTS idx_otel_spans_eval ON otel_spans(eval_id);
