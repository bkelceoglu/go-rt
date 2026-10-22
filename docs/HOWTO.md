# GORT (`go-rt`) — Step-by-Step How-To & Practical Operations Guide

This guide walks through every practical workflow in **GORT (`gort`)**, from inspecting the built-in SQLite catalog to configuring local/cloud LLMs, chaining multi-turn strategies, and querying OpenTelemetry evaluation results.

---

## 1. Building & Verifying the Binary

GORT compiles with `CGO_ENABLED=0` into a standalone static binary (`./bin/gort`):

```bash
# Run formatting, static analysis (go vet), unit tests, and build ./bin/gort
make all

# Check version and CLI help
./bin/gort --version
./bin/gort --help
```

On first run, GORT automatically creates `.gort/gort.db`, executes GORM `AutoMigrate` for all 7 tables (`plugins`, `plugin_collections`, `dataset_sources`, `prompt_bank`, `eval_runs`, `eval_results`, `otel_spans`), seeds the **155 plugins** and **14 collections**, and ensures all **20 benchmark datasets (~8,256 prompts)** are populated.

---

## 2. Exploring Datasets & Plugins from the Terminal

### 2.1. Listing & Inspecting Prompt Datasets (`gort ls`)
```bash
# 1. List all 20 named datasets and their prompt counts
./bin/gort ls

# 2. View all prompts inside a dataset by its list number (e.g. #9 enterprise_security_suite)
./bin/gort ls 9

# 3. View all prompts inside a dataset by name or substring
./bin/gort ls lakera_gandalf
./bin/gort ls harmbench
```

### 2.2. Listing & Inspecting Red-Team Plugins (`gort plugins`)
```bash
# 1. List all 155 plugins with category, severity badge, and description
./bin/gort plugins

# 2. View the detailed usage card for a plugin by index or ID (0ms, 100% offline)
./bin/gort plugins 38
./bin/gort plugins sql-injection
./bin/gort plugins gort:plugin:agentic:memory-poisoning

# 3. Request a live sample attack prompt from your configured Attacker LLM
./bin/gort plugins sql-injection --ai
```

### 2.3. Re-syncing Remote Datasets (`gort pull`)
```bash
# Re-download and synchronize all remote datasets into .gort/gort.db and ./prompts/*.csv
./bin/gort pull
```

---

## 3. Configuring Providers (`config.yaml`)

GORT supports local runtimes (**llama.cpp**, **vLLM**, **Ollama**, **LocalAI**), cloud APIs (**OpenAI** / compatible gateways), and custom **REST Webhooks**.

### Recipe A: `llama.cpp` (`llama-server`) on Port `8080` with Speculative Decoding (Default Setup)
Start `llama-server` on `:8080` with Intel Arc GPU OpenVINO acceleration and speculative decoding:
```bash
GGML_OPENVINO_DEVICE="GPU" llama-server \
  -m ~/.models/Qwen2.5-7B-Instruct-Q4_K_M.gguf \
  -md ~/.models/qwen2.5-1.5b-instruct-q4_k_m.gguf \
  -ngl 99 -ngld 99 -c 4096 --port 8080
```
Then configure `config.yaml`:
```yaml
maxConcurrency: 2

targets:
  - id: "llamacpp:chat:Qwen2.5-7B-Instruct"
    label: "Target Assistant (Qwen 2.5 7B + 1.5B Draft via llama.cpp)"
    baseUrl: "http://localhost:8080"
    temperature: 0.1
    maxTokens: 512
    think: false # Optional: false -> prepends '/no_think', true -> prepends '/think', omitted -> untouched

attacker:
  id: "llamacpp:chat:Qwen2.5-7B-Instruct"
  baseUrl: "http://localhost:8080"
  temperature: 0.7
  maxTokens: 512
  think: false

judge:
  id: "llamacpp:chat:Qwen2.5-7B-Instruct"
  baseUrl: "http://localhost:8080"
  temperature: 0.0
  maxTokens: 256
  think: false
```
> **Environment Variable Override:** You can also set `export LLAMACPP_BASE_URL="http://localhost:8080/v1"`.

