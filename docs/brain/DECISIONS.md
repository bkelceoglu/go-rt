# GO-RT Shared Brain — Architecture Decision Records (ADR)

> **Kural:** Hem `agy` hem de `antigravity-ide` buradaki mimari kararlara uymak zorundadır. Yeni bir mimari karar alındığında buraya yeni bir `ADR-00X` maddesi eklenir.

---

## ADR-001: Dil ve Derleyici Standartları (Go 1.27+, Sıfır CGO)
- **Karar:** Proje Go 1.27+ standartlarıyla, `CGO_ENABLED=0` uyumlu olarak geliştirilecektir.
- **Gerekçe:** Harici C++ derleme sorunlarını ve Python/Pandas çalışma zamanı bağımlılıklarını ortadan kaldırmak, <15 MB RAM tüketimi ve tek bir statik binary (`gort`) ile her ortamda anında çalışmasını sağlamak.

## ADR-002: Atomic Functions (10–15 Satır Sınırı) ve Pure Domain
- **Karar:** Fonksiyonlar tek bir görev yapmalı ve 10–15 satırı geçmemelidir. Sıralı adımlar yorum satırları yerine açıklayıcı `private` yardımcı fonksiyonlara bölünmelidir. `internal/domain` paketi hiçbir dış kütüphaneye bağımlı olmamalıdır.
- **Gerekçe:** Hem insan okuyucu hem de AI ajanları (`agy` ve `antigravity-ide`) için sıfır bilişsel yük, kolay birim testi ve kod stili bütünlüğü.

## ADR-003: Agentic Multi-Turn Stratejiler İçin Provider Decorator Pattern
- **Karar:** `gort:strategy:crescendo`, `gort:strategy:goat`, `gort:strategy:hydra` gibi çok turlu stratejiler doğrudan `domain.Provider` arayüzünü implemente eden ve gerçek hedef `domain.Provider`'ı sarmalayan (wrap eden) Decorator struct'ları olarak yazılacaktır.
- **Gerekçe:** Evaluator motorunun strateji detaylarını bilmesine gerek kalmaz (`Open/Closed Principle`). Her strateji kendi konuşma hafızasını, backtracking (geri sarma) ve unblocking mantığını izole olarak yönetir.

## ADR-004: İki Aşamalı SSTI-Korumalı Şablon Enjeksiyonu (`SafeRenderer`)
- **Karar:** Saldırı değişkeni (`injectVar`) şablon motoruna doğrudan değişken olarak verilmeyecek; önce şablon diğer değişkenlerle render edilip `injectVar` yerinde benzersiz bir boundary token bırakılacak, ardından `strings.ReplaceAll` ile ham saldırı metni yerleştirilecektir.
- **Gerekçe:** Saldırı payload'ları içinde yer alan `{{...}}` veya `{% ... %}` ifadelerinin `gort` şablon motorunu tetiklemesini (self-SSTI) %100 oranında engellemek.

## ADR-005: Yerel `gort:*` İsimlendirme Standardı ve Gömülü YAML Kataloğu (`//go:embed`)
- **Karar:** Müşteri veya üçüncü parti araç isimlendirmeleri kullanılmayacaktır. Tüm eklenti, strateji, grader ve telemetri kimlikleri kendi `gort:*` isim alanımızı (`gort:plugin:sql-injection`, `gort:strategy:crescendo`, `gort:grader:rubric`, `.gort/gort.db`) kullanacaktır.
- **Gerekçe:** Kendi bağımsız kurumsal ürün standardımızı oluşturmak.

## ADR-006: Paylaşımlı Beyin (Shared Brain) Senkronizasyonu
- **Karar:** `agy` (CLI) ve `antigravity-ide` kendi oturum loglarını kendi dizinlerinde tutsa da, projenin kalıcı beyni `~/codes/go-rt/docs/brain/` dizinidir. Her iki ajan da `AGENTS.md` kuralları gereği bu dizini okur ve günceller.

