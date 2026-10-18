# GO-RT Shared Brain — Project Context & Live State

> **Amaç:** Bu dosya hem **Antigravity CLI (`agy`)** hem de **Antigravity IDE (`antigravity-ide`)** tarafından okunan ortak hafıza (Shared Brain) durum dosyasıdır. Herhangi bir ajan (CLI veya IDE) yeni bir oturumda çalışmaya başladığında önce bu dosyayı ve `TASKS.md` dosyasını okur; yaptığı değişiklikten sonra buradaki durumu günceller.

---

## 1. Güncel Durum Özeti
- **Aktif Faz:** Faz 0, Faz 1, Faz 2, Faz 3 ve Faz 4 Tamamlandı ✅ $\rightarrow$ Sıradaki: **Faz 5 (Statik ve Agentic Multi-Turn Stratejiler: Crescendo, GOAT, Jailbreak)**
- **Go Sürümü:** `go1.27.0 linux/amd64` (`CGO_ENABLED=0`)
- **Modül Adı:** `github.com/caveman/go-rt`
- **SQLite Durumu (`.gort/gort.db`):**
  - `plugins`: **155 yerleşik plugin** (`gort:plugin:*`)
  - `plugin_collections`: **14 hazır koleksiyon** (`gort:collection:*`)
  - `dataset_sources`: **17 uzak ve yerleşik veri seti kaynağı** (GitHub CSV/JSON/MD + HuggingFace REST API)
  - `prompt_bank`: **20 isimlendirilmiş veri seti (`dataset_name`)**, toplam **~8,250+ tekilleştirilmiş saldırı promptu** (`advbench_harmful_goals`, `aegis_safety`, `beavertails_unsafe`, `bipia_ipi`, `chatgpt_jailbreak_templates`, `cyberseceval_injections`, `deepset_injections`, `donotanswer`, `enterprise_security_suite`, `harmbench_functional`, `injecagent_ipi`, `jackhhao_jailbreaks`, `lakera_gandalf`, `pliny_l1b3rt4s`, `shomi28_injections`, `toxicchat_wild`, `xstest_prompts`, `master_red_team_100`, `master_red_team_1000`, `master_red_team_full`)
- **Ergonomik CLI Komut Standardı:**
  - `gort [config.yaml] <dataset_name|file.csv>` $\rightarrow$ Sadece seçilen veri setini koşturur (Attacker LLM devreye girmez).
  - `gort [config.yaml] all` $\rightarrow$ SQLite `prompt_bank` içindeki tüm veri setlerini sırayla koşturur.
  - `gort config.yaml` $\rightarrow$ Veri seti verilmezse `config.yaml` içindeki aktif pluginleri listeler, kullanıcıdan `[y/N]` onayı alır ve Attacker LLM ile üretim yapar.
  - `gort ls` / `gort ls <numara|isim>` $\rightarrow$ Tüm veri setlerini listeler veya seçilen veri setinin promptlarını tablo olarak basar (`cmd/gort/inspector.go`).
  - `gort plugins` / `gort plugins <numara|id>` $\rightarrow$ 155 plugini listeler veya seçilen pluginin tehdit vektörünü, Judge kriterini ve `config.yaml` kullanım örneğini offline kart olarak basar (`--ai` ile canlı prompt üretir).
  - `gort pull` $\rightarrow$ Tüm uzak veri setlerini indirip SQLite `prompt_bank` ve `prompts/` klasörünü senkronize eder.
- **Derleme Durumu:** `make all` (`go fmt`, `go vet`, `go test`, `go build`) temiz (Yeşil).

---

## 2. Mimari Değişmezler (Architectural Invariants)
1. **Sıfır Hard-Coded Katalog — SQLite Single Source of Truth (ADR-009):**
   - Go kaynak dosyalarında sabit plugin dizileri, URL listeleri veya prompt metinleri tutulmaz. Tüm katalog ve prompt havuzu `internal/store/schema.sql` ve `seed_catalog.sql` üzerinden `.gort/gort.db` (SQLite) içinde yaşar.
   - Açılışta `EnsureBootstrapped` eksik veri setlerini otomatik indirir ve kullanıcıya veritabanı hazır özetini (`[gort] SQLite DB Hazır: 155 plugin | 14 koleksiyon | 20 veri seti`) sunar.