### Recipe B: OpenAI / Cloud API Gateway
```yaml
targets:
  - id: "openai:chat:gpt-4o-mini"
    apiKey: "${OPENAI_API_KEY}"
    temperature: 0.0
    maxTokens: 512
```

### Recipe C: Custom Internal REST Application / API Gateway (`webhook:`)
Use `webhook:` when testing an internal microservice, RAG endpoint, or chat API gateway with custom JSON schemas:
```yaml
targets:
  - id: "webhook:https://internal-rag.company.local/api/v2/ask"
    headers:
      Authorization: "Bearer ${INTERNAL_RAG_TOKEN}"
      X-Tenant-ID: "redteam-sandbox"
    bodyTemplate: |
      {
        "query": "{{prompt}}",
        "stream": false
      }
    responsePath: "data.answer.text"
```

---

## 4. Configuring Attack Strategies (`gort:strategy:*`)

Strategies transform seed prompts (from either SQLite `prompt_bank` or Attacker LLM synthesis) before sending them to the target. Plugins marked `strategyExempt: true` (such as static safety benchmarks) automatically bypass mutation so they are never corrupted.

### 4.1. Available Strategy IDs
| Strategy ID | Type | Description |
| :--- | :--- | :--- |
| `gort:strategy:basic` | Static | Direct pass-through of the seed prompt without obfuscation. |
| `gort:strategy:base64` | Static | Encodes the payload in Base64 and instructs the target to decode and execute. |
| `gort:strategy:rot13` | Static | Encodes the payload with ROT13 cipher substitution. |
| `gort:strategy:hex` | Static | Encodes the payload as hexadecimal ASCII bytes. |
| `gort:strategy:leetspeak` | Static | Replaces characters with leetspeak equivalents (`a->4, e->3, i->1, o->0, s->5, t->7`). |
| `gort:strategy:homoglyph` | Static | Replaces Latin letters with visually identical Cyrillic Unicode homoglyphs. |
| `gort:strategy:jailbreak-templates` | Framing | Wraps the prompt in Sandbox Audit, Academic Simulation, or Defensive Analysis framing. |
| `gort:strategy:layer` | Chained | Applies multiple strategies sequentially (e.g., framing first, then Base64 encoding). |
| `gort:strategy:crescendo` | Multi-Turn | Gradual multi-turn escalation with refusal `RollbackLastTurn()` and `Unblocker`. |
| `gort:strategy:goat` | Multi-Turn | Generative Offensive Agent Tester (multi-technique adaptive reasoning across turns). |
| `gort:strategy:hydra` | Multi-Turn | Multi-head pivoting that switches to orthogonal technical angles upon resistance. |

### 4.2. Example Strategy Block in `config.yaml`
```yaml
strategies:
  # 1. Standard unmutated baseline
  - id: "gort:strategy:basic"

  # 2. Single-turn encodings & framing
  - id: "gort:strategy:base64"
  - id: "gort:strategy:homoglyph"
  - id: "gort:strategy:jailbreak-templates"

  # 3. Layered chaining (wraps in jailbreak template first, then Base64-encodes the result)
  - id: "gort:strategy:layer"
    steps:
      - "gort:strategy:jailbreak-templates"
      - "gort:strategy:base64"

  # 4. Agentic multi-turn escalation with backtracking
  - id: "gort:strategy:crescendo"
    maxTurns: 5
    maxBacktracks: 3
```

---

## 5. Running Red-Teaming Assessments & Live `--verbose` (`-v`) Tracing

