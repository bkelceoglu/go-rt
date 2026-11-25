package config

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bkelceoglu/go-rt/internal/domain"
)

// ProviderReachabilityResult holds health and latency info for a single provider.
type ProviderReachabilityResult struct {
	Role       string `json:"role"`
	ID         string `json:"id"`
	BaseURL    string `json:"baseUrl"`
	Reachable  bool   `json:"reachable"`
	StatusCode int    `json:"statusCode,omitzero"`
	LatencyMs  int64  `json:"latencyMs"`
	Message    string `json:"message"`
}

// CheckProvidersReachability performs non-blocking pre-flight reachability checks on all configured providers.
func CheckProvidersReachability(ctx context.Context, cfg domain.Config) []ProviderReachabilityResult {
	results := []ProviderReachabilityResult{}
	for i, t := range cfg.Targets {
		results = append(results, checkSingleProvider(ctx, fmt.Sprintf("target[%d]", i), t))
	}
	if strings.TrimSpace(cfg.Attacker.ID) != "" {
		results = append(results, checkSingleProvider(ctx, "attacker", cfg.Attacker))
	}
	return appendJudgeReachability(ctx, results, cfg.Judge)
}

// FirstUnreachableProvider returns the first unreachable provider result and true if any endpoint is offline.
func FirstUnreachableProvider(results []ProviderReachabilityResult) (ProviderReachabilityResult, bool) {
	for _, r := range results {
		if !r.Reachable {
			return r, true
		}
	}
	return ProviderReachabilityResult{}, false
}

func appendJudgeReachability(ctx context.Context, results []ProviderReachabilityResult, judge domain.JudgeSpec) []ProviderReachabilityResult {
	if strings.TrimSpace(judge.ID) == "" {
		return results
	}
	jSpec := domain.ProviderSpec{ID: judge.ID, BaseURL: judge.BaseURL, APIKey: judge.APIKey}
	return append(results, checkSingleProvider(ctx, "judge", jSpec))
}

func checkSingleProvider(ctx context.Context, role string, spec domain.ProviderSpec) ProviderReachabilityResult {
	res := ProviderReachabilityResult{
		Role:    role,
		ID:      spec.ID,
		BaseURL: spec.BaseURL,
	}
	if strings.TrimSpace(spec.ID) == "" {
		res.Message = "Provider ID is empty"
		return res
	}
	res.BaseURL = resolveProbeURL(spec)
	return executeProbeRequest(ctx, spec, res)
}

func resolveProbeURL(spec domain.ProviderSpec) string {
	raw := strings.TrimRight(strings.TrimSpace(spec.BaseURL), "/")
	if raw == "" {
		raw = defaultBaseURLForID(spec.ID)
	}
	if strings.HasPrefix(spec.ID, "webhook:") || strings.HasPrefix(raw, "http") {
		return raw
	}
	return "http://localhost:8080"
}

func defaultBaseURLForID(id string) string {
	switch {
	case strings.HasPrefix(id, "openai"):
		return "https://api.openai.com/v1"
	case strings.HasPrefix(id, "localai"):
		return "http://localhost:8090/v1"
	case strings.HasPrefix(id, "vllm"):
		return "http://localhost:8000/v1"
	case strings.HasPrefix(id, "ollama"):
		return "http://localhost:11434"
	default:
		return "http://localhost:8080/v1"
	}
}

func executeProbeRequest(ctx context.Context, spec domain.ProviderSpec, res ProviderReachabilityResult) ProviderReachabilityResult {
	reqCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := buildProbeRequest(reqCtx, res.BaseURL, spec)
	if err != nil {
		res.Message = fmt.Sprintf("invalid url: %v", err)
		return res
	}
	return runTimedProbe(req, res)
}

func buildProbeRequest(ctx context.Context, probeURL string, spec domain.ProviderSpec) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, buildModelsOrHealthURL(probeURL), nil)
	if err != nil {
		return nil, err
	}
	setProbeHeaders(req, spec)
	return req, nil
}

func runTimedProbe(req *http.Request, res ProviderReachabilityResult) ProviderReachabilityResult {
	client := &http.Client{Timeout: 3 * time.Second}
	start := time.Now()
	resp, err := client.Do(req)
	res.LatencyMs = time.Since(start).Milliseconds()
	if err != nil {
		res.Message = parseProbeError(err)
		return res
	}
	defer resp.Body.Close()
	return evaluateProbeResponse(resp.StatusCode, res)
}

func buildModelsOrHealthURL(baseURL string) string {
	clean := strings.TrimRight(baseURL, "/")
	if strings.Contains(clean, "/v1") {
		return clean + "/models"
	}
	return clean + "/v1/models"
}

func setProbeHeaders(req *http.Request, spec domain.ProviderSpec) {
	key := spec.APIKey
	if key == "" && strings.HasPrefix(spec.ID, "openai") {
		key = os.Getenv("OPENAI_API_KEY")
	}
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+os.ExpandEnv(key))
	}
}

func evaluateProbeResponse(code int, res ProviderReachabilityResult) ProviderReachabilityResult {
	res.StatusCode = code
	switch {
	case code >= 200 && code < 300:
		res.Reachable = true
		res.Message = fmt.Sprintf("Online (HTTP %d, %dms)", code, res.LatencyMs)
	case code == http.StatusUnauthorized || code == http.StatusForbidden:
		res.Message = fmt.Sprintf("Authentication required (HTTP %d). Check API key.", code)
	case code == http.StatusNotFound || code == http.StatusMethodNotAllowed:
		res.Reachable = true
		res.Message = fmt.Sprintf("Server reachable (HTTP %d, endpoint ready)", code)
	default:
		res.Message = fmt.Sprintf("HTTP %d returned by server", code)
	}
	return res
}

func parseProbeError(err error) string {
	msg := err.Error()
	if strings.Contains(msg, "connection refused") {
		return "Connection refused. Is the LLM service running on this port?"
	}
	if strings.Contains(msg, "context deadline exceeded") || strings.Contains(msg, "Client.Timeout") {
		return "Connection timed out (3s). Service is unreachable."
	}
	return msg
}