## ADR-007: Koşullu Prompt Besleme, Kullanıcı Onaylı Attacker LLM ve CLI Sözleşmesi
- **Karar:**
  1. **Veri Seti Dosyası Verildiğinde (`gort config.yaml prompts/hedehude.csv`):** Sadece argüman olarak verilen prompt veri seti dosyası kullanılır; Attacker LLM devreye **girmez**. Dosya veya `prompts/` klasörü yoksa otomatik oluşturulur/indirilir.
  2. **Veri Seti Dosyası Verilmediğinde (`gort config.yaml`):** Attacker LLM **sadece** CLI'da prompt dosyası argümanı verilmediğinde devreye girer. Devreye girmeden önce `config.yaml` içindeki `plugins` ID listesi (`gort:plugin:*`) ve üretilecek prompt sayısı ekrana listelenir ve kullanıcıdan onay (`[y/N]`) alınır; üretilen promptlar `prompts/` klasörüne kaydedilerek koşturulur.
- **Gerekçe:** Gereksiz LLM maliyetini/beklemesini önlemek ve kullanıcıya hangi plugin'lerle dinamik prompt üretileceğini şeffafça onaylatmak.

## ADR-008: OpenTelemetry (OTel) GenAI Uyumlu Çoklu Veritabanı Saklama Katmanı
- **Karar:** Tüm değerlendirme çalışmaları (`EvalRun`), test sonuçları (`EvalResult`), çok turlu konuşma geçmişleri ve model çağrıları **OpenTelemetry GenAI Semantic Conventions (`gen_ai.*` ve `gort.*`)** standardına uygun `TraceID` ve `SpanID` ile etiketlenecek; `domain.ResultStore` arayüzü üzerinden hem **SQLite** (`.gort/gort.db`), hem **PostgreSQL 18+** (`JSONB` & `vector`), hem de **OTLP Exporter** hedeflerine yazılabilecektir.
- **Gerekçe:** Hem taşınabilir yerel SQLite hem merkezi PostgreSQL analitiği hem de kurumsal OTel/APM sistemleriyle tam uyumluluk.

## ADR-009: Hard-Coded Veri Yerine SQLite Katalog & İsimlendirilmiş `prompt_bank` (`dataset_name` & `all`)
- **Karar:**
  1. Go kaynak dosyalarında hard-coded plugin listesi, koleksiyon haritası, veri seti URL'leri veya sabit prompt dizileri tutulmaz. Tüm bu katalog verileri `internal/store/schema.sql` ve `seed_catalog.sql` ile `.gort/gort.db` (SQLite) içindeki `plugins`, `plugin_collections`, `dataset_sources` ve `prompt_bank` tablolarında saklanır.
  2. `prompt_bank` tablosunda her prompt seti `dataset_name` (örn. `lakera_gandalf`, `harmbench_functional`, `enterprise_security_suite`, `hedehude`) sütunuyla ayrıştırılır.
  3. Kullanıcı `gort [config.yaml] <dataset_name>` (veya `prompts/hedehude.csv`) dediğinde sadece o isimli set koşturulur; `gort [config.yaml] all` dediğinde `prompt_bank` içindeki tüm veri setleri sırayla tek tek koşturulur.
- **Gerekçe:** Go kodunu %100 saf motor mantığı olarak tutmak, yeniden derleme gerektirmeden veritabanından plugin/veri seti yönetebilmek ve isim bazlı (`<dataset_name>` veya `all`) esnek test koşumu sağlamak.

## ADR-010: Ergonomik CLI Alt Komutları, Tablo Görünümü (`gort ls 1`) ve Offline Plugin Kartları (`gort plugins 1`)
- **Karar:**
  1. CLI argüman çözümleyici (`cmd/gort/cli.go`) kısa alt komutları (`ls`, `plugins`, `pull`) ve opsiyonel `config.yaml` kullanımını (`gort lakera_gandalf` $\rightarrow$ otomatik `config.yaml`) destekler.
  2. `gort ls <numara|isim>` (örn. `gort ls 1` veya `gort ls advbench_harmful_goals`) seçilen veri setinin içindeki promptları formatlı terminal tablosu olarak basar (`cmd/gort/inspector.go`).
  3. `gort plugins <numara|id>` (örn. `gort plugins 1` veya `gort plugins gort:plugin:aegis`) SQLite `plugins` tablosundan anlık (0 ms, %100 offline) olarak plugin açıklamasını, tehdit vektörünü, Judge kriterini ve kopyalanabilir `config.yaml` kullanım bloğunu basar.
- **Gerekçe:** Güvenlik uzmanının terminalden ayrılmadan tüm veri setlerini ve 155 plugini sıfır gecikmeyle (ve offline/air-gapped ortamlarda bile) inceleyebilmesi.