You can add `--verbose` (or `-v`) to **any** assessment command to stream a live, boxed trace for every test case:
- **`[SYNTHESIZED]`**: Displays each attack prompt generated by the Attacker LLM before strategy expansion.
- **`Strategy` & `Prompt`**: Shows which strategy (`basic`, `base64`, `crescendo`, etc.) was applied and the exact mutated payload sent to the Target LLM.
- **`Output` & Multi-Turn `[BACKTRACKED]` Transcript**: Shows the Target LLM's response—or for `crescendo`/`goat`/`hydra`, every `Turn #N User` and `Turn #N Assistant` exchange, explicitly marking rolled-back refusals as `Turn #N [BACKTRACKED]`.
- **`Verdict`**: Prints the `Score` (`1.0` = `PASS`, `0.0` = `FAIL`) and the Judge LLM's reasoning.

### Scenario 1: Running Built-in SQLite Datasets by Name or Index (`1..20`)
When you specify a dataset name or its `gort ls` index (`1..20`), GORT loads the seed prompts directly from `.gort/gort.db` (`prompt_bank`) **without** calling the Attacker LLM for synthesis, then expands non-exempt prompts across the `strategies` configured in your YAML:

```bash
# Run by dataset name or by 'gort ls' numeric index (e.g. 9 = enterprise_security_suite, 18 = shomi28_injections)
./bin/gort enterprise_security_suite
./bin/gort 18 --verbose

# Run a SQLite dataset using a custom config file (e.g. _first_dry_config.yaml) with live verbose trace
./bin/gort _first_dry_config.yaml 18 --verbose
./bin/gort _first_dry_config.yaml shomi28_injections -v
./bin/gort _first_dry_config.yaml cyberseceval_injections -v

# Run BIPIA Indirect Prompt Injection (#4) or UIUC InjecAgent (#11)
./bin/gort 4 -v
./bin/gort 11 -v

# Run Lakera Gandalf (#13, 310 prompt extraction levels) or stratified master suites (#14, #15)
./bin/gort lakera_gandalf
./bin/gort master_red_team_100
./bin/gort master_red_team_1000
```

> **Note on `StrategyExempt` Isolation in Datasets:**
> Academic safety benchmarks and system-override baselines (such as `gort:plugin:cyberseceval`, `gort:plugin:system-prompt-override`, `gort:plugin:harmbench`, `gort:plugin:beavertails`, `gort:plugin:donotanswer`, `gort:plugin:xstest`) have `strategy_exempt = true` in the SQLite `plugins` table. Even when `strategies` contains `base64` and `crescendo`, those exempt prompts run **only** with `gort:strategy:basic`, while non-exempt prompts in the same dataset (e.g., `gort:plugin:hijacking`, `gort:plugin:rbac`, `gort:plugin:harmful:cybercrime:malicious-code`) automatically multiply across all configured strategies.

### Scenario 2: Dynamic Attacker LLM Synthesis (`gort <config.yaml> --verbose`)
When you invoke `gort` with **only** a config file (and no dataset argument), GORT expands all collections and plugins configured under `redteam.plugins`, displays the exact plugin list and `numTests` counts, asks for `[y/N]` confirmation, and uses the **Attacker LLM** to synthesize custom multilingual prompts tailored to your `purpose`:

```bash
# Run the fast 2-plugin x 3-strategy dry-run configuration with live verbose output
./bin/gort _first_dry_config.yaml --verbose

# Run the full enterprise configuration
./bin/gort config.yaml -v
```

### Preset Collections (`gort:collection:*`) Available for `redteam.plugins`
- `gort:collection:default`
- `gort:collection:owasp-llm-top-10`
- `gort:collection:coding-agent:all`
- `gort:collection:harmful`
- `gort:collection:pii`
- `gort:collection:bias`
- `gort:collection:financial`
- `gort:collection:medical`
- `gort:collection:telecom`
- `gort:collection:ecommerce`
- `gort:collection:insurance`
- `gort:collection:pharmacy`
- `gort:collection:realestate`
- `gort:collection:guardrails`

---

## 6. Inspecting Stored Results (`gort results`) & OpenTelemetry Spans

All evaluation runs, individual verdicts, multi-turn conversation histories (including `[BACKTRACKED]` rolled-back turns), and OpenTelemetry spans are persisted in `.gort/gort.db`.

