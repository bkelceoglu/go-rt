# GO-RT Shared Brain — Master Task Tracker

> **Kural:** Hem `agy` (CLI) hem de `antigravity-ide` bir göreve başlamadan önce burayı kontrol eder, görevi bitirip testlerini (`make check`) geçirdikten sonra ilgili kutucuğu `[x]` olarak işaretler ve `CONTEXT.md` dosyasını günceller.

---

## Faz 0: Proje İskeleti & Vibe-Coding Ortak Hafıza Altyapısı ✅
- [x] `~/codes/go-rt` dizin yapısının oluşturulması
- [x] `docs/MASTER_PLAN.md` detaylı Go 1.27+ mimari master planının (`gort:*` isim alanı, İkili Prompt Stratejisi + OTel Multi-DB) yazılması
- [x] `AGENTS.md`, `GEMINI.md` ve `.agents/` (rules + skills) ortak ajan kurallarının kurulması
- [x] `docs/brain/` (`CONTEXT.md`, `TASKS.md`, `DECISIONS.md`) paylaşımlı beyin yapısının oluşturulması
- [x] `go.mod` (Go 1.27), `Makefile`, `.golangci.yml`, örnek `config.yaml` ve `internal/domain` (`domain.go`, `telemetry.go`, `config.go`) çekirdek tiplerinin oluşturulması
- [x] `cmd/gort/main.go` CLI başlangıç noktasının ve birim testlerinin doğrulanması

---

## Faz 1: `config.yaml` Yükleyici, SQLite Katalog & `prompt_bank` Bootstrap Motoru, Ergonomik CLI ve SSTI-Safe Şablonlama ✅
- [x] `internal/config/loader.go`: `config.yaml` ayrıştırıcı, ortam değişkeni çözümleyici ve varsayılan değer atayıcı
- [x] `internal/store/sqlite.go`, `prompt_bank.go`, `schema.sql`, `seed_catalog.sql`: Hard-coded Go dizileri yerine CGO'suz SQLite (`modernc.org/sqlite`, WAL + `busy_timeout=10000`, `MaxOpenConns=1`) üzerinde **155 plugin**, **14 koleksiyon**, **17 veri seti kaynağı** ve `dataset_name` ayrımına sahip `prompt_bank` tablosu
- [x] `internal/plugin/catalog.go`: SQLite üzerindeki 155 plugin ve 14 `gort:collection:*` koleksiyonunu dinamik açan (`ExpandSpecs`) katalog modülü
- [x] `internal/dataset/downloader.go`, `huggingface.go`, `benchmarks.go`: Açılışta otomatik `EnsureBootstrapped` ile eksik veri setlerini tespit edip rate-limit korumalı (`g.SetLimit(3)` + 3x retry) olarak HuggingFace REST API ve GitHub'dan indiren, `master_red_team_100` / `1000` / `full` setlerini üreten motor (toplam **20 veri seti, ~8,250+ prompt**)
- [x] `internal/dataset/loader.go`: Tek bir isimli veri setini (`gort lakera_gandalf` veya `prompts/hedehude.csv`) ya da `all` ile tüm veri setlerini sırayla SQLite `prompt_bank` üzerinden koşturan yükleyici
- [x] `cmd/gort/cli.go` & `inspector.go`: Kullanıcı dostu alt komutlar (`gort ls`, `gort ls 1` / `gort ls <isim>` tablo görünümü, `gort plugins`, `gort plugins 1` / `gort plugins <id>` offline detay ve kullanım kartı, `gort pull`, ve veri seti verilmediğinde kullanıcı onaylı `[y/N]` Attacker LLM akışı)
- [x] `internal/template/extractor.go` & `renderer.go`: `SkipRenderVars` destekli, SSTI-korumalı iki aşamalı `SafeRenderer`
- [x] `internal/ui/ansi.go`: Harici bağımlılıksız, terminal/pipe duyarlı, görünür karakter uzunluğuna göre hizalamayı bozmayan ANSI renklendirme ve badge motoru (`ADR-012`)

---

## Faz 2: Hedef & Hakem LLM İstemcileri (Ollama, vLLM, OpenAI, Webhook) & Grader Motoru ✅
- [x] `internal/provider/openai_compatible.go`: Sıfır dış bağımlılıklı (`net/http`) Ollama (`llama3.1:8b`, `qwen2.5:3b`), vLLM ve resmi OpenAI (`api.openai.com`) uyumlu, ortam değişkeni öncelikli istemci
- [x] `internal/provider/webhook.go`: Özel REST/Webhook chat ve API Gateway servisleri (`headers`, `bodyTemplate` ve `responsePath` destekli) istemcisi
- [x] `internal/provider/ratelimit.go`: Token-bucket hız sınırı (`rate.Limiter`) + eşzamanlılık semafor kalkanı (`RateLimitedProvider` decorator)
- [x] `internal/grader/refusal.go` & `rubric.go`: Çoklu dil destekli deterministik `notContains` / `IsBasicRefusal` kontrolleri ve `RubricGrader` LLM-as-a-Judge Hakem değerlendiricisi
- [x] `cmd/gort/inspector.go` (`--ai` eklentisi): `gort plugins <id> --ai` çağrıldığında offline karta ek olarak sağlayıcı üzerinden canlı örnek saldırı üretimi

