# GO-RT Workspace Rules & Shared Brain Protocol (`agy` CLI + `antigravity-ide`)

Bu proje (`~/codes/go-rt`), kurumsal AI Red Teaming & Evaluation motorunun (`gort`) **Go 1.27+** ile sıfırdan, yüksek eşzamanlı, OpenTelemetry uyumlu ve tek-binary olarak gerçeklemesidir. Hem **Antigravity CLI (`agy`)** hem de **Antigravity IDE (`antigravity-ide`)** bu kurallara **istisnasız** uymak zorundadır.

---

## 🧠 1. Shared Brain Protokolü (Zorunlu Senkronizasyon)

Hem `agy` hem de `antigravity-ide` aynı kod tabanı üzerinde çalıştığından, oturumlar arası hafıza kaybını ve mimari sapmaları önlemek için `docs/brain/` dizini **Tek Doğruluk Kaynağı (Single Source of Truth)** olarak kullanılır:

1. **Göreve Başlamadan Önce:**
   - Mimari genel görünüm için [`docs/MASTER_PLAN.md`](docs/MASTER_PLAN.md) dosyasını referans al.
   - Güncel durum için [`docs/brain/CONTEXT.md`](docs/brain/CONTEXT.md) ve [`docs/brain/TASKS.md`](docs/brain/TASKS.md) dosyalarını oku.
   - Mimari kısıtlar için [`docs/brain/DECISIONS.md`](docs/brain/DECISIONS.md) (ADR) kayıtlarını kontrol et.
2. **Görevi Tamamladıktan Sonra:**
   - `make check` (`go fmt`, `go vet`, `go test ./...`) komutunu çalıştırıp tüm testlerin geçtiğini doğrula.
   - [`docs/brain/TASKS.md`](docs/brain/TASKS.md) içindeki tamamlanan maddeleri `[x]` olarak işaretle.
   - [`docs/brain/CONTEXT.md`](docs/brain/CONTEXT.md) içindeki aktif faz/paket durumunu güncelle.

---

## 🏷️ 2. İsimlendirme, Uygulama Dili (`ADR-011`), CLI ve SQLite Standartları

1. **%100 İngilizce Uygulama Dili (`ADR-011`):**
   - Kullanıcı ile iletişim Türkçe olsa bile, **`gort` uygulamasının kendisi %100 İngilizce** olmak zorundadır (tüm CLI çıktıları, tablo başlıkları, hata mesajları, interaktif sorular, Go kod yorumları, SQL şeması ve `config.yaml` yorumları).
2. **Tarafsız ve Kendi İsim Alanımız (`gort:*`):**
   - Hiçbir kodda, yorumda veya veri setinde müşteri ismi kullanılmaz.
   - Üçüncü parti araç önekleri yerine kendi `gort:plugin:*`, `gort:collection:*`, `gort:strategy:*`, `gort:grader:*` isimlendirmemiz ve `.gort/gort.db` SQLite veritabanımız kullanılır.
3. **Sıfır Hardcoded Katalog & SQLite `prompt_bank` (`ADR-009`):**
   - Tüm 155 plugin, 14 koleksiyon, 17 veri seti kaynağı ve 20 isimli prompt seti (`dataset_name` ayrımıyla) `.gort/gort.db` içindedir; Go dosyalarında hardcoded katalog dizisi tutulmaz.
4. **CLI Çalıştırma ve İnceleme Standartları (`ADR-010`):**
   - `gort [config.yaml] <dataset_name|file.csv>`: Sadece seçilen veri setini koşar (Attacker LLM devreye girmez).
   - `gort [config.yaml] all`: SQLite `prompt_bank` içindeki tüm veri setlerini sırayla koşar.
   - `gort config.yaml`: Veri seti verilmezse kullanıcıdan `[y/N]` onayı alarak aktif pluginlere göre Attacker LLM ile prompt üretir.
   - `gort ls [no|dataset_name]` ve `gort plugins [no|plugin_id]`: Veri setlerini ve pluginleri listeler veya tablo/kart olarak detaylarını gösterir.

---

## 🐹 3. Go 1.27+ Kodlama Stili & Clean Architecture Kuralları

1. **Atomik Fonksiyonlar (10–15 Satır Kuralı):**
   - Her fonksiyon **yalnızca tek bir iş** yapmalı ve **10–15 satırı** aşmamalıdır.
   - Bir fonksiyonun içinde sıralı adımları anlatan yorumlar gerekiyorsa, o adımları açıklayıcı isimlere sahip `private` yardımcı fonksiyonlara böl.
2. **Sıfır Tekrar (DRY) & Saf Fonksiyonlar (Pure Functions):**
   - İş mantığı yan etkisiz (side-effect-free) ve deterministik olmalıdır.
   - `internal/domain` paketi hiçbir dış kütüphaneye bağımlı olamaz.
3. **Kesin Hata Yönetimi (Zero Panics):**
   - İş mantığında **asla** `panic` kullanma. Tüm hataları bağlam ekleyerek sarmala: `fmt.Errorf("render template %s: %w", id, err)`.
4. **Yapılandırılmış Eşzamanlılık (Structured Concurrency):**
   - Her I/O fonksiyonu ilk parametre olarak `ctx context.Context` alır.
   - Paralel işlemler için `golang.org/x/sync/errgroup` veya kontrollü worker pool kullan.
