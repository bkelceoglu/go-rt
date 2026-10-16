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
    run_id TEXT PRIMARY KEY,
    trace_id TEXT NOT NULL,
    config_desc TEXT NOT NULL,
    target_models TEXT NOT NULL,
    judge_model TEXT NOT NULL,
    total_tests INTEGER NOT NULL DEFAULT 0,
    passed_count INTEGER NOT NULL DEFAULT 0,
    failed_count INTEGER NOT NULL DEFAULT 0,
    pass_rate REAL NOT NULL DEFAULT 0.0,
    started_at TEXT NOT NULL,
    completed_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS eval_results (
    result_id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL,
    trace_id TEXT NOT NULL,
    span_id TEXT NOT NULL,
    dataset_name TEXT NOT NULL DEFAULT 'default',
    test_id TEXT NOT NULL,
    plugin_id TEXT NOT NULL,
    strategy_id TEXT NOT NULL,
    severity TEXT NOT NULL,
    target_id TEXT NOT NULL,
    prompt TEXT NOT NULL,
    output TEXT NOT NULL,
    passed INTEGER NOT NULL,
    score REAL NOT NULL,
    reason TEXT NOT NULL,
    is_refusal INTEGER NOT NULL,
    latency_ms INTEGER NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_eval_results_run ON eval_results(run_id);
CREATE INDEX IF NOT EXISTS idx_eval_results_dataset ON eval_results(dataset_name);

CREATE TABLE IF NOT EXISTS otel_spans (
    span_id TEXT PRIMARY KEY,
    trace_id TEXT NOT NULL,
    parent_span_id TEXT NOT NULL DEFAULT '',
    run_id TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL,
    kind TEXT NOT NULL,
    start_time_unix_nano INTEGER NOT NULL,
    end_time_unix_nano INTEGER NOT NULL,
    duration_ms INTEGER NOT NULL,
    status_code TEXT NOT NULL,
    attributes_json TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_otel_spans_trace ON otel_spans(trace_id);
