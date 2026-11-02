package config

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/caveman/go-rt/internal/domain"
)

// ProviderReachabilityResult holds health and latency info for a single provider.
type ProviderReachabilityResult struct {
	Role       string `json:"role"`
	ID         string `json:"id"`
	BaseURL    string `json:"baseUrl"`
	Reachable  bool   `json:"reachable"`
	StatusCode int    `json:"statusCode,omitempty"`
	LatencyMs  int64  `json:"latencyMs"`
	Message    string `json:"message"`
}

// CheckProvidersReachability performs non-blocking pre-flight reachability checks on all configured providers.
func CheckProvidersReachability(ctx context.Context, cfg domain.Config) []ProviderReachabilityResult {
	var results []ProviderReachabilityResult
	for i, t := range cfg.Targets {
		role := fmt.Sprintf("target[%d]", i)
		results = append(results, checkSingleProvider(ctx, role, t))
	}
	if strings.TrimSpace(cfg.Attacker.ID) != "" {
		results = append(results, checkSingleProvider(ctx, "attacker", cfg.Attacker))
	}
	if strings.TrimSpace(cfg.Judge.ID) != "" {
		jSpec := domain.ProviderSpec{
			ID:      cfg.Judge.ID,
			BaseURL: cfg.Judge.BaseURL,
			APIKey:  cfg.Judge.APIKey,
		}
		results = append(results, checkSingleProvider(ctx, "judge", jSpec))
	}
	return results
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
	probeURL := resolveProbeURL(spec)
	res.BaseURL = probeURL
	return executeProbeRequest(ctx, probeURL, spec, res)
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

func executeProbeRequest(ctx context.Context, probeURL string, spec domain.ProviderSpec, res ProviderReachabilityResult) ProviderReachabilityResult {
	reqCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, buildModelsOrHealthURL(probeURL), nil)
	if err != nil {
		res.Message = fmt.Sprintf("invalid URL: %v", err)
		return res
	}
	setProbeHeaders(req, spec)
	start := time.Now()
	resp, err := client.Do(req)
	res.LatencyMs = time.Since(start).Milliseconds()
	if err != nil {
		res.Message = parseProbeError(err)
		return res
	}
	defer resp.Body.Close()
	return evaluateProbeResponse(resp, res)
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

func evaluateProbeResponse(resp *http.Response, res ProviderReachabilityResult) ProviderReachabilityResult {
	res.StatusCode = resp.StatusCode
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		res.Reachable = true
		res.Message = fmt.Sprintf("Online (HTTP %d, %dms)", resp.StatusCode, res.LatencyMs)
		return res
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		res.Message = fmt.Sprintf("Authentication required (HTTP %d). Check API key.", resp.StatusCode)
		return res
	}
	// Many local engines return 404/405 for /models but the server is running
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed {
		res.Reachable = true
		res.Message = fmt.Sprintf("Server reachable (HTTP %d, endpoint ready)", resp.StatusCode)
		return res
	}
	res.Message = fmt.Sprintf("HTTP %d returned by server", resp.StatusCode)
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
