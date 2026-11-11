package config

import (
	"cmp"
	"errors"
	"fmt"
	"os"

	"github.com/caveman/go-rt/internal/domain"
	"gopkg.in/yaml.v3"
)

const (
	defaultPromptsDir     = "prompts"
	defaultPromptTemplate = "{{prompt}}"
	defaultInjectVar      = "prompt"
	defaultSQLitePath     = ".gort/gort.db"
	defaultMaxConcurrency = 4
)

// Load reads, expands environment variables, parses, and validates a config.yaml file.
func Load(path string) (domain.Config, error) {
	rawBytes, err := os.ReadFile(path)
	if err != nil {
		return domain.Config{}, fmt.Errorf("read config file %s: %w", path, err)
	}
	return ParseBytes(rawBytes)
}

// ParseBytes unmarshals YAML bytes with environment expansion and applies defaults.
func ParseBytes(raw []byte) (domain.Config, error) {
	expanded := os.ExpandEnv(string(raw))
	var cfg domain.Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return domain.Config{}, fmt.Errorf("parse config yaml: %w", err)
	}
	applyDefaults(&cfg)
	if err := validate(cfg); err != nil {
		return domain.Config{}, err
	}
	return cfg, nil
}

func applyDefaults(cfg *domain.Config) {
	cfg.PromptsDir = cmp.Or(cfg.PromptsDir, defaultPromptsDir)
	cfg.PromptTemplate = cmp.Or(cfg.PromptTemplate, defaultPromptTemplate)
	cfg.InjectVar = cmp.Or(cfg.InjectVar, defaultInjectVar)
	applyRuntimeDefaults(cfg)
}

func applyRuntimeDefaults(cfg *domain.Config) {
	if cfg.MaxConcurrency <= 0 {
		cfg.MaxConcurrency = defaultMaxConcurrency
	}
	cfg.Storage.SQLitePath = cmp.Or(cfg.Storage.SQLitePath, defaultSQLitePath)
	if cfg.Redteam.NumTestsPerPlugin <= 0 {
		cfg.Redteam.NumTestsPerPlugin = 5
	}
}

func validate(cfg domain.Config) error {
	if len(cfg.Targets) == 0 {
		return errors.New("config validation failed: at least one target provider is required")
	}
	for i, target := range cfg.Targets {
		if target.ID == "" {
			return fmt.Errorf("config validation failed: target[%d] is missing id", i)
		}
	}
	return nil
}