### 6.1. Built-in Terminal Result Inspector (`gort results`)
You do **not** need an external `sqlite3` CLI binary installed to inspect past runs:

```bash
# 1. List the 25 most recent evaluation runs stored in .gort/gort.db
./bin/gort results

# 2. View the summary table for a run by its list index (e.g. 1 = latest run) or eval_id
./bin/gort results 1
./bin/gort results eval-10b3cbf96d8bf10e

# 3. Replay the full verbose transcript (prompts, target outputs, Crescendo turns, and Judge reasons)
./bin/gort results 1 --verbose
```

### 6.2. Real-Time File Log Streaming (`tail -f .gort/gort.log`)
Even if you run `gort` **without** `--verbose` (or as a background daemon / CI job), GORT continuously appends timestamped RFC3339 events (`[RUN_START]`, `[SYNTH]`, `[PASS]`/`[FAIL]`, `[DETAIL]`, `[TURN]`, `[RUN_COMPLETE]`) to **`.gort/gort.log`**:

```bash
# Watch all live attack prompts, Crescendo turns, and Judge verdicts from another terminal:
tail -f .gort/gort.log

# Filter only failed test cases (successful jailbreaks) in real time:
tail -f .gort/gort.log | grep "\[FAIL  \]"
```

### 6.3. Optional Direct SQLite Queries (`sqlite3`)
If you also want to query `.gort/gort.db` from external BI tools or scripts:

```bash
# 1. View recent evaluation run summaries
sqlite3 -header -column .gort/gort.db \
  "SELECT eval_id, trace_id, total_tests, passed_tests, failed_tests, duration_ms, created_at FROM eval_runs ORDER BY created_at DESC LIMIT 10;"

# 2. Inspect failed test cases (successful jailbreaks / vulnerabilities found)
sqlite3 -header -column .gort/gort.db \
  "SELECT plugin_id, strategy_id, substr(raw_prompt,1,45) AS prompt, substr(target_output,1,45) AS output, judge_reason FROM eval_results WHERE pass = 0 ORDER BY created_at DESC LIMIT 20;"

# 3. Inspect OpenTelemetry spans
sqlite3 -header -column .gort/gort.db \
  "SELECT span_id, trace_id, name, status_code, end_time_unix_nano - start_time_unix_nano AS duration_ns FROM otel_spans LIMIT 10;"
```

---

## 7. Web Command Center & Real-Time Studio (`gort view`)

GORT embeds a complete, zero-dependency Web Command Center (`//go:embed static`) directly inside the single Go binary:

```bash
# Start the Web Command Center on default port 8585
./bin/gort view

# Or specify a custom port
./bin/gort view 9090
```

Once running, navigate to `http://localhost:8585` in your browser. The Command Center provides five interconnected operational modules:

### 7.1. Config Builder Wizard (`wizard`)
- **Interactive 5-Step Pipeline:**
  1. **General & Purpose:** Set evaluation title, system purpose, prompt template, injection variable (`{{prompt}}`), and max concurrency.
  2. **Target Models:** Add multiple target models (LocalAI, OpenAI, vLLM, Webhook) with base URLs, API keys, temperature, and thinking mode toggles.
  3. **Attacker & Judge LLMs:** Configure the automated red-team attacker generator and evaluation Judge LLM with safety rubric.
  4. **Plugins & Datasets:** 
     - **4A Preset Collections:** Select high-level suites (`owasp-llm-top-10`, `coding-agent:all`, `harmful`, `pii`, `financial`).
     - **4B Individual Plugins:** Fine-grained selection across all 155 plugins, automatically kept in bidirectional sync with 4A collections.
     - **Benchmark Datasets:** Pick standard benchmark datasets or create custom sub-sampled prompt sets.
  5. **Attack Strategies:** Enable static mutations (`base64`, `rot13`, `hex`, `leetspeak`, `homoglyph`), framing (`jailbreak-templates`), sequential chaining (`layer`), and multi-turn agentic loops (`crescendo`).