## ADR-011: Uygulama Dili Standardı (%100 İngilizce Uygulama, Kod, Log ve Konfigürasyon)
- **Karar:** Geliştirici ile sohbet Türkçe yürütülse dahi, `gort` uygulamasının kendisi (tüm CLI çıktıları, tablo başlıkları, hata mesajları, interaktif onay soruları, Go yorum satırları, SQL şemaları ve `config.yaml` içeriği) **%100 İngilizce** olmak zorundadır.
- **Gerekçe:** Kurumsal uluslararası ürün standardı, OpenTelemetry / log analitiği uyumluluğu ve kod tabanı dil bütünlüğü.

## ADR-012: Sıfır Bağımlılıklı ANSI CLI ve Dış Sarmalayıcı (Full-Encapsulation) Web UI Mimarisi
- **Karar:**
  1. **ANSI CLI Çıktıları (`internal/ui`):** Harici kütüphane eklemeden Go standart kütüphanesiyle ANSI renklendirme (`BoldCyan`, `Badge`, `PadRightVisible`) sağlanır; pipe/yönlendirme veya `NO_COLOR` tespit edildiğinde otomatik olarak saf metne geri döner ve görünür karakter uzunluğuna göre tablo sütunlarını milimetrik hizalı tutar.
  2. **Web UI Zamanlaması ve Mimarisi (Faz 6):** Web UI, CLI motoru (Faz 2 $\rightarrow$ Faz 5) tamamen bitip test edildikten sonra **en son adım (Faz 6)** olarak eklenecektir. Temel motor paketlerinde (`internal/*`) hiçbir UI mantığı bulunmaz; Web UI (`internal/webui`), motoru ve `.gort/gort.db` veritabanını dışarıdan sarmalayan (**full-encapsulation**) bir katmandır ve tek `gort` binary'si içine `//go:embed` ile gömülecektir.
- **Gerekçe:** Clean Architecture prensibini korumak, önce çekirdek motoru CLI üzerinden kaya gibi sağlamlaştırmak, ardından arayüzü ek bir iş mantığı üretmeden saf bir sunum katmanı olarak bağlamak.

## ADR-013: Mikro-Adım (Micro-Step) Geliştirme ve Çoklu-Uç Nokta (OpenAI & Ollama) Sağlayıcı Standardı
- **Karar:**
  1. **Geliştirme Metodolojisi:** Kod tabanında büyük sıçramalar yerine, her biri tek sorumluluğa sahip mikro-adımlarla (Dosya $\rightarrow$ Birim Testi $\rightarrow$ `make all` $\rightarrow$ Git Commit) ilerlenecektir.
  2. **Çift Yönlü Sağlayıcı (Provider) Desteği:** `internal/provider` katmanı yalnızca yerel `ollama` değil, doğrudan uzaktaki resmi **OpenAI (`api.openai.com`)**, Azure OpenAI, uzak vLLM, Groq, LiteLLM ve DeepSeek API'lerini standart HTTP `v1/chat/completions` REST ve SSE protokolleriyle tam olarak destekleyecektir.
- **Gerekçe:** Kod kalitesinde ve mimarisinde model geçişlerinden kaynaklanabilecek olası sapmaları sıfıra indirmek ve hem yerel hem de kurumsal bulut AI uç noktalarını eşit güçte test edebilmek.

## ADR-014: Sıfır Bağımlılıklı GORM + Pure-Go SQLite Depolama Mimarisi (PostgreSQL & Ham SQL İptali)
- **Karar:** PostgreSQL 18+ gereksinimi ve ham `database/sql` sorguları (`schema.sql` / manuel `rows.Scan`) mimariden çıkarılmıştır. Tüm değerlendirme kayıtları (`eval_runs`, `eval_results`), prompt havuzu (`prompt_bank`), katalog (`plugins`, `plugin_collections`, `dataset_sources`) ve OpenTelemetry span'leri (`otel_spans`) **`gorm.io/gorm` + `github.com/glebarez/sqlite` (`modernc.org/sqlite` tabanlı, `CGO_ENABLED=0`)** üzerinden yerel `.gort/gort.db` içinde yönetilir. Tablo şemaları başlangıçta `db.AutoMigrate(...)` ile otomatik senkronize edilir ve JSON alanları `gorm:"serializer:json"` ile serileştirilir.
- **Gerekçe:** 27 sütunlu kırılgan `rows.Scan` kodlarını ve şema göçü (`ALTER TABLE`) hatalarını ortadan kaldırmak, <15 MB CGO'suz tekil binary taşınabilirliğini korumak ve Faz 6 Web UI analitik sorgularını tip güvenli hale getirmek.