2. **Marka/Üçüncü Parti Bağımsız `gort:*` İsim Alanı (ADR-005):**
   - Hiçbir yerde müşteri ismi kullanılmaz. Tüm eklenti, koleksiyon, strateji ve grader kimlikleri `gort:plugin:*`, `gort:collection:*`, `gort:strategy:*`, `gort:grader:*` isim alanındadır.
3. **Koşullu Attacker LLM & İsimlendirilmiş `prompt_bank` (ADR-007, ADR-009):**
   - Attacker LLM **sadece** CLI'da veri seti argümanı verilmediğinde (`gort config.yaml`) ve kullanıcı `[y/N]` onayı verdiğinde devreye girer.
4. **OpenTelemetry (OTel) GenAI Uyumlu SQLite Depolama (ADR-008, ADR-014):**
   - Tüm değerlendirme sonuçları, W3C `TraceID`/`SpanID` ve `gen_ai.*` / `gort.*` semantik öznitelikleriyle harici bağımlılık olmadan doğrudan SQLite (`.gort/gort.db`) içinde saklanır. PostgreSQL iptal edilmiş olup tek kaynak SQLite'tır.
5. **Sıfır CGO & Atomik Fonksiyonlar (ADR-001, ADR-002):**
   - Binary her zaman `CGO_ENABLED=0` (`modernc.org/sqlite`) ile derlenebilir olmalıdır.
   - Her fonksiyon **tek bir atomik iş** yapmalı ve **10–15 satırı** aşmamalıdır.
6. **%100 İngilizce Uygulama Dili (ADR-011):**
   - Kullanıcıyla iletişim Türkçe olsa da, `gort` binary'sinin tüm CLI çıktıları, tabloları, hata mesajları, Go yorumları ve `config.yaml` içeriği **%100 İngilizce** olmalıdır.

---

## 3. Paket Haritası & Durumları
| Paket | Sorumluluk | Durum |
| :--- | :--- | :--- |
| `cmd/gort` | Ergonomik CLI (`main.go`, `cli.go`, `inspector.go`, `cli_test.go`) | Tamamlandı ✅ |
| `internal/domain` | Saf veri modelleri (`domain.go`, `telemetry.go`, `config.go`) | Tamamlandı ✅ |
| `internal/config` | `config.yaml` okuyucu, `${ENV}` çözümleyici ve doğrulayıcı | Tamamlandı ✅ |
| `internal/store` | CGO'suz SQLite: Katalog, `prompt_bank`, `eval_runs`, `eval_results`, `otel_spans` | Tamamlandı ✅ |
| `internal/plugin` | SQLite tabanlı 155 plugin, 14 koleksiyon ve Attacker LLM `Synthesizer` | Tamamlandı ✅ |
| `internal/dataset` | Otomatik Bootstrap, HF/GitHub/OWASP indirici, `prompt_bank` & CSV senkronizasyonu | Tamamlandı ✅ |
| `internal/template` | SSTI-korumalı şablonlama (`SafeRenderer`) ve `injectVar` çıkarımı | Tamamlandı ✅ |
| `internal/provider` | LocalAI (`:8090`), Ollama, vLLM, OpenAI, Webhook istemcileri ve RateLimiter (`Provider` arayüzü) | Tamamlandı ✅ (Faz 5 Agentic Decorator) |
| `internal/grader` | Çoklu dil `notContains`, `IsBasicRefusal` ve `RubricGrader` Hakem LLM değerlendiricisi | Tamamlandı ✅ |
| `internal/telemetry` | W3C `TraceID`/`SpanID` ve OTel GenAI Semantik Conventions (`tracer.go`) | Tamamlandı ✅ |
| `internal/strategy` | Statik mutasyonlar ve multi-turn strateji sarmalayıcıları (`gort:strategy:*`) | Faz 5'te yazılacak ⏳ |
| `internal/engine` | Paralel OTel Evaluator ve Canlı ANSI akışı (`evaluator.go`) | Tamamlandı ✅ |
