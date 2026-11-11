package domain

// Config represents the root configuration loaded from config.yaml.
type Config struct {
	Version             string         `json:"version" yaml:"version"`
	Description         string         `json:"description" yaml:"description"`
	Purpose             string         `json:"purpose" yaml:"purpose"`
	PromptsDir          string         `json:"promptsDir" yaml:"promptsDir"`
	PromptTemplate      string         `json:"promptTemplate" yaml:"promptTemplate"`
	InjectVar           string         `json:"injectVar" yaml:"injectVar"`
	Datasets            []string       `json:"datasets,omitempty" yaml:"datasets,omitempty"`
	Targets             []ProviderSpec `json:"targets" yaml:"targets"`
	Attacker            ProviderSpec   `json:"attacker" yaml:"attacker"`
	Judge               JudgeSpec      `json:"judge" yaml:"judge"`
	Redteam             RedteamSpec    `json:"redteam" yaml:"redteam"`
	Strategies          []StrategySpec `json:"strategies" yaml:"strategies"`
	Storage             StorageSpec    `json:"storage" yaml:"storage"`
	MaxConcurrency      int            `json:"maxConcurrency" yaml:"maxConcurrency"`
	SequentialSharedLLM bool           `json:"sequentialSharedLlm,omitzero" yaml:"sequentialSharedLlm,omitempty"`
}

// ProviderSpec defines connection and inference settings for a Target, Attacker, or Judge provider.
type ProviderSpec struct {
	ID       string            `json:"id" yaml:"id"`
	Label    string            `json:"label,omitempty" yaml:"label,omitempty"`
	BaseURL  string            `json:"baseUrl,omitempty" yaml:"baseUrl,omitempty"`
	APIKey   string            `json:"apiKey,omitempty" yaml:"apiKey,omitempty"`
	Temp     float64           `json:"temperature,omitzero" yaml:"temperature,omitempty"`
	MaxToks  int               `json:"maxTokens,omitzero" yaml:"maxTokens,omitempty"`
	Think    *bool             `json:"think,omitempty" yaml:"think,omitempty"`
	Headers  map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	BodyTmpl string            `json:"bodyTemplate,omitempty" yaml:"bodyTemplate,omitempty"`
	RespPath string            `json:"responsePath,omitempty" yaml:"responsePath,omitempty"`
}

// JudgeSpec configures the LLM-as-a-Judge and deterministic refusal rules.
type JudgeSpec struct {
	ID          string   `json:"id" yaml:"id"`
	BaseURL     string   `json:"baseUrl,omitempty" yaml:"baseUrl,omitempty"`
	APIKey      string   `json:"apiKey,omitempty" yaml:"apiKey,omitempty"`
	Temp        float64  `json:"temperature,omitzero" yaml:"temperature,omitempty"`
	MaxToks     int      `json:"maxTokens,omitzero" yaml:"maxTokens,omitempty"`
	Think       *bool    `json:"think,omitempty" yaml:"think,omitempty"`
	Rubric      string   `json:"rubric" yaml:"rubric"`
	NotContains []string `json:"notContains,omitempty" yaml:"notContains,omitempty"`
}

// RedteamSpec configures dynamic LLM plugin synthesis.
type RedteamSpec struct {
	NumTestsPerPlugin int          `json:"numTestsPerPlugin" yaml:"numTestsPerPlugin"`
	Languages         []string     `json:"languages,omitempty" yaml:"languages,omitempty"`
	Plugins           []PluginSpec `json:"plugins,omitempty" yaml:"plugins,omitempty"`
}

// PluginSpec identifies a single redteam plugin and its optional overrides.
type PluginSpec struct {
	ID       string            `json:"id" yaml:"id"`
	NumTests int               `json:"numTests,omitzero" yaml:"numTests,omitempty"`
	Severity Severity          `json:"severity,omitempty" yaml:"severity,omitempty"`
	Config   map[string]string `json:"config,omitempty" yaml:"config,omitempty"`
}

// StrategySpec identifies an attack mutation or multi-turn strategy.
type StrategySpec struct {
	ID            string   `json:"id" yaml:"id"`
	Steps         []string `json:"steps,omitempty" yaml:"steps,omitempty"`
	MaxTurns      int      `json:"maxTurns,omitzero" yaml:"maxTurns,omitempty"`
	MaxBacktracks int      `json:"maxBacktracks,omitzero" yaml:"maxBacktracks,omitempty"`
}

// StorageSpec configures OpenTelemetry-compatible SQLite, PostgreSQL, and OTLP persistence.
type StorageSpec struct {
	SQLitePath      string `json:"sqlitePath" yaml:"sqlitePath"`
	PostgresDSNEnv  string `json:"postgresDsnEnv,omitempty" yaml:"postgresDsnEnv,omitempty"`
	OTLPEndpointEnv string `json:"otlpEndpointEnv,omitempty" yaml:"otlpEndpointEnv,omitempty"`
}