---

## Faz 3: OpenTelemetry (OTel) GenAI & SQLite Depolama Katmanı (`.gort/gort.db`) ✅
- [x] `internal/telemetry/tracer.go`: W3C `TraceID`/`SpanID` üretici, `gen_ai.*` ve `gort.*` semantik özniteliklerini formatlayan OTel Tracer modülü
- [x] `internal/store/sqlite_eval.go`: SQLite `.gort/gort.db` içindeki `eval_runs`, `eval_results`, `otel_spans` tablolarına test sonuçlarını ve telemetriyi kaydeden/sorgulayan `SQLiteResultStore` (`domain.ResultStore`)
- [x] `internal/store/eval_test.go`: Değerlendirme koşumlarının, sonuçlarının ve OTel span'lerinin SQLite üzerinde sorgulanabilirlik birim testleri

---

## Faz 4: Dinamik LLM Plugin Motoru & Hibrit Orkestratör ✅
- [x] `internal/plugin/generator.go`: Attacker LLM ile dinamik olarak aktif eklentilere çok dilli (`tr`, `en`) saldırı promptları üreten `Synthesizer` motoru
- [x] `internal/engine/evaluator.go`: `gort <dataset_name|all>` veya Attacker LLM ile üretilen test vakalarını paralel koşturan, Judge LLM / Refusal ile puanlayan, canlı ANSI badge basan ve OTel Store'a kaydeden ana orkestratör
- [x] `cmd/gort/main.go`: `gort <dataset>` ve `gort [y/N]` Attacker LLM akışlarının uçtan uca canlı yürütmeye bağlanması

---

## Faz 5: Statik ve Agentic Multi-Turn Stratejiler (`gort:strategy:*`) ⏳
- [x] **Adım 5.1 — Statik Kodlama & Obfuscation Stratejileri (`internal/strategy/static.go` + `static_test.go`):**
  - `gort:strategy:basic` (kimlik geçişi), `gort:strategy:base64`, `gort:strategy:rot13`, `gort:strategy:hex`, `gort:strategy:leetspeak` ve `gort:strategy:homoglyph` dönüşümleri (`Metadata.StrategyID` ve `Metadata.OriginalText` korunarak).
- [x] **Adım 5.2 — Jailbreak Şablonları & Katmanlı Strateji Hattı (`internal/strategy/templates.go` + `pipeline.go` + `pipeline_test.go`):**
  - `gort:strategy:jailbreak-templates` (kurumsal rol yapma / çerçeveleme sarmalayıcıları) ve `gort:strategy:layer` / `ExpandWithStrategies` boru hattı (`StrategyExempt` pluginleri koruyarak tüm aktif stratejileri çoğaltan/uygulayan orkestratör).
- [ ] **Adım 5.3 — Agentic Konuşma Hafızası & Engel Açıcı (`internal/provider/agentic/memory.go` + `unblocking.go` + `memory_test.go`):**
  - Çok turlu konuşma geçmişi (`ConversationMemory`), hedef reddettiğinde son turu geri saran `RollbackLastTurn()` (Backtracking) ve hedefin karşı sorularını yanıtlayan `Unblocker` modülü.
- [ ] **Adım 5.4 — Çok Turlu Crescendo / GOAT / Hydra Dekoratörleri & Motor Entegrasyonu (`internal/provider/agentic/crescendo.go` + `crescendo_test.go` + `evaluator.go` + `cmd/gort/main.go`):**
  - `gort:strategy:crescendo` (ve `goat` / `hydra` varyantları) için `TargetOverride` sarmalayıcısı (`CrescendoDecorator`), `Evaluator` içinde `tc.TargetOverride` desteği ve CLI akışına tam entegrasyon.

---

## Faz 6: Web Dashboard, Yönetici Raporu & Docker Compose Altyapısı ⏳
- [ ] SQLite ve PostgreSQL üzerindeki OTel verilerini görselleştiren yerleşik Web UI (`gort --view`)
- [ ] Kurumsal Yönetici Özeti (Executive Summary) HTML/PDF rapor üreticisi ve Docker Compose yapılandırması
