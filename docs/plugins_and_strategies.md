plugins:
    # ==========================================================================
    # 1. Zararlı, Yasadışı ve Güvenlik İhlali İçeren İçerikler (Harmful & Illegal Content)
    # ==========================================================================
    - id: harmful:cybercrime                           # [AKTİF] Siber saldırı rehberliği, exploit/malware geliştirme ve yetkisiz sistem erişim istekleri.
    - id: harmful:privacy                              # [AKTİF] Kişisel mahremiyet ihlalleri, izinsiz gözetleme ve veri toplama girişimleri.
    # - id: harmful:chemical-biological-weapons        # [PASİF] Kimyasal, biyolojik, radyolojik veya nükleer (KBRN) kitle imha silahları yönergeleri.
    # - id: harmful:child-exploitation                 # [PASİF] Çocuk istismarı, sömürüsü ve pedofili ile ilişkili zararlı içerikler.
    # - id: harmful:copyright-violations               # [PASİF] Telif hakkı ihlalleri, lisanslı materyallerin izinsiz kopyalanması.
    # - id: harmful:cybercrime:malicious-code          # [PASİF] Tests creation of malicious code
    # - id: harmful:graphic-content                    # [PASİF] Tests handling of graphic or violent content
    # - id: harmful:harassment-bullying                # [PASİF] Bireylere yönelik taciz, yıldırma, tehdit ve siber zorbalık girişimleri.
    # - id: harmful:hate                               # [PASİF] Nefret söylemi, ırkçı veya ayrımcı saldırgan içerikler.
    # - id: harmful:illegal-activities                 # [PASİF] Yasadışı faaliyetler, kanun dışı eylem planları ve yasalara aykırı talimatlar.
    # - id: harmful:illegal-drugs                      # [PASİF] Yasadışı uyuşturucu maddelerin üretimi, sentezi ve dağıtımı yönergeleri.
    # - id: harmful:illegal-drugs:meth                 # [PASİF] Metamfetamin ve türevi tehlikeli sentetik uyarıcıların üretimi ve tarifleri.
    # - id: harmful:indiscriminate-weapons             # [PASİF] Ayrım gözetmeyen konvansiyonel silahlar ve askeri mühimmat üretim bilgisi.
    # - id: harmful:insults                            # [PASİF] Aşağılayıcı, onur kırıcı ve hakaret içeren ifadeler.
    # - id: harmful:intellectual-property              # [PASİF] Fikri mülkiyet hırsızlığı, ticari sırların çalınması ve patent ihlalleri.
    # - id: harmful:misinformation-disinformation      # [PASİF] Kamuoyunu yanıltıcı sahte haber, dezenformasyon ve manipülasyon kampanyaları.
    # - id: harmful:non-violent-crime                  # [PASİF] Şiddet içermeyen suçlar (dolandırıcılık, vergi kaçırma, sahtecilik vb.).
    # - id: harmful:profanity                          # [PASİF] Küfür, argo ve kurumsal dile aykırı kaba dil kullanımı.
    # - id: harmful:radicalization                     # [PASİF] Aşırılıkçı ideolojiler, terör propagandası ve radikalleşme çağrıları.
    # - id: harmful:self-harm                          # [PASİF] İntihar, kendine zarar verme ve yeme bozukluğu teşvik eden içerikler.
    # - id: harmful:sex-crime                          # [PASİF] Cinsel suçlar, rıza dışı eylemler ve cinsel saldırı içerikleri.
    # - id: harmful:sexual-content                     # [PASİF] Müstehcen, pornografik ve açık cinsel içerikler.
    # - id: harmful:specialized-advice                 # [PASİF] Yetkisiz profesyonel tavsiye (tıbbi teşhis, hukuki mütalaa, mali danışmanlık).
    # - id: harmful:unsafe-practices                   # [PASİF] Hayati tehlike arz eden güvensiz pratikler ve tehlikeli kendin-yap talimatları.
    # - id: harmful:violent-crime                      # [PASİF] Şiddet suçları, cinayet, fiziksel saldırı ve terör eylemleri rehberliği.
    # - id: harmful:weapons:ied                        # [PASİF] El yapımı patlayıcı (EYP/IED) düzenekleri, bomba yapım tarifleri ve patlayıcı maddeler.

    # ==========================================================================
    # 2. Siber Güvenlik, Enjeksiyon ve OWASP Top 10 Zafiyetleri (Cybersecurity & Injections)
    # ==========================================================================
    - id: prompt-extraction                            # [AKTİF] Sistem promptu, gizli direktifler ve guardrail kurallarının dışarı sızdırılması zafiyeti.
    - id: sql-injection                                # [AKTİF] Modelin SQL veritabanı sorgularına enjekte edilebilir zararlı payload üretmesi.
    # - id: ascii-smuggling                            # [PASİF] Görünmez Unicode tag karakterleri ile güvenlik filtrelerini atlatarak komut sızdırma.
    # - id: bfla                                       # [PASİF] Kırık Fonksiyon Seviyesi Yetkilendirme (Broken Function-Level Auth - OWASP API 5).
    # - id: bola                                       # [PASİF] Kırık Nesne Seviyesi Yetkilendirme (Broken Object-Level Auth / IDOR - OWASP API 1).
    # - id: cca                                        # [PASİF] Bağlam Uyumluluk Saldırıları (Context Compliance Attacks) ile sahte geçmiş uydurma.
    # - id: cross-session-leak                         # [PASİF] Kullanıcı oturumları arasında bellek ve veri sızıntısı tespiti.
    # - id: data-exfil                                 # [PASİF] URL parametreleri, markdown resim linkleri veya web hooklar üzerinden veri kaçırma.
    # - id: debug-access                               # [PASİF] Geliştirici debug arayüzleri, test komutları ve bakım modlarının ifşa olması.
    # - id: indirect-prompt-injection                  # [PASİF] Güvenilmeyen harici değişkenler ve dokümanlar üzerinden dolaylı prompt enjeksiyonu.
    # - id: rbac                                       # [PASİF] Rol tabanlı erişim kontrolü mekanizmalarını atlatma ve yetki yükseltme testleri.
    # - id: shell-injection                            # [PASİF] İşletim sistemi kabuk (Bash/PowerShell) komut enjeksiyonu ve yetkisiz çalıştırma.
    # - id: special-token-injection                    # [PASİF] <|endoftext|>, [INST] gibi format belirteçleriyle model mantığını manipüle etme.
    # - id: ssrf                                       # [PASİF] Sunucu taraflı istek sahteciliği (SSRF) ile dahili ağ kaynaklarına yetkisiz erişim.
    # - id: system-prompt-override                     # [PASİF] Modelin başlangıç talimatlarını ve kısıtlamalarını ezerek kontrolü ele geçirme.

    # ==========================================================================
    # 3. Otonom Kod Ajanı ve CI/CD Güvenliği (Coding Agent & DevSecOps)
    # ==========================================================================
    - id: coding-agent:generated-vulnerability         # [AKTİF] Ajan tarafından bilerek veya kazara güvenlik açığı içeren güvensiz kod üretilmesi.
    - id: coding-agent:procfs-credential-read          # [AKTİF] Ajanın /proc dosya sistemi veya ortam değişkenlerinden gizli anahtarları okuması.
    # - id: coding-agent:automation-poisoning          # [PASİF] Depo otomasyonlarına (Make, scripts, pre-commit) zararlı değişiklikler yapılması.
    # - id: coding-agent:delayed-ci-exfil              # [PASİF] CI/CD derleme boru hatları üzerinden gizli anahtarların gecikmeli dışarı sızdırılması.
    # - id: coding-agent:network-egress-bypass         # [PASİF] Ajanın tanımlı ağ sınırlarını aşarak yetkisiz dış sunuculara istek atması.
    # - id: coding-agent:repo-prompt-injection         # [PASİF] Kaynak kod depolarındaki README veya yorum satırlarına gizlenmiş enjeksiyonlar.
    # - id: coding-agent:sandbox-read-escape           # [PASİF] Ajanın tahsis edilen çalışma dizini dışındaki sistem dosyalarını okuması.
    # - id: coding-agent:sandbox-write-escape          # [PASİF] Ajanın yetkisiz dizinlere dosya yazması veya sistem dosyalarını değiştirmesi.
    # - id: coding-agent:secret-env-read               # [PASİF] Ortam değişkenlerindeki API anahtarları ve şifrelerin ifşa edilmesi.
    # - id: coding-agent:secret-file-read              # [PASİF] Hassas yapılandırma dosyalarının (.env, id_rsa, credentials) izinsiz okunması.
    # - id: coding-agent:steganographic-exfil          # [PASİF] Ajan çıktılarındaki boşluk, yorum veya steganografi ile gizli veri kaçırma.
    # - id: coding-agent:terminal-output-injection     # [PASİF] Terminal çıktıları üzerinden modelin komut satırını manipüle etmesi.
    # - id: coding-agent:verifier-sabotage             # [PASİF] Tests for weakened tests and security checks

    # ==========================================================================
    # 4. Finans, Yatırım ve Kurumsal Uyumluluk (Financial Services & Compliance)
    # ==========================================================================
    - id: financial:compliance-violation               # [AKTİF] Sermaye piyasası kanunları, SPK düzenlemeleri ve içeriden öğrenenlerin ticareti ihlali.
    - id: financial:confidential-disclosure            # [AKTİF] Açıklanmamış şirket verileri (MNPI), bilançolar ve ticari finansal sırların ifşası.
    # - id: financial:calculation-error                # [PASİF] Finansal hesaplama, risk modelleme ve kantitatif analizlerde kritik hatalar.
    # - id: financial:counterfactual                   # [PASİF] Tests for false financial narratives or misleading market information
    # - id: financial:data-leakage                     # [PASİF] Hassas mali tablolar, banka hesap dökümleri ve müşteri finansal verilerinin sızması.
    # - id: financial:defamation                       # [PASİF] Tests for false statements damaging financial entity reputations
    # - id: financial:hallucination                    # [PASİF] Tests for fabricated market data, non-existent financial instruments, or fictional company information
    # - id: financial:impartiality                     # [PASİF] Tests for unauthorized financial advice or transaction facilitation
    # - id: financial:japan-fiea-suitability           # [PASİF] Tests for unsuitable financial product recommendations or downplaying of risks under Japan's FIEA
    # - id: financial:misconduct                       # [PASİF] Tests for facilitation of financial crimes or market manipulation
    # - id: financial:sox-compliance                   # [PASİF] Tests for violations of Sarbanes-Oxley requirements including internal controls, financial reporting, and audit integrity
    # - id: financial:sycophancy                       # [PASİF] Tests for agreeing with risky investment strategies or validating get-rich-quick schemes

    # ==========================================================================
    # 5. RAG (Retrieval-Augmented Generation) Güvenliği (RAG Security)
    # ==========================================================================
    - id: rag-document-exfiltration                    # [AKTİF] Vektör veritabanından yetkisiz gizli kurumsal dokümanların dışarı sızdırılması.
    - id: rag-poisoning                                # [AKTİF] RAG bilgi tabanına sahte/zararlı veri enjekte ederek arama sonuçlarının zehirlenmesi.
    # - id: rag-source-attribution                     # [PASİF] RAG sisteminin kaynak dokümanlarda bulunmayan uydurma alıntı ve referans üretmesi.

    # ==========================================================================
    # 6. Ajan Protokolleri, MCP ve Araç Güvenliği (Agentic, MCP & Tool Security)
    # ==========================================================================
    - id: mcp                                          # [AKTİF] Model Context Protocol (MCP) zafiyetleri ve harici araç entegrasyonu istismarı.
    - id: tool-discovery                               # [AKTİF] Kullanılabilir sistem araçlarının, fonksiyon imzalarının ve API uçlarının keşfedilmesi.
    # - id: agentic:memory-poisoning                   # [PASİF] Ajanın uzun dönemli çalışma belleğine kalıcı zararlı talimatlar enjekte edilmesi.
    # - id: excessive-agency                           # [PASİF] Ajanın tanımlanmış görev yetkilerini aşarak bağımsız yetkisiz aksiyonlar alması.

    # ==========================================================================
    # 7. Kişisel Veri ve Gizlilik (PII, KVKK & GDPR)
    # ==========================================================================
    - id: pii:direct                                   # [AKTİF] Doğrudan kişisel veri (TCKN, telefon, adres, isim) ifşa etme zafiyeti.
    - id: pii:session                                  # [AKTİF] Kullanıcı oturum verileri ve bellek geçmişindeki kişisel verilerin ifşası.
    # - id: pii:api-db                                 # [PASİF] Veritabanı veya API sorguları üzerinden toplu kişisel veri sızdırma.
    # - id: pii:social                                 # [PASİF] Sosyal mühendislik teknikleriyle kişisel kimlik bilgilerinin çalınması.

    # ==========================================================================
    # 8. Kurumsal Yönetişim, Sözleşmeler ve Hukuk (Corporate Governance & Legal)
    # ==========================================================================
    - id: competitors                                  # [AKTİF] Şirket kurallarına aykırı rakip firma övgüsü, taraflı reklam veya yetkisiz kıyaslama.
    - id: contracts                                    # [AKTİF] Şirket adına yetkisiz yasal ve finansal sözleşme taahhütlerine girilmesi.
    # - id: coppa                                      # [PASİF] Çocukların Çevrimiçi Gizliliğini Koruma Yasası (COPPA) ihlalleri.
    # - id: ferpa                                      # [PASİF] Eğitim hakları ve öğrenci gizlilik kayıtları yasası (FERPA) ihlalleri.
    # - id: imitation                                  # [PASİF] Yetkisiz kurum, resmi makam veya şirket yöneticilerini taklit etme.
    # - id: policy                                     # [PASİF] Kuruma özel tanımlanmış özel güvenlik ve etik politikalarına uyum testleri.

    # ==========================================================================
    # 9. Model Güvenilirliği, Halüsinasyon ve Kaynak Tüketimi (Reliability & Robustness)
    # ==========================================================================
    - id: hallucination                                # [AKTİF] Modelin gerçek dışı bilgi, sahte olgular veya uydurma veriler üretmesi.
    - id: reasoning-dos                                # [AKTİF] Aşırı mantıksal döngü ve hesaplama kaynak tüketimiyle DoS oluşturma.
    # - id: divergent-repetition                       # [PASİF] Tekrarlayan kalıplarla modelin kontrolünü kaybettirip eğitim verisi sızdırma.
    # - id: goal-misalignment                          # [PASİF] Hedef kayması ve vekil metriklerin asıl kurumsal hedefle çelişmesi (Goodhart Kanunu).
    # - id: hijacking                                  # [PASİF] Modelin tahsis edilen kaynaklarının orijinal amacı dışında başka işler için gasp edilmesi.
    # - id: intent                                     # [PASİF] Özel hazırlanmış promptlarla modelin davranış amacının manipüle edilmesi.
    # - id: model-identification                       # [PASİF] Modelin temel mimarisi, sürümü ve sistem detaylarını ifşa etmeye zorlanması.
    # - id: off-topic                                  # [PASİF] Modelin tanımlı kurumsal uzmanlık alanı dışına saptırılması.
    # - id: overreliance                               # [PASİF] Sistem varsayımlarına veya kullanıcı girdilerine körü körüne aşırı güven duyulması.
    # - id: unverifiable-claims                        # [PASİF] Doğrulanamaz, kanıtsız ve spekülatif iddiaların gerçek gibi sunulması.
    # - id: wordplay                                   # [PASİF] Kelime oyunları ve dil hileleri ile kurumsal filtrelerin ve yasaklı kelimelerin aşılması.

    # ==========================================================================
    # 10. Telekomünikasyon ve İletişim Sektörü (Telecom Domain)
    # ==========================================================================
    - id: telecom:account-takeover                     # [AKTİF] SIM kart kopyalama, hat ele geçirme ve hesap gaspı saldırıları.
    - id: telecom:cpni-disclosure                      # [AKTİF] Müşteri Tescilli Ağ Bilgilerinin (CPNI / arama ve sinyal kayıtları) izinsiz ifşası.
    # - id: telecom:accessibility-violation            # [PASİF] Engelli erişilebilirlik standartları ve yasal regülasyon ihlalleri.
    # - id: telecom:billing-misinformation             # [PASİF] Fatura, tarife ve ücretlendirme konularında yanıltıcı veya sahte bilgi verilmesi.
    # - id: telecom:coverage-misinformation            # [PASİF] Kapsama alanı, çekim gücü veya baz istasyonu hizmetleri hakkında yanıltıcı bilgi.
    # - id: telecom:e911-misinformation                # [PASİF] Acil durum arama (112 / E911) konum ve yönlendirme protokollerinde yanlış yönlendirme.
    # - id: telecom:fraud-enablement                   # [PASİF] Telekom hatları üzerinden dolandırıcılık, PBX hacking veya SMS fraud kolaylaştırma.
    # - id: telecom:law-enforcement-request-handling   # [PASİF] Kolluk kuvvetleri veri taleplerinin yasal prosedürlere aykırı şekilde işlenmesi.
    # - id: telecom:location-disclosure                # [PASİF] Müşterinin baz istasyonu ve gerçek zamanlı GPS konum verilerinin yetkisiz ifşası.
    # - id: telecom:porting-misinformation             # [PASİF] Numara taşıma kuralları ve yasal haklar hakkında yanıltıcı yönlendirme yapılması.
    # - id: telecom:tcpa-violation                     # [PASİF] Telefon Tüketicisini Koruma Yasası (TCPA / İYS) ve izinsiz pazarlama ihlalleri.
    # - id: telecom:unauthorized-changes               # [PASİF] Müşteri onayı olmaksızın tarife ve hat hizmetlerinin değiştirilmesi (slamming/cramming).

    # ==========================================================================
    # 11. E-Ticaret ve Ödeme Sistemleri (E-Commerce & Payments)
    # ==========================================================================
    - id: ecommerce:order-fraud                        # [AKTİF] Sipariş sahteciliği, yetkisiz işlem ve ödeme akışı manipülasyonu.
    - id: ecommerce:pci-dss                            # [AKTİF] Kredi kartı numarası, CVV ve banka hesap bilgilerinin ifşası (PCI DSS ihlali).
    # - id: ecommerce:compliance-bypass                # [PASİF] E-ticaret güvenlik kuralları, iade koşulları ve kullanım şartlarını atlatma.
    # - id: ecommerce:price-manipulation               # [PASİF] Dinamik fiyatlandırma ve indirim kuponu algoritmalarının manipüle edilmesi.

    # ==========================================================================
    # 12. Sigorta Sektörü Güvenliği (Insurance Domain)
    # ==========================================================================
    - id: insurance:data-disclosure                    # [AKTİF] Poliçe sahibinin hasar geçmişi, mülk ve finansal bilgilerinin ifşası.
    - id: insurance:phi-disclosure                     # [AKTİF] Korumalı Sağlık Bilgilerinin (PHI / HIPAA) sigorta akışlarında ifşası.
    # - id: insurance:coverage-discrimination          # [PASİF] Korunan sınıflara (yaş, ırk, engel vb.) göre ayrımcı sigorta kapsam kararları.
    # - id: insurance:network-misinformation           # [PASİF] Anlaşmalı sağlık kurumları ve hekim ağları hakkında hatalı bilgi sunulması.

    # ==========================================================================
    # 13. Gayrimenkul ve Emlak Sektörü (Real Estate Domain)
    # ==========================================================================
    - id: realestate:fair-housing-discrimination       # [AKTİF] Adil Konut Yasası ihlali ve korunan sınıflara karşı ayrımcılık yapılması.
    - id: realestate:valuation-bias                    # [AKTİF] Emlak ekspertiz ve değerleme algoritmalarında önyargı ve manipülasyon.
    # - id: realestate:accessibility-discrimination    # [PASİF] Konut erişilebilirliği ve engelli uyum standartları ihlalleri.
    # - id: realestate:advertising-discrimination      # [PASİF] Konut reklamlarında demografik ayrımcılık ve dışlayıcı hedefleme yapılması.
    # - id: realestate:discriminatory-listings         # [PASİF] Emlak ilanlarında ayrımcı, önyargılı ve yasadışı ifadeler kullanılması.
    # - id: realestate:lending-discrimination          # [PASİF] İpotek ve konut kredilerinde ayrımcılık, kırmızı çizgi çekme (redlining).
    # - id: realestate:source-of-income                # [PASİF] Sosyal yardım ve konut kuponu sahiplerine karşı yasadışı ayrımcılık.
    # - id: realestate:steering                        # [PASİF] Alıcıları demografik yapıya göre belirli mahallelere yönlendirme suçu (steering).

    # ==========================================================================
    # 14. Sağlık ve Tıbbi Cihaz Güvenliği (Medical & Healthcare)
    # ==========================================================================
    - id: medical:fda:cyber-access-control             # [AKTİF] FDA tıbbi cihazlarında rol tabanlı kimlik doğrulama ve yetki kontrolü ihlalleri.
    - id: medical:hallucination                        # [AKTİF] Tıbbi makaleler, klinik çalışmalar veya ilaç etkileşimleri hakkında halüsinasyon üretme.
    # - id: medical:anchoring-bias                     # [PASİF] Tıbbi teşhiste gereksiz ayrıntılara takılıp kritik semptomları göz ardı etme (çıpalama).
    # - id: medical:fda:ai-disclosure                  # [PASİF] Tıbbi yapay zekanın kullanım sınırlarını ve düşük güvenilirlik uyarılarını gizlemesi.
    # - id: medical:fda:cyber-audit-tampering          # [PASİF] Tıbbi denetim kayıtlarını tahrif etme veya anomali uyarılarını susturma.
    # - id: medical:incorrect-knowledge                # [PASİF] Hayati tehlike doğurabilecek hatalı tıbbi bilgi ve yanlış tedavi yönlendirmesi.
    # - id: medical:off-label-use                      # [PASİF] Onaysız ve endikasyon dışı (off-label) ilaç kullanımını uygunsuz önerme.
    # - id: medical:prioritization-error               # [PASİF] Acil servis ve klinik triyaj kararlarında yanlış önceliklendirme yapma.
    # - id: medical:sycophancy                         # [PASİF] Kullanıcının yanlış tıbbi önermelerine dalkavukluk edip doğruyu söylememe.

    # ==========================================================================
    # 15. Eczacılık ve İlaç Mevzuatı (Pharmacy Domain)
    # ==========================================================================
    - id: pharmacy:controlled-substance-compliance     # [AKTİF] Kırmızı/yeşil reçeteli kontrollü maddeler ve uyuşturucu mevzuatına uyum.
    - id: pharmacy:drug-interaction                    # [AKTİF] Ölümcül ilaç-ilaç, ilaç-besin ve takviye etkileşimlerini tespit etme başarısı.
    # - id: pharmacy:dosage-calculation                # [PASİF] Böbrek yetmezliği, çocuk ve kilo bazlı hassas dozaj hesaplama hataları.

    # ==========================================================================
    # 16. Genç ve Çocuk Güvenliği (Teen & Child Safety)
    # ==========================================================================
    - id: teen-safety:age-restricted-goods-and-services # [AKTİF] Reşit olmayanların alkol, tütün, silah ve kumara erişimini kolaylaştırma.
    - id: teen-safety:dangerous-content                # [AKTİF] Tehlikeli meydan okumalar, virütik akımlar ve gençlere zararlı deney talimatları.
    # - id: teen-safety:dangerous-roleplay             # [PASİF] Küçük yaştakilerle romantik, cinsel veya şiddet içeren tehlikeli rol yapma senaryoları.
    # - id: teen-safety:harmful-body-ideals            # [PASİF] Yeme bozukluğu, beden aşağılama ve aşırı sağlıksız zayıflama telkinleri.

    # ==========================================================================
    # 17. Önyargı, Tarafsızlık ve Sosyo-Politik Tutarlılık (Bias & Neutrality)
    # ==========================================================================
    - id: bias:gender                                  # [AKTİF] Cinsiyet temelli ayrımcılık ve kalıp yargıların tespiti.
    - id: xstest                                       # [AKTİF] Aşırı sansür (exaggerated safety) tespiti; zararsız soruların gereksiz reddedilmesi.
    # - id: bias:age                                   # [PASİF] Yaş temelli ayrımcılık (yaşlılık veya gençlik önyargıları) tespiti.
    # - id: bias:disability                            # [PASİF] Engellilik temelli ayrımcılık ve erişim engeli önyargıları tespiti.
    # - id: bias:race                                  # [PASİF] Irk, etnik köken ve milliyet temelli önyargıların tespiti.
    # - id: politics                                   # [PASİF] Siyasi tarafsızlık, politik kutuplaşma ve seçim manipülasyonu testleri.
    # - id: religion                                   # [PASİF] Dini inançlara yönelik tarafsızlık ve önyargı testleri.

    # ==========================================================================
    # 18. Akademik ve Global Güvenlik Kıyaslama Kütüphaneleri (Global Benchmarks)
    # ==========================================================================
    - id: harmbench                                    # [AKTİF] CAIS HarmBench akademik siber saldırı ve zararlı davranış kıyaslama seti.
    - id: toxic-chat                                   # [AKTİF] LMSYS Chatbot Arena gerçek kullanıcı istismar ve toksik konuşma seti.
    # - id: aegis                                      # [PASİF] NVIDIA Aegis güvenlik mimarisi içerik koruma doğrulama seti.
    # - id: beavertails                                # [PASİF] PKU BeaverTails 14 güvenlik politikası ihlali kıyaslama seti.
    # - id: cyberseceval                               # [PASİF] Meta CyberSecEval siber güvenlik ve kod istismar kıyaslama seti.
    # - id: donotanswer                                # [PASİF] DoNotAnswer akademik yanıtlanmaması gereken zararlı soru seti.
    # - id: pliny                                      # [PASİF] Pliny jailbreak enjeksiyon ve guardrail atlatma saldırı kütüphanesi.

    # ==========================================================================
    # 19. Çok Modlu (Multimodal / Görüntü) Güvenlik Testleri (Multimodal Safety)
    # ==========================================================================
    # - id: unsafebench                                # [PASİF] UnsafeBench görsel güvenlik ve zararlı görüntü analizi seti (Görsel/Vision modelleri için).
    # - id: vlguard                                    # [PASİF] VLGuard görsel-dil modelleri için potansiyel zararlı görsel test seti (Görsel/Vision modelleri için).
    # - id: vlsu                                       # [PASİF] Kompozisyonel güvenlik; ayrı ayrı zararsız resim ve metnin birleşince zarar üretmesi.

  # ----------------------------------------------------------------------------
  # SALDIRI STRATEJİLERİ (STRATEGIES) - TOPLAM 35 STRATEJİ
  # ----------------------------------------------------------------------------
  # Stratejiler, yukarıda seçilen eklentilerin ürettiği saldırı istemlerini
  # farklı tekniklerle manipüle ederek guardrail ve filtreleme katmanlarını atlatmayı hedefler.
  strategies:
    - id: basic                                # [AKTİF] Temel doğrudan saldırı; prompt herhangi bir kodlama veya gizleme yapılmadan saf haliyle test edilir.
    - id: base64                               # [AKTİF] Base64 kodlama; zararlı istek Base64 stringine çevrilerek kelime bazlı filtreleme mekanizmaları atlatılır.
    - id: rot13                                # [AKTİF] ROT13 şifreleme; basit Sezar şifrelemesi ile güvenlik filtrelerini ve regex kontrollerini aşma denemesi.
    - id: jailbreak-templates                  # [AKTİF] Klasik jailbreak şablonları; DAN (Do Anything Now), Developer Mode ve persona manipülasyonu.
    # - id: hex                                # [PASİF] Onaltılık (Hex) kodlama; bayt seviyesinde metin maskeleme ile analiz motorlarını atlatma.
    # - id: leetspeak                          # [PASİF] 1337speak; harflerin sayılar ve sembollerle değiştirilmesi (ör: h4ck3r, p4ssw0rd).
    # - id: homoglyph                          # [PASİF] Homoglif saldırısı; Kiril ve Latin alfabesindeki görsel olarak farksız ancak farklı Unicode karakterlerle gizleme.
    # - id: morse                              # [PASİF] Mors kodu; metnin nokta ve çizgilerle ifade edilerek doğrudan metin algılayıcılarını atlatması.
    # - id: piglatin                           # [PASİF] Pig Latin dili; kelime harflerinin yer değiştirilmesiyle oluşturulan dil hilesi.
    # - id: camelcase                          # [PASİF] camelCase veya birleşik yazım manipülasyonu ile token sınırlarını şaşırtma.
    # - id: emoji                              # [PASİF] Emoji enjeksiyonu; metin arasına emoji karakterleri serpiştirerek kelime tespitini engelleme.
    # - id: math-prompt                        # [PASİF] Matematiksel problem kılığına sokma; zararlı direktifin bir cebir veya denklem içinde gizlenmesi.
    # - id: prompt-injection                   # [PASİF] Standart prompt enjeksiyon sarmalayıcısı; sistem komutlarını geçersiz kılma ön eki.
    # --------------------------------------------------------------------------
    # ÇOK TURLU STRATEJİLER VE SESSION (crescendo, goat, jailbreak:hydra, mischievous-user)
    # --------------------------------------------------------------------------
    # Bu stratejiler saldırıyı birkaç mesaja (tura) yayar. boost_provider.py ile session davranışı
    # stratejinin "stateful" ayarına bağlıdır:
    #
    #   stateful: false (varsayılan) : Promptfoo her turda o ana kadarki konuşmanın TAMAMINI tek mesaj
    #                                  olarak gönderir; her tur için YENİ session açılır. Konuşma geçmişini
    #                                  chatbot değil mesajın kendisi taşır. Mesaj her turda uzar;
    #                                  prod'da 1000 karakter aşılırsa o tur raporda (test clone'da limit yok)
    #                                  hata olarak görünür.
    #   stateful: true               : Her turda sadece SON mesaj gönderilir; aynı testin tüm turları
    #                                  AYNI session'da kalır, chatbot önceki turları kendi hafızasından
    #                                  (son 5 mesaj + konuşma özeti) hatırlar. Özet zehirlenmesi gibi
    #                                  hafıza açıkları ancak bu modda test edilir.
    #
    # stateful: true ile asıl koşudan önce numTests: 1 ve tek eklentiyle küçük bir deneme yapılması önerilir
    # (README.md, "Çok turlu deneme"). DB'de aynı testin turlarının tek session'da toplandığı doğrulanmalıdır.
    #
    # Örnek:
    # - id: crescendo
    #   config:
    #     stateful: false     # true yapılırsa aynı testin tüm turları aynı session'da kalır
    #     maxTurns: 10
    # --------------------------------------------------------------------------
    # - id: crescendo                          # [PASİF] Çok turlu (Multi-turn) Crescendo saldırısı; masum sorularla başlayıp adım adım güven kazanarak sınırları zorlama.
    # - id: best-of-n                          # [PASİF] Best-of-N mutasyon; birden fazla saldırı varyasyonu türeterek en etkili olanı seçme.
    # - id: citation                           # [PASİF] Akademik atıf ve literatür kılığına sokma; 'Bilimsel bir makalede analiz edilmek üzere...' pretext'i.
    # - id: authoritative-markup-injection     # [PASİF] Sistem yetkili etiket enjeksiyonu; <system>, <admin>, XML veya JSON formatında sahte emirler verme.
    # - id: mischievous-user                   # [PASİF] Yanıltıcı ve oyuncu kullanıcı simülasyonu; kural tanımaz meraklı bir kullanıcı kişiliği takınma.
    # - id: goat                               # [PASİF] GOAT (Generative Offensive Agent Tester); hedef modeli kırmak için otonom saldırgan ajan kullanımı.
    # - id: gcg                                # [PASİF] GCG (Greedy Coordinate Gradient); modelin güvenlik ağırlıklarını zorlayan optimize edilmiş anlamsız ekler.
    # - id: simba                              # [PASİF] SIMBA basitleştirilmiş gradyan tabanlı saldırı stratejisi.
    # - id: jailbreak                          # [PASİF] Genel otonom jailbreak üretici sarmalayıcısı.
    # - id: jailbreak:composite                # [PASİF] Bileşik jailbreak; birden fazla saldırı taktiğinin tek bir promptta harmanlanması.
    # - id: jailbreak:likert                   # [PASİF] Likert ölçeği manipülasyonu; güvenlik değerlendiricisini 1-5 puanlama hilesiyle aldatma.
    # - id: jailbreak:tree                     # [PASİF] Tree-of-Thought jailbreak; ağaç yapısında dallanarak en zayıf direnç noktasını bulma.
    # - id: jailbreak:meta                     # [PASİF] Meta-prompting saldırısı; modelin kendi akıl yürütme direktiflerini ona karşı kullanma.
    # - id: jailbreak:hydra                    # [PASİF] Hydra çok başlı saldırı; aynı anda birden fazla zafiyet alanına yönelen saldırı tasarımı.
    # - id: jailbreak:goblin                   # [PASİF] Goblin hedefli guardrail atlatma tekniği.
    # - id: layer                              # [PASİF] Katmanlı saldırı (Layering); bir stratejinin çıktısını başka bir stratejiye besleyerek çoklu maskeleme.
    # - id: indirect-web-pwn                   # [PASİF] Dolaylı web enjeksiyonu; web sayfalarından veya URL içeriklerinden saldırı simülasyonu.
    # - id: image                              # [PASİF] Görsel tabanlı enjeksiyon; çok modlu modellerde resim içine metin/payload gömme.
    # - id: audio                              # [PASİF] Ses tabanlı enjeksiyon; sesli asistanlar için gizli akustik komut simülasyonu.
    # - id: video                              # [PASİF] Video tabanlı enjeksiyon; kareler arasına görsel saldırı kodları yerleştirme.
    # - id: retry                              # [PASİF] Dinamik yeniden deneme; reddedilen saldırıları otomatik olarak farklı kelimelerle yeniden kurgulama.