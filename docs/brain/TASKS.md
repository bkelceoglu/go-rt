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

## Faz 5: Statik ve Agentic Multi-Turn Stratejiler (`gort:strategy:*`) ✅
- [x] **Adım 5.1 — Statik Kodlama & Obfuscation Stratejileri (`internal/strategy/static.go` + `static_test.go`):**
  - `gort:strategy:basic` (kimlik geçişi), `gort:strategy:base64`, `gort:strategy:rot13`, `gort:strategy:hex`, `gort:strategy:leetspeak` ve `gort:strategy:homoglyph` dönüşümleri (`Metadata.StrategyID` ve `Metadata.OriginalText` korunarak).
- [x] **Adım 5.2 — Jailbreak Şablonları & Katmanlı Strateji Hattı (`internal/strategy/templates.go` + `pipeline.go` + `pipeline_test.go`):**
  - `gort:strategy:jailbreak-templates` (kurumsal rol yapma / çerçeveleme sarmalayıcıları) ve `gort:strategy:layer` / `ExpandWithStrategies` boru hattı (`StrategyExempt` pluginleri koruyarak tüm aktif stratejileri çoğaltan/uygulayan orkestratör).
- [x] **Adım 5.3 — Agentic Konuşma Hafızası & Engel Açıcı (`internal/provider/agentic/memory.go` + `unblocking.go` + `memory_test.go`):**
  - Çok turlu konuşma geçmişi (`ConversationMemory`), hedef reddettiğinde son turu geri saran `RollbackLastTurn()` (Backtracking) ve hedefin karşı sorularını yanıtlayan `Unblocker` modülü.
- [x] **Adım 5.4 — Çok Turlu Crescendo / GOAT / Hydra Dekoratörleri & Motor Entegrasyonu (`internal/provider/agentic/crescendo.go` + `crescendo_test.go` + `evaluator.go` + `cmd/gort/main.go`):**
  - `gort:strategy:crescendo` (ve `goat` / `hydra` varyantları) için `TargetOverride` sarmalayıcısı (`CrescendoDecorator`), `Evaluator` içinde `tc.TargetOverride` desteği ve CLI akışına tam entegrasyon.
- [x] **Adım 5.5 — SQLite Strateji Kataloğu (`strategies` Tablosu), 28 Strateji Seed & Ergonomik CLI (`gort strategies`):**
  - SQLite `.gort/gort.db` içine `strategies` tablosu eklendi; 28 saldırı stratejisinin tamamı `seed_catalog.sql` ile tohumlandı.
  - `gort strategies` (özet liste) ve `gort strategies <index|strategy_id>` (detay kartı ve hazır `config.yaml` örneği) CLI komutları eklendi.
  - Web UI (`/api/catalog`) hardcoded kod yapısından arındırılarak doğrudan SQLite `strategies` tablosuna bağlandı.

---

## Faz 6: Yerleşik Web Command Center, Canlı Saldırı Konsolu & Denetim Raporları (`gort view`) ✅
- [x] **Adım 6.1 — Web Command Center Altyapısı, Config Builder Wizard & Katalog Gezgini (`internal/web`):**
  - Tek binary sıfır bağımlılıklı `//go:embed static` mimarisi ve Go 1.27 `http.ServeMux` REST API yönlendirme (`api_catalog.go`, `api_config.go`).
  - 3 kolonlu modern UI: Sol gezinti paneli, dinamik çalışma alanı, sağ canlı YAML önizleme ve test senaryosu hesaplayıcı.
  - 5 adımlı modüler Config Wizard (`wizard.js`): Genel/Amaç, Hedef Modeller (LocalAI, OpenAI, vLLM, Webhook), Attacker & Judge LLM, 4A Preset Koleksiyonlar ile 4B Tekil Pluginler arası çift yönlü tam senkronizasyon, ve Saldırı Stratejileri.
  - YAML yoksa boş başlama (`+ Blank Config`), tek tıkla şablon getirme (`⚡ Starter Template`) ve iki yönlü canlı YAML editörü.
  - SQLite Katalog Gezgini (`catalog.js`): 155 plugin, 14 koleksiyon ve 20 benchmark veri seti için anlık arama, filtreleme ve detaylı prompt önizleme.