- **Live Bidirectional YAML Editor & Test Case Estimator:**
  - Real-time serialization of configuration into clean YAML.
  - Test case multiplier calculator (`Seeds × Strategies = Total Test Cases`).
  - Blank configuration generator (`+ Blank Config`) or instant template loader (`⚡ Starter Template`).
  - Disk save & load controls (`Save to Disk`, `Reload`).

### 7.2. SQLite Plugin & Prompt Bank Explorer (`catalog`)
- Instant offline search across **155 redteam plugins**, **14 compliance collections**, and **20 benchmark datasets**.
- Filter by category, severity level, or compliance tag.
- Deep prompt inspector: View exact prompt bank contents directly in the browser.

### 7.3. Executive Security Dashboard (`dashboard`)
- **Top-line Posture KPIs:** Overall defense pass rate (%), total test cases evaluated, breaches/jailbreaks detected, total token spend, and average latency.
- **Scope Filter:** Toggle between:
  - `Latest Run` (Default): Analyzes only the most recent evaluation session.
  - `All-Time Cumulative`: Aggregates every evaluation run in SQLite history.
  - Specific past runs by `eval_id`.
- **Strategy Bypass Analytics:** Direct visual comparison of which adversarial strategies bypassed the target's defenses.
- **OWASP LLM Top 10 Risk Breakdown:** Severity metrics (Critical, High, Medium, Low) and most vulnerable plugin areas.

### 7.4. Assessment Runs & Deep Transcript Inspector (`runs`)
- List and inspect all historical evaluation runs stored in `.gort/gort.db`.
- Filter results by outcome (`All`, `Failed / Vulnerable`, `Passed / Defended`), plugin category, or attack strategy.
- Detailed modal inspector:
  - Side-by-side comparison of Raw Prompt vs. Mutated Adversarial Prompt vs. Target Response.
  - Judge assessment reasoning and rubric score.
  - **Multi-Turn Crescendo Chat Bubbles:** Visual turn-by-turn conversation view with red `[BACKTRACKED]` badges indicating when the attacker rolled back following a target refusal.
  - **OpenTelemetry GenAI Span Waterfall Tree:** Full distributed trace spans showing internal operations, latencies, and semantic attributes.

### 7.5. Live Attack Launcher & SSE Terminal Console (`live`)
- **Launch Controls:** Trigger red-team assessments directly from the browser using either the in-memory Wizard YAML or a saved configuration file on disk.
- **Operator Stop:** Gracefully terminate running assessments with a single click.
- **Live Telemetry HUD:** Real-time counters for active target model, dataset, progress (`X / Y` cases), pass/fail tallies, and elapsed time.
- **Real-Time Terminal Console:**
  - Streams `.gort/gort.log` live via Server-Sent Events (`GET /api/logs/stream`).
  - Dark terminal styling with automatic event color-coding (`[FAIL]`, `[PASS]`, `[WARN]`, `[ERROR]`, `[INFO]`, `[SYSTEM]`).
  - Auto-scroll lock toggle, real-time keyword search filter, and console clear controls.

### 7.6. Executive Audit & Compliance Reports (`reports`)
- **Official Security Audit Document:**
  - Standardized corporate reporting format mapping results to **OWASP Top 10 for LLMs** (LLM01–LLM10), **NIST AI RMF 1.0**, and **MITRE ATLAS**.
  - Top-line security posture status badge: `SECURE / LOW RESIDUAL RISK`, `MODERATE RISK`, or `CRITICAL VULNERABILITIES DETECTED`.
  - Comprehensive Strategy Bypass analysis table.
  - Critical Vulnerability Findings with full prompt and target response evidence.
  - Formal Lead Auditor Attestation & Signature block.
- **Export & Print Capabilities:**
  - `⬇ Export JSON`: Download the full structured audit data package.
  - `⎙ Print / Save as PDF`: Native `@media print` CSS engine produces clean, high-contrast, professional A4/Letter audit PDFs with zero UI clutter.