## ADR-015: Dinamik Provider Protokolleri, Düzenlenebilir Endpoint'ler ve Pre-flight Erişim Doğrulaması
- **Karar:**
  1. Frontend ve backend kodlarında hardcoded model isimleri (`Qwen2.5-7B`, `llama3.1:8b`, `gpt-4o-mini` vb.) ve `~/.models` gibi keyfi yerel dizin varsayımları tutulmaz.
  2. Standart provider protokol şablonları (`ollama:chat:`, `llamacpp:chat:`, `vllm:chat:`, `localai:chat:`, `openai:chat:`, `webhook:`) ve bunların resmi varsayılan portları tanımlanır; kullanıcının Base URL alanını serbestçe düzenleyebilmesi (farklı port veya IP kullanımı) garanti altına alınır.
  3. Konfigürasyon doğrulama (`Validate`) ve kaydetme (`Save`) öncesinde `CheckProvidersReachability` mekanizması ile tüm yapılandırılmış hedeflere (`targets`, `attacker`, `judge`) hafif bir pre-flight ağ kontrolü (ping) yapılır; erişilemeyen veya yetkilendirme hatası (401) veren modeller hem CLI'da hem Web UI'da anında `Online` / `Unreachable` rozetleriyle raporlanır.
- **Gerekçe:** Testleri başlattıktan sonra yanlış port veya bağlantı reddi nedeniyle sürecin ortasında patlamasını önlemek ve kullanıcıya esnek, dinamik model yapılandırması sunmak.

## ADR-015: LocalAI (`:8090`) Varsayılan Yerel Çalışma Zamanı & İkili Model Dağılımı
- **Karar:** Yerel makinedeki Ollama kurulumu kaldırılmış olup, varsayılan yerel LLM çalışma zamanı `http://localhost:8090` (`~/localAI`, Intel GPU SYCL) üzerinde çalışan **LocalAI** olarak güncellenmiştir. `OpenAICompatibleProvider` içinde `localai:` / `localai:chat:` önekleri ve `LOCALAI_BASE_URL` ortam değişkeni birinci sınıf olarak desteklenir. Varsayılan `config.yaml` yapılandırmasında Target ve Attacker rolleri için `localai:chat:qwen3.8-9b-q4` (~5.8 GB), Judge rolü için `localai:chat:qwen3-4b` (~3.0 GB + DFlash speculative decoding) kullanılır (`maxConcurrency: 2`).
- **Gerekçe:** 30 GiB sistem belleğinde (~23 GiB kullanılabilir) iki modelin toplam bellek ayak izi (~9.5–10 GiB) rahatça aynı anda sığmakta ve bağımsız bir Judge LLM ile yüksek doğruluklu değerlendirme sağlamaktadır.

## ADR-016: Yerleşik Web Command Center & Sıfır Bağımlılıklı Vanilla JS Mimarisi (Faz 6)
- **Karar:** Web Command Center (`gort view [port]`), sıfır Node.js/npm dış bağımlılığıyla Vanilla JavaScript (modern ES Modules) ve Go 1.27 standart kütüphanesinin `//go:embed static` altyapısı kullanılarak doğrudan `./bin/gort` ikili dosyası içine gömülmüştür. Arayüz; 5 adımlı interaktif Config Builder Wizard'ı, SQLite Katalog ve Veri Seti Gezginini, Yönetici Güvenlik Panosunu (Executive Dashboard), Çok Turlu Transkript Müfettişini (Runs Inspector), Gerçek Zamanlı SSE Log Akış Konsolunu ve OWASP Top 10 uyumlu, yazdırılabilir (`@media print` / PDF) Denetim Raporu Motorunu içerir.
- **Gerekçe:** Dağıtımda harici web sunucusu, paket yöneticisi (npm) veya derleme aracı (webpack, vite) gereksinimini ortadan kaldırmak; tek bir statik Go binary'si ile hem terminal hem de kurumsal web arayüzünü anında çalıştırmak.