- [x] **Adım 6.2 — Executive Security Dashboard & Runs Inspector (`api_runs.go`, `dashboard.js`, `runs.js`):**
  - Güvenlik Duruşu KPI'ları: Savunma oranı (% pass rate), tespit edilen zafiyetler, toplam işlem maliyeti (token), ortalama gecikme.
  - Strateji Bypass Analitiği: Her bir saldırı stratejisinin başarı/savunma oranları ve bypass yüzdeleri.
  - Önem derecesi kırılımı (Critical, High, Medium, Low) ve en çok ihlal edilen pluginlerin dağılımı.
  - Kapsam Filtresi: "Sadece Son Oturum (Latest Run)" varsayılan, "Tüm Zamanlar (All-Time Cumulative)" veya geçmiş oturumlar arasında anlık geçiş.
  - Runs & Deep Transcript Inspector (`runs.js`): Değerlendirme oturumları listesi, filtreleme, ham prompt vs. mutate edilmiş prompt vs. hedef LLM yanıtı yan yana kıyaslama, Judge rubric analizi, tam OTel GenAI span ağacı ve çok turlu Crescendo chat balonları (`[BACKTRACKED]` etiketleriyle).
- [x] **Adım 6.3 — Canlı Saldırı Başlatıcı & SSE Real-Time Terminal Log Akışı (`api_live.go`, `assessment_runner.go`, `live.js`):**
  - Web üzerinden canlı değerlendirme başlatma (`POST /api/assessments/start`) ve durdurma (`POST /api/assessments/stop`).
  - Live Telemetry HUD: Aktif hedef model, veri seti, tamamlanan vaka oranı (`X / Y`), Pass/Fail sayaçları ve geçen süre sayacı.
  - SSE Tabanlı Canlı Terminal Akışı: `.gort/gort.log` dosyasını `GET /api/logs/stream` üzerinden gerçek zamanlı terminal ekranında akıtma, auto-scroll kilidi, canlı filtreleme ve log seviyesi renklendirmesi (`[FAIL]`, `[PASS]`, `[WARN]`, `[ERROR]`, `[INFO]`, `[SYSTEM]`).
- [x] **Adım 6.4 — Kurumsal Güvenlik & Denetim Raporları (`api_reports.go`, `reports.js`):**
  - Yönetici denetim API'si (`GET /api/reports/executive?evalId=...`).
  - Yazdırılabilir ve PDF uyumlu (`@media print`) kurumsal güvenlik ve uyumluluk denetim raporu.
  - OWASP Top 10 for LLMs Uyumluluk Matrisi (LLM01 - LLM10 uyumluluk durumu, test ve breach sayıları).
  - Saldırı stratejisi etkinlik tablosu ve kritik zafiyet bulguları (ham prompt, kanıt niteliğindeki hedef yanıtı, Judge gerekçesi).
  - NIST AI RMF 1.0 ve OWASP standartlarında Baş Denetçi Tasdiki & İmza alanı.
  - JSON dışa aktarma (`⬇ Export JSON`) ve tek tıkla vektörel PDF / Yazıcı çıktısı (`⎙ Print / Save as PDF`).

## Faz 7: YAML Doğrulama, Mantık Kontrolü (Attacker/Judge Sıralı Kilit) & Otomatik Düzeltme (`docs/validation.md`) — ✅ TAMAMLANDI
- [x] **Format ve Yapı Doğrulayıcı (`internal/config/validator.go`):**
  - Sıkı YAML sözdizimi (`KnownFields(true)`), `targets`/`judge` yapı kontrolü ve SQLite kataloğundaki (155 plugin, 14 koleksiyon, 28 strateji) ID eşleşme doğrulaması.
- [x] **Mantık Kontrolü & Sıralı Judge/Attacker Çalışma Kilidi (`internal/provider/serialized.go`):**
  - Attacker LLM gerektiren stratejiler (`crescendo`, `goat`, `hydra`, `retry`) seçildiğinde `attacker.id` boşsa kullanıcıya 3 seçenek sunulması: 1.1 Özel Attacker LLM girilmesi, 1.2.1 Attacker stratejilerinin çıkarılması veya 1.2.2 Judge LLM'in sıralı modda (`SequentialSharedLLM: true`, `MaxConcurrency: 1`, `SerializedProvider` paylaşımlı `sync.Mutex`) Attacker olarak kullanılması.
- [x] **Çift Katmanlı Otomatik Düzeltme Motoru (`internal/config/autofix.go`):**
  - **2.1 Mini Engine (`AutoFixMiniEngine`):** `<1ms` deterministik sekme/girinti, anahtar yazım hatası (`target:` -> `targets:`, `strategy:` -> `strategies:`) ve katalog ID önek (`crescendo` -> `gort:strategy:crescendo`) onarımı.
  - **2.2 Judge LLM Repair (`AutoFixWithJudgeLLM`):** Judge LLM üzerinden ağır bozuk YAML onarımı + Mini Engine doğrulaması ve her iki modda da kullanıcı onayı zorunluluğu.
- [x] **CLI ve Web UI Tam Parite (`cmd/gort/validator_cli.go`, `internal/web/api_config.go`, `validator_ui.js`):**
  - `./bin/gort validate [config.yaml]` komutu ve değerlendirme öncesi interaktif doğrulama/onarım döngüsü.
  - Web UI'da `POST /api/configs/validate`, `POST /api/configs/autofix` uç noktaları ve `✓ Validate` interaktif modalı.
- [x] **Dinamik Provider Protokolleri & Pre-flight Uç Nokta Erişim Doğrulaması (`internal/config/reachability.go`, `api_catalog.go`, `wizard.js`):**
  - Hardcoded model isimlerinin (`Qwen2.5-7B`, `llama3.1:8b`, `gpt-4o-mini`) ve `~/.models` klasör varsayımının kod tabanından tamamen kaldırılması.
  - Dinamik provider şablonları (`ollama`, `llamacpp`, `vllm`, `localai`, `openai`, `webhook`) ve resmi varsayılan portlarla otomatik dolan ancak kullanıcı tarafından %100 düzenlenebilir Base URL mimarisi.
  - Doğrulama/Kaydetme öncesinde `CheckProvidersReachability` ile hem CLI (`gort validate`) hem Web UI (`⚡ Test Ping` & `✓ Validate`) üzerinden canlı network socket ve kimlik doğrulama kontrolleri (`● Online (HTTP 200)` / `○ Unreachable`).

---

## Faz 8: `golang-pro` (Go 1.27+) Kod Tabanı Denetimi, Güvenlik & Eşzamanlılık Modernizasyonu — ✅ TAMAMLANDI
- [x] **Adım 8.1 — P0 Eşzamanlılık, Yarış Durumu (Race Condition) ve `synctest` İyileştirmeleri (`internal/ui`, `internal/provider`, `internal/dataset`):**
  - `internal/ui/ansi.go`: `ForceColor` global değişkeni veri yarışını önlemek için `atomic.Bool` (`SetForceColor`, `IsForceColor`) yapısına geçirildi.
  - `internal/provider/serialized.go`: `sync.Mutex` yerine context iptaline duyarlı 1-kapasiteli kanal semaforu (`sem chan struct{}`) ile donma (deadlock/hang) koruması sağlandı.
  - `internal/provider/ratelimit_test.go`: `testing/synctest` sanal saat balonu ile deterministik ve milisaniye seviyesinde hız sınırı testleri eklendi.
  - `internal/dataset/benchmarks.go` & `huggingface.go`: Deterministik karıştırma (`math/rand/v2` PCG tohumu) ve context-duyarlı `time.NewTimer` geri çekilme (backoff) mekanizması eklendi.
- [x] **Adım 8.2 — P1 Güvenlik (`os.OpenRoot`) ve Web Sunucu Yaşam Döngüsü (`internal/web`):**
  - `internal/web/api_config.go` & `assessment_runner.go`: `os.OpenRoot` ile sembolik bağ (symlink) ve dizin aşımı (path traversal) koruması eklendi.
  - `internal/web/api_config.go`: `handleGetConfig` içinde `${ENV}` değişkenlerinin web arayüzüne sızmasını önlemek için ham YAML (`LoadRawFile`) okuma ayrımı yapıldı.
  - `internal/web/assessment_runner.go`: Arka plan değerlendirme goroutine'leri `sync.WaitGroup.Go` ve sunucu kapanış (`Shutdown`) kancası ile birleştirildi.
- [x] **Adım 8.3 — P2 Go 1.27 Modern Stdlib Deyimleri (Tüm Paketler):**
  - `json:"...,omitzero"` etiketleri (`internal/domain/config.go`), `new(expr)` yardımcı fonksiyon sadeleştirmesi (`internal/provider/openai_compatible.go`), `strings.SplitSeq` / `FieldsSeq` / `CutPrefix` / `CutLast` sıfır tahsisli (zero-allocation) iteratörler, `slices.Sorted(maps.Keys(m))` ve `slices.ContainsFunc` tüm paketlere uygulandı.
- [x] **Adım 8.4 — P3 Mimari, Atomik Fonksiyonlar (≤15 Satır, ≤4 Parametre) ve Test İskeleti:**
  - Tüm paketlerde derleme zamanı arayüz kontrolleri (`var _ domain.Provider = (*Client)(nil)` vb.) eklendi.
  - Uzun veya >4 parametreli fonksiyonlar bağlam yapıları (`evalRunContext`, `assessmentContext`, `targetBatchRunner`, `repairContext` vb.) ve tablo güdümlü (map-based) alt komut yönlendiricileri ile ≤15 satırlık atomik fonksiyonlara bölündü.
  - Tüm test dosyaları `1:1` dosya eşleşmesi (`loader_test.go`, `sqlite_eval_test.go`, `extractor_test.go`, `renderer_test.go`, `validator_cli_test.go`, `autofix_test.go`, `factory_test.go`, `unblocking_test.go`), `t.Context()` ve `t.Run` alt testleri ile modernize edildi; `Makefile` içine `make test-race` eklendi.
